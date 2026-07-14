// Package director — DIRECTOR SERVICE, trái tim của engine (server-authoritative).
// Client KHÔNG bao giờ tự quyết scene tiếp theo hay tự mở video.
package director

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"fmv-game/backend/internal/engine"
	"fmv-game/backend/internal/media"
	"fmv-game/backend/internal/store"
)

const AutosaveSlot = 0

// Error — lỗi nghiệp vụ có HTTP status + code máy đọc được.
type Error struct {
	Status int            `json:"-"`
	Code   string         `json:"code"`
	Msg    string         `json:"message"`
	Data   map[string]any `json:"data,omitempty"`
}

func (e *Error) Error() string { return fmt.Sprintf("%s: %s", e.Code, e.Msg) }

func errf(status int, code, msg string) *Error {
	return &Error{Status: status, Code: code, Msg: msg}
}

type Director struct {
	St *store.Store
}

func New(st *store.Store) *Director { return &Director{St: st} }

// ===== Response DTOs =====

type ChoiceDTO struct {
	ID        int64   `json:"id"`
	Label     string  `json:"label"`
	TimerMs   *int64  `json:"timerMs"`
	IsDefault bool    `json:"isDefault"`
	// PreloadURL: video đoạn đầu của nhánh — cho BranchPreloader phía client.
	PreloadURL *string `json:"preloadUrl"`
	// Hotspot: nếu có, choice này là vùng bấm định vị trên khung video (màn
	// "Tương tác") thay vì nút text. nil = giữ nút text như thường.
	Hotspot *HotspotDTO `json:"hotspot,omitempty"`
}

// HotspotDTO — vùng bấm định vị trên khung video (toạ độ 0..1 tương đối theo
// khung hình). Style = kiểu hiển thị client ('door' | 'marker').
type HotspotDTO struct {
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	W     float64 `json:"w"`
	H     float64 `json:"h"`
	Style string  `json:"style,omitempty"`
}

type EndingDTO struct {
	Code  string  `json:"code"`
	Title string  `json:"title"`
	Rank  *string `json:"rank"`
}

// NotifyDTO — thông báo in-app do SERVER phát (server-authoritative): client chỉ
// hiển thị, không tự suy ra. Hiện chỉ phát khi mở khoá gallery; mở rộng sau:
// 'ending', 'chapter'...
type NotifyDTO struct {
	Kind  string `json:"kind"` // 'gallery'
	Title string `json:"title"`
	Body  string `json:"body"`
}

type SceneResponse struct {
	Scene struct {
		ID           int64  `json:"id"`
		Code         string `json:"code"`
		Type         string `json:"type"`
		IsCheckpoint bool   `json:"isCheckpoint"`
	} `json:"scene"`
	Chapter struct {
		ID    int64  `json:"id"`
		Idx   int    `json:"idx"`
		Title string `json:"title"`
	} `json:"chapter"`
	VideoURL *string `json:"videoUrl"`
	// Choices ĐÃ LỌC theo condition — client không thấy nhánh chưa đủ điều kiện.
	Choices    []ChoiceDTO `json:"choices"`
	Ending     *EndingDTO  `json:"ending"`
	State      StateDTO    `json:"state"`
	Characters []CharDTO   `json:"characters"`
	// Notifications: sự kiện server phát ra cho riêng lượt advance này (vd: mở
	// khoá gallery). Rỗng trên Current/load — chỉ enterScene mới sinh ra.
	Notifications []NotifyDTO `json:"notifications"`
}

type StateDTO struct {
	Affinity map[string]int  `json:"affinity"`
	Flags    map[string]bool `json:"flags"`
}

type CharDTO struct {
	Code        string `json:"code"`
	DisplayName string `json:"displayName"`
}

// ===== Helpers =====

func (d *Director) assertEntitled(userID int64, ch *store.Chapter) error {
	if ch.IsFree {
		return d.St.GrantEntitlement(userID, ch.ID, "free")
	}
	ok, err := d.St.IsEntitled(userID, ch.ID)
	if err != nil {
		return err
	}
	if !ok {
		e := errf(http.StatusPaymentRequired, "CHAPTER_LOCKED",
			fmt.Sprintf("Chương %q chưa mở khoá", ch.Title))
		e.Data = map[string]any{
			"chapterId": ch.ID, "idx": ch.Idx, "title": ch.Title,
			"priceCents": ch.PriceCents, "sku": ch.SKU.String,
		}
		return e
	}
	return nil
}

