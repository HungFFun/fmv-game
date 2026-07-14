package store

import (
	"database/sql"
	"errors"
	"fmt"
)

// ===== Admin write DAL (Phase 2 — CRUD nội dung) =====
//
// SQL portable SQLite↔Postgres (placeholder ?, ON CONFLICT/standard). Mutation
// nhiều bước gói trong transaction. Cascade xoá theo thứ tự FK-safe.

// nzs/nzi — ghi NULL khi rỗng (giữ cột optional gọn gàng).
func nzs(s string) any {
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

// ----- models -----

type ModelInput struct {
	Code         string `json:"code"`
	DisplayName  string `json:"displayName"`
	Avatar       string `json:"avatar"`
	Age          int    `json:"age"`
	Birthday     string `json:"birthday"`
	Relationship string `json:"relationship"`
	Occupation   string `json:"occupation"`
	HeightCm     int    `json:"heightCm"`
	WeightKg     int    `json:"weightKg"`
	Family       string `json:"family"`
	Bio          string `json:"bio"`
}

// CreateModel — tạo model + story-shell (draft) đi kèm (1 model ≈ 1 story) để
// publish/engine có chỗ bám. slug shell = code.
func (s *Store) CreateModel(in ModelInput) (int64, error) {
	var modelID int64
	err := s.tx(func(tx *sql.Tx) error {
		res, err := tx.Exec(
			`INSERT INTO models (code, display_name, avatar, age, birthday, relationship, occupation, height_cm, weight_kg, family, bio)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			in.Code, in.DisplayName, nzs(in.Avatar), nzi(in.Age), nzs(in.Birthday), nzs(in.Relationship),
			nzs(in.Occupation), nzi(in.HeightCm), nzi(in.WeightKg), nzs(in.Family), nzs(in.Bio))
		if err != nil {
			return err
		}
		modelID, _ = res.LastInsertId()
		title := in.DisplayName
		if title == "" {
			title = in.Code
		}
		_, err = tx.Exec(`INSERT INTO stories (model_id, slug, title, status) VALUES (?, ?, ?, 'draft')`,
			modelID, in.Code, title)
		return err
	})
	return modelID, err
}

// UpdateModel — PUT semantics: ghi đè toàn bộ field hồ sơ (code không đổi).
func (s *Store) UpdateModel(id int64, in ModelInput) error {
	res, err := s.DB.Exec(
		`UPDATE models SET display_name=?, avatar=?, age=?, birthday=?, relationship=?, occupation=?,
		   height_cm=?, weight_kg=?, family=?, bio=? WHERE id=?`,
		in.DisplayName, nzs(in.Avatar), nzi(in.Age), nzs(in.Birthday), nzs(in.Relationship),
		nzs(in.Occupation), nzi(in.HeightCm), nzi(in.WeightKg), nzs(in.Family), nzs(in.Bio), id)
	if err != nil {
		return err
	}
	return mustAffect(res)
}

// DeleteModel — xoá model + toàn bộ nội dung sở hữu (cascade FK-safe).
func (s *Store) DeleteModel(id int64) error {
	return s.tx(func(tx *sql.Tx) error {
		return DeleteModelContentTx(tx, id)
	})
}

// SetModelPublished — bật/tắt publish story-shell của model (publish 1 model = tắt các model khác).
func (s *Store) SetModelPublished(modelID int64, published bool) error {
	return s.tx(func(tx *sql.Tx) error {
		if published {
			if _, err := tx.Exec(`UPDATE stories SET status='draft' WHERE status='published'`); err != nil {
				return err
			}
			res, err := tx.Exec(`UPDATE stories SET status='published' WHERE model_id=?`, modelID)
			if err != nil {
				return err
			}
			return mustAffect(res)
		}
		_, err := tx.Exec(`UPDATE stories SET status='draft' WHERE model_id=?`, modelID)
		return err
	})
}

// ----- chapters -----

type ChapterInput struct {
	Idx        int    `json:"idx"`
	Title      string `json:"title"`
	IsFree     bool   `json:"isFree"`
	PriceCents int    `json:"priceCents"`
	SKU        string `json:"sku"`
	Poster     string `json:"poster"`
	MapJSON    string `json:"mapJson"` // layer layout (canvas + positions + edges)
}

func (s *Store) CreateChapter(modelID int64, in ChapterInput) (int64, error) {
	mapJSON := in.MapJSON
	if mapJSON == "" {
		mapJSON = "{}"
	}
	res, err := s.DB.Exec(
		`INSERT INTO chapters (model_id, idx, title, is_free, price_cents, sku, poster, map_json)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		modelID, in.Idx, in.Title, boolToInt(in.IsFree), in.PriceCents, nzs(in.SKU), nzs(in.Poster), mapJSON)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) UpdateChapter(id int64, in ChapterInput) error {
	mapJSON := in.MapJSON
	if mapJSON == "" {
		mapJSON = "{}"
	}
	res, err := s.DB.Exec(
		`UPDATE chapters SET idx=?, title=?, is_free=?, price_cents=?, sku=?, poster=?, map_json=? WHERE id=?`,
		in.Idx, in.Title, boolToInt(in.IsFree), in.PriceCents, nzs(in.SKU), nzs(in.Poster), mapJSON, id)
	if err != nil {
		return err
	}
	return mustAffect(res)
}

// SetChapterMap — cập nhật RIÊNG map_json (layout bản đồ) của 1 chapter.
// Đường ghi nhẹ cho editor kéo-thả layout, không đụng nội dung story.
func (s *Store) SetChapterMap(id int64, mapJSON string) error {
	if mapJSON == "" {
		mapJSON = "{}"
	}
	res, err := s.DB.Exec(`UPDATE chapters SET map_json=? WHERE id=?`, mapJSON, id)
	if err != nil {
		return err
	}
	return mustAffect(res)
}

// DeleteChapter — cascade nội dung riêng của chapter. Từ chối nếu có cạnh từ chapter
// KHÁC trỏ vào (tránh làm hỏng đồ thị liên-chapter).
func (s *Store) DeleteChapter(id int64) error {
	return s.tx(func(tx *sql.Tx) error {
		var inbound int
		err := tx.QueryRow(`SELECT COUNT(*) FROM scenes WHERE chapter_id != ? AND next_scene_id IN
			(SELECT id FROM scenes WHERE chapter_id = ?)`, id, id).Scan(&inbound)
		if err != nil {
			return err
		}
		var inboundChoice int
		err = tx.QueryRow(`SELECT COUNT(*) FROM choices WHERE next_scene_id IN (SELECT id FROM scenes WHERE chapter_id=?)
			AND scene_id IN (SELECT id FROM scenes WHERE chapter_id != ?)`, id, id).Scan(&inboundChoice)
		if err != nil {
			return err
		}
		if inbound+inboundChoice > 0 {
			return errInUse
		}
		execs := []struct {
			q    string
			args []any
		}{
			{`DELETE FROM entitlements WHERE chapter_id=?`, []any{id}},
			{`DELETE FROM gallery_items WHERE chapter_id=? OR unlock_scene_id IN (SELECT id FROM scenes WHERE chapter_id=?)`, []any{id, id}},
			{`DELETE FROM endings WHERE chapter_id=?`, []any{id}},
			{`DELETE FROM choices WHERE scene_id IN (SELECT id FROM scenes WHERE chapter_id=?)`, []any{id}},
			{`UPDATE scenes SET next_scene_id=NULL WHERE chapter_id=?`, []any{id}},
			{`DELETE FROM scenes WHERE chapter_id=?`, []any{id}},
			{`DELETE FROM chapter_videos WHERE chapter_id=?`, []any{id}},
			{`DELETE FROM chapters WHERE id=?`, []any{id}},
		}
		for _, e := range execs {
			if _, err := tx.Exec(e.q, e.args...); err != nil {
				return err
			}
		}
		return nil
	})
}

// ----- chapter_videos -----

type VideoInput struct {
	Code       string `json:"code"`
	Idx        int    `json:"idx"`
	Title      string `json:"title"`
	MediaID    int64  `json:"mediaId"`
	DurationMs int    `json:"durationMs"`
	Poster     string `json:"poster"`
}

func (s *Store) CreateChapterVideo(chapterID int64, in VideoInput) (int64, error) {
	res, err := s.DB.Exec(
		`INSERT INTO chapter_videos (chapter_id, code, idx, title, media_id, duration_ms, poster)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		chapterID, in.Code, in.Idx, in.Title, nzi64(in.MediaID), nzi(in.DurationMs), nzs(in.Poster))
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) UpdateChapterVideo(id int64, in VideoInput) error {
	res, err := s.DB.Exec(
		`UPDATE chapter_videos SET idx=?, title=?, media_id=?, duration_ms=?, poster=? WHERE id=?`,
		in.Idx, in.Title, nzi64(in.MediaID), nzi(in.DurationMs), nzs(in.Poster), id)
	if err != nil {
		return err
	}
	return mustAffect(res)
}

// DeleteChapterVideo — từ chối nếu còn scene tham chiếu (tránh node flow mất video).
func (s *Store) DeleteChapterVideo(id int64) error {
	var used int
	if err := s.DB.QueryRow(`SELECT COUNT(*) FROM scenes WHERE video_id=?`, id).Scan(&used); err != nil {
		return err
	}
	if used > 0 {
		return errInUse
	}
	res, err := s.DB.Exec(`DELETE FROM chapter_videos WHERE id=?`, id)
	if err != nil {
		return err
	}
	return mustAffect(res)
}

// ReorderChapterVideos — đặt lại idx theo thứ tự ids (2 pha tránh đụng UNIQUE(chapter_id,idx)).
func (s *Store) ReorderChapterVideos(chapterID int64, orderedIDs []int64) error {
	return s.tx(func(tx *sql.Tx) error {
		for i, id := range orderedIDs {
			if _, err := tx.Exec(`UPDATE chapter_videos SET idx=? WHERE id=? AND chapter_id=?`, -1-i, id, chapterID); err != nil {
				return err
			}
		}
		for i, id := range orderedIDs {
			if _, err := tx.Exec(`UPDATE chapter_videos SET idx=? WHERE id=? AND chapter_id=?`, i, id, chapterID); err != nil {
				return err
			}
		}
		return nil
	})
}

// Scenes — toàn bộ node flow của 1 chapter (cho admin flow GET).
func (s *Store) Scenes(chapterID int64) ([]Scene, error) {
	rows, err := s.DB.Query(`SELECT `+sceneCols+` FROM scenes WHERE chapter_id = ? ORDER BY id`, chapterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Scene
	for rows.Next() {
		sc, err := scanScene(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *sc)
	}
	return out, rows.Err()
}

// ===== helpers =====

var errInUse = errors.New("in use")

// ErrInUse — entity còn được tham chiếu, không thể xoá.
var ErrInUse = errInUse

func nzi64(i int64) any {
	if i == 0 {
		return nil
	}
	return i
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func mustAffect(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) tx(fn func(*sql.Tx) error) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// DeleteModelContentTx — xoá toàn bộ nội dung + runtime phụ thuộc của 1 model (FK-safe).
// Export để seed.Replace tái dùng trong tx của nó.
func DeleteModelContentTx(tx *sql.Tx, modelID int64) error {
	chapSub := `(SELECT id FROM chapters WHERE model_id=?)`
	sceneSub := `(SELECT s.id FROM scenes s JOIN chapters c ON c.id=s.chapter_id WHERE c.model_id=?)`
	steps := []string{
		`DELETE FROM entitlements WHERE chapter_id IN ` + chapSub,
		`DELETE FROM saves WHERE story_id IN (SELECT id FROM stories WHERE model_id=?)`,
		`DELETE FROM gallery_items WHERE model_id=?`,
		`DELETE FROM endings WHERE model_id=?`,
		`DELETE FROM choices WHERE scene_id IN ` + sceneSub,
		`UPDATE scenes SET next_scene_id=NULL WHERE chapter_id IN ` + chapSub,
		`DELETE FROM scenes WHERE chapter_id IN ` + chapSub,
		`DELETE FROM chapter_videos WHERE chapter_id IN ` + chapSub,
		`DELETE FROM chapters WHERE model_id=?`,
		`DELETE FROM characters WHERE model_id=?`,
		`DELETE FROM stories WHERE model_id=?`,
		`DELETE FROM models WHERE id=?`,
	}
	for _, q := range steps {
		if _, err := tx.Exec(q, modelID); err != nil {
			return fmt.Errorf("delete model content: %w", err)
		}
	}
	return nil
}
