# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

FMV branching dating-sim ("Năm Trái Tim Dưới Một Mái Nhà"). A Go backend (`backend/`)
serves a **server-authoritative Director API**; a React + Vite frontend (`frontend/`, plain
JavaScript — **not TypeScript**) is a thin player. The story is **data, not code** — authored
as JSON, loaded into a DB.

> A `fmv-rules` skill (`.claude/skills/fmv-rules/`) restates these invariants and routes to
> content-authoring / backend-go / frontend-player sub-skills — consult it for deeper detail.
> Source language is **Vietnamese**; code comments are in Vietnamese.

## Commands

```bash
# Backend (Go ≥ 1.23; uses modernc.org/sqlite — pure Go, no cgo)
cd backend
go test ./...                  # RUN FIRST — engine + director are the most-tested layers
go test ./internal/engine -run TestEvaluate   # single test by name
go run ./cmd/seed              # load content/demo-story.json → data/game.db
go run ./cmd/server            # serve :8080
rm backend/data/game.db        # reset DB after editing content JSON, then re-seed

# Frontend (React 19 + MUI 9 + Emotion, no test runner — verify with build)
cd frontend
npm install
npm run dev                    # :5173, Vite proxies /api + /media → :8080
npm run build                  # vite build (no tsc — pure JS with JSX-in-.js)
npm run crowdin:processor      # crowdin-imports/*.json → public/locales/{lang}/common.json
```

Env overrides (server): `DB_PATH`, `MEDIA_DIR`, `ADDR`, `MEDIA_SIGNING_SECRET`,
`ADMIN_TOKEN` (defaults to `dev-admin` — **must** be set in prod).

> Go was written in a sandbox without a Go toolchain — **always run `go test ./...` first**
> to confirm it compiles. Frontend has no tests — verify with `npm run build`.

## Architecture: the non-negotiable invariants

These are the rules the whole design depends on. Violating one means moving logic to the
wrong layer.

1. **Server-authoritative, absolute.** All branching, condition evaluation, effect
   application, and entitlement checks live in `internal/director`. The client never
   decides the next scene, never computes affinity, never receives an unsigned video URL.
   If a new feature needs the client to "know" something, add a field to `SceneResponse` —
   do not move logic clientward.
2. **`internal/engine` is pure.** No I/O imports (db, http, os). `Evaluate(condition, state)`
   and `Apply(effects, state)` are pure functions; `Apply` clones, never mutates input.
   Every engine change requires a table-driven test in `engine_test.go`.
3. **Choices are filtered server-side.** `buildSceneResponse` returns only choices whose
   condition passes. Never return the full list with a `disabled` flag — that leaks spoilers.
4. **Entitlement is checked at chapter boundaries.** In `Advance`, when
   `next.ChapterID != scene.ChapterID`, and at game start. A locked chapter returns
   HTTP 402 `CHAPTER_LOCKED` with chapter data so the frontend opens the store.
5. **All media goes through signed URLs.** `media.SignURL` (HMAC + 5-min TTL, per-user);
   `GET /media/{file}` calls `media.Verify` before serving. The signature is enforced, not
   decorative.

## Request flow (the core loop)

`GET /api/play/current` (auto-starts a new game if no save) and
`POST /api/play/advance {choiceId?}` are the heart. `Advance` does, in order:
load state from autosave → validate the choice belongs to the scene + its condition holds
(anti-spoof) → apply choice effects → resolve next scene → if crossing a chapter, check
entitlement → run `on_enter` effects + gallery/ending unlocks (`enterScene`) → autosave
(slot 0, every clip) → log to `choice_events` → return the new `SceneResponse` (signed
`videoUrl`, filtered `choices` each with a `preloadUrl` for client-side branch preloading).

Business errors are `director.Error{Status, Code, Data}` → JSON `{error:{code,message,data}}`.
The frontend keys off `code` (e.g. `CHAPTER_LOCKED`).

Other player endpoints (all under `auth`, `uid`-cookie scoped): `POST /api/play/restart`,
`GET|POST /api/saves` + `POST /api/saves/load` (slots), `GET /api/gallery`, `GET /api/store` +
`POST /api/store/purchase`, and the data-driven browse endpoints `GET /api/chapters` (overview)
+ `GET /api/chapters/{id}/map` that feed the pre-game title → chapter-select → chapter-map screens.

## Admin CRUD (Phase 2 — `internal/api/admin.go`)