func (d *Director) videoURL(sc *store.Scene, userID int64) (*string, error) {
	if !sc.VideoID.Valid {
		return nil, nil
	}
	// scenes.video_id → chapter_videos → media_assets (resolve 1 hop).
	m, err := d.St.VideoMedia(sc.VideoID.Int64)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	path := m.StorageKey
	if m.HLSManifest.Valid && m.HLSManifest.String != "" {
		path = m.HLSManifest.String
	}
	url := media.SignURL(path, userID)
	return &url, nil
}

func (d *Director) buildSceneResponse(sc *store.Scene, userID int64, state engine.State) (*SceneResponse, error) {
	ch, err := d.St.Chapter(sc.ChapterID)
	if err != nil {
		return nil, err
	}
	story, err := d.St.PublishedStory()
	if err != nil {
		return nil, err
	}

	resp := &SceneResponse{Choices: []ChoiceDTO{}, Notifications: []NotifyDTO{}}
	resp.Scene.ID = sc.ID
	resp.Scene.Code = sc.Code
	resp.Scene.Type = sc.Type
	resp.Scene.IsCheckpoint = sc.IsCheckpoint
	resp.Chapter.ID = ch.ID
	resp.Chapter.Idx = ch.Idx
	resp.Chapter.Title = ch.Title
	resp.State = StateDTO{Affinity: state.Affinity, Flags: state.Flags}

	if resp.VideoURL, err = d.videoURL(sc, userID); err != nil {
		return nil, err
	}

	if sc.Type == "choice" {
		choices, err := d.St.ChoicesForScene(sc.ID)
		if err != nil {
			return nil, err
		}
		for _, c := range choices {
			cond, err := engine.ParseCondition(c.ConditionRaw)
			if err != nil {
				return nil, fmt.Errorf("choice %d: %w", c.ID, err)
			}
			if !engine.Evaluate(cond, state) {
				continue // lọc server-side: client không bao giờ thấy
			}
			dto := ChoiceDTO{ID: c.ID, Label: c.Label}
			if c.TimerMs.Valid {
				v := c.TimerMs.Int64
				dto.TimerMs = &v
			}
			dto.IsDefault = c.DefaultChoiceID.Valid && c.DefaultChoiceID.Int64 == c.ID
			if c.Hotspot.Valid && c.Hotspot.String != "" {
				var hs HotspotDTO
				if err := json.Unmarshal([]byte(c.Hotspot.String), &hs); err != nil {
					return nil, fmt.Errorf("choice %d hotspot: %w", c.ID, err)
				}
				dto.Hotspot = &hs
			}
			if c.NextSceneID.Valid {
				if nextScene, err := d.St.Scene(c.NextSceneID.Int64); err == nil {
					if dto.PreloadURL, err = d.videoURL(nextScene, userID); err != nil {
						return nil, err
					}
				}
			}
			resp.Choices = append(resp.Choices, dto)
		}
	}

	if sc.Type == "ending" {
		if e, err := d.St.EndingByScene(sc.ID); err == nil {
			dto := &EndingDTO{Code: e.Code, Title: e.Title}
			if e.Rank.Valid {
				dto.Rank = &e.Rank.String
			}
			resp.Ending = dto
		}
	}

	chars, err := d.St.Characters(story.ModelID)
	if err != nil {
		return nil, err
	}
	resp.Characters = make([]CharDTO, 0, len(chars))
	for _, c := range chars {
		resp.Characters = append(resp.Characters, CharDTO{Code: c.Code, DisplayName: c.DisplayName})
	}
	return resp, nil
}

