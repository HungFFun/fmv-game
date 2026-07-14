-- PostgreSQL schema (PRODUCTION) — nguyên bản theo thiết kế.
-- Dev dùng SQLite (internal/store/schema.sql) mirror 1:1 file này.

-- ===== CONTENT =====

-- models — persona top-level (vd "Hana"), SỞ HỮU chapters. Hồ sơ lấy từ màn Album.
CREATE TABLE models (
  id           BIGSERIAL PRIMARY KEY,
  code         TEXT UNIQUE NOT NULL,
  display_name TEXT NOT NULL,
  avatar       TEXT,
  age          INT,
  birthday     TEXT,
  relationship TEXT,
  occupation   TEXT,
  height_cm    INT,
  weight_kg    INT,
  family       TEXT,
  bio          TEXT,
  profile_json JSONB DEFAULT '{}',
  created_at   TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE stories (
  id          BIGSERIAL PRIMARY KEY,
  model_id    BIGINT REFERENCES models(id),
  slug        TEXT UNIQUE NOT NULL,
  title       TEXT NOT NULL,
  season      INT  DEFAULT 1,
  status      TEXT DEFAULT 'draft',
  created_at  TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE characters (
  id           BIGSERIAL PRIMARY KEY,
  model_id     BIGINT REFERENCES models(id),
  code         TEXT NOT NULL,
  display_name TEXT NOT NULL,
  archetype    TEXT,
  meta         JSONB DEFAULT '{}',
  UNIQUE (model_id, code)
);

CREATE TABLE chapters (
  id             BIGSERIAL PRIMARY KEY,
  model_id       BIGINT REFERENCES models(id),
  idx            INT NOT NULL,
  title          TEXT NOT NULL,
  entry_scene_id BIGINT,
  is_free        BOOLEAN DEFAULT false,
  price_cents    INT DEFAULT 0,
  sku            TEXT,
  poster         TEXT,                        -- storage key ảnh thẻ chapter (màn select)
  map_json       JSONB DEFAULT '{}',          -- LAYER layout: canvas + vị trí node + cạnh trang trí
  UNIQUE (model_id, idx)
);

CREATE TABLE media_assets (
  id            BIGSERIAL PRIMARY KEY,
  kind          TEXT NOT NULL,
  storage_key   TEXT NOT NULL,
  hls_manifest  TEXT,
  duration_ms   INT,
  drm_protected BOOLEAN DEFAULT true,
  meta          JSONB DEFAULT '{}'
);

-- chapter_videos — danh sách video PHẲNG của mỗi chapter; scenes.video_id trỏ vào đây.
CREATE TABLE chapter_videos (
  id          BIGSERIAL PRIMARY KEY,
  chapter_id  BIGINT REFERENCES chapters(id),
  code        TEXT NOT NULL,
  idx         INT NOT NULL,
  title       TEXT NOT NULL,
  media_id    BIGINT REFERENCES media_assets(id),
  duration_ms INT,
  poster      TEXT,
  UNIQUE (chapter_id, code),
  UNIQUE (chapter_id, idx)
);

CREATE TABLE scenes (
  id            BIGSERIAL PRIMARY KEY,
  chapter_id    BIGINT REFERENCES chapters(id),
  code          TEXT NOT NULL,
  type          TEXT NOT NULL,
  video_id      BIGINT REFERENCES chapter_videos(id),  -- NODE flow → video phẳng
  next_scene_id BIGINT REFERENCES scenes(id),
  on_enter      JSONB DEFAULT '{}',
  is_checkpoint BOOLEAN DEFAULT false,
  UNIQUE (chapter_id, code)
);

CREATE TABLE choices (
  id                BIGSERIAL PRIMARY KEY,
  scene_id          BIGINT REFERENCES scenes(id),
  idx               INT NOT NULL,
  label             TEXT NOT NULL,
  condition         JSONB DEFAULT '{}',
  effects           JSONB DEFAULT '{}',
  next_scene_id     BIGINT REFERENCES scenes(id),
  timer_ms          INT,
  default_choice_id BIGINT,
  hotspot           JSONB,                   -- optional {x,y,w,h,style} — vùng bấm trên khung video (hotspot)
  UNIQUE (scene_id, idx)
);

CREATE TABLE endings (
  id         BIGSERIAL PRIMARY KEY,
  model_id   BIGINT REFERENCES models(id),
  chapter_id BIGINT REFERENCES chapters(id),
  scene_id   BIGINT REFERENCES scenes(id),
  code       TEXT NOT NULL,
  title      TEXT NOT NULL,
  rank       TEXT
);

CREATE TABLE gallery_items (
  id              BIGSERIAL PRIMARY KEY,
  model_id        BIGINT REFERENCES models(id),
  chapter_id      BIGINT REFERENCES chapters(id),   -- "Obtainable in Chapter N"
  ending_id       BIGINT REFERENCES endings(id),    -- "Obtainable in Ending …"
  media_id        BIGINT REFERENCES media_assets(id),
  title           TEXT,
  unlock_scene_id BIGINT REFERENCES scenes(id),
  is_bonus        BOOLEAN DEFAULT false
  -- đúng 1 trong chapter_id / ending_id được set (enforce ở seed/app)
);

-- ===== RUNTIME =====
CREATE TABLE users (
  id            BIGSERIAL PRIMARY KEY,
  email         TEXT UNIQUE,
  auth_provider TEXT,
  created_at    TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE entitlements (
  user_id    BIGINT REFERENCES users(id),
  chapter_id BIGINT REFERENCES chapters(id),
  source     TEXT,
  granted_at TIMESTAMPTZ DEFAULT now(),
  PRIMARY KEY (user_id, chapter_id)
);

CREATE TABLE saves (
  id               BIGSERIAL PRIMARY KEY,
  user_id          BIGINT REFERENCES users(id),
  slot             INT NOT NULL,
  story_id         BIGINT REFERENCES stories(id),
  current_scene_id BIGINT REFERENCES scenes(id),
  state            JSONB NOT NULL DEFAULT '{}',
  updated_at       TIMESTAMPTZ DEFAULT now(),
  UNIQUE (user_id, slot)
);

CREATE TABLE user_unlocks (
  user_id     BIGINT REFERENCES users(id),
  kind        TEXT NOT NULL,
  ref_id      BIGINT NOT NULL,
  unlocked_at TIMESTAMPTZ DEFAULT now(),
  PRIMARY KEY (user_id, kind, ref_id)
);

CREATE TABLE choice_events (
  id             BIGSERIAL PRIMARY KEY,
  user_id        BIGINT,
  scene_id       BIGINT,
  choice_id      BIGINT,
  state_snapshot JSONB,
  created_at     TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_chapter_videos_chapter ON chapter_videos(chapter_id);
CREATE INDEX idx_scenes_chapter ON scenes(chapter_id);
CREATE INDEX idx_choices_scene  ON choices(scene_id);
CREATE INDEX idx_events_scene   ON choice_events(scene_id);
CREATE INDEX idx_gallery_unlock ON gallery_items(unlock_scene_id);
CREATE INDEX idx_gallery_chapter ON gallery_items(chapter_id);