A separate `/api/admin/*` surface for content tooling, guarded by `adminAuth` (header
`X-Admin-Token`, `Authorization: Bearer`, or `?admin_token=`; compared to `ADMIN_TOKEN`).
CRUD for models / chapters / chapter-videos (+ reorder + publish), plus:
- `PUT /api/admin/models/{id}/content` — import/replace a whole model's story graph via
  `seed.Replace` (same DSL/ref/dead-end validation as `cmd/seed`; invalid → 400 `CONTENT_INVALID`).
- `GET /api/admin/chapters/{id}/flow` — full scene/choice graph (nodes + edges + `map_json`
  layout) for a visual flow editor.

DAL errors map to HTTP via `handleWrite`: `store.ErrNotFound` → 404, `store.ErrInUse` → 409.

## The mini-DSL (in `condition` / `effects` / `on_enter` JSON columns)

```json
{"affinity": {"malsook": {">=": 30}}, "flags": {"saw_secret": true}}
{"affinity": {"minjung": 5, "malsook": -2}, "set_affinity": {"x": 0}, "flags": {"f": true}}
```

- **Condition**: ANDs all clauses. Bare number = `>=` shorthand. Unset affinity = 0,
  unset flag = false. Operators: `>= > <= < == !=`.
- **Effects**: `affinity` is a DELTA, clamped 0..100; `set_affinity` is absolute and runs
  AFTER the delta. Parsing lives in `engine.go` (`ParseCondition`/`ParseEffects`,
  `CmpSet.UnmarshalJSON` handles the shorthand).

## Editing the story (`backend/content/demo-story.json`)

Authoring references everything by **`code`** (string), never numeric id; `cmd/seed`
runs a two-pass loader (insert all scenes, then wire edges by code) and validates the
whole DSL + dead-ends + refs before writing. Rules:

- `linear` scenes must have `next`; `choice` scenes need ≥1 choice; `ending` scenes must
  have a matching record in the `endings` array.
- Every `next` references a scene `code` — seed fails on a bad ref or dead-end.
- After editing: `rm backend/data/game.db`, then `go run ./cmd/seed` (idempotent by slug).
- A missing media clip is fine — the player skips it, the game never gets stuck.

## Layout

```
backend/
  cmd/server, cmd/seed          # HTTP entrypoint :8080 / JSON→DB loader
  internal/engine               # pure DSL Evaluate/Apply (test heaviest)
  internal/director             # the Director service + integration tests
  internal/store                # SQLite DAL: content.go, runtime.go, admin.go; schema.sql
  internal/api                  # net/http routes (Go 1.22+), dev auth, media, admin.go (Phase 2)
  internal/seed                 # JSON authoring loader + validation (Load + Replace)
  internal/media                # signed-URL sign/verify
  content/demo-story.json       # the demo story
  db/postgres-schema.sql        # production schema (Postgres, JSONB)
  media/                        # put .mp4 files here (see media/README.md)

frontend/src/                   # plain JS; path aliases @app @models @containers @components
                                #   @contexts @providers @hooks (jsconfig.json + vite.config.js)
  main.js, App.js               # entry; App = ThemeProvider → ProviderComposer → GameShell
  models/game.js                # typed fetch wrapper `api` + `ApiError` (NO axios/SWR); mirrors DTOs
  containers/game/              # the "game" feature (container pattern)
    index.js                    #   GameShell — pure presentation, renders from context
    context, provider, hooks/   #   GameContext + GameProvider + useGame (state + play loop)
    components/                 #   playerCore, titleScreen, chapterSelectScreen,
                                #     chapterMapScreen, albumScreen, panels (SaveLoad/Store), ...
  providers/, contexts/         # notification + authentication providers/contexts
  hooks/useTranslation.js       # react-i18next wrapper (vi default, en; LANGUAGES list)
  i18n/                         # i18next init; locales in public/locales/{vi,en}/common.json
frontend/scripts/crowdin-processor.js  # Crowdin export → flat locale JSON
```

## Dev stubs to replace for production

These are intentional dev shortcuts, marked in code:

- **Auth**: `uid` cookie (`api/server.go` `currentUser`, auto-creates a user) → real JWT/session.
- **Admin auth**: `ADMIN_TOKEN` shared secret (default `dev-admin`, `admin.go`) → RBAC/JWT with an admin scope.
- **Purchase**: `POST /api/store/purchase` grants entitlement immediately (simulated IAP) →
  only grant inside a Stripe / App Store webhook after verifying the receipt.
- **DB**: SQLite → Postgres via `db/postgres-schema.sql`; the DAL writes standard SQL, so
  swap the driver and change `?` placeholders to `$n`.
- **Media**: static mp4 → HLS/DRM; the `hls_manifest` column already exists, only
  `internal/media` signing changes.