// enterScene — side-effects khi VÀO scene: on_enter, gallery unlock, ending record.
// Trả thêm []NotifyDTO: thông báo phát cho client (hiện: gallery mở khoá LẦN ĐẦU).
func (d *Director) enterScene(userID int64, sc *store.Scene, state engine.State) (engine.State, []NotifyDTO, error) {
	eff, err := engine.ParseEffects(sc.OnEnterRaw)
	if err != nil {
		return state, nil, fmt.Errorf("scene %s on_enter: %w", sc.Code, err)
	}
	next := engine.Apply(eff, state)

	var notifs []NotifyDTO
	items, err := d.St.GalleryItemsByUnlockScene(sc.ID)
	if err != nil {
		return next, notifs, err
	}
	if len(items) > 0 {
		// Chỉ thông báo item mở khoá LẦN ĐẦU — tránh phát lại khi đi qua scene
		// lần nữa (load save cũ rồi advance lại).
		unlocked, err := d.St.Unlocks(userID, "gallery")
		if err != nil {
			return next, notifs, err
		}
		for _, item := range items {
			wasUnlocked := unlocked[item.ID]
			if err := d.St.Unlock(userID, "gallery", item.ID); err != nil {
				return next, notifs, err
			}
			if !wasUnlocked {
				title := "Ảnh mới"
				if item.Title.Valid && item.Title.String != "" {
					title = item.Title.String
				}
				notifs = append(notifs, NotifyDTO{Kind: "gallery", Title: "Mở khoá ảnh", Body: title})
			}
		}
	}
	if sc.Type == "ending" {
		if e, err := d.St.EndingByScene(sc.ID); err == nil {
			if err := d.St.Unlock(userID, "ending", e.ID); err != nil {
				return next, notifs, err
			}
		}
	}
	return next, notifs, nil
}

// ===== Public API =====

// Current — scene hiện tại; chưa có save thì bắt đầu game mới từ chương 1.
func (d *Director) Current(userID int64) (*SceneResponse, error) {
	save, err := d.St.GetSave(userID, AutosaveSlot)
	if errors.Is(err, store.ErrNotFound) {
		return d.StartNewGame(userID)
	}
	if err != nil {
		return nil, err
	}
	sc, err := d.St.Scene(save.CurrentSceneID)
	if err != nil {
		return nil, errf(http.StatusInternalServerError, "BROKEN_SAVE", "save trỏ tới scene không tồn tại")
	}
	return d.buildSceneResponse(sc, userID, save.State)
}

// JumpToScene — nhảy tới 1 scene cụ thể (bấm đúng node video trên bản đồ chapter) để xem/chơi lại.
// Server-authoritative: chỉ cho nhảy trong chương đã sở hữu. Giữ state hiện tại, đổi scene hiện tại
// của autosave rồi trả SceneResponse. KHÔNG chạy lại on_enter (tránh cộng dồn affinity khi xem lại).
func (d *Director) JumpToScene(userID int64, code string) (*SceneResponse, error) {
	sc, err := d.St.SceneByCode(code)
	if errors.Is(err, store.ErrNotFound) {
		return nil, errf(http.StatusNotFound, "NO_SCENE", "scene không tồn tại")
	}
	if err != nil {
		return nil, err
	}
	ch, err := d.St.Chapter(sc.ChapterID)
	if err != nil {
		return nil, errf(http.StatusInternalServerError, "NO_CHAPTER", "scene không thuộc chương nào")
	}
	if err := d.assertEntitled(userID, ch); err != nil {
		return nil, err
	}
	// Chỉ cho nhảy tới scene đã MỞ KHÓA: đã ghé qua HOẶC là entry của chương.
	visited, err := d.St.VisitedScenes(userID, ch.ID)
	if err != nil {
		return nil, err
	}
	isEntry := ch.EntrySceneID.Valid && ch.EntrySceneID.Int64 == sc.ID
	if !visited[code] && !isEntry {
		return nil, errf(http.StatusForbidden, "SCENE_LOCKED", "video chưa mở khóa — xem video trước để mở")
	}
	story, err := d.St.PublishedStory()
	if err != nil {
		return nil, errf(http.StatusInternalServerError, "NO_STORY", "chưa seed story")
	}
	state := engine.NewState()
	if save, err := d.St.GetSave(userID, AutosaveSlot); err == nil {
		state = save.State
	} else if !errors.Is(err, store.ErrNotFound) {
		return nil, err
	}
	state.Chapter = ch.Idx
	if err := d.St.UpsertSave(userID, AutosaveSlot, story.ID, sc.ID, state); err != nil {
		return nil, err
	}
	return d.buildSceneResponse(sc, userID, state)
}

