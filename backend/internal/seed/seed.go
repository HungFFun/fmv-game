// Package seed — nạp kịch bản từ file JSON authoring vào bảng CONTENT.
//
// Format authoring dùng CODE (không dùng id số) để biên kịch tham chiếu;
// loader chạy 2 pass: (1) insert toàn bộ scene, (2) nối cạnh theo code.
// Đây cũng chính là chỗ cắm script compile Ink/Twine → JSON này sau.
package seed

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"fmv-game/backend/internal/engine"
	"fmv-game/backend/internal/store"
)

// ModelDef — persona top-level (hồ sơ màn Album); SỞ HỮU chapters.
type ModelDef struct {
	Code         string `json:"code"`
	DisplayName  string `json:"display_name"`
	Avatar       string `json:"avatar,omitempty"`
	Age          int    `json:"age,omitempty"`
	Birthday     string `json:"birthday,omitempty"`
	Relationship string `json:"relationship,omitempty"`
	Occupation   string `json:"occupation,omitempty"`
	HeightCm     int    `json:"height_cm,omitempty"`
	WeightKg     int    `json:"weight_kg,omitempty"`
	Family       string `json:"family,omitempty"`
	Bio          string `json:"bio,omitempty"`
}

type MediaDef struct {
	Key        string `json:"key"`  // tham chiếu nội bộ file JSON
	Kind       string `json:"kind"` // video | image
	File       string `json:"file"` // "/media/xxx.mp4" (dev)
	DurationMs int    `json:"duration_ms"`
}

// VideoDef — 1 video trong danh sách phẳng của chapter; flow (scene) tham chiếu theo code.
type VideoDef struct {
	Code       string `json:"code"`
	Idx        int    `json:"idx"`
	Title      string `json:"title"`
	Media      string `json:"media"` // media key
	Poster     string `json:"poster,omitempty"`
	DurationMs int    `json:"duration_ms,omitempty"`
}

type ChoiceDef struct {
	Label     string          `json:"label"`
	Condition json.RawMessage `json:"condition,omitempty"`
	Effects   json.RawMessage `json:"effects,omitempty"`
	Next      string          `json:"next"` // scene code
	TimerMs   int             `json:"timer_ms,omitempty"`
	Default   bool            `json:"default,omitempty"` // chọn nếu hết giờ
	Hotspot   *HotspotDef     `json:"hotspot,omitempty"` // vùng bấm trên khung video (choice định vị)
}

// HotspotDef — vùng bấm định vị trên khung video (toạ độ 0..1 tương đối). Style
// chọn kiểu hiển thị phía client ('door' = viền glow, 'marker' = chỉ dấu ❗).
type HotspotDef struct {
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	W     float64 `json:"w"`
	H     float64 `json:"h"`
	Style string  `json:"style,omitempty"`
}

type SceneDef struct {
	Code       string          `json:"code"`
	Type       string          `json:"type"`            // linear | choice | ending
	Video      string          `json:"video,omitempty"` // VIDEO CODE (chapter_videos.code)
	Next       string          `json:"next,omitempty"`  // scene code (linear)
	OnEnter    json.RawMessage `json:"on_enter,omitempty"`
	Checkpoint bool            `json:"checkpoint,omitempty"`
	Choices    []ChoiceDef     `json:"choices,omitempty"`
}

type ChapterDef struct {
	Idx        int             `json:"idx"`
	Title      string          `json:"title"`
	IsFree     bool            `json:"is_free"`
	PriceCents int             `json:"price_cents"`
	SKU        string          `json:"sku,omitempty"`
	Entry      string          `json:"entry"`            // scene code
	Poster     string          `json:"poster,omitempty"` // storage key ảnh thẻ chapter
	Map        json.RawMessage `json:"map,omitempty"`    // layer layout bản đồ nhánh
	Videos     []VideoDef      `json:"videos"`           // danh sách video phẳng của chapter
	Scenes     []SceneDef      `json:"scenes"`
}

type EndingDef struct {
	Scene string `json:"scene"` // scene code
	Code  string `json:"code"`
	Title string `json:"title"`
	Rank  string `json:"rank"`
}

