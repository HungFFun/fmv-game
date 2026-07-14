package director

import (
	"errors"
	"testing"

	"fmv-game/backend/internal/seed"
	"fmv-game/backend/internal/store"
)

// newTestDirector: DB in-memory + seed demo story thật → test đi đúng đồ thị
// mà người chơi sẽ đi (chống lỗi kiểu "Meow ending" — nhánh kẹt vì test thiếu).
func newTestDirector(t *testing.T) (*Director, int64) {
	t.Helper()
	st, err := store.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	if err := seed.LoadFile(st, "../../content/demo-story.json"); err != nil {
		t.Fatalf("seed: %v", err)
	}
	uid, err := st.GetOrCreateUser("test@dev.local")
	if err != nil {
		t.Fatal(err)
	}
	return New(st), uid
}

func choiceByLabel(t *testing.T, resp *SceneResponse, label string) int64 {
	t.Helper()
	for _, c := range resp.Choices {
		if c.Label == label {
			return c.ID
		}
	}
	t.Fatalf("không thấy choice %q trong scene %s; có: %+v", label, resp.Scene.Code, resp.Choices)
	return 0
}

func mustAdvance(t *testing.T, d *Director, uid int64, choiceID *int64) *SceneResponse {
	t.Helper()
	resp, err := d.Advance(uid, choiceID)
	if err != nil {
		t.Fatalf("advance: %v", err)
	}
	return resp
}

func dirErr(t *testing.T, err error) *Error {
	t.Helper()
	var de *Error
	if !errors.As(err, &de) {
		t.Fatalf("muốn *director.Error, nhận %T: %v", err, err)
	}
	return de
}

// enterBar: từ đầu game bấm hotspot "cửa Moonlit Bar" (ch1_intro) → vào ch1_bar.
func enterBar(t *testing.T, d *Director, uid int64) *SceneResponse {
	t.Helper()
	intro, err := d.Current(uid)
	if err != nil {
		t.Fatal(err)
	}
	id := choiceByLabel(t, intro, "Bước vào Moonlit Bar")
	return mustAdvance(t, d, uid, &id)
}

// Playthrough đầy đủ story design (Moonlit Bar, 7 shot): phố (hotspot cửa) → vào
// quán → nhánh "để ý cô gái" → hỏi tên → hỏi số → ending "Đừng vội về nhà".
func TestFullPlaythroughHanaEnding(t *testing.T) {
	d, uid := newTestDirector(t)

	// Vào game → chương 1 free, scene phố (ch1_intro) — CHOICE có hotspot cửa.
	cur, err := d.Current(uid)
	if err != nil {
		t.Fatal(err)
	}
	if cur.Scene.Code != "ch1_intro" || cur.Chapter.Idx != 1 {
		t.Fatalf("entry sai: %+v", cur.Scene)
	}
	if cur.VideoURL == nil {
		t.Error("scene phố phải có signed video URL")
	}
	if !cur.State.Flags["game_started"] {
		t.Error("on_enter của intro phải set flag game_started")
	}
	if len(cur.Choices) != 1 || cur.Choices[0].Hotspot == nil {
		t.Fatalf("ch1_intro phải có đúng 1 choice hotspot (cửa): %+v", cur.Choices)
	}

	// Bấm cửa → vào quán (ch1_bar) — CHOICE dạng thẻ (không hotspot).
	id := choiceByLabel(t, cur, "Bước vào Moonlit Bar")
	cur = mustAdvance(t, d, uid, &id)
	if cur.Scene.Code != "ch1_bar" || len(cur.Choices) != 2 {
		t.Fatalf("ch1_bar sai: %+v", cur)
	}
	for _, c := range cur.Choices {
		if c.PreloadURL == nil {
			t.Errorf("choice %q thiếu preloadUrl cho BranchPreloader", c.Label)
		}
		if c.Hotspot != nil {
			t.Errorf("choice %q ở ch1_bar không nên có hotspot (dùng thẻ câu hỏi)", c.Label)
		}
	}

	// Chú ý cô gái (+10, on_enter nhánh quiet +8 → 18)
	id = choiceByLabel(t, cur, "Chú ý đến cô gái ở góc quầy")
	cur = mustAdvance(t, d, uid, &id)
	if cur.Scene.Code != "ch1_quiet" {
		t.Fatalf("đi nhánh sai: %s", cur.Scene.Code)
	}
	if got := cur.State.Affinity["hana"]; got != 18 {
		t.Errorf("affinity hana = %d, muốn 18 (10 choice + 8 on_enter)", got)
	}

	// Scene quiet unlock gallery
	items, err := d.Gallery(uid)
	if err != nil {
		t.Fatal(err)
	}
	var quietUnlocked bool
	for _, it := range items {
		if it.Unlocked && it.Title == "Ánh mắt bên khung cửa" {
			quietUnlocked = true
			if it.MediaURL == nil {
				t.Error("item đã unlock phải có signed mediaUrl")
			}
		}
	}
	if !quietUnlocked {
		t.Error("tới ch1_quiet phải unlock 'Ánh mắt bên khung cửa'")
	}

	// → màn hỏi tên (choice có timer + default)
	cur = mustAdvance(t, d, uid, nil)
	if cur.Scene.Code != "ch1_name" {
		t.Fatalf("muốn ch1_name, nhận %s", cur.Scene.Code)
	}
	if cur.Choices[0].TimerMs == nil || *cur.Choices[0].TimerMs != 10000 {
		t.Error("choice hỏi tên phải có timer 10s")
	}
	var hasDefault bool
	for _, c := range cur.Choices {
		if c.IsDefault {
			hasDefault = true
		}
	}
	if !hasDefault {
		t.Error("phải có default choice khi hết giờ")
	}

	// Hỏi tên (+10 → 28) → ch1_number (checkpoint)
	id = choiceByLabel(t, cur, "Xin phép hỏi tên em")
	cur = mustAdvance(t, d, uid, &id)
	if cur.Scene.Code != "ch1_number" || !cur.Scene.IsCheckpoint {
		t.Fatalf("ch1_number checkpoint sai: %+v", cur.Scene)
	}
	if got := cur.State.Affinity["hana"]; got != 28 {
		t.Errorf("affinity hana = %d, muốn 28", got)
	}

	// → ending "Đừng vội về nhà"
	cur = mustAdvance(t, d, uid, nil)
	if cur.Scene.Type != "ending" || cur.Ending == nil || cur.Ending.Code != "hana_good" {
		t.Fatalf("ending sai: %+v", cur.Ending)
	}
	if cur.Ending.Rank == nil || *cur.Ending.Rank != "good" {
		t.Errorf("ending hana_good phải rank 'good': %+v", cur.Ending)
	}

	// Advance từ ending → 409; restart → quay lại phố, gallery vẫn giữ unlock
	if _, err := d.Advance(uid, nil); dirErr(t, err).Code != "AT_ENDING" {
		t.Error("advance từ ending phải trả AT_ENDING")
	}
	cur, err = d.Restart(uid)
	if err != nil {
		t.Fatal(err)
	}
	if cur.Scene.Code != "ch1_intro" {
		t.Errorf("restart phải về ch1_intro, nhận %s", cur.Scene.Code)
	}
	items, _ = d.Gallery(uid)
	var stillUnlocked int
	for _, it := range items {
		if it.Unlocked {
			stillUnlocked++
		}
	}
	if stillUnlocked < 2 { // 'Ánh mắt bên khung cửa' + 'Nụ cười của Hana' (ending)
		t.Errorf("gallery unlock phải vĩnh viễn xuyên lượt chơi, còn %d", stillUnlocked)
	}
}