func (d *Director) StartNewGame(userID int64) (*SceneResponse, error) {
	story, err := d.St.PublishedStory()
	if err != nil {
		return nil, errf(http.StatusInternalServerError, "NO_STORY", "chưa seed story nào (chạy cmd/seed)")
	}
	chapters, err := d.St.Chapters(story.ModelID)
	if err != nil || len(chapters) == 0 {
		return nil, errf(http.StatusInternalServerError, "NO_CHAPTER", "story chưa có chương")
	}
	ch1 := chapters[0]
	if !ch1.EntrySceneID.Valid {
		return nil, errf(http.StatusInternalServerError, "NO_ENTRY", "chương 1 chưa có entry_scene_id")
	}
	if err := d.assertEntitled(userID, &ch1); err != nil {
		return nil, err
	}

	entry, err := d.St.Scene(ch1.EntrySceneID.Int64)
	if err != nil {
		return nil, err
	}
	state := engine.NewState()
	var notifs []NotifyDTO
	if state, notifs, err = d.enterScene(userID, entry, state); err != nil {
		return nil, err
	}
	state.Chapter = ch1.Idx
	if err := d.St.UpsertSave(userID, AutosaveSlot, story.ID, entry.ID, state); err != nil {
		return nil, err
	}
	resp, err := d.buildSceneResponse(entry, userID, state)
	if err != nil {
		return nil, err
	}
	if len(notifs) > 0 {
		resp.Notifications = notifs
	}
	return resp, nil
}

// Advance — đúng pseudocode trong thiết kế:
//  1. nạp state từ save        2. validate choice + condition (chống spoof)
//  3. apply effects + on_enter 4. check entitlement chương mới
//  5. autosave + log event     6. trả scene mới.
func (d *Director) Advance(userID int64, choiceID *int64) (*SceneResponse, error) {
	save, err := d.St.GetSave(userID, AutosaveSlot)
	if errors.Is(err, store.ErrNotFound) {
		return nil, errf(http.StatusNotFound, "NO_SAVE", "chưa có save — gọi GET /api/play/current trước")
	}
	if err != nil {
		return nil, err
	}
	sc, err := d.St.Scene(save.CurrentSceneID)
	if err != nil {
		return nil, errf(http.StatusInternalServerError, "BROKEN_SAVE", "save trỏ tới scene không tồn tại")
	}

	state := save.State
	var nextSceneID int64

	switch sc.Type {
	case "choice":
		if choiceID == nil {
			return nil, errf(http.StatusBadRequest, "CHOICE_REQUIRED", "scene này yêu cầu chọn một lựa chọn")
		}
		choice, err := d.St.Choice(*choiceID)
		if err != nil || choice.SceneID != sc.ID {
			return nil, errf(http.StatusBadRequest, "INVALID_CHOICE", "choice không thuộc scene hiện tại")
		}
		cond, err := engine.ParseCondition(choice.ConditionRaw)
		if err != nil {
			return nil, err
		}
		if !engine.Evaluate(cond, state) {
			return nil, errf(http.StatusForbidden, "CONDITION_FAILED", "không đủ điều kiện cho lựa chọn này")
		}
		eff, err := engine.ParseEffects(choice.EffectsRaw)
		if err != nil {
			return nil, err
		}
		state = engine.Apply(eff, state)
		if !choice.NextSceneID.Valid {
			return nil, errf(http.StatusInternalServerError, "DEAD_END", "choice không có next_scene_id (lỗi data)")
		}
		nextSceneID = choice.NextSceneID.Int64
		if err := d.St.LogChoiceEvent(userID, sc.ID, choice.ID, state); err != nil {
			return nil, err
		}
	case "ending":
		return nil, errf(http.StatusConflict, "AT_ENDING", "đã tới ending — dùng POST /api/play/restart để chơi lại")
	default: // linear
		if !sc.NextSceneID.Valid {
			return nil, errf(http.StatusInternalServerError, "DEAD_END",
				fmt.Sprintf("scene %s không có nhánh đi tiếp (lỗi data)", sc.Code))
		}
		nextSceneID = sc.NextSceneID.Int64
	}

	next, err := d.St.Scene(nextSceneID)
	if err != nil {
		return nil, errf(http.StatusInternalServerError, "BROKEN_GRAPH", "next_scene_id không tồn tại")
	}

	if next.ChapterID != sc.ChapterID {
		nextCh, err := d.St.Chapter(next.ChapterID)
		if err != nil {
			return nil, err
		}
		if err := d.assertEntitled(userID, nextCh); err != nil {
			return nil, err // 402 CHAPTER_LOCKED — khoá theo chương đã mua
		}
		state.Chapter = nextCh.Idx
	}

	var notifs []NotifyDTO
	if state, notifs, err = d.enterScene(userID, next, state); err != nil {
		return nil, err
	}
	// FMV autosave mỗi clip cho an toàn (is_checkpoint chỉ là metadata hiển thị).
	if err := d.St.UpsertSave(userID, AutosaveSlot, save.StoryID, next.ID, state); err != nil {
		return nil, err
	}
	resp, err := d.buildSceneResponse(next, userID, state)
	if err != nil {
		return nil, err
	}
	if len(notifs) > 0 {
		resp.Notifications = notifs
	}
	return resp, nil
}

