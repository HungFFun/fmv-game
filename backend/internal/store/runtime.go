package store

import (
	"database/sql"
	"encoding/json"
	"errors"

	"fmv-game/backend/internal/engine"
)

// ===== Models (nhóm RUNTIME) =====

type Save struct {
	ID             int64
	UserID         int64
	Slot           int
	StoryID        int64
	CurrentSceneID int64
	State          engine.State
	UpdatedAt      string
}

// ===== users =====

func (s *Store) GetOrCreateUser(email string) (int64, error) {
	var id int64
	err := s.DB.QueryRow(`SELECT id FROM users WHERE email = ?`, email).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	res, err := s.DB.Exec(`INSERT INTO users (email, auth_provider) VALUES (?, 'dev')`, email)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) UserExists(id int64) (bool, error) {
	var one int
	err := s.DB.QueryRow(`SELECT 1 FROM users WHERE id = ?`, id).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// ===== entitlements =====

func (s *Store) IsEntitled(userID, chapterID int64) (bool, error) {
	var one int
	err := s.DB.QueryRow(`SELECT 1 FROM entitlements WHERE user_id = ? AND chapter_id = ?`, userID, chapterID).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (s *Store) GrantEntitlement(userID, chapterID int64, source string) error {
	_, err := s.DB.Exec(
		`INSERT OR IGNORE INTO entitlements (user_id, chapter_id, source) VALUES (?, ?, ?)`,
		userID, chapterID, source)
	return err
}

// ===== saves =====

func scanSave(row interface{ Scan(...any) error }) (*Save, error) {
	var sv Save
	var stateRaw string
	err := row.Scan(&sv.ID, &sv.UserID, &sv.Slot, &sv.StoryID, &sv.CurrentSceneID, &stateRaw, &sv.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	sv.State = engine.NewState()
	if err := json.Unmarshal([]byte(stateRaw), &sv.State); err != nil {
		return nil, err
	}
	if sv.State.Affinity == nil {
		sv.State.Affinity = map[string]int{}
	}
	if sv.State.Flags == nil {
		sv.State.Flags = map[string]bool{}
	}
	return &sv, nil
}

const saveCols = `id, user_id, slot, story_id, current_scene_id, state, updated_at`

func (s *Store) GetSave(userID int64, slot int) (*Save, error) {
	return scanSave(s.DB.QueryRow(`SELECT `+saveCols+` FROM saves WHERE user_id = ? AND slot = ?`, userID, slot))
}

func (s *Store) ListSaves(userID int64) ([]Save, error) {
	rows, err := s.DB.Query(`SELECT `+saveCols+` FROM saves WHERE user_id = ? ORDER BY slot`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Save
	for rows.Next() {
		sv, err := scanSave(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *sv)
	}
	return out, rows.Err()
}

func (s *Store) UpsertSave(userID int64, slot int, storyID, currentSceneID int64, state engine.State) error {
	raw, err := json.Marshal(state)
	if err != nil {
		return err
	}
	_, err = s.DB.Exec(`
		INSERT INTO saves (user_id, slot, story_id, current_scene_id, state, updated_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT (user_id, slot) DO UPDATE SET
		  story_id = excluded.story_id,
		  current_scene_id = excluded.current_scene_id,
		  state = excluded.state,
		  updated_at = CURRENT_TIMESTAMP`,
		userID, slot, storyID, currentSceneID, string(raw))
	return err
}

// ===== user_unlocks (vĩnh viễn xuyên lượt chơi) =====

func (s *Store) Unlock(userID int64, kind string, refID int64) error {
	_, err := s.DB.Exec(
		`INSERT OR IGNORE INTO user_unlocks (user_id, kind, ref_id) VALUES (?, ?, ?)`,
		userID, kind, refID)
	return err
}

func (s *Store) Unlocks(userID int64, kind string) (map[int64]bool, error) {
	rows, err := s.DB.Query(`SELECT ref_id FROM user_unlocks WHERE user_id = ? AND kind = ?`, userID, kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]bool{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}

// ===== analytics =====

func (s *Store) LogChoiceEvent(userID, sceneID, choiceID int64, state engine.State) error {
	raw, err := json.Marshal(state)
	if err != nil {
		return err
	}
	_, err = s.DB.Exec(
		`INSERT INTO choice_events (user_id, scene_id, choice_id, state_snapshot) VALUES (?, ?, ?, ?)`,
		userID, sceneID, choiceID, string(raw))
	return err
}

// VisitedScenes — tập scene code user đã GHÉ QUA trong 1 chapter, suy từ:
//   (A) scene có lựa chọn (choice_events.scene) + (B) scene đích của lựa chọn đó
//   (C) scene hiện tại trong autosave (slot 0) nếu thuộc chapter.
// Đủ để dựng đường đã đi: 1 cạnh coi là "đã đi" khi CẢ 2 đầu đều nằm trong tập này
// (bao gồm cả bước linear giữa các choice, vì scene linear là đích của choice trước đó).
func (s *Store) VisitedScenes(userID, chapterID int64) (map[string]bool, error) {
	visited := map[string]bool{}
	rows, err := s.DB.Query(
		`SELECT s.code, ns.code
		   FROM choice_events ce
		   JOIN choices c  ON c.id = ce.choice_id
		   JOIN scenes  s  ON s.id = ce.scene_id
		   JOIN scenes  ns ON ns.id = c.next_scene_id
		  WHERE ce.user_id = ? AND s.chapter_id = ?`,
		userID, chapterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var from, to string
		if err := rows.Scan(&from, &to); err != nil {
			return nil, err
		}
		visited[from] = true
		visited[to] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// (C) scene hiện tại (kết thúc đường đi) — vd cạnh cuối tới ending.
	var cur string
	err = s.DB.QueryRow(
		`SELECT sc.code FROM saves sv
		   JOIN scenes sc ON sc.id = sv.current_scene_id
		  WHERE sv.user_id = ? AND sv.slot = 0 AND sc.chapter_id = ?`,
		userID, chapterID).Scan(&cur)
	if err == nil {
		visited[cur] = true
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	return visited, nil
}
