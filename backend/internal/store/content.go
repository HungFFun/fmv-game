package store

import (
	"database/sql"
	"errors"
)

// ===== Models (nhóm CONTENT) =====

// Model — persona top-level sở hữu chapters (hồ sơ màn Album).
type Model struct {
	ID           int64
	Code         string
	DisplayName  string
	Avatar       sql.NullString
	Age          sql.NullInt64
	Birthday     sql.NullString
	Relationship sql.NullString
	Occupation   sql.NullString
	HeightCm     sql.NullInt64
	WeightKg     sql.NullInt64
	Family       sql.NullString
	Bio          sql.NullString
	ProfileJSON  string
}

type Story struct {
	ID      int64
	ModelID int64
	Slug    string
	Title   string
}

type Character struct {
	ID          int64
	ModelID     int64
	Code        string
	DisplayName string
	Archetype   string
}

type Chapter struct {
	ID           int64
	ModelID      int64
	Idx          int
	Title        string
	EntrySceneID sql.NullInt64
	IsFree       bool
	PriceCents   int
	SKU          sql.NullString
	Poster       sql.NullString // storage key ảnh thẻ chapter (màn select)
	MapJSON      string         // layer layout bản đồ nhánh cho màn map
}

// ChapterVideo — 1 video trong danh sách phẳng của chapter (scenes.video_id trỏ vào đây).
type ChapterVideo struct {
	ID         int64
	ChapterID  int64
	Code       string
	Idx        int
	Title      string
	MediaID    sql.NullInt64
	DurationMs sql.NullInt64
	Poster     sql.NullString
}

type MediaAsset struct {
	ID          int64
	Kind        string
	StorageKey  string
	HLSManifest sql.NullString
	DurationMs  sql.NullInt64
}

type Scene struct {
	ID           int64
	ChapterID    int64
	Code         string
	Type         string // 'linear' | 'choice' | 'ending'
	VideoID      sql.NullInt64
	NextSceneID  sql.NullInt64
	OnEnterRaw   string // Effects JSON
	IsCheckpoint bool
}

type Choice struct {
	ID              int64
	SceneID         int64
	Idx             int
	Label           string
	ConditionRaw    string // Condition JSON
	EffectsRaw      string // Effects JSON
	NextSceneID     sql.NullInt64
	TimerMs         sql.NullInt64
	DefaultChoiceID sql.NullInt64
	Hotspot         sql.NullString // optional {x,y,w,h,style} JSON — vùng bấm trên khung video
}

type Ending struct {
	ID        int64
	ModelID   int64
	ChapterID sql.NullInt64
	SceneID   int64
	Code      string
	Title     string
	Rank      sql.NullString
}

type GalleryItem struct {
	ID            int64
	ModelID       int64
	ChapterID     sql.NullInt64
	EndingID      sql.NullInt64
	MediaID       int64
	Title         sql.NullString
	UnlockSceneID sql.NullInt64
	IsBonus       bool
}

var ErrNotFound = errors.New("not found")

// ===== Queries: models =====

const modelCols = `id, code, display_name, avatar, age, birthday, relationship, occupation, height_cm, weight_kg, family, bio, COALESCE(profile_json,'{}')`

func scanModel(row interface{ Scan(...any) error }) (*Model, error) {
	var m Model
	err := row.Scan(&m.ID, &m.Code, &m.DisplayName, &m.Avatar, &m.Age, &m.Birthday,
		&m.Relationship, &m.Occupation, &m.HeightCm, &m.WeightKg, &m.Family, &m.Bio, &m.ProfileJSON)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &m, nil
}

func (s *Store) Model(id int64) (*Model, error) {
	return scanModel(s.DB.QueryRow(`SELECT `+modelCols+` FROM models WHERE id = ?`, id))
}

func (s *Store) ModelByCode(code string) (*Model, error) {
	return scanModel(s.DB.QueryRow(`SELECT `+modelCols+` FROM models WHERE code = ?`, code))
}