func (d *Director) Restart(userID int64) (*SceneResponse, error) {
	return d.StartNewGame(userID)
}

// ===== Save slots thủ công =====

type SaveDTO struct {
	Slot      int            `json:"slot"`
	SceneCode string         `json:"sceneCode"`
	Chapter   int            `json:"chapter"`
	Affinity  map[string]int `json:"affinity"`
	UpdatedAt string         `json:"updatedAt"`
}

func (d *Director) ListSaves(userID int64) ([]SaveDTO, error) {
	saves, err := d.St.ListSaves(userID)
	if err != nil {
		return nil, err
	}
	out := []SaveDTO{}
	for _, sv := range saves {
		code := ""
		if sc, err := d.St.Scene(sv.CurrentSceneID); err == nil {
			code = sc.Code
		}
		out = append(out, SaveDTO{
			Slot: sv.Slot, SceneCode: code, Chapter: sv.State.Chapter,
			Affinity: sv.State.Affinity, UpdatedAt: sv.UpdatedAt,
		})
	}
	return out, nil
}

// SaveToSlot copy autosave → slot thủ công.
func (d *Director) SaveToSlot(userID int64, slot int) error {
	if slot <= AutosaveSlot {
		return errf(http.StatusBadRequest, "INVALID_SLOT", "slot thủ công phải >= 1")
	}
	auto, err := d.St.GetSave(userID, AutosaveSlot)
	if err != nil {
		return errf(http.StatusNotFound, "NO_SAVE", "chưa có autosave để lưu")
	}
	return d.St.UpsertSave(userID, slot, auto.StoryID, auto.CurrentSceneID, auto.State)
}

// LoadFromSlot copy slot thủ công → autosave rồi trả scene hiện tại.
func (d *Director) LoadFromSlot(userID int64, slot int) (*SceneResponse, error) {
	sv, err := d.St.GetSave(userID, slot)
	if err != nil {
		return nil, errf(http.StatusNotFound, "NO_SAVE", "slot trống")
	}
	if err := d.St.UpsertSave(userID, AutosaveSlot, sv.StoryID, sv.CurrentSceneID, sv.State); err != nil {
		return nil, err
	}
	return d.Current(userID)
}

// ===== Gallery & Store =====

type GalleryItemDTO struct {
	ID        int64   `json:"id"`
	Title     string  `json:"title"`
	IsBonus   bool    `json:"isBonus"`
	Unlocked  bool    `json:"unlocked"`
	MediaURL  *string `json:"mediaUrl"`
	MediaKind *string `json:"mediaKind"`
}

// Gallery — chỉ item đã unlock mới có title thật + signed URL.
func (d *Director) Gallery(userID int64) ([]GalleryItemDTO, error) {
	story, err := d.St.PublishedStory()
	if err != nil {
		return []GalleryItemDTO{}, nil
	}
	unlocked, err := d.St.Unlocks(userID, "gallery")
	if err != nil {
		return nil, err
	}
	items, err := d.St.GalleryItems(story.ModelID)
	if err != nil {
		return nil, err
	}
	out := []GalleryItemDTO{}
	for _, item := range items {
		dto := GalleryItemDTO{ID: item.ID, Title: "???", IsBonus: item.IsBonus}
		if unlocked[item.ID] {
			dto.Unlocked = true
			if item.Title.Valid {
				dto.Title = item.Title.String
			}
			if m, err := d.St.Media(item.MediaID); err == nil {
				url := media.SignURL(m.StorageKey, userID)
				dto.MediaURL = &url
				dto.MediaKind = &m.Kind
			}
		}
		out = append(out, dto)
	}
	return out, nil
}

