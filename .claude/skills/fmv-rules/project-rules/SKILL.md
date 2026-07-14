---
name: project-rules
description: MANDATORY code conventions for the FMV dating-sim — Go Director backend + Vite/React/TS player. Applies to ALL code changes (new code, refactor, bugfix, review). Covers tech stack, code style, imports, file/folder structure, naming, element IDs, the api client layer, styling, and backend (Go) rules. For the architecture invariants and domain depth, see the parent fmv-rules skill and its backend-go / frontend-player / content-authoring sub-skills.
globs: "backend/**/*.go, frontend/src/**/*.{ts,tsx}"
alwaysApply: true
---

# Project Rules — FMV Dating-Sim

**MANDATORY** — applies to ALL code changes. Every new file, edit, refactor, or bugfix MUST
comply. This is the conventions layer; the **6 architectural invariants** live in the parent
[fmv-rules](../SKILL.md) and override everything here if they ever conflict.

---

## 1. Tech Stack & Versions

| Area | Tech | Version | Notes |
|---|---|---|---|
| Backend | Go | 1.23 | stdlib `net/http` (Go 1.22+ method+path patterns) |
| DB driver | `modernc.org/sqlite` | 1.34.x | pure Go, **no cgo**; SQLite (dev) mirrors Postgres (prod) |
| Frontend | React | 19.x | `react-jsx` runtime — do NOT `import React` just for JSX |
| Language | JavaScript (JSX) | ES2022 | **plain JS, no TypeScript**; every file is `.js` (incl. JSX) |
| Build | Vite | 6.x | dev `:5173`, proxies `/api` + `/media` → `:8080`; `npm run build` = `vite build` |
| UI kit | MUI Material + Emotion | `@mui/material`, `@mui/icons-material`, `@emotion/{react,styled}` | styling via `sx` + a custom theme (§9) |
| i18n | `i18next` + `react-i18next` + `i18next-http-backend` | 26 / 17 / 4 | source = **Vietnamese**, target = `en`; Crowdin manual workflow (§9a) |

**JSX lives in `.js` files** (Tevi-style `index.js` convention). `vite.config.js` makes esbuild
treat `src/**/*.js` as JSX (`esbuild: { loader: 'jsx', include: /src\/.*\.js$/ }` +
`optimizeDeps.esbuildOptions.loader['.js'] = 'jsx'`) — in production build `@vitejs/plugin-react`
defers JSX to esbuild, so this loader is **load-bearing**, don't remove it. No TypeScript: no
`tsconfig`, no type annotations/interfaces/generics; editor IntelliSense via `jsconfig.json`.

**Deliberately absent — do NOT add without discussion:** no Next.js, no external state library
(state is local `useState` + a `Context`/`Provider`, all React built-ins — no Redux/Zustand).
Runtime deps: React + i18next trio + MUI/Emotion — do not add others without discussion. The
frontend is a thin client; the Go server is authoritative.

**Path aliases ARE configured** (`jsconfig.json` `paths` + `vite.config.js` `resolve.alias`):
`@app/* → src/*`, `@models/* → src/models/*`, `@containers/* → src/containers/*`,
`@components/* → src/components/*`, `@contexts/* → src/contexts/*`,
`@providers/* → src/providers/*`, `@hooks/* → src/hooks/*`. Use an alias to cross a
layer/feature boundary; relative imports within the same feature.

---

## 2. Code Style

No linter/formatter is configured. **Match the surrounding file**; the de-facto style is:

### Go
- **`gofmt` is law** (tabs, standard layout). Run `gofmt -w` / rely on your editor. Run
  `go vet ./...` and `go test ./...` before considering a change done.
- Package-level doc comment on every package (see existing packages — keep the style).
- Wrap errors with `%w`: `fmt.Errorf("scene %s on_enter: %w", sc.Code, err)`. Never discard a
  meaningful `err` (no `x, _ := ...` when the error can signal a real failure).
- Single-letter receivers, consistent per type (`d *Director`, `s *Store`, `e Effects`).

