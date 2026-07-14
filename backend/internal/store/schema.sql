-- SQLite schema (dev). Mirror 1:1 của db/postgres-schema.sql (prod).
-- JSONB (Postgres) → TEXT chứa JSON (SQLite).

-- ===== CONTENT (biên kịch tạo) =====

-- models — persona top-level (vd "Hana"), SỞ HỮU chapters. Hồ sơ lấy từ màn Album.
CREATE TABLE IF NOT EXISTS models (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  code         TEXT UNIQUE NOT NULL,         -- 'hana'
  display_name TEXT NOT NULL,                -- Album: name
  avatar       TEXT,                         -- storage key ảnh đại diện
  age          INTEGER,                      -- Album: Age
  birthday     TEXT,                         -- Album: Birthday (free-text)
  relationship TEXT,                         -- Album: Relationship
  occupation   TEXT,                         -- Album: Occupation
  height_cm    INTEGER,                      -- Album: Physical / Height
  weight_kg    INTEGER,                      -- Album: Physical / Weight
  family       TEXT,                         -- Album: Family
  bio          TEXT,
  profile_json TEXT DEFAULT '{}',            -- field hồ sơ mở rộng (future-proof)
  created_at   TEXT DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS stories (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  model_id   INTEGER REFERENCES models(id),  -- shell publish của 1 model
  slug       TEXT UNIQUE NOT NULL,
  title      TEXT NOT NULL,
  season     INTEGER DEFAULT 1,
  status     TEXT DEFAULT 'draft',           -- draft | published
  created_at TEXT DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS characters (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  model_id     INTEGER REFERENCES models(id),
  code         TEXT NOT NULL,                -- 'malsook', 'minjung'... (affinity key)
  display_name TEXT NOT NULL,
  archetype    TEXT,
  meta         TEXT DEFAULT '{}',
  UNIQUE (model_id, code)
);

CREATE TABLE IF NOT EXISTS chapters (
  id             INTEGER PRIMARY KEY AUTOINCREMENT,
  model_id       INTEGER REFERENCES models(id),
  idx            INTEGER NOT NULL,
  title          TEXT NOT NULL,
  entry_scene_id INTEGER,                    -- scene đầu chương
  is_free        INTEGER DEFAULT 0,
  price_cents    INTEGER DEFAULT 0,
  sku            TEXT,
  poster         TEXT,                        -- storage key ảnh thẻ chapter (màn select)
  map_json       TEXT DEFAULT '{}',           -- LAYER layout: canvas + vị trí node + cạnh trang trí
  UNIQUE (model_id, idx)
);

CREATE TABLE IF NOT EXISTS media_assets (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  kind          TEXT NOT NULL,               -- 'video' | 'image' | 'audio'
  storage_key   TEXT NOT NULL,               -- dev: file dưới backend/media; prod: key S3/R2
  hls_manifest  TEXT,
  duration_ms   INTEGER,
  drm_protected INTEGER DEFAULT 0,
  meta          TEXT DEFAULT '{}'
);

-- chapter_videos — DANH SÁCH VIDEO PHẲNG của mỗi chapter (title hiển thị trên map/Album).
-- scenes.video_id trỏ vào đây; media_assets là lớp lưu trữ phía dưới.
CREATE TABLE IF NOT EXISTS chapter_videos (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  chapter_id  INTEGER REFERENCES chapters(id),
  code        TEXT NOT NULL,                 -- 'v_itaewon' — ref khi seed/đấu flow
  idx         INTEGER NOT NULL,              -- thứ tự trong danh sách phẳng
  title       TEXT NOT NULL,                 -- "A Long Day in Itaewon"
  media_id    INTEGER REFERENCES media_assets(id),
  duration_ms INTEGER,
  poster      TEXT,                          -- storage key thumbnail
  UNIQUE (chapter_id, code),
  UNIQUE (chapter_id, idx)
);

CREATE TABLE IF NOT EXISTS scenes (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  chapter_id    INTEGER REFERENCES chapters(id),
  code          TEXT NOT NULL,               -- 'ch1_kitchen_01'
  type          TEXT NOT NULL,               -- 'linear' | 'choice' | 'ending'
  video_id      INTEGER REFERENCES chapter_videos(id),  -- NODE flow → video phẳng
  next_scene_id INTEGER REFERENCES scenes(id),
  on_enter      TEXT DEFAULT '{}',           -- Effects JSON
  is_checkpoint INTEGER DEFAULT 0,
  UNIQUE (chapter_id, code)
);

CREATE TABLE IF NOT EXISTS choices (
  id                INTEGER PRIMARY KEY AUTOINCREMENT,
  scene_id          INTEGER REFERENCES scenes(id),
  idx               INTEGER NOT NULL,
  label             TEXT NOT NULL,
  condition         TEXT DEFAULT '{}',       -- Condition JSON
  effects           TEXT DEFAULT '{}',       -- Effects JSON
  next_scene_id     INTEGER REFERENCES scenes(id),
  timer_ms          INTEGER,
  default_choice_id INTEGER,
  hotspot           TEXT,                    -- optional {x,y,w,h,style} JSON — vùng bấm trên khung video (hotspot)
  UNIQUE (scene_id, idx)
);

CREATE TABLE IF NOT EXISTS endings (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  model_id   INTEGER REFERENCES models(id),
  chapter_id INTEGER REFERENCES chapters(id),  -- ending thuộc chương nào (nhãn Album)
  scene_id   INTEGER REFERENCES scenes(id),
  code       TEXT NOT NULL,                    -- 'malsook_good', 'true_end'...
  title      TEXT NOT NULL,
  rank       TEXT                              -- 'good' | 'normal' | 'bad' | 'true'
);

CREATE TABLE IF NOT EXISTS gallery_items (
  id              INTEGER PRIMARY KEY AUTOINCREMENT,
  model_id        INTEGER REFERENCES models(id),
  chapter_id      INTEGER REFERENCES chapters(id),  -- "Obtainable in Chapter N"
  ending_id       INTEGER REFERENCES endings(id),   -- "Obtainable in Ending …"
  media_id        INTEGER REFERENCES media_assets(id),
  title           TEXT,
  unlock_scene_id INTEGER REFERENCES scenes(id),
  is_bonus        INTEGER DEFAULT 0
  -- đúng 1 trong chapter_id / ending_id được set (enforce ở seed/app)
);

-- ===== RUNTIME (người chơi) =====
CREATE TABLE IF NOT EXISTS users (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  email         TEXT UNIQUE,
  auth_provider TEXT,
  created_at    TEXT DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS entitlements (
  user_id    INTEGER REFERENCES users(id),
  chapter_id INTEGER REFERENCES chapters(id),
  source     TEXT,                           -- 'purchase' | 'bundle' | 'promo' | 'free'
  granted_at TEXT DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (user_id, chapter_id)
);

CREATE TABLE IF NOT EXISTS saves (
  id               INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id          INTEGER REFERENCES users(id),
  slot             INTEGER NOT NULL,         -- 0 = autosave, 1..n thủ công
  story_id         INTEGER REFERENCES stories(id),
  current_scene_id INTEGER REFERENCES scenes(id),
  state            TEXT NOT NULL DEFAULT '{}',
  updated_at       TEXT DEFAULT CURRENT_TIMESTAMP,
  UNIQUE (user_id, slot)
);

CREATE TABLE IF NOT EXISTS user_unlocks (
  user_id     INTEGER REFERENCES users(id),
  kind        TEXT NOT NULL,                 -- 'gallery' | 'ending' | 'scene'
  ref_id      INTEGER NOT NULL,
  unlocked_at TEXT DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (user_id, kind, ref_id)
);

CREATE TABLE IF NOT EXISTS choice_events (
  id             INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id        INTEGER,
  scene_id       INTEGER,
  choice_id      INTEGER,
  state_snapshot TEXT,
  created_at     TEXT DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_chapter_videos_chapter ON chapter_videos(chapter_id);
CREATE INDEX IF NOT EXISTS idx_scenes_chapter  ON scenes(chapter_id);
CREATE INDEX IF NOT EXISTS idx_choices_scene   ON choices(scene_id);
CREATE INDEX IF NOT EXISTS idx_events_scene    ON choice_events(scene_id);
CREATE INDEX IF NOT EXISTS idx_gallery_unlock  ON gallery_items(unlock_scene_id);
CREATE INDEX IF NOT EXISTS idx_gallery_chapter ON gallery_items(chapter_id);