type StoreChapterDTO struct {
	ID         int64  `json:"id"`
	Idx        int    `json:"idx"`
	Title      string `json:"title"`
	IsFree     bool   `json:"isFree"`
	PriceCents int    `json:"priceCents"`
	SKU        string `json:"sku"`
	Owned      bool   `json:"owned"`
}

func (d *Director) StoreChapters(userID int64) ([]StoreChapterDTO, error) {
	story, err := d.St.PublishedStory()
	if err != nil {
		return []StoreChapterDTO{}, nil
	}
	chapters, err := d.St.Chapters(story.ModelID)
	if err != nil {
		return nil, err
	}
	out := []StoreChapterDTO{}
	for _, ch := range chapters {
		owned := ch.IsFree
		if !owned {
			if owned, err = d.St.IsEntitled(userID, ch.ID); err != nil {
				return nil, err
			}
		}
		out = append(out, StoreChapterDTO{
			ID: ch.ID, Idx: ch.Idx, Title: ch.Title, IsFree: ch.IsFree,
			PriceCents: ch.PriceCents, SKU: ch.SKU.String, Owned: owned,
		})
	}
	return out, nil
}

// Purchase — GIẢ LẬP IAP: cấp entitlement ngay. Prod: chỉ cấp trong webhook
// Stripe/App Store sau khi verify receipt.
func (d *Director) Purchase(userID, chapterID int64) error {
	ch, err := d.St.Chapter(chapterID)
	if err != nil {
		return errf(http.StatusNotFound, "NO_CHAPTER", "chương không tồn tại")
	}
	return d.St.GrantEntitlement(userID, ch.ID, "purchase")
}

// ===== Chapter browse (màn Chapter select + Chapter map, data-driven) =====

type ChapterOverviewDTO struct {
	ID        int64   `json:"id"`
	Idx       int     `json:"idx"`
	Title     string  `json:"title"`
	Locked    bool    `json:"locked"`
	PosterURL *string `json:"posterUrl"`
}

// ChaptersOverview — danh sách chapter cho màn select; locked theo entitlement.
func (d *Director) ChaptersOverview(userID int64) ([]ChapterOverviewDTO, error) {
	story, err := d.St.PublishedStory()
	if err != nil {
		return []ChapterOverviewDTO{}, nil
	}
	chapters, err := d.St.Chapters(story.ModelID)
	if err != nil {
		return nil, err
	}
	out := []ChapterOverviewDTO{}
	for _, ch := range chapters {
		owned := ch.IsFree
		if !owned {
			if owned, err = d.St.IsEntitled(userID, ch.ID); err != nil {
				return nil, err
			}
		}
		dto := ChapterOverviewDTO{ID: ch.ID, Idx: ch.Idx, Title: ch.Title, Locked: !owned}
		if ch.Poster.Valid && ch.Poster.String != "" {
			u := media.SignURL(ch.Poster.String, userID)
			dto.PosterURL = &u
		}
		out = append(out, dto)
	}
	return out, nil
}

