---
name: frontend-player
description: Use when changing the React 19 + Vite player of the FMV dating-sim (plain JS + MUI) — the GameShell play loop, PlayerCore (video/choice/timer/preloader), panels (HUD/save/gallery/store), or the models/game.js api client. Covers the thin-client contract, the play loop, and the load-bearing video/timer gotchas.
---

# Frontend (React player) Conventions

## Purpose

The frontend is a THIN client over a server-authoritative backend: the server decides the
next scene, computes affinity, and signs every URL. The client plays video and posts choices.
Trigger: any edit under `frontend/src/`. Read the parent [fmv-rules](../SKILL.md) invariants first.

## Thin-client contract

- **Never derive game state locally.** Render affinity, flags, chapter, choices straight from
  the `SceneResponse` the server returns each advance. Do not compute "what affinity will be"
  or "which choice should be hidden" on the client.
- All decisions come back as a new `SceneResponse`; locked chapters arrive as an
  `ApiError` with `code === 'CHAPTER_LOCKED'` and `data.chapterId`.
- `models/game.js` is the only network layer: a `fetch` wrapper, `credentials: 'include'`
  (cookie auth), throwing `ApiError(status, code, message, data)` on non-2xx.

## Stack & style

React 19 + Vite 6, **plain JavaScript (JSX in `.js`, no TypeScript)**. Runtime deps = React +
i18next trio + **MUI Material/Emotion** — nothing else without discussion. Arrow-function
components, default exports, `useCallback` for handlers passed down, refs to guard one-shot
actions. **Styling = MUI `sx` + a custom theme** (no `styles.css`); modals are MUI `Dialog`. UI
strings go through `t()` (i18n; source Vietnamese, target `en`). See project-rules §1/§9/§9a for
the JS+MUI+i18n conventions; do not hardcode user-facing literals.

## The play loop (`containers/game/index.js` GameShell)

```
start() → api.current() → setScene
  scene.type === 'linear'  → PlayerCore plays video → onLinearEnded → advance()
  scene.type === 'choice'  → video ends → ChoiceOverlay → onChoose(id) → advance(id)
  scene.type === 'ending'  → ending overlay (restart / gallery)
  ApiError CHAPTER_LOCKED  → open StoreScreen; after purchase → advance() to cross boundary
```

**Pre-game navigation** (`!started`, App `screen` state): `title → chapterSelect → chapterMap →`
`start()` (game). All three are Figma flat-art screens (`public/*.png`) on the shared `.fig-stage`
(694:391 letterbox) + transparent `.fig-hotspot`s positioned in %:
- `TitleScreen` (node 1:24, `title-bg.png` + `title-character.png` in the circle): Click to Start /
  Story → `chapterSelect`; Album → gallery; Ranking/Settings → toast.
- `ChapterSelectScreen` (node 4:557) — **DATA-DRIVEN, not a flat image**: fetches `api.chapters()`
  and renders DOM chapter cards (poster + number badge + lock) on a CSS gradient backdrop. Unlocked
  card → `onOpenChapter(id)` → `chapterMap`; locked → toast; back → title.
- `ChapterMapScreen` (nodes 4:724/1012/1213) — **DATA-DRIVEN**: fetches `api.chapterMap(id)` and
  renders a branching map on a wide horizontal-scroll canvas (`.cmap-canvas`, width = `W/694`),
  nodes absolutely positioned in % of the map's logical `width×height`, dashed edges drawn in an
  `<svg viewBox=0 0 W H preserveAspectRatio=none>` (main line through start/video/finish + stubs to
  lock diamonds). Video card → `onPlayVideo` = `start()` (the real PlayerCore scene flow IS the
  fullscreen video, per node 27:15). The Figma layout (node positions, posters, locks) lives in
  `chapters.map_json` content data — see [content-authoring](../content-authoring/SKILL.md).

