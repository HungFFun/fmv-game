---
name: backend-go
description: Use when changing the Go backend of the FMV dating-sim — internal/engine (pure DSL), internal/store (SQLite DAL), internal/director (orchestration), internal/api (HTTP), internal/media (signed URLs), or adding an endpoint. Covers the layering, the error/test conventions, and SQLite→Postgres portability rules.
---

# Backend (Go) Conventions

## Purpose

How the Go layers fit together and the rules each enforces. Trigger: any edit under
`backend/`. Read the 6 invariants in the parent [fmv-rules](../SKILL.md) first.

## Layering (dependencies point one way)

```
api  →  director  →  store        (HTTP)        (orchestration)   (DAL/SQLite)
                  →  engine        (pure DSL — no I/O)
                  →  media         (HMAC sign/verify)
seed →  store + engine             (JSON authoring loader + validation)
```

- **`engine`** — `Evaluate(Condition, State)` and `Apply(Effects, State)` are PURE: no
  database/sql, net/http, os, or file imports. `Apply` returns a new State via `Clone()`,
  never mutates input. Affinity clamps 0..100. State is the JSON blob stored in `saves.state`.
- **`store`** — typed DAL over `*sql.DB`. SQLite driver is `modernc.org/sqlite` (pure Go, no
  cgo). `Open` runs the embedded `schema.sql` (idempotent), sets `SetMaxOpenConns(1)` + WAL +
  `foreign_keys=ON`. `OpenMemory()` (`:memory:`) is for tests. `ErrNotFound` is the
  not-found sentinel.
- **`director`** — the only place branching/affinity/entitlement decisions are made
  (invariant #1). Builds DTOs (`SceneResponse`, etc.); `buildSceneResponse` filters choices.
- **`api`** — `net/http` with Go 1.22+ method+path patterns (`"GET /api/play/current"`).
  Dev auth = `uid` cookie. CORS allows only localhost origins.
- **`media`** — `SignURL`/`Verify`: HMAC-SHA256 over `path:userID:exp`, 5-min TTL, secret
  from `MEDIA_SIGNING_SECRET` env (dev fallback `dev-secret-change-me`).

## Error convention

Business errors are `*director.Error{Status int, Code string, Msg string, Data map}`
(built via `errf(...)`). The HTTP layer's `writeErr` unwraps it with `errors.As` →
`{error:{code,message,data}}` at the right status; anything else → 500 `INTERNAL` (logged).
Add a new failure mode as a new `Code` + status, never a bare string error to the client.

## Testing convention (REQUIRED for engine/director changes)

- **`engine_test.go`** — table-driven. Every new operator, shorthand, or clamp rule gets a
  case in `TestEvaluate` / the Apply tests.
- **`director_test.go`** — integration: `OpenMemory()` + `seed.LoadFile(...demo-story.json)`
  → walk the real scene graph (`choiceByLabel`, `mustAdvance`). This is what catches a
  stranded branch in content. When you change branching logic, assert against the real demo
  graph, not a hand-built fixture.
- Always `go test ./...` first (the toolchain may be absent when code was written).

## Adding an endpoint (checklist)

1. Add the business method to `director` (returns DTO + `*director.Error`). Keep all decision
   logic here — never in the handler.
2. If the client needs new data, add a field to the relevant DTO (invariant #1), don't push
   logic to the client.
3. Register the route in `api/server.go` `Handler()` with a `"METHOD /path"` pattern, wrapped
   in `s.auth(...)` for per-user endpoints; decode the body with `decodeBody[T]`.
4. Emit any media URL via `media.SignURL` only (invariant #5).
5. Add/extend a `director_test.go` case driving it through the real seeded story.

## SQLite → Postgres portability (keep the DAL portable)

`internal/store/schema.sql` mirrors `db/postgres-schema.sql` 1:1 (JSONB→TEXT-holding-JSON).
Write standard SQL in the DAL. The only prod changes should be: swap the driver, and change
`?` placeholders to `$n`. Do NOT use SQLite-only syntax in queries. All queries are
parameterized — never string-concat user input into SQL.

## Gotchas

- `SetMaxOpenConns(1)`: a single writer connection. Don't introduce code assuming parallel
  writes; long-running queries serialize. WAL + `busy_timeout=5000` cover read concurrency.
- `default_choice_id` is written onto **every** choice of a scene by the seeder (pointing at
  the default one); `buildSceneResponse` marks `IsDefault` only when `choice.id ==
  default_choice_id` — so exactly one choice gets the flag. Preserve this if you touch either side.
- Autosave is slot 0 and happens on every advance; `is_checkpoint` is display metadata only,
  not a save trigger.

## Chapter-browse API (data-driven Chapter screens)

`GET /api/chapters` → `ChaptersOverview`: chapter list with `locked` (entitlement) + signed
`posterUrl` (from `chapters.poster`). `GET /api/chapters/{id}/map` → `ChapterMap`: parses
`chapters.map_json` and signs each node's `poster` into `posterUrl`. Both honor the invariants
(server decides lock state; posters via `media.SignURL`). Layout/posters are content data
(`poster` + `map` on a ChapterDef), not hardcoded — see content-authoring.

## Changelog
- v1.0 (2026-06-12): Initial — layering, error/test conventions, portability, gotchas.