// MapNodeDTO — 1 node trên bản đồ chapter (video/lock/start/finish).
type MapNodeDTO struct {
	ID        string  `json:"id,omitempty"`
	Kind      string  `json:"kind"` // 'video' | 'lock' | 'start' | 'finish'
	Scene     string  `json:"scene,omitempty"` // scene code node này đại diện (bấm để nhảy tới)
	Title     string  `json:"title,omitempty"`
	PosterURL *string `json:"posterUrl,omitempty"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	W         float64 `json:"w"`
	H         float64 `json:"h"`
	Locked    bool    `json:"locked"`
	Hotspot   bool    `json:"hotspot,omitempty"` // scene của node này dùng HotspotArea (đánh dấu icon trên bản đồ)
}

// MapEdgeDTO — 1 cạnh nối 2 node theo `id` (đồ thị nhánh của story).
// Nhãn choice/điều kiện KHÔNG gửi cho player (tránh lộ spoiler) — chỉ from/to.
// Chosen = user đã đi qua nhánh này (bấm choice tương ứng) → vẽ đường liền + icon kim cương.
type MapEdgeDTO struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Chosen bool   `json:"chosen,omitempty"`
}

type ChapterMapDTO struct {
	Width  float64      `json:"width"`
	Height float64      `json:"height"`
	Locked bool         `json:"locked"`
	Nodes  []MapNodeDTO `json:"nodes"`
	Edges  []MapEdgeDTO `json:"edges"`
}

// rawMap — cấu trúc map_json trong content (poster lưu là storage key).
type rawMap struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	Nodes  []struct {
		ID     string  `json:"id"`
		Kind   string  `json:"kind"`
		Title  string  `json:"title"`
		Poster string  `json:"poster"`
		Scene  string  `json:"scene"` // scene code node này đại diện (để map cạnh đã-đi)
		X       float64 `json:"x"`
		Y       float64 `json:"y"`
		W       float64 `json:"w"`
		H       float64 `json:"h"`
		Locked  bool    `json:"locked"`
		Hotspot bool    `json:"hotspot"`
	} `json:"nodes"`
	Edges []struct {
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"edges"`
}

// ChapterMap — layout bản đồ nhánh cho 1 chapter; poster ký thành signed URL.
func (d *Director) ChapterMap(userID, chapterID int64) (*ChapterMapDTO, error) {
	ch, err := d.St.Chapter(chapterID)
	if err != nil {
		return nil, errf(http.StatusNotFound, "NO_CHAPTER", "chương không tồn tại")
	}
	owned := ch.IsFree
	if !owned {
		if owned, err = d.St.IsEntitled(userID, ch.ID); err != nil {
			return nil, err
		}
	}
	out := &ChapterMapDTO{Locked: !owned, Nodes: []MapNodeDTO{}, Edges: []MapEdgeDTO{}}
	if ch.MapJSON != "" && ch.MapJSON != "{}" {
		var rm rawMap
		if err := json.Unmarshal([]byte(ch.MapJSON), &rm); err != nil {
			return nil, fmt.Errorf("map_json chapter %d lỗi: %w", ch.ID, err)
		}
		out.Width = rm.Width
		out.Height = rm.Height

		// Tiến trình user trong chương (chỉ khi đã sở hữu). visited = scene đã ghé qua;
		// entry scene luôn mở khóa để bắt đầu chơi. Node video KHÓA nếu scene chưa ghé qua
		// (và không phải entry) → frontend chặn click, buộc xem video trước để mở tiếp.
		visited := map[string]bool{}
		entryCode := ""
		if owned {
			if visited, err = d.St.VisitedScenes(userID, ch.ID); err != nil {
				return nil, err
			}
			if ch.EntrySceneID.Valid {
				if es, err := d.St.Scene(ch.EntrySceneID.Int64); err == nil {
					entryCode = es.Code
				}
			}
		}
		unlocked := func(scene string) bool { return scene != "" && (visited[scene] || scene == entryCode) }

		sceneByNode := map[string]string{} // node id → scene code (để đối chiếu cạnh đã-đi)
		for _, n := range rm.Nodes {
			if n.Scene != "" {
				sceneByNode[n.ID] = n.Scene
			}
			locked := n.Locked
			if n.Kind == "video" {
				locked = !unlocked(n.Scene) // khóa theo tiến trình user
			}
			dto := MapNodeDTO{ID: n.ID, Kind: n.Kind, Scene: n.Scene, Title: n.Title, X: n.X, Y: n.Y, W: n.W, H: n.H, Locked: locked, Hotspot: n.Hotspot}
			if n.Poster != "" {
				u := media.SignURL(n.Poster, userID)
				dto.PosterURL = &u
			}
			out.Nodes = append(out.Nodes, dto)
		}
		// Cạnh "đã đi" = cả 2 node đầu-cuối đều là scene user đã ghé qua.
		for _, e := range rm.Edges {
			edge := MapEdgeDTO{From: e.From, To: e.To}
			if from, to := sceneByNode[e.From], sceneByNode[e.To]; from != "" && to != "" && visited[from] && visited[to] {
				edge.Chosen = true
			}
			out.Edges = append(out.Edges, edge)
		}
	}
	return out, nil
}
