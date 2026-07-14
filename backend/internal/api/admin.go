// Admin CRUD (Phase 2) — quản lý models/chapters/videos + import flow + publish.
// Bảo vệ bằng admin token (DEV stub). Prod: thay bằng RBAC/JWT có scope admin.
package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"

	"fmv-game/backend/internal/director"
	"fmv-game/backend/internal/seed"
	"fmv-game/backend/internal/store"
)

// ===== admin auth (DEV stub) =====

func (s *Server) adminToken() string {
	if t := os.Getenv("ADMIN_TOKEN"); t != "" {
		return t
	}
	return "dev-admin" // DEV fallback — prod PHẢI set ADMIN_TOKEN.
}

func (s *Server) adminAuth(h http.HandlerFunc) http.HandlerFunc {
	want := s.adminToken()
	return func(w http.ResponseWriter, r *http.Request) {
		tok := r.Header.Get("X-Admin-Token")
		if tok == "" {
			if a := r.Header.Get("Authorization"); strings.HasPrefix(a, "Bearer ") {
				tok = strings.TrimPrefix(a, "Bearer ")
			}
		}
		if tok == "" {
			tok = r.URL.Query().Get("admin_token")
		}
		if tok != want {
			writeErr(w, &director.Error{Status: http.StatusUnauthorized, Code: "ADMIN_UNAUTHORIZED", Msg: "thiếu/sai admin token"})
			return
		}
		h(w, r)
	}
}

// ===== DTOs (camelCase sạch — không lộ sql.Null*) =====

type modelDTO struct {
	ID           int64  `json:"id"`
	Code         string `json:"code"`
	DisplayName  string `json:"displayName"`
	Avatar       string `json:"avatar,omitempty"`
	Age          int    `json:"age,omitempty"`
	Birthday     string `json:"birthday,omitempty"`
	Relationship string `json:"relationship,omitempty"`
	Occupation   string `json:"occupation,omitempty"`
	HeightCm     int    `json:"heightCm,omitempty"`
	WeightKg     int    `json:"weightKg,omitempty"`
	Family       string `json:"family,omitempty"`
	Bio          string `json:"bio,omitempty"`
}

func toModelDTO(m *store.Model) modelDTO {
	return modelDTO{
		ID: m.ID, Code: m.Code, DisplayName: m.DisplayName,
		Avatar: m.Avatar.String, Age: int(m.Age.Int64), Birthday: m.Birthday.String,
		Relationship: m.Relationship.String, Occupation: m.Occupation.String,
		HeightCm: int(m.HeightCm.Int64), WeightKg: int(m.WeightKg.Int64),
		Family: m.Family.String, Bio: m.Bio.String,
	}
}

type chapterDTO struct {
	ID         int64           `json:"id"`
	Idx        int             `json:"idx"`
	Title      string          `json:"title"`
	IsFree     bool            `json:"isFree"`
	PriceCents int             `json:"priceCents"`
	SKU        string          `json:"sku,omitempty"`
	Poster     string          `json:"poster,omitempty"`
	MapJSON    json.RawMessage `json:"mapJson,omitempty"`
}

func toChapterDTO(c store.Chapter) chapterDTO {
	d := chapterDTO{ID: c.ID, Idx: c.Idx, Title: c.Title, IsFree: c.IsFree, PriceCents: c.PriceCents, SKU: c.SKU.String, Poster: c.Poster.String}
	if c.MapJSON != "" && c.MapJSON != "{}" {
		d.MapJSON = json.RawMessage(c.MapJSON)
	}
	return d
}

type videoDTO struct {
	ID         int64  `json:"id"`
	ChapterID  int64  `json:"chapterId"`
	Code       string `json:"code"`
	Idx        int    `json:"idx"`
	Title      string `json:"title"`
	MediaID    int64  `json:"mediaId,omitempty"`
	DurationMs int    `json:"durationMs,omitempty"`
	Poster     string `json:"poster,omitempty"`
}

func toVideoDTO(v store.ChapterVideo) videoDTO {
	return videoDTO{
		ID: v.ID, ChapterID: v.ChapterID, Code: v.Code, Idx: v.Idx, Title: v.Title,
		MediaID: v.MediaID.Int64, DurationMs: int(v.DurationMs.Int64), Poster: v.Poster.String,
	}
}

