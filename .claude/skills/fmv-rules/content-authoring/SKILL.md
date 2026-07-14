---
name: content-authoring
description: Use when editing the FMV story — adding/changing scenes, choices, endings, gallery, characters, or writing mini-DSL conditions/effects in backend/content/demo-story.json. Covers the authoring JSON format, the DSL spec, what cmd/seed validates, and the reset-and-reseed cycle.
---

# Content Authoring (story JSON + mini-DSL)

## Purpose

The story is **data, not code**. Writers edit `backend/content/demo-story.json`; `cmd/seed`
validates and loads it. This skill is the authoring contract. Trigger: any change to story
content or DSL.

## Authoring file structure

Top-level keys (`StoryFile` in `internal/seed/seed.go`): `story` (slug/title),
`characters[]`, `media[]`, `chapters[]`, `endings[]`, `gallery[]`. **Everything references
by `code` / `key` strings, never numeric ids** — the loader resolves them in a second pass.

```jsonc
"characters": [{ "code": "malsook", "display_name": "Malsook", "archetype": "..." }],
"media":      [{ "key": "ch1_intro", "kind": "video", "file": "/media/ch1_intro.mp4", "duration_ms": 12000 }],
"chapters":   [{ "idx": 1, "title": "...", "is_free": true, "price_cents": 0, "sku": "...",
                 "entry": "<scene code>", "scenes": [ ...SceneDef... ] }],
"endings":    [{ "scene": "<scene code>", "code": "true_end", "title": "...", "rank": "true" }],
"gallery":    [{ "title": "...", "media": "<media key>", "unlock_scene": "<scene code>", "is_bonus": false }]
```

### ChapterDef extras (for the data-driven Chapter screens)
- `poster` (optional): storage key of the chapter's select-card thumbnail, e.g.
  `/media/thumb_ch1.jpg`. Served as a signed URL by `GET /api/chapters`.
- `map` (optional): the branching-map layout for `GET /api/chapters/{id}/map`, rendered by
  `ChapterMapScreen`. Shape: `{ width, height, nodes: [...] }` where each node is
  `{ kind: "video"|"lock"|"start"|"finish", title?, poster? (storage key), x, y, w, h, locked? }`
  in logical canvas coords. Positions came from the Figma map frames (node 4:724/1012/1213);
  the director signs each `poster` into a URL. Stored in `chapters.map_json` (TEXT/JSONB).

### SceneDef
- `code` (unique across the whole story), `type`: `linear` | `choice` | `ending`.
- `video` (optional media key), `on_enter` (Effects DSL), `checkpoint` (bool, display only).
- `linear` → needs `next` (scene code).
- `choice` → needs `choices[]` (≥1); each: `label`, `next` (scene code), optional
  `condition`, `effects`, `timer_ms`, `default` (bool — auto-picked when the timer runs out).
- `ending` → no outgoing edges; **must** have a matching record in the `endings[]` array.

## Mini-DSL (condition / effects / on_enter)

```jsonc
// condition — ANDs all clauses. Bare number = ">=" shorthand. Unset affinity=0, unset flag=false.
{ "affinity": { "malsook": { ">=": 30 }, "minjung": 5 }, "flags": { "saw_secret": true } }
// effects — affinity is a DELTA (clamped 0..100); set_affinity is absolute, runs AFTER delta.
{ "affinity": { "minjung": 5, "malsook": -2 }, "set_affinity": { "x": 0 }, "flags": { "confessed": true } }
```

Operators: `>= > <= < == !=`. Affinity keys must be real character `code`s. `""` or `{}` =
empty (condition always true / effects no-op). Implemented in `engine.go`
(`ParseCondition`/`ParseEffects`, `CmpSet.UnmarshalJSON` does the shorthand).

## What `cmd/seed` validates (it fails the load on any of these)

- Every `condition`/`effects`/`on_enter` parses as valid DSL (`validateDSL`).
- Duplicate scene `code` → error.
- Scene `video`, gallery `media`, ending/gallery `scene`, chapter `entry`, and every `next`
  resolve to a defined `key`/`code` — unknown ref fails.
- `linear` scene without `next` → "dead end" error; `choice` scene with 0 choices → error;
  unknown scene `type` → error.

What seed does **not** catch: an unreachable scene that is otherwise well-formed, or a
logically stranded player (e.g. a choice whose condition can never be true). Trace reachability
by hand for new branches.

## Edit workflow (checklist)

1. Edit `backend/content/demo-story.json`. Reference scenes/media/characters by code/key only.
2. `rm backend/data/game.db` — seed is **idempotent by story slug**, so it skips a story that
   already exists; you must drop the DB to re-seed changed content.
3. `cd backend && go run ./cmd/seed` — fix any validation error it prints.
4. `go test ./internal/director` — integration tests seed the real demo story and walk the
   graph; they catch stranded branches a fresh playthrough would hit.
5. Missing media file is fine — the player treats a missing/broken clip as a 1.5s silent scene
   and advances; the game never gets stuck.

## Verification (4C)

- **Correctness**: every `next` resolves; endings have an `endings[]` record; affinity keys are
  real character codes; delta vs `set_affinity` used as intended.
- **Completeness**: new branch is reachable AND exitable (no orphan, no dead-end loop).
- **Context-fit**: codes follow existing convention (e.g. `ch1_kitchen_01`); shorthand vs
  explicit operator used consistently.
- **Consequence**: a bad ref fails seed loudly (safe); an unreachable/stranded scene passes
  seed silently and strands players (dangerous) — verify reachability.

## Changelog
- v1.0 (2026-06-12): Initial — authoring format, DSL, seed validation, reset cycle.
