package api_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"fmv-game/backend/internal/api"
	"fmv-game/backend/internal/store"
)

const adminTok = "dev-admin"

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	st, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("open mem: %v", err)
	}
	ts := httptest.NewServer(api.New(st, "media").Handler())
	t.Cleanup(func() { ts.Close(); st.Close() })
	return ts
}

// do gửi request admin; auth=true → kèm X-Admin-Token. Trả status + body decode.
func do(t *testing.T, ts *httptest.Server, method, path string, body any, auth bool) (int, map[string]any) {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, ts.URL+path, rdr)
	if err != nil {
		t.Fatalf("new req: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if auth {
		req.Header.Set("X-Admin-Token", adminTok)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer res.Body.Close()
	var out map[string]any
	raw, _ := io.ReadAll(res.Body)
	_ = json.Unmarshal(raw, &out)
	return res.StatusCode, out
}

func TestAdminAuthRequired(t *testing.T) {
	ts := newTestServer(t)
	if code, _ := do(t, ts, "GET", "/api/admin/models", nil, false); code != http.StatusUnauthorized {
		t.Fatalf("không token → muốn 401, được %d", code)
	}
	if code, _ := do(t, ts, "GET", "/api/admin/models", nil, true); code != http.StatusOK {
		t.Fatalf("có token → muốn 200, được %d", code)
	}
}

func TestAdminModelChapterVideoCRUD(t *testing.T) {
	ts := newTestServer(t)

	// create model
	code, body := do(t, ts, "POST", "/api/admin/models",
		map[string]any{"code": "hana", "displayName": "Hana", "age": 24}, true)
	if code != http.StatusCreated {
		t.Fatalf("create model: %d %v", code, body)
	}
	modelID := int64(body["id"].(float64))

	// create chapter
	code, body = do(t, ts, "POST", "/api/admin/models/"+itoa(modelID)+"/chapters",
		map[string]any{"idx": 1, "title": "Chapter 1", "isFree": true}, true)
	if code != http.StatusCreated {
		t.Fatalf("create chapter: %d %v", code, body)
	}
	chID := int64(body["id"].(float64))

	// create 2 videos
	_, b1 := do(t, ts, "POST", "/api/admin/chapters/"+itoa(chID)+"/videos",
		map[string]any{"code": "v1", "idx": 0, "title": "V1"}, true)
	_, b2 := do(t, ts, "POST", "/api/admin/chapters/"+itoa(chID)+"/videos",
		map[string]any{"code": "v2", "idx": 1, "title": "V2"}, true)
	v1 := int64(b1["id"].(float64))
	v2 := int64(b2["id"].(float64))

	// reorder → v2 trước v1
	if c, _ := do(t, ts, "POST", "/api/admin/chapters/"+itoa(chID)+"/videos/reorder",
		map[string]any{"ids": []int64{v2, v1}}, true); c != http.StatusOK {
		t.Fatalf("reorder: %d", c)
	}
	_, lst := do(t, ts, "GET", "/api/admin/chapters/"+itoa(chID)+"/videos", nil, true)
	vids := lst["videos"].([]any)
	if first := vids[0].(map[string]any); first["code"] != "v2" {
		t.Fatalf("reorder không áp dụng: %v", vids)
	}

	// publish
	if c, _ := do(t, ts, "POST", "/api/admin/models/"+itoa(modelID)+"/publish", nil, true); c != http.StatusOK {
		t.Fatalf("publish: %d", c)
	}
}

func TestAdminContentImportAndFlow(t *testing.T) {
	ts := newTestServer(t)

	sf := map[string]any{
		"model": map[string]any{"code": "imp", "display_name": "Imp"},
		"story": map[string]any{"slug": "imp-s1", "title": "Imp S1"},
		"characters": []any{map[string]any{"code": "a", "display_name": "A"}},
		"media": []any{map[string]any{"key": "m1", "kind": "video", "file": "/media/a.mp4", "duration_ms": 1000}},
		"chapters": []any{map[string]any{
			"idx": 1, "title": "C1", "is_free": true, "entry": "s1",
			"videos": []any{
				map[string]any{"code": "v1", "idx": 0, "title": "V1", "media": "m1"},
				map[string]any{"code": "v2", "idx": 1, "title": "V2", "media": "m1"},
			},
			"scenes": []any{
				map[string]any{"code": "s1", "type": "linear", "video": "v1", "next": "s2"},
				map[string]any{"code": "s2", "type": "choice", "video": "v2", "choices": []any{
					map[string]any{"label": "L", "next": "s_end"},
					map[string]any{"label": "R", "next": "s_end"},
				}},
				map[string]any{"code": "s_end", "type": "ending", "video": "v1"},
			},
		}},
		"endings": []any{map[string]any{"scene": "s_end", "code": "e1", "title": "End", "rank": "good"}},
	}

	if c, b := do(t, ts, "PUT", "/api/admin/models/0/content", sf, true); c != http.StatusOK {
		t.Fatalf("import content: %d %v", c, b)
	}

	// tìm model "imp" → chapter → flow
	_, ml := do(t, ts, "GET", "/api/admin/models", nil, true)
	var modelID int64
	for _, m := range ml["models"].([]any) {
		mm := m.(map[string]any)
		if mm["code"] == "imp" {
			modelID = int64(mm["id"].(float64))
		}
	}
	if modelID == 0 {
		t.Fatal("không thấy model imp sau import")
	}
	_, chs := do(t, ts, "GET", "/api/admin/models/"+itoa(modelID)+"/chapters", nil, true)
	chID := int64(chs["chapters"].([]any)[0].(map[string]any)["id"].(float64))

	code, flow := do(t, ts, "GET", "/api/admin/chapters/"+itoa(chID)+"/flow", nil, true)
	if code != http.StatusOK {
		t.Fatalf("flow: %d %v", code, flow)
	}
	if flow["entry"] != "s1" {
		t.Fatalf("entry sai: %v", flow["entry"])
	}
	if n := len(flow["nodes"].([]any)); n != 3 {
		t.Fatalf("muốn 3 node, được %d", n)
	}
	if e := len(flow["edges"].([]any)); e != 3 { // s1→s2, s2→s_end (x2)
		t.Fatalf("muốn 3 edge, được %d", e)
	}
}

func itoa(i int64) string {
	return strconv.FormatInt(i, 10)
}