type GalleryDef struct {
	Title       string `json:"title"`
	Media       string `json:"media"`        // media key
	UnlockScene string `json:"unlock_scene"` // scene code
	IsBonus     bool   `json:"is_bonus,omitempty"`
}

type CharacterDef struct {
	Code        string `json:"code"`
	DisplayName string `json:"display_name"`
	Archetype   string `json:"archetype"`
}

type StoryFile struct {
	Model ModelDef `json:"model"`
	Story struct {
		Slug  string `json:"slug"`
		Title string `json:"title"`
	} `json:"story"`
	Characters []CharacterDef `json:"characters"`
	Media      []MediaDef     `json:"media"`
	Chapters   []ChapterDef   `json:"chapters"`
	Endings    []EndingDef    `json:"endings"`
	Gallery    []GalleryDef   `json:"gallery"`
}

func rawOrEmpty(r json.RawMessage) string {
	if len(r) == 0 {
		return "{}"
	}
	return string(r)
}

// nz/nzi — trả nil để ghi NULL khi giá trị rỗng (hồ sơ optional gọn gàng).
func nz(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nzi(i int) any {
	if i == 0 {
		return nil
	}
	return i
}

// validateDSL parse thử mọi condition/effects để chặn lỗi data từ lúc seed.
func validateDSL(sf *StoryFile) error {
	for _, ch := range sf.Chapters {
		for _, sc := range ch.Scenes {
			if _, err := engine.ParseEffects(rawOrEmpty(sc.OnEnter)); err != nil {
				return fmt.Errorf("scene %s on_enter: %w", sc.Code, err)
			}
			for i, c := range sc.Choices {
				if _, err := engine.ParseCondition(rawOrEmpty(c.Condition)); err != nil {
					return fmt.Errorf("scene %s choice %d condition: %w", sc.Code, i, err)
				}
				if _, err := engine.ParseEffects(rawOrEmpty(c.Effects)); err != nil {
					return fmt.Errorf("scene %s choice %d effects: %w", sc.Code, i, err)
				}
			}
		}
	}
	return nil
}

// LoadFile đọc file JSON rồi nạp vào DB (idempotent theo slug: đã có thì bỏ qua).
func LoadFile(st *store.Store, path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var sf StoryFile
	if err := json.Unmarshal(raw, &sf); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return Load(st, &sf)
}

// Load — nạp story mới (idempotent theo slug: đã có thì bỏ qua).
func Load(st *store.Store, sf *StoryFile) error {
	if err := validateDSL(sf); err != nil {
		return err
	}
	var exists int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM stories WHERE slug = ?`, sf.Story.Slug).Scan(&exists); err != nil {
		return err
	}
	if exists > 0 {
		return nil // đã seed
	}
	tx, err := st.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := insertContent(tx, sf); err != nil {
		return err
	}
	return tx.Commit()
}

// Replace — admin import: thay TOÀN BỘ nội dung của model (theo slug) bằng StoryFile mới.
// Xoá nội dung cũ (FK-safe) rồi insert lại — đây là đường ghi an toàn cho đồ thị flow
// (whole-graph replace, tránh lỗi cạnh liên-chapter khi sửa từng phần).
func Replace(st *store.Store, sf *StoryFile) error {
	if err := validateDSL(sf); err != nil {
		return err
	}
	tx, err := st.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var modelID sql.NullInt64
	err = tx.QueryRow(`SELECT model_id FROM stories WHERE slug = ?`, sf.Story.Slug).Scan(&modelID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if modelID.Valid {
		if err := store.DeleteModelContentTx(tx, modelID.Int64); err != nil {
			return err
		}
	}
	if err := insertContent(tx, sf); err != nil {
		return err
	}
	return tx.Commit()
}

// insertContent — 2-pass insert toàn bộ nội dung trong 1 tx (KHÔNG commit; caller commit).
func insertContent(tx *sql.Tx, sf *StoryFile) error {
	// model (owner) → story (publish shell) → characters.
	m := sf.Model
	res, err := tx.Exec(
		`INSERT INTO models (code, display_name, avatar, age, birthday, relationship, occupation, height_cm, weight_kg, family, bio)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.Code, m.DisplayName, nz(m.Avatar), nzi(m.Age), nz(m.Birthday), nz(m.Relationship),
		nz(m.Occupation), nzi(m.HeightCm), nzi(m.WeightKg), nz(m.Family), nz(m.Bio))
	if err != nil {
		return err
	}
	modelID, _ := res.LastInsertId()

	if _, err := tx.Exec(`INSERT INTO stories (model_id, slug, title, status) VALUES (?, ?, ?, 'published')`,
		modelID, sf.Story.Slug, sf.Story.Title); err != nil {
		return err
	}

	for _, c := range sf.Characters {
		if _, err := tx.Exec(
			`INSERT INTO characters (model_id, code, display_name, archetype) VALUES (?, ?, ?, ?)`,
			modelID, c.Code, c.DisplayName, c.Archetype); err != nil {
			return err
		}
	}

	mediaIDs := map[string]int64{}
	for _, md := range sf.Media {
		res, err := tx.Exec(
			`INSERT INTO media_assets (kind, storage_key, duration_ms, drm_protected) VALUES (?, ?, ?, 0)`,
			md.Kind, md.File, md.DurationMs)
		if err != nil {
			return err
		}
		mediaIDs[md.Key], _ = res.LastInsertId()
	}

	// Pass 1: insert chapters → chapter_videos → scenes (chưa nối cạnh).
	chapterIDs := map[int]int64{}
	sceneIDs := map[string]int64{}      // scene code → id (code unique toàn story trong format này)
	sceneChapter := map[string]int64{}  // scene code → chapter id (cho ending/gallery)
	for _, ch := range sf.Chapters {
		res, err := tx.Exec(
			`INSERT INTO chapters (model_id, idx, title, is_free, price_cents, sku, poster, map_json) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			modelID, ch.Idx, ch.Title, boolInt(ch.IsFree), ch.PriceCents, ch.SKU, nz(ch.Poster), rawOrEmpty(ch.Map))
		if err != nil {
			return err
		}
		chapterID, _ := res.LastInsertId()
		chapterIDs[ch.Idx] = chapterID

		// chapter_videos: danh sách video phẳng của chapter này (code scope theo chapter).
		chVideoIDs := map[string]int64{}
		for _, v := range ch.Videos {
			mediaID, ok := mediaIDs[v.Media]
			if !ok {
				return fmt.Errorf("chapter %d video %s: media key %q không tồn tại", ch.Idx, v.Code, v.Media)
			}
			res, err := tx.Exec(
				`INSERT INTO chapter_videos (chapter_id, code, idx, title, media_id, duration_ms, poster) VALUES (?, ?, ?, ?, ?, ?, ?)`,
				chapterID, v.Code, v.Idx, v.Title, mediaID, nzi(v.DurationMs), nz(v.Poster))
			if err != nil {
				return err
			}
			chVideoIDs[v.Code], _ = res.LastInsertId()
		}

		for _, sc := range ch.Scenes {
			if _, dup := sceneIDs[sc.Code]; dup {
				return fmt.Errorf("scene code trùng: %s", sc.Code)
			}
			var videoID any
			if sc.Video != "" {
				id, ok := chVideoIDs[sc.Video]
				if !ok {
					return fmt.Errorf("scene %s: video code %q không có trong chapter %d", sc.Code, sc.Video, ch.Idx)
				}
				videoID = id
			}
			res, err := tx.Exec(
				`INSERT INTO scenes (chapter_id, code, type, video_id, on_enter, is_checkpoint) VALUES (?, ?, ?, ?, ?, ?)`,
				chapterID, sc.Code, sc.Type, videoID, rawOrEmpty(sc.OnEnter), boolInt(sc.Checkpoint))
			if err != nil {
				return err
			}
			sceneIDs[sc.Code], _ = res.LastInsertId()
			sceneChapter[sc.Code] = chapterID
		}
	}

	resolve := func(code, ctx string) (int64, error) {
		id, ok := sceneIDs[code]
		if !ok {
			return 0, fmt.Errorf("%s: scene code %q không tồn tại", ctx, code)
		}
		return id, nil
	}

	// Pass 2: nối cạnh — entry, next, choices.
	for _, ch := range sf.Chapters {
		entryID, err := resolve(ch.Entry, fmt.Sprintf("chapter %d entry", ch.Idx))
		if err != nil {
			return err
		}
		if _, err := tx.Exec(`UPDATE chapters SET entry_scene_id = ? WHERE id = ?`, entryID, chapterIDs[ch.Idx]); err != nil {
			return err
		}

		for _, sc := range ch.Scenes {
			switch sc.Type {
			case "linear":
				if sc.Next == "" {
					return fmt.Errorf("scene linear %s thiếu next (dead end)", sc.Code)
				}
				nextID, err := resolve(sc.Next, "scene "+sc.Code)
				if err != nil {
					return err
				}
				if _, err := tx.Exec(`UPDATE scenes SET next_scene_id = ? WHERE id = ?`, nextID, sceneIDs[sc.Code]); err != nil {
					return err
				}
			case "choice":
				if len(sc.Choices) == 0 {
					return fmt.Errorf("scene choice %s không có choice nào", sc.Code)
				}
				var defaultID int64
				ids := make([]int64, len(sc.Choices))
				for i, c := range sc.Choices {
					nextID, err := resolve(c.Next, fmt.Sprintf("scene %s choice %d", sc.Code, i))
					if err != nil {
						return err
					}
					var timer any
					if c.TimerMs > 0 {
						timer = c.TimerMs
					}
					var hotspot any
					if c.Hotspot != nil {
						b, err := json.Marshal(c.Hotspot)
						if err != nil {
							return fmt.Errorf("scene %s choice %d hotspot: %w", sc.Code, i, err)
						}
						hotspot = string(b)
					}
					res, err := tx.Exec(
						`INSERT INTO choices (scene_id, idx, label, condition, effects, next_scene_id, timer_ms, hotspot) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
						sceneIDs[sc.Code], i, c.Label, rawOrEmpty(c.Condition), rawOrEmpty(c.Effects), nextID, timer, hotspot)
					if err != nil {
						return err
					}
					ids[i], _ = res.LastInsertId()
					if c.Default {
						defaultID = ids[i]
					}
				}
				if defaultID != 0 {
					// default_choice_id ghi trên MỌI choice của scene (chọn gì nếu hết giờ)
					for _, id := range ids {
						if _, err := tx.Exec(`UPDATE choices SET default_choice_id = ? WHERE id = ?`, defaultID, id); err != nil {
							return err
						}
					}
				}
			case "ending":
				// không có cạnh đi ra
			default:
				return fmt.Errorf("scene %s: type %q không hợp lệ", sc.Code, sc.Type)
			}
		}
	}

	// endings: model_id + chapter_id (suy từ scene). Lưu ending theo scene để gắn nhãn gallery.
	endingByScene := map[string]int64{}
	for _, e := range sf.Endings {
		sceneID, err := resolve(e.Scene, "ending "+e.Code)
		if err != nil {
			return err
		}
		res, err := tx.Exec(
			`INSERT INTO endings (model_id, chapter_id, scene_id, code, title, rank) VALUES (?, ?, ?, ?, ?, ?)`,
			modelID, sceneChapter[e.Scene], sceneID, e.Code, e.Title, e.Rank)
		if err != nil {
			return err
		}
		endingByScene[e.Scene], _ = res.LastInsertId()
	}

	// gallery: nhãn "Obtainable in…" suy từ unlock scene — ending scene → ending_id, còn lại → chapter_id.
	for _, g := range sf.Gallery {
		sceneID, err := resolve(g.UnlockScene, "gallery "+g.Title)
		if err != nil {
			return err
		}
		mediaID, ok := mediaIDs[g.Media]
		if !ok {
			return fmt.Errorf("gallery %s: media key %q không tồn tại", g.Title, g.Media)
		}
		var chapterID, endingID any
		if eid, isEnding := endingByScene[g.UnlockScene]; isEnding {
			endingID = eid
		} else {
			chapterID = sceneChapter[g.UnlockScene]
		}
		if _, err := tx.Exec(
			`INSERT INTO gallery_items (model_id, chapter_id, ending_id, media_id, title, unlock_scene_id, is_bonus) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			modelID, chapterID, endingID, mediaID, g.Title, sceneID, boolInt(g.IsBonus)); err != nil {
			return err
		}
	}

	return nil
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
