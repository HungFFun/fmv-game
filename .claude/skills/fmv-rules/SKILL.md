---
name: fmv-rules
description: Use for ANY work in the FMV dating-sim repo (Go Director backend + React player) — the non-negotiable architectural invariants, the request/branching flow, and a router to the content-authoring / backend-go / frontend-player sub-skills.
---

# FMV Dating-Sim — Architecture Rules (router)

## Purpose

The single source of truth for the invariants every change must respect. Read this
first; then jump to the sub-skill that matches the task. Triggers: any edit to `backend/`
or `frontend/`, adding a feature, debugging branching/entitlement/media, editing the story.

## Sub-skills

| Task | Sub-skill |
|------|-----------|
| Code conventions for ANY change — style, imports, naming, folder structure, element IDs, api client, Go rules | [project-rules](project-rules/SKILL.md) |
| Edit the story, add scenes/choices/endings/gallery, DSL conditions/effects | [content-authoring](content-authoring/SKILL.md) |
| Go backend: engine, store/DAL, director, api, add an endpoint, tests | [backend-go](backend-go/SKILL.md) |
| React player: play loop, video/choice/timer UI, panels | [frontend-player](frontend-player/SKILL.md) |

There is also a `.cursor/rules/project.mdc` and root `CLAUDE.md` — they overlap with this
skill on purpose (multi-tool). If they ever disagree with the code, the code wins; fix the docs.

## The 6 invariants (DO NOT VIOLATE)

1. **Server-authoritative, absolute.** All branching, condition evaluation, effect
   application, and entitlement checks run in `backend/internal/director`. The client never
   decides the next scene, never computes affinity, never receives an unsigned media URL.
   New feature needs the client to "know" something? → add a field to `SceneResponse`, do
   **not** move logic clientward.
2. **`internal/engine` is pure.** No I/O imports (database/sql, net/http, os, file). Every
   engine change ships with a table-driven test in `engine_test.go`. `Apply` clones state,
   never mutates its input.
3. **Choices are filtered server-side.** `buildSceneResponse` returns only choices whose
   condition passes. Never return the full list with a `disabled` flag — that leaks spoilers.
4. **Entitlement at chapter boundaries.** In `Advance`, when
   `next.ChapterID != scene.ChapterID`, and again at game start. A locked chapter → HTTP 402
   `CHAPTER_LOCKED` with chapter data so the FE opens the store.
5. **All media via signed URLs.** Emit URLs only through `media.SignURL` (HMAC + 5-min TTL,
   per-user). `GET /media/{file}` calls `media.Verify` before serving. The signature is
   enforced, not decorative.
6. **Content is DATA, not code.** Story lives in `backend/content/*.json`, loaded by
   `cmd/seed` (validates DSL + refs + dead-ends). Never hardcode scenes/choices in Go.

## The core request flow

Two endpoints drive everything:
- `GET /api/play/current` — current scene; auto-starts a new game from chapter 1 if no save.
- `POST /api/play/advance {choiceId?}` — linear scenes need no `choiceId`.

`Advance` order (see `director.go`):
1. Load state from autosave (slot 0).
2. Validate the choice belongs to the current scene **and** its condition passes (anti-spoof:
   `INVALID_CHOICE` 400 / `CONDITION_FAILED` 403).
3. Apply the choice's effects.
4. Resolve next scene; if it crosses a chapter, check entitlement (402 `CHAPTER_LOCKED`).
5. `enterScene`: run `on_enter` effects + unlock gallery items / record ending.
6. Autosave (slot 0, **every clip** — `is_checkpoint` is display metadata only) + log to
   `choice_events`.
7. Return `SceneResponse` (signed `videoUrl`, filtered `choices` each with a `preloadUrl`,
   and `notifications[]` — server-emitted events for this advance, e.g. gallery unlock).

**Server-emitted notifications** (invariant #1): user-facing event toasts are produced by the
server in `enterScene`, not derived on the client. `SceneResponse.notifications []NotifyDTO`
carries them; it's empty on `Current`/load (only an advance that enters a new scene emits).
Currently only gallery first-unlock fires; only newly-unlocked items emit (no replay spam).

Business errors are `director.Error{Status, Code, Data}` → JSON
`{error:{code,message,data}}`; the FE keys off `code` (notably `CHAPTER_LOCKED`).

## Commands

```bash
cd backend && go test ./...        # RUN FIRST — confirms it compiles; engine+director tested
cd backend && go run ./cmd/seed    # load content/demo-story.json → data/game.db
cd backend && go run ./cmd/server  # :8080  (env: DB_PATH MEDIA_DIR ADDR MEDIA_SIGNING_SECRET)
rm backend/data/game.db            # reset after editing content JSON, then re-seed
cd frontend && npm install && npm run dev   # :5173, proxies /api + /media → :8080
cd frontend && npm run build       # tsc -b && vite build
```

> Go is authored in a sandbox without a Go toolchain — **always `go test ./...` first**.
> No linter is configured on either side.

## Dev stubs to replace for production (intentional, marked in code)

- **Auth**: `uid` cookie (`api/server.go` `currentUser`, auto-creates a user) → JWT/session.
- **Admin auth** (Phase 2): `/api/admin/*` guarded by a static `X-Admin-Token` (env `ADMIN_TOKEN`,
  dev fallback `dev-admin` in `api/admin.go` `adminToken`) → real RBAC/JWT with an admin scope.
- **Purchase**: `POST /api/store/purchase` grants entitlement immediately (simulated IAP) →
  grant only inside a verified Stripe / App Store webhook.
- **DB**: SQLite → Postgres via `db/postgres-schema.sql`; DAL SQL is portable — swap driver,
  change `?` placeholders to `$n`.
- **Media**: static mp4 → HLS/DRM; the `hls_manifest` column already exists, only
  `internal/media` signing changes.

## Changelog
- v1.2 (2026-06-13): Content model = **models → chapters → {chapter_videos, scenes/choices flow}**
  (story_id→model_id; scenes.video_id→chapter_videos). Added Phase 2 **admin CRUD** at `/api/admin/*`
  (models/chapters/videos + reorder + publish + `PUT …/content` whole-graph import via `seed.Replace`
  + `GET …/chapters/{id}/flow`), guarded by `X-Admin-Token`.
- v1.1 (2026-06-12): Added server-emitted `SceneResponse.notifications` (gallery unlock toasts).
- v1.0 (2026-06-12): Initial — extracted from code + README + cursor rules during onboarding.