func (s *Store) Models() ([]Model, error) {
	rows, err := s.DB.Query(`SELECT ` + modelCols + ` FROM models ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Model
	for rows.Next() {
		m, err := scanModel(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}

// PublishedModel — model sở hữu story đang published.
func (s *Store) PublishedModel() (*Model, error) {
	row := s.DB.QueryRow(`SELECT ` + modelCols + ` FROM models WHERE id = (
		SELECT model_id FROM stories WHERE status = 'published' ORDER BY id LIMIT 1)`)
	return scanModel(row)
}

// ===== Queries: story / characters =====

func (s *Store) PublishedStory() (*Story, error) {
	row := s.DB.QueryRow(`SELECT id, COALESCE(model_id,0), slug, title FROM stories WHERE status = 'published' ORDER BY id LIMIT 1`)
	var st Story
	if err := row.Scan(&st.ID, &st.ModelID, &st.Slug, &st.Title); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &st, nil
}

func (s *Store) Characters(modelID int64) ([]Character, error) {
	rows, err := s.DB.Query(
		`SELECT id, model_id, code, display_name, COALESCE(archetype,'') FROM characters WHERE model_id = ? ORDER BY id`, modelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Character
	for rows.Next() {
		var c Character
		if err := rows.Scan(&c.ID, &c.ModelID, &c.Code, &c.DisplayName, &c.Archetype); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ===== Queries: chapters =====

func scanChapter(row interface{ Scan(...any) error }) (*Chapter, error) {
	var c Chapter
	var isFree int
	err := row.Scan(&c.ID, &c.ModelID, &c.Idx, &c.Title, &c.EntrySceneID, &isFree, &c.PriceCents, &c.SKU, &c.Poster, &c.MapJSON)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	c.IsFree = isFree != 0
	return &c, nil
}

const chapterCols = `id, model_id, idx, title, entry_scene_id, is_free, price_cents, sku, poster, COALESCE(map_json,'{}')`

func (s *Store) Chapter(id int64) (*Chapter, error) {
	return scanChapter(s.DB.QueryRow(`SELECT `+chapterCols+` FROM chapters WHERE id = ?`, id))
}

func (s *Store) Chapters(modelID int64) ([]Chapter, error) {
	rows, err := s.DB.Query(`SELECT `+chapterCols+` FROM chapters WHERE model_id = ? ORDER BY idx`, modelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Chapter
	for rows.Next() {
		c, err := scanChapter(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

// ===== Queries: chapter_videos =====

const chapterVideoCols = `id, chapter_id, code, idx, title, media_id, duration_ms, poster`

func scanChapterVideo(row interface{ Scan(...any) error }) (*ChapterVideo, error) {
	var v ChapterVideo
	err := row.Scan(&v.ID, &v.ChapterID, &v.Code, &v.Idx, &v.Title, &v.MediaID, &v.DurationMs, &v.Poster)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &v, nil
}

func (s *Store) ChapterVideo(id int64) (*ChapterVideo, error) {
	return scanChapterVideo(s.DB.QueryRow(`SELECT `+chapterVideoCols+` FROM chapter_videos WHERE id = ?`, id))
}

func (s *Store) ChapterVideos(chapterID int64) ([]ChapterVideo, error) {
	rows, err := s.DB.Query(`SELECT `+chapterVideoCols+` FROM chapter_videos WHERE chapter_id = ? ORDER BY idx`, chapterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ChapterVideo
	for rows.Next() {
		v, err := scanChapterVideo(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}

// VideoMedia — resolve media_assets phía dưới 1 chapter_videos (1 hop, dùng cho videoURL).
func (s *Store) VideoMedia(videoID int64) (*MediaAsset, error) {
	row := s.DB.QueryRow(`SELECT m.id, m.kind, m.storage_key, m.hls_manifest, m.duration_ms
		FROM chapter_videos cv JOIN media_assets m ON m.id = cv.media_id WHERE cv.id = ?`, videoID)
	var m MediaAsset
	if err := row.Scan(&m.ID, &m.Kind, &m.StorageKey, &m.HLSManifest, &m.DurationMs); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &m, nil
}

// ===== Queries: scenes / choices =====

const sceneCols = `id, chapter_id, code, type, video_id, next_scene_id, COALESCE(on_enter,'{}'), is_checkpoint`

func scanScene(row interface{ Scan(...any) error }) (*Scene, error) {
	var sc Scene
	var checkpoint int
	err := row.Scan(&sc.ID, &sc.ChapterID, &sc.Code, &sc.Type, &sc.VideoID, &sc.NextSceneID, &sc.OnEnterRaw, &checkpoint)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	sc.IsCheckpoint = checkpoint != 0
	return &sc, nil
}

func (s *Store) Scene(id int64) (*Scene, error) {
	return scanScene(s.DB.QueryRow(`SELECT `+sceneCols+` FROM scenes WHERE id = ?`, id))
}

func (s *Store) SceneByCode(code string) (*Scene, error) {
	return scanScene(s.DB.QueryRow(`SELECT `+sceneCols+` FROM scenes WHERE code = ?`, code))
}

const choiceCols = `id, scene_id, idx, label, COALESCE(condition,'{}'), COALESCE(effects,'{}'), next_scene_id, timer_ms, default_choice_id, hotspot`

func scanChoice(row interface{ Scan(...any) error }) (*Choice, error) {
	var c Choice
	err := row.Scan(&c.ID, &c.SceneID, &c.Idx, &c.Label, &c.ConditionRaw, &c.EffectsRaw, &c.NextSceneID, &c.TimerMs, &c.DefaultChoiceID, &c.Hotspot)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (s *Store) Choice(id int64) (*Choice, error) {
	return scanChoice(s.DB.QueryRow(`SELECT `+choiceCols+` FROM choices WHERE id = ?`, id))
}

func (s *Store) ChoicesForScene(sceneID int64) ([]Choice, error) {
	rows, err := s.DB.Query(`SELECT `+choiceCols+` FROM choices WHERE scene_id = ? ORDER BY idx`, sceneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Choice
	for rows.Next() {
		c, err := scanChoice(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

// ===== Queries: media / endings / gallery =====

func (s *Store) Media(id int64) (*MediaAsset, error) {
	row := s.DB.QueryRow(`SELECT id, kind, storage_key, hls_manifest, duration_ms FROM media_assets WHERE id = ?`, id)
	var m MediaAsset
	if err := row.Scan(&m.ID, &m.Kind, &m.StorageKey, &m.HLSManifest, &m.DurationMs); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &m, nil
}

func (s *Store) EndingByScene(sceneID int64) (*Ending, error) {
	row := s.DB.QueryRow(`SELECT id, model_id, chapter_id, scene_id, code, title, rank FROM endings WHERE scene_id = ?`, sceneID)
	var e Ending
	if err := row.Scan(&e.ID, &e.ModelID, &e.ChapterID, &e.SceneID, &e.Code, &e.Title, &e.Rank); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &e, nil
}

const galleryCols = `id, model_id, chapter_id, ending_id, media_id, title, unlock_scene_id, is_bonus`

func (s *Store) GalleryItemsByUnlockScene(sceneID int64) ([]GalleryItem, error) {
	return s.galleryQuery(`SELECT `+galleryCols+` FROM gallery_items WHERE unlock_scene_id = ?`, sceneID)
}

func (s *Store) GalleryItems(modelID int64) ([]GalleryItem, error) {
	return s.galleryQuery(`SELECT `+galleryCols+` FROM gallery_items WHERE model_id = ? ORDER BY id`, modelID)
}

func (s *Store) galleryQuery(query string, arg any) ([]GalleryItem, error) {
	rows, err := s.DB.Query(query, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GalleryItem
	for rows.Next() {
		var g GalleryItem
		var bonus int
		if err := rows.Scan(&g.ID, &g.ModelID, &g.ChapterID, &g.EndingID, &g.MediaID, &g.Title, &g.UnlockSceneID, &bonus); err != nil {
			return nil, err
		}
		g.IsBonus = bonus != 0
		out = append(out, g)
	}
	return out, rows.Err()
}
