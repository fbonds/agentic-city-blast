# Session handoff — Core redesign complete

**Last touched:** 2026-05-29. Terminal Claude Code session in this directory.

---

## Committed and pushed

Everything is committed and pushed on `main`. Working tree is clean.

- **Phase 1:** Blast radius → building height. `internal/deps/blastradius.go`
  computes transitive dependents via reverse BFS.
  `heightFromBlastRadius` uses `clamp(3 + 3·log₂(1+BR), 3, 30)`.

- **Phase 1.5a — Footprint refactor:** `footprint()` switched from LOC to
  BlastRadius. `w = clamp(4 + √BR, 4, 12)`, `h = 0.8w`.

- **Phase 1.5b — MergeBuildings bug fix:** Preserves BlastRadius and Churn
  on incremental edits.

- **Phase 1.5c — Frontend null guard:** `districtThresholds` crash fix.

- **Phase 2a — Treemap district sizing:** Uniform grid replaced with
  `squarify()` weighted by total footprint area per district.

- **Phase 2b — HUD legend update:** RightRail shows blast radius, churn,
  and LOC when a building is selected.

- **Phase 3 — Churn as color:** `internal/repo/churn.go` runs
  `git log --since=90days`, log-normalizes to [0,1]. Frontend replaces
  language-based tint with 5-stop churn ramp (cyan → red).

- **Road arcs:** `web/src/canvas/RoadRenderer.ts` rewritten. Roads hidden
  by default; on building selection, shows rooftop-to-rooftop bezier arcs
  (cyan = outgoing, blue = incoming) with endpoint dots.

## What's running

Two processes should be active (check with `lsof -i :8080,5173`):

- **Backend:** `go run ./cmd/agentic-city -repo . -addr :8080`
- **Vite dev server:** `npm run dev` from `web/`

System node is 14.15.0 (too old for Vite 5); use v22.20.0:
```bash
/opt/homebrew/bin/go run ./cmd/agentic-city -repo . -addr :8080
PATH="/Users/fletcherbonds/.nvm/versions/node/v22.20.0/bin:$PATH" npm --prefix web run dev
```

Browser: <http://localhost:5173>.

## What's next

| Priority | Item | Notes |
|----------|------|-------|
| — | `encoding-redesign.md` §6 | DESIGN.md pointer note. Low priority. |

The core encoding redesign is complete. All three visual channels are live:
height = blast radius, footprint = blast radius, color = churn.

## Known issues

- **Agent overlay line cap:** Backend logs `parse failed: line exceeds size
  cap: 1368464 > 1048576` warnings from agentwatch hitting an upstream JSONL
  line cap. Agent tracking still works but some session updates are dropped.
- **Agent-driven churn noise:** Agents are much of the churn, so the signal
  partly re-encodes the live UFO layer. Filtering would require correlating
  agentwatch sessions with git authorship — flagged, not yet addressed.

## How to verify visually

Reload <http://localhost:5173> after the backend is up.

- Buildings **tinted by churn**: cyan (cold) → red (hot).
- Districts are **different sizes** (treemap by footprint area).
- Selecting a building shows **blast radius**, **churn**, and **LOC** in
  the RightRail, plus **rooftop bezier arcs** for dependency edges.