func toVideoDTOs(vs []store.ChapterVideo) []videoDTO {
	out := make([]videoDTO, 0, len(vs))
	for _, v := range vs {
		out = append(out, toVideoDTO(v))
	}
	return out
}

// ===== helpers =====

func pathID(r *http.Request, name string) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	return id, err == nil
}

func badReq(w http.ResponseWriter, msg string) {
	writeErr(w, &director.Error{Status: http.StatusBadRequest, Code: "VALIDATION", Msg: msg})
}

// handleWrite map lỗi DAL → HTTP. Trả true nếu OK (err == nil).
func handleWrite(w http.ResponseWriter, err error) bool {
	switch {
	case err == nil:
		return true
	case errors.Is(err, store.ErrNotFound):
		writeErr(w, &director.Error{Status: http.StatusNotFound, Code: "NOT_FOUND", Msg: "không tìm thấy"})
	case errors.Is(err, store.ErrInUse):
		writeErr(w, &director.Error{Status: http.StatusConflict, Code: "IN_USE", Msg: "đang được tham chiếu, không thể xoá"})
	default:
		writeErr(w, err) // 500
	}
	return false
}

// ===== routes =====

func (s *Server) registerAdmin(mux *http.ServeMux) {
	// ----- models -----
	mux.HandleFunc("GET /api/admin/models", s.adminAuth(func(w http.ResponseWriter, r *http.Request) {
		models, err := s.St.Models()
		if !handleWrite(w, err) {
			return
		}
		out := make([]modelDTO, 0, len(models))
		for i := range models {
			out = append(out, toModelDTO(&models[i]))
		}
		writeJSON(w, http.StatusOK, map[string]any{"models": out})
	}))

	mux.HandleFunc("POST /api/admin/models", s.adminAuth(func(w http.ResponseWriter, r *http.Request) {
		in, err := decodeBody[store.ModelInput](r)
		if err != nil {
			badReq(w, err.Error())
			return
		}
		if in.Code == "" || in.DisplayName == "" {
			badReq(w, "code và displayName là bắt buộc")
			return
		}
		id, err := s.St.CreateModel(in)
		if !handleWrite(w, err) {
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"id": id})
	}))

	mux.HandleFunc("GET /api/admin/models/{id}", s.adminAuth(func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(r, "id")
		if !ok {
			badReq(w, "id không hợp lệ")
			return
		}
		m, err := s.St.Model(id)
		if !handleWrite(w, err) {
			return
		}
		writeJSON(w, http.StatusOK, toModelDTO(m))
	}))

	mux.HandleFunc("PATCH /api/admin/models/{id}", s.adminAuth(func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(r, "id")
		if !ok {
			badReq(w, "id không hợp lệ")
			return
		}
		in, err := decodeBody[store.ModelInput](r)
		if err != nil {
			badReq(w, err.Error())
			return
		}
		if !handleWrite(w, s.St.UpdateModel(id, in)) {
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}))

	mux.HandleFunc("DELETE /api/admin/models/{id}", s.adminAuth(func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(r, "id")
		if !ok {
			badReq(w, "id không hợp lệ")
			return
		}
		if !handleWrite(w, s.St.DeleteModel(id)) {
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}))

	mux.HandleFunc("POST /api/admin/models/{id}/publish", s.adminAuth(func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(r, "id")
		if !ok {
			badReq(w, "id không hợp lệ")
			return
		}
		body, err := decodeBody[struct {
			Published *bool `json:"published"`
		}](r)
		if err != nil {
			badReq(w, err.Error())
			return
		}
		published := body.Published == nil || *body.Published // mặc định publish
		if !handleWrite(w, s.St.SetModelPublished(id, published)) {
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true, "published": published})
	}))

	// Export toàn bộ nội dung model → StoryFile (round-trip với PUT .../content). Editor load bằng đây.
	mux.HandleFunc("GET /api/admin/models/{id}/content", s.adminAuth(func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(r, "id")
		if !ok {
			badReq(w, "id không hợp lệ")
			return
		}
		sf, err := seed.Export(s.St, id)
		if !handleWrite(w, err) {
			return
		}
		writeJSON(w, http.StatusOK, sf)
	}))

	// Import/replace toàn bộ nội dung model (đường ghi an toàn cho đồ thị flow).
	mux.HandleFunc("PUT /api/admin/models/{id}/content", s.adminAuth(func(w http.ResponseWriter, r *http.Request) {
		sf, err := decodeBody[seed.StoryFile](r)
		if err != nil {
			badReq(w, err.Error())
			return
		}
		if sf.Story.Slug == "" || sf.Model.Code == "" {
			badReq(w, "story.slug và model.code là bắt buộc")
			return
		}
		if err := seed.Replace(s.St, &sf); err != nil {
			// lỗi validate DSL / ref / dead-end → 400 cho client biết sửa.
			writeErr(w, &director.Error{Status: http.StatusBadRequest, Code: "CONTENT_INVALID", Msg: err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}))

	// ----- chapters -----
	mux.HandleFunc("GET /api/admin/models/{id}/chapters", s.adminAuth(func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(r, "id")
		if !ok {
			badReq(w, "id không hợp lệ")
			return
		}
		chs, err := s.St.Chapters(id)
		if !handleWrite(w, err) {
			return
		}
		out := make([]chapterDTO, 0, len(chs))
		for _, c := range chs {
			out = append(out, toChapterDTO(c))
		}
		writeJSON(w, http.StatusOK, map[string]any{"chapters": out})
	}))

	mux.HandleFunc("POST /api/admin/models/{id}/chapters", s.adminAuth(func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(r, "id")
		if !ok {
			badReq(w, "id không hợp lệ")
			return
		}
		in, err := decodeBody[store.ChapterInput](r)
		if err != nil {
			badReq(w, err.Error())
			return
		}
		if in.Title == "" {
			badReq(w, "title là bắt buộc")
			return
		}
		cid, err := s.St.CreateChapter(id, in)
		if !handleWrite(w, err) {
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"id": cid})
	}))

	mux.HandleFunc("PATCH /api/admin/chapters/{id}", s.adminAuth(func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(r, "id")
		if !ok {
			badReq(w, "id không hợp lệ")
			return
		}
		in, err := decodeBody[store.ChapterInput](r)
		if err != nil {
			badReq(w, err.Error())
			return
		}
		if !handleWrite(w, s.St.UpdateChapter(id, in)) {
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}))

	mux.HandleFunc("DELETE /api/admin/chapters/{id}", s.adminAuth(func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(r, "id")
		if !ok {
			badReq(w, "id không hợp lệ")
			return
		}
		if !handleWrite(w, s.St.DeleteChapter(id)) {
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}))

	// Lưu RIÊNG layout bản đồ (map_json) — editor kéo-thả gọi khi Save layout.
	mux.HandleFunc("PUT /api/admin/chapters/{id}/map", s.adminAuth(func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(r, "id")
		if !ok {
			badReq(w, "id không hợp lệ")
			return
		}
		raw, err := decodeBody[json.RawMessage](r)
		if err != nil {
			badReq(w, err.Error())
			return
		}
		if len(raw) == 0 || !json.Valid(raw) {
			badReq(w, "map JSON không hợp lệ")
			return
		}
		if !handleWrite(w, s.St.SetChapterMap(id, string(raw))) {
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}))

	// ----- chapter videos -----
	mux.HandleFunc("GET /api/admin/chapters/{id}/videos", s.adminAuth(func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(r, "id")
		if !ok {
			badReq(w, "id không hợp lệ")
			return
		}
		vids, err := s.St.ChapterVideos(id)
		if !handleWrite(w, err) {
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"videos": toVideoDTOs(vids)})
	}))

	mux.HandleFunc("POST /api/admin/chapters/{id}/videos", s.adminAuth(func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(r, "id")
		if !ok {
			badReq(w, "id không hợp lệ")
			return
		}
		in, err := decodeBody[store.VideoInput](r)
		if err != nil {
			badReq(w, err.Error())
			return
		}
		if in.Code == "" || in.Title == "" {
			badReq(w, "code và title là bắt buộc")
			return
		}
		vid, err := s.St.CreateChapterVideo(id, in)
		if !handleWrite(w, err) {
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"id": vid})
	}))

	mux.HandleFunc("PATCH /api/admin/videos/{id}", s.adminAuth(func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(r, "id")
		if !ok {
			badReq(w, "id không hợp lệ")
			return
		}
		in, err := decodeBody[store.VideoInput](r)
		if err != nil {
			badReq(w, err.Error())
			return
		}
		if !handleWrite(w, s.St.UpdateChapterVideo(id, in)) {
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}))

	mux.HandleFunc("DELETE /api/admin/videos/{id}", s.adminAuth(func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(r, "id")
		if !ok {
			badReq(w, "id không hợp lệ")
			return
		}
		if !handleWrite(w, s.St.DeleteChapterVideo(id)) {
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}))

	mux.HandleFunc("POST /api/admin/chapters/{id}/videos/reorder", s.adminAuth(func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(r, "id")
		if !ok {
			badReq(w, "id không hợp lệ")
			return
		}
		body, err := decodeBody[struct {
			IDs []int64 `json:"ids"`
		}](r)
		if err != nil {
			badReq(w, err.Error())
			return
		}
		if !handleWrite(w, s.St.ReorderChapterVideos(id, body.IDs)) {
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}))

	// ----- flow (đồ thị scenes+choices) — READ đầy đủ cho editor -----
	mux.HandleFunc("GET /api/admin/chapters/{id}/flow", s.adminAuth(func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(r, "id")
		if !ok {
			badReq(w, "id không hợp lệ")
			return
		}
		flow, err := s.adminFlow(id)
		if !handleWrite(w, err) {
			return
		}
		writeJSON(w, http.StatusOK, flow)
	}))
}