### JavaScript / React
- **2-space** indent, **semicolons**, **single quotes**, **trailing commas**. Plain JS — no
  type annotations, interfaces, or generics (they won't parse).
- Components are **arrow functions with default export**; helper components may be local
  function declarations in the same file (see `components/playerCore/index.js`).
- No unused imports/vars (no linter enforces it — keep it clean by hand).
- StrictMode is on (`main.js`) → effects run twice in dev; effects must be idempotent and
  clean up (clear timers/intervals).

---

## 3. Import Conventions (frontend)

Use **aliases across a layer/feature boundary** (`@models/*`, `@containers/*`, `@app/*`) and
**relative imports within the same feature** (`./context`, `../constant`,
`./components/panels`). Order imports in groups separated by a blank line; each non-React group
is preceded by a **capitalized comment** (`// Models`, `// Context`, `// Hooks`,
`// Components`, `// Constants`):

Groups in order: React → third-party → **MUI** → App layers (Models, Context, Hooks,
Components, Constants). MUI uses **specific path imports** (tree-shakeable).

```js
// React (first, no comment)
import { useCallback, useEffect, useState } from 'react';

// MUI
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import CloseIcon from '@mui/icons-material/Close';

// Models
import { api, ApiError } from '@models/game';

// Context
import { useGameContext } from './context';

// Hooks
import useTranslation from '@hooks/useTranslation';

// Components
import PlayerCore from './components/playerCore';

// Constants
import { TOAST_TTL_MS } from '../constant';
```

Keep one component family per file when small (e.g. `components/panels/index.js` exports
AffinityHUD, SaveLoadMenu, GalleryScreen, StoreScreen + a shared `Modal`).

---

## 4. File & Folder Structure

### Backend (Go) — layered, dependencies point one way

```
backend/
  cmd/server/      # HTTP entrypoint :8080
  cmd/seed/        # JSON authoring → DB loader
  internal/
    engine/        # PURE DSL (no I/O) — Evaluate/Apply + table tests
    store/         # SQLite DAL; schema.sql mirrors db/postgres-schema.sql
    director/      # the only place game decisions are made + integration tests
    api/           # net/http routes, dev auth, media handler
    seed/          # authoring loader + validation
    media/         # signed-URL sign/verify
  content/*.json   # the story (DATA, not code)
  db/postgres-schema.sql
```

Dependency rule: `api → director → {store, engine, media}`, `seed → {store, engine}`.
`engine` imports **no I/O**. Never reverse these arrows.

### Frontend — feature container pattern (Context/Provider separated)

The player is organized as a `game` feature container (adapted from the Tevi web-app
convention — see the parent project rules):

```
frontend/src/                    # all .js (JSX in .js); no .ts/.tsx, no styles.css
  main.js                        # entry → renders <App/>; side-effect imports '@app/i18n'
  App.js                         # root composer — ThemeProvider+CssBaseline → ProviderComposer
  theme/index.js                 # MUI createTheme (token map: accent→primary, panel→bg.paper…)
  i18n/index.js                  # i18next init (vi source) — side-effect imported in main.js
  hooks/useTranslation.js        # wrapper hook: t, changeLanguage, currentLanguage, LANGUAGES
  models/
    game.js                      # the api client (fetch wrapper + ApiError; mirrors director DTOs)
  contexts/                      # GLOBAL context DEFINITIONS only (createContext + hook)
    notification/index.js        #   useNotificationContext — { items, notify, dismiss }
    authentication/index.js      #   useAuthenticationContext — { currentUser, isAuthenticated… }
  providers/                     # GLOBAL provider IMPLEMENTATIONS (state + logic)
    composer/index.js            #   ProviderComposer (reduceRight + cloneElement)
    notification/index.js        #   owns toast queue + renders <NotificationStack/>
    authentication/index.js      #   session (DEV STUB: anonymous uid-cookie session)
  components/                    # SHARED presentational components (cross-feature)
    notificationStack/index.js
    languageSwitcher/index.js
  containers/game/               # the `game` feature container
    index.js                     # GameShell — presentational, consumes the context
    constant.js                  # feature consts (TOAST_TTL_MS)
    context/index.js             # createContext + useGameContext (definition only)
    provider/index.js            # GameProvider — wires the hooks into context (useMemo)
    hooks/
      useGame.js                 # play-loop state + actions (start/advance/restart…)
      useNotifications.js        # pushes server-emitted toasts into the global provider
    components/
      playerCore/index.js
      titleScreen/index.js
      chapterSelectScreen/index.js
      chapterMapScreen/index.js
      panels/index.js
```

Rules: **Context defines, Provider implements** — at BOTH levels. Top-level cross-cutting
concerns live in `contexts/` (definition) + `providers/` (implementation); a feature's own
context/provider live inside its container (`containers/<feature>/context|provider`). `context*`
files hold only `createContext` + the `useXContext` hook; all state/logic lives in hooks and is
composed in the provider. Component folders are camelCase with an `index.js`; shared
cross-feature components go in top-level `components/`, feature-local ones under the container. A
component that fetches data owns its **loading / empty / data** states explicitly. Grow by adding
sibling containers under `containers/`.

### Provider composition

Compose global providers in `App.js` via `ProviderComposer` (analog of Tevi's `_app.js`); MUI
`ThemeProvider`+`CssBaseline` wrap everything (outside the composer). Within the composer the
**first array element is the OUTERMOST** layer — order matters, a provider may consume any
provider above it:

```js
<ThemeProvider theme={theme}>
  <CssBaseline />
  <ProviderComposer providers={[
    <NotificationProvider key="notification" />,   // outermost — everyone can notify()
    <AuthenticationProvider key="authentication" />, // consumes useNotificationContext()
    <GameProvider key="game" />,                    // game state; pushes toasts up to Notification
  ]}>
    <GameShell />
  </ProviderComposer>
</ThemeProvider>
```

---

## 5. Naming Conventions

| Type | Convention | Example |
|---|---|---|
| Go exported / unexported | `MixedCaps` / `mixedCaps` | `Advance`, `buildSceneResponse` |
| Go error codes | `UPPER_SNAKE` string in `director.Error` | `CHAPTER_LOCKED`, `INVALID_CHOICE` |
| React component / provider | `PascalCase`, default export | `PlayerCore`, `NotificationProvider` |
| Context + hook | `XContext` + `useXContext` | `NotificationContext`, `useAuthenticationContext` |
| Hooks / handlers | hook `use{Feature}`, handlers `handleX` / `onX` props | `useGame`, `handleError`, `onChoose` |
| JS constants | `UPPER_SNAKE` | `NOTIF_TTL_MS`, `TOAST_TTL_MS` |
| Folders | camelCase concern dir + `index.js` | `containers/game/`, `providers/notification/`, `contexts/authentication/` |
| Files | all frontend files `.js` (incl. JSX); Go files `lower.go` | `models/game.js`, `hooks/useGame.js`, `director.go` |
| Story refs | string `code`/`key`, never numeric id | `ch1_kitchen`, media `cg_kitchen` |

---

## 6. Element IDs for Automation (keep — adopt going forward)

Interactive and meaningful display elements SHOULD carry a kebab-case `id` for automation.
Pattern: `{feature}-{area}-{name}-{type}`. Put `id` as the **first prop**.

```jsx
<Button id='topbar-saves-btn' onClick={() => setPanel('saves')}>{t('topbar_w1_save', '💾 Save')}</Button>
<Button id={`player-choice-${c.id}-btn`} onClick={() => choose(c.id)} sx={{ ... }} />
<ToggleButton id={`lang-${l.code}-btn`} value={l.code} />
```

Required on: buttons, inputs/checkboxes, clickable items, and display elements asserted in
tests (chapter title, affinity value, ending title, notification body). Skip pure layout
wrappers and decorative icons. Kebab-case only; never put a task id (`tev-123`) in a DOM id;
for lists use a stable key: ``id={`choice-${c.id}-btn`}``.

---

## 7. Component Structure Order (inside a React component)

```
1. Contexts (useGameContext)   2. Hooks (custom: useGame, useNotifications)
3. State (useState)            4. Refs (useRef)
5. Memo (useMemo)              6. Callbacks (useCallback)
7. Effects (useEffect)         8. Render (return)
```

Feature state/logic lives in `hooks/` and is exposed via the `game` Context (see
`provider/index.js`); a presentational component (`containers/game/index.js`) consumes it
through `useGameContext` and stays declarative. Guard one-shot actions with a `useRef` flag
(see `chosenRef`/`firedRef` in `components/playerCore/index.js`), and remember the load-bearing
`key={scene.scene.id}` that resets per-scene state.

---

## 8. Data Fetching — the `api` client is mandatory

**Never call `fetch` directly in a component.** All network access goes through the
client in [models/game.js](../../../frontend/src/models/game.js) (the analog of Tevi's ApiModel
rule); components and hooks import it as `import { api } from '@models/game'`:

```js
export const api = {
  current: () => request('/api/play/current'),
  advance: (choiceId) => request('/api/play/advance', { ... }),
  // ...
};
```

- `request` injects `credentials: 'include'`, parses JSON, and throws `ApiError(status,
  code, message, data)` on non-2xx.
- Handle errors by **`code`**, not message text — e.g. `e.code === 'CHAPTER_LOCKED'` opens the
  store. Add new fields to a DTO rather than computing game state on the client.
- Never trust or derive game state client-side; render straight from the server response.

---

## 9. Styling — MUI (`sx` + theme)

**MUI Material + Emotion; no `styles.css`.** Style with the `sx` prop and use MUI components
(`Box`, `Stack`, `Typography`, `Button`, `Dialog`, `IconButton`, `ToggleButton`…). The old CSS
`:root` tokens now live in [theme/index.js](../../../frontend/src/theme/index.js) palette — refer
to them by name, never hardcode hex for brand colors:

| Token (old CSS var) | Theme key | `sx` usage |
|---|---|---|
| `--accent` `#ff5c8a` | `primary.main` | `color: 'primary.main'` |
| `--accent-2` `#8a5cff` | `secondary.main` | `bgcolor: 'secondary.main'` |
| `--bg` `#0d0b12` | `background.default` | |
| `--panel` `#1a1622` | `background.paper` | `bgcolor: 'background.paper'` |
| `--text` | `text.primary` | |
| `--muted` | `text.secondary` | `color: 'text.secondary'` |

- Gradients/dynamic styles → `sx` theme callback: ``sx={(th) => ({ background: `linear-gradient(135deg, ${th.palette.primary.main}, ${th.palette.secondary.main})` })}``.
- Keyframes inline in `sx`: `{ animation: 'pulse 0.8s infinite', '@keyframes pulse': { from: {…}, to: {…} } }`.
- Modals → MUI `Dialog` (free backdrop / Escape / focus-trap); icons → specific imports
  (`@mui/icons-material/Close`). The pixel-positioned Figma screens stay as `Box` with absolute
  `%` positioning in `sx` (same numbers as the old art layout).
- `CssBaseline` (in `App.js`) provides the global reset + body `overflow:hidden` + font.
- UI strings go through `t()` (see §9a), never inline literals.

---

## 9a. Internationalization (i18n)

i18next + react-i18next, HTTP backend loads `/locales/{lng}/common.json` on demand. **Source
language is Vietnamese (`vi`)**; `en` is a target. Init singleton: [src/i18n/index.js]; wired by
a side-effect `import '@app/i18n'` in `main.js`. Consume via the wrapper hook:

```jsx
import useTranslation from '@hooks/useTranslation';
const { t, currentLanguage, changeLanguage, LANGUAGES } = useTranslation();
<h1>{t('ending_w1_restart', 'Chơi lại từ đầu')}</h1>
<p>{t('game_w1_coming_soon', '{{label}} sắp ra mắt', { label })}</p>
```

**Hard rules:**
- **All user-facing strings use `t()`** — no inline literals. Server-emitted strings
  (`NotifyDTO`, `chapter.title`, `ending.title`) are NOT translated client-side (they come from
  the backend) — render them as-is.
- **2-arg form always**: `t('key', 'Vietnamese source')`. The 2nd arg is the **Vietnamese**
  fallback (= source language) so UI is correct before JSON loads / if a key is missing. Never
  `t('key')` (blank-key risk) and never `t('key', { opts })` (options ≠ fallback).
- **Key convention** `{page}_{version}_{key}`, snake_case, e.g. `save_w1_load_btn`,
  `chapter_select_w1_empty`. Page prefixes in use: `common_ title_ chapter_select_ map_ player_
  save_ gallery_ store_ game_ ending_ topbar_ auth_`.
- **Every new key** must be added to BOTH `public/locales/vi/common.json` (source) and
  `en/common.json`; keep the two files at key parity.

**Adding a language** — keep 3 spots in sync: `supportedLanguages` (src/i18n/index.js),
`LANGUAGES` (src/hooks/useTranslation.js), `LANGUAGE_MAPPING` (scripts/crowdin-processor.js) +
add the target in Crowdin. **Crowdin flow** (manual dashboard): upload `vi/common.json` as
source → translate in web UI → download per-lang JSON into `crowdin-imports/` →
`npm run crowdin:processor` flattens them to `public/locales/{lng}/common.json`.

**Audit before shipping** (all should be empty):
```bash
grep -rnE "\bt\(\s*['\"][^'\"]+['\"]\s*\)" src/        # single-arg (no fallback)
grep -rnE "\bt\(\s*['\"][^'\"]+['\"]\s*,\s*\{" src/    # 2nd arg is options, not a string
```

---

## 10. Backend (Go) Rules — always on

These codify the invariants + audit findings into enforced rules. Depth in
[backend-go](../backend-go/SKILL.md).

1. **All game decisions in `director`.** Branching, condition eval, effect apply, entitlement —
   never in `api` handlers or the client. New data the client needs → new field on a DTO.
2. **`engine` stays pure.** No `database/sql`/`net/http`/`os`/file imports. `Apply` clones,
   never mutates. Every engine change ships a table-driven case in `engine_test.go`.
3. **Client errors are `director.Error` only.** Build with `errf(status, "CODE", "msg")` and
   send via `writeErr`. Never write a bare-string error envelope from a handler (this includes
   `decodeBody` failures — wrap them as e.g. `BAD_BODY`).
4. **Portable SQL.** Write SQL that ports SQLite→Postgres with only `?`→`$n` + driver change.
   Use `ON CONFLICT … DO NOTHING/UPDATE` (NOT SQLite-only `INSERT OR IGNORE`). Always
   parameterize — never concatenate user input.
5. **Content is data.** Scenes/choices/endings live in `content/*.json`, loaded by `cmd/seed`
   (which validates DSL + refs + dead-ends). Never hardcode story in Go.
6. **Signed media.** Emit URLs only via `media.SignURL`; serve only after `media.Verify`.
7. **New endpoint checklist:** director method (returns DTO + `*director.Error`) → route in
   `api/server.go` wrapped in `s.auth(...)` → integration test driving it through the seeded
   demo story. Multi-statement mutations belong in a transaction.
8. **Hygiene:** `defer rows.Close()` + check `rows.Err()` after every `Query`; distinguish
   `store.ErrNotFound` from real DB errors (don't swallow the latter as "absent").

---

## 11. Security Rules

- **Secrets via env, never in code.** `MEDIA_SIGNING_SECRET` must be set in prod (fail-closed —
  do not ship the dev fallback). No secrets sent to or stored on the client.
- **Server-authoritative validation.** Re-validate every client input server-side (choice
  belongs to scene + condition passes); the client list is already filtered, but never trust it.
- **Media gating.** Signed URL + entitlement at the chapter boundary. (Known prod hardening:
  bind the `/media` signature to the session user — see the audit.)
- **Dev stubs are NOT production:** `uid` cookie auth and simulated purchase must be replaced
  by real session/JWT + a verified payment webhook before any non-dev deploy.

---

## 12. Testing & Commands

```bash
# Backend (run before every commit)
cd backend && go vet ./... && go test ./...     # table-driven engine + director integration
cd backend && go run ./cmd/seed && go run ./cmd/server   # :8080
rm backend/data/game.db                          # reset after editing content JSON, then reseed

# Frontend
cd frontend && npm run dev      # :5173
cd frontend && npm run build    # vite build (no TS step) — MUST build clean (no test runner yet)
```

Director/engine changes ship with a Go test. Frontend has no test runner configured; a clean
`npm run build` is the gate (plus the §9a i18n audit greps).

---

## 13. Git & Commits

This game lives outside the Tevi git repo (no branch flow here). Still use **Conventional
Commits** (`feat`, `fix`, `chore`, `refactor`, `docs`) if/when versioned. Never `--no-verify`
past hooks in repos that have them.

---

## Changelog
- v1.4 (2026-06-13): **Frontend migrated TS → plain JS + adopted MUI** (reverses "no MUI" + the
  TypeScript rule, per explicit request). All `.ts/.tsx` → `.js` (JSX in `.js`; `vite.config.js`
  esbuild `loader:'jsx'` for `src/**/*.js` — load-bearing); removed `tsconfig`, added
  `jsconfig.json`. Removed `styles.css` — styling is MUI `sx` + a custom theme
  ([theme/index.js]) mapping the old CSS tokens to the palette; modals → `Dialog`. `App.js` wraps
  `ThemeProvider`+`CssBaseline`. Updated §1 (stack/deps/aliases), §2 (JS style), §3 (MUI import
  group), §4 (tree), §5 (naming), §6/§7/§8 (paths), §9 (MUI styling), §12 (build = `vite build`).
- v1.3 (2026-06-13): **Adopted i18n** (reverses the old "no i18n / inline VN" rule per explicit
  request). Added i18next + react-i18next + i18next-http-backend; source = Vietnamese, target =
  `en`, manual Crowdin workflow (`scripts/crowdin-processor.js`, `npm run crowdin:processor`). All
  inline UI strings migrated to `t()` (2-arg, VN fallback); new §9a documents the hard rules +
  key convention + audit greps. Added `src/i18n/`, `src/hooks/useTranslation.ts`,
  `components/languageSwitcher`, `public/locales/{vi,en}/common.json`, `@hooks/*` alias. Updated
  §1 (deps + i18n row), §4 (tree), §9 (no longer inline).
- v1.2 (2026-06-13): Added **global Context/Provider layer** — `src/contexts/` (notification,
  authentication) + `src/providers/` (composer, notification, authentication). `ProviderComposer`
  composes them in `App.tsx` (Notification → Authentication → Game; first = outermost).
  NotificationStack moved to shared `components/`; toasts are now global (game pushes via
  `notify()`). Aliases `@components/@contexts/@providers` added. AuthenticationProvider is a dev
  stub over the anonymous `uid`-cookie session. Updated §1, §4 (folder tree + composition), §5.
- v1.1 (2026-06-13): Frontend reorganized into the Tevi-style **feature container pattern** —
  `containers/game/` with Context/Provider separation (`context/`, `provider/`, `hooks/`,
  `components/`, `constant.ts`), `api.ts` → `models/game.ts`. Added path aliases (`@app/@models/
  @containers`) + capitalized import-group comments. Updated §1 (aliases now present), §3
  (imports), §4 (folder structure), §5 (naming), §7 (context order), §8 (api client path).
- v1.0 (2026-06-12): Rewritten from the Tevi web-app project-rules to fit fmv-game (Go + Vite/
  React/TS): real stack/style/imports, kept folder-structure + element-ID + api-client-layer +
  state-handling conventions, added a Backend (Go) rules section codifying the audit findings.