// Video đầu tiên (phố) là màn Tương tác: choice "cửa" mang hotspot (Frame 57);
// choice trong quán (ch1_bar) không hotspot → dùng thẻ câu hỏi. Toạ độ do server trả.
func TestChoiceHotspotsFromContent(t *testing.T) {
	d, uid := newTestDirector(t)
	intro, err := d.Current(uid)
	if err != nil {
		t.Fatal(err)
	}
	if intro.Scene.Code != "ch1_intro" {
		t.Fatalf("scene đầu sai: %s", intro.Scene.Code)
	}
	door := &intro.Choices[0]
	if door.Label != "Bước vào Moonlit Bar" || door.Hotspot == nil {
		t.Fatalf("choice cửa phải có hotspot; nhận %+v", door)
	}
	if door.Hotspot.Style != "door" || door.Hotspot.W != 0.179 {
		t.Errorf("hotspot cửa sai (phải khớp Frame 57): %+v", door.Hotspot)
	}

	// Vào quán → ch1_bar: choice KHÔNG có hotspot (thẻ câu hỏi).
	bar := mustAdvance(t, d, uid, &door.ID)
	if bar.Scene.Code != "ch1_bar" {
		t.Fatalf("scene sai: %s", bar.Scene.Code)
	}
	for _, c := range bar.Choices {
		if c.Hotspot != nil {
			t.Errorf("choice ch1_bar %q không nên có hotspot: %+v", c.Label, c.Hotspot)
		}
	}
}

func TestAdvanceRejectsSpoofedChoice(t *testing.T) {
	d, uid := newTestDirector(t)

	if _, err := d.Current(uid); err != nil { // ch1_intro (choice hotspot)
		t.Fatal(err)
	}

	// Spoof 1: choiceId không thuộc scene hiện tại
	var bogus int64 = 999999
	if _, err := d.Advance(uid, &bogus); dirErr(t, err).Code != "INVALID_CHOICE" {
		t.Error("phải chặn choice không thuộc scene")
	}

	// Spoof 2: gửi id của choice thuộc scene khác (một choice của ch1_name) khi
	// đang ở ch1_intro → phải bị chặn (INVALID_CHOICE).
	var otherID int64
	if err := d.St.DB.QueryRow(
		`SELECT c.id FROM choices c JOIN scenes s ON s.id = c.scene_id WHERE s.code = 'ch1_name' LIMIT 1`).Scan(&otherID); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Advance(uid, &otherID); dirErr(t, err).Code != "INVALID_CHOICE" {
		t.Error("phải chặn choice thuộc scene khác")
	}
}