// adminFlow — dựng đồ thị đầy đủ của chapter (nodes=scenes, edges=linear/choice, layout=map_json).
func (s *Server) adminFlow(chapterID int64) (map[string]any, error) {
	ch, err := s.St.Chapter(chapterID)
	if err != nil {
		return nil, err
	}
	scenes, err := s.St.Scenes(chapterID)
	if err != nil {
		return nil, err
	}
	videos, err := s.St.ChapterVideos(chapterID)
	if err != nil {
		return nil, err
	}
	videoCodeByID := map[int64]string{}
	for _, v := range videos {
		videoCodeByID[v.ID] = v.Code
	}
	sceneCodeByID := map[int64]string{}
	for _, sc := range scenes {
		sceneCodeByID[sc.ID] = sc.Code
	}
	// resolve code cho scene đích (có thể ở chapter khác — ranh giới chapter).
	codeOf := func(id int64) string {
		if c, ok := sceneCodeByID[id]; ok {
			return c
		}
		if sc, err := s.St.Scene(id); err == nil {
			return sc.Code
		}
		return ""
	}

	entry := ""
	if ch.EntrySceneID.Valid {
		entry = codeOf(ch.EntrySceneID.Int64)
	}

	nodes := []map[string]any{}
	edges := []map[string]any{}
	for _, sc := range scenes {
		node := map[string]any{"id": sc.Code, "type": sc.Type}
		if sc.VideoID.Valid {
			node["videoCode"] = videoCodeByID[sc.VideoID.Int64]
		}
		if sc.Type == "ending" {
			if e, err := s.St.EndingByScene(sc.ID); err == nil {
				node["endingCode"] = e.Code
				node["endingTitle"] = e.Title
				if e.Rank.Valid {
					node["rank"] = e.Rank.String
				}
			}
		}
		nodes = append(nodes, node)

		switch sc.Type {
		case "linear":
			if sc.NextSceneID.Valid {
				edges = append(edges, map[string]any{"from": sc.Code, "to": codeOf(sc.NextSceneID.Int64)})
			}
		case "choice":
			choices, err := s.St.ChoicesForScene(sc.ID)
			if err != nil {
				return nil, err
			}
			for _, c := range choices {
				edge := map[string]any{"from": sc.Code, "label": c.Label}
				if c.NextSceneID.Valid {
					edge["to"] = codeOf(c.NextSceneID.Int64)
				}
				if c.ConditionRaw != "" && c.ConditionRaw != "{}" {
					edge["condition"] = json.RawMessage(c.ConditionRaw)
				}
				if c.EffectsRaw != "" && c.EffectsRaw != "{}" {
					edge["effects"] = json.RawMessage(c.EffectsRaw)
				}
				if c.TimerMs.Valid {
					edge["timerMs"] = c.TimerMs.Int64
				}
				edges = append(edges, edge)
			}
		}
	}

	var layout any
	if ch.MapJSON != "" && ch.MapJSON != "{}" {
		layout = json.RawMessage(ch.MapJSON)
	}
	return map[string]any{
		"chapterId": ch.ID,
		"idx":       ch.Idx,
		"title":     ch.Title,
		"entry":     entry,
		"videos":    toVideoDTOs(videos),
		"nodes":     nodes,
		"edges":     edges,
		"layout":    layout,
	}, nil
}