`PlayerCore` = `VideoSurface` + `ChoiceOverlay` (with countdown timer) + `BranchPreloader`
(hidden `<video preload>` of each branch's first clip so the next clip plays instantly).

**Notifications**: `SceneResponse.notifications[]` (server-emitted, e.g. gallery unlock) are
pushed into the **global** `NotificationProvider` (`providers/notification`) via the game's
`useNotifications(scene)` hook (`notify()` each, guarded by scene identity so StrictMode's double
effect doesn't duplicate). The provider owns the queue (monotonic `id` from a `useRef`) and
renders `NotificationStack` (`components/notificationStack`). Each toast (`NotifToast`) owns its
own auto-dismiss `setTimeout` — **do not** manage dismiss timers in the queue effect, or its
cleanup would cancel pending dismissals when scenes advance fast.

## Load-bearing gotchas (DO NOT break these)

- **`<VideoSurface key={scene.scene.id}>` is load-bearing.** `firedRef` (the one-shot
  ended/error guard) only resets because the `key` forces a fresh mount per scene. Remove the
  key to "avoid reload flicker" and every scene after the first auto-advances instantly via
  the stale guard. Keep the key; comment it if you touch it.
- **Missing/broken video must not strand the player.** No `src` → treated as a 1.5s silent
  scene (`setTimeout`); `onError` on a real `<video>` also fires `onEnded`. Preserve both
  fallbacks. (Known gap: a video that *stalls* mid-buffer fires neither `ended` nor `error` —
  there is no watchdog yet; see Known issues.)
- **Choice idempotency:** `ChoiceOverlay` guards manual-click-vs-timer with `chosenRef`, and
  the timer auto-picks `isDefault` (or `choices[0]`) on timeout. This is only safe if the
  overlay remounts per scene — see Known issue #1.

## Verification (4C)

- **Correctness**: state rendered from `scene.state`/`scene.choices`, never derived locally;
  one-shot guards reset per scene.
- **Completeness**: handle loading (between Start and first scene), error (failed fetch),
  and empty (`scene === null`) states — don't leave a black screen with no retry.
- **Context-fit**: plain JS arrow components, `useCallback` handlers, MUI `sx` + theme (no
  `styles.css`), user-facing strings via `t()` (2-arg, VN fallback), no new runtime deps beyond
  React + the i18next trio + MUI/Emotion.
- **Consequence**: a broken guard/`key` either freezes the player or silently disables
  choices on the next scene — both are player-facing dead-ends.

## Known issues / lessons learned (from onboarding audit 2026-06-12)

1. **Bug — `<ChoiceOverlay>` lacks `key={scene.scene.id}` (`components/playerCore/index.js`).**
   Back-to-back choice scenes with equal `timerMs` reuse the same overlay instance; its
   `chosenRef` stays `true`, silently disabling all choices on the second scene (and the timer
   can auto-pick a stale choice ID). **Fix:** add `key={scene.scene.id}` to the overlay.
   Highest-priority fix (still open after the JS+MUI migration).
2. **Robustness — no retry on initial-fetch failure (`hooks/useGame.js` `start()`).** `start()`
   sets `started=true` before `api.current()` resolves; if it rejects, the user sees a black
   shell with a 5s toast and no Start button. **Fix:** keep `started=false` until it resolves, or
   render a retry when `started && !scene && !busy`.
3. **Robustness — no loading indicator** between Start and the first scene (the busy dot lives
   inside the not-yet-mounted `PlayerCore`).
4. **Robustness — stalled real video has no watchdog** (only null-src has the 1.5s net).
5. **Bandwidth — BranchPreloader uses `preload="auto"` on every branch**; consider
   `preload="metadata"` or preloading only the default branch.
6. **A11y (LOW)** — modals now use MUI `Dialog` (Escape/focus-trap/`role` provided ✓); remaining:
   timed choice overlay has no autofocus/focus management; emoji-only buttons rely on `aria-label`.

> Resolved by the JS+MUI migration: the old TS non-null-assertion nit (`c.preloadUrl!`) is gone
> (JS `filter`), and modal a11y is largely handled by MUI `Dialog`.

## Changelog
- v1.3 (2026-06-13): **Migrated TS → plain JS + MUI.** All files `.js` (JSX in `.js`); styling is
  MUI `sx` + theme (no `styles.css`); modals → MUI `Dialog`. Network layer renamed `api.ts` →
  `models/game.js`. Refreshed paths + Known-issues (TS nit resolved, modal a11y via Dialog). See
  project-rules §1/§9 (v1.4).
- v1.2 (2026-06-13): Reversed "no i18n" — adopted i18next (source VN, target `en`); UI strings
  now via `t()` 2-arg. Container/provider refactor + global Notification/Authentication providers.
  See project-rules §9a + §4.
- v1.1 (2026-06-12): Added NotificationStack (server-emitted toasts) + per-item self-dismiss pattern.
- v1.0 (2026-06-12): Initial — thin-client contract, play loop, gotchas + audit findings.