func TestGalleryUnlockEmitsNotification(t *testing.T) {
	d, uid := newTestDirector(t)

	cur, err := d.Current(uid)
	if err != nil {
		t.Fatal(err)
	}
	if len(cur.Notifications) != 0 {
		t.Errorf("intro không unlock gì → không được có notification: %+v", cur.Notifications)
	}

	cur = enterBar(t, d, uid) // → ch1_bar (không unlock gallery)
	if len(cur.Notifications) != 0 {
		t.Errorf("ch1_bar không được có notification: %+v", cur.Notifications)
	}

	// Chú ý cô gái → ch1_quiet unlock "Ánh mắt bên khung cửa" → phát đúng 1 notification.
	id := choiceByLabel(t, cur, "Chú ý đến cô gái ở góc quầy")
	cur = mustAdvance(t, d, uid, &id)
	if cur.Scene.Code != "ch1_quiet" {
		t.Fatalf("đi nhánh sai: %s", cur.Scene.Code)
	}
	if len(cur.Notifications) != 1 {
		t.Fatalf("tới ch1_quiet phải phát đúng 1 notification, có %d: %+v", len(cur.Notifications), cur.Notifications)
	}
	if n := cur.Notifications[0]; n.Kind != "gallery" || n.Body != "Ánh mắt bên khung cửa" {
		t.Errorf("notification sai: %+v", n)
	}

	// Chơi lại và đi qua CHÍNH scene đó lần nữa: item đã unlock vĩnh viễn →
	// KHÔNG phát lại (tránh spam khi load save cũ / replay).
	if _, err := d.Restart(uid); err != nil {
		t.Fatal(err)
	}
	cur = enterBar(t, d, uid)
	id = choiceByLabel(t, cur, "Chú ý đến cô gái ở góc quầy")
	cur = mustAdvance(t, d, uid, &id) // ch1_quiet lần 2
	if len(cur.Notifications) != 0 {
		t.Errorf("item đã unlock rồi → không được phát lại: %+v", cur.Notifications)
	}
}

func TestChoiceSceneRequiresChoiceId(t *testing.T) {
	d, uid := newTestDirector(t)
	if _, err := d.Current(uid); err != nil { // ch1_intro là scene choice
		t.Fatal(err)
	}
	// Scene choice mà không gửi choiceId → CHOICE_REQUIRED
	if _, err := d.Advance(uid, nil); dirErr(t, err).Code != "CHOICE_REQUIRED" {
		t.Error("scene choice phải đòi choiceId")
	}
	// Scene linear (ch1_counter) → advance không cần choiceId
	bar := enterBar(t, d, uid)
	id := choiceByLabel(t, bar, "Ngồi xuống quầy bar")
	counter := mustAdvance(t, d, uid, &id) // → ch1_counter (linear)
	if counter.Scene.Code != "ch1_counter" {
		t.Fatalf("muốn ch1_counter, nhận %s", counter.Scene.Code)
	}
	name := mustAdvance(t, d, uid, nil) // linear advance
	if name.Scene.Code != "ch1_name" {
		t.Fatalf("linear advance sai: %s", name.Scene.Code)
	}
}

// Chương 2 (Mal-sook/Min-jung) không nằm trong story design 7-shot: vẫn được seed
// nhưng khoá (chưa mua) — entitlement vẫn được thực thi ở tầng dữ liệu.
func TestChapterTwoStaysLocked(t *testing.T) {
	d, uid := newTestDirector(t)
	chs, err := d.ChaptersOverview(uid)
	if err != nil {
		t.Fatal(err)
	}
	if len(chs) < 2 {
		t.Fatalf("cần ≥2 chương, có %d", len(chs))
	}
	if chs[0].Locked {
		t.Error("chương 1 free phải mở")
	}
	if !chs[1].Locked {
		t.Error("chương 2 (chưa mua) phải khoá")
	}
}

func TestSaveLoadSlots(t *testing.T) {
	d, uid := newTestDirector(t)
	bar := enterBar(t, d, uid) // → ch1_bar

	if err := d.SaveToSlot(uid, 1); err != nil {
		t.Fatal(err)
	}

	// Đi tiếp một nhánh rồi load lại slot 1 → quay về ch1_bar
	id := choiceByLabel(t, bar, "Ngồi xuống quầy bar")
	mustAdvance(t, d, uid, &id)

	back, err := d.LoadFromSlot(uid, 1)
	if err != nil {
		t.Fatal(err)
	}
	if back.Scene.Code != "ch1_bar" {
		t.Errorf("load slot phải về ch1_bar, nhận %s", back.Scene.Code)
	}
	if back.State.Affinity["hana"] != 0 {
		t.Errorf("state phải khôi phục về trước khi chọn, hana = %d", back.State.Affinity["hana"])
	}

	if err := d.SaveToSlot(uid, 0); err == nil {
		t.Error("không được ghi đè autosave slot 0 qua API save thủ công")
	}
}
