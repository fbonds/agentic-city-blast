# Session handoff — UI rendering changes not taking effect

**Last touched:** 2026-05-29 ~9:30pm. Terminal Claude Code session.

---

## Critical issue: frontend changes not rendering

Multiple changes were made to frontend canvas renderers but **none of them
appear in the browser**. The Vite dev server was killed, cache cleared
(`rm -rf node_modules/.vite`), source files touched, server restarted,
browser hard-refreshed, Chrome DevTools "Clear site data" attempted —
**nothing works**. The browser continues to render the old transparent
buildings with impact ellipses and dashed selection outlines.

### What was changed (files on disk are confirmed correct via grep):

1. **`web/src/canvas/AgentRenderer.ts`** (~line 612):
   - Tractor beam impact ellipse code removed (replaced with comment)
   - Cow abduction function added (~line 665+)
   - Verified: `grep "impactAlpha" AgentRenderer.ts` returns nothing

2. **`web/src/canvas/BuildingRenderer.ts`**:
   - Side faces changed from outline-only to opaque solid fills using
     computed RGB colors blended with dark background:
     ```
     rightFill = rgb(tR*0.4+22, tG*0.4+27, tB*0.4+33)
     leftFill  = rgb(tR*0.3+16, tG*0.3+21, tB*0.3+26)
     roofFill  = rgb(tR*0.5+30, tG*0.5+35, tB*0.5+40)
     ```
   - Hidden back edges (B→C dashed line) removed
   - Window dots call removed (drawWindowDots)
   - `drawCursorHighlight` changed from dashed outline to pulsing glow
     fill on visible faces (requires `time` parameter)
   - `drawHoverHighlight` changed from cyan outline to subtle glow fill
   - Verified: `grep "rightFill" BuildingRenderer.ts` shows line 226

3. **`web/src/canvas/CityRenderer.ts`**:
   - `drawCursorHighlight` calls updated to pass `now` as time parameter

4. **`web/src/hud/GuideOverlay.tsx`** (new file):
   - Guide narration overlay component, centered bottom of viewport
   - Shows GUIDE activity messages with fade in/out

5. **`web/src/hud/HudOverlay.tsx`**:
   - Added `<GuideOverlay />` import and render

### What was tried to force the browser to pick up changes:
- Killed Vite dev server (kill -9), restarted
- Cleared Vite cache: `rm -rf web/node_modules/.vite`
- Touched source files: `touch web/src/main.tsx`, `touch` on changed files
- Browser hard refresh (Cmd+Shift+R — but app intercepts this hotkey)
- Chrome DevTools > right-click reload > "Empty Cache and Hard Reload"
- Chrome DevTools > Application > Clear site data
- **None of these worked.** Browser serves identical old rendering.

### Theories for next session to investigate:
1. **The Vite proxy may be serving from a different source.** Check
   `web/vite.config.ts` for any unusual root/base/publicDir settings.
2. ~~**Browser loading from :8080 instead of :5173**~~ — RULED OUT.
   User confirmed URL has always been localhost:5173.
3. **Service worker caching.** Check Chrome DevTools > Application >
   Service Workers. If one is registered, unregister it.
4. **Try a completely different browser** (Firefox, Safari) to rule out
   Chrome-specific caching.
5. **Check if `npm run build` + serving from :8080 picks up the changes.**
   If it does, the issue is Vite HMR specifically.

---

## Committed and pushed

Everything through the road arcs commit is pushed. The following are
**uncommitted on disk**:

### Backend (committed separately, already pushed)
- Demo mode rewrite (`cmd/agentic-city/main.go`) — scripted timeline with
  guided narration, blast radius on buildings, cow abduction choreography
- Demo rewrite IS committed and pushed (verify with `git log --oneline`)

### Frontend (UNCOMMITTED — these are the changes not rendering)
- `web/src/canvas/AgentRenderer.ts` — impact ellipse removed, cow abduction
- `web/src/canvas/BuildingRenderer.ts` — opaque buildings, no windows,
  pulse selection
- `web/src/canvas/CityRenderer.ts` — time param for cursor highlight
- `web/src/hud/GuideOverlay.tsx` — new file, guide narration overlay
- `web/src/hud/HudOverlay.tsx` — added GuideOverlay

### Also uncommitted
- `.gitignore` — screenshot patterns
- `DESIGN.md` — fork note at top

### User's requested visual changes (the intent behind the code):
1. **Remove tractor beam impact ellipses** — visual noise
2. **Remove window dots on buildings** — visual noise
3. **Make buildings opaque** — seeing through buildings is confusing
4. **Replace dashed selection outline with pulsing glow** — the dotted
   line showed back edges through transparent buildings
5. **Guide narration overlay** — centered bottom of viewport, not buried
   in the right rail activity log

All code changes compile, pass TypeScript check, and pass all 116 tests.
The issue is purely that the browser is not rendering the updated code.

---

## What's running (kill before restarting)

```bash
lsof -i :8080,5173
```

- Backend (demo mode): `go run ./cmd/agentic-city --demo -addr :8080`
- Vite dev server: `npm run dev` from `web/`

To restart cleanly:
```bash
/opt/homebrew/bin/go run ./cmd/agentic-city --demo -addr :8080
PATH="/Users/fletcherbonds/.nvm/versions/node/v22.20.0/bin:$PATH" npm --prefix web run dev
```

## What's next

1. **Fix the rendering issue** — the changes are on disk but not in the
   browser. See theories above.
2. Once rendering works, verify the visual changes look right.
3. Commit all frontend changes.
4. Update docs.

## Known issues

- **Agent overlay line cap:** agentwatch JSONL size limit warnings.
- **Agent-driven churn noise:** known limitation, not filtered.
