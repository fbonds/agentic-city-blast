# Session handoff — Phase 3 complete

**Last touched:** 2026-05-29. Terminal Claude Code session in this directory.

---

## Committed and pushed

Everything through Phase 2b is committed on `main`.

## Uncommitted on disk right now

### Phase 3 — Churn as color (the encoding redesign's final piece)

**Backend:**
- `internal/model/model.go`: `Churn float64` field added to Building.
- `internal/repo/churn.go` (new): `ComputeChurn` runs
  `git log --since=90days --name-only --format="" --diff-filter=AMRC`,
  `parseChurnOutput` counts commits per file, `NormalizeChurn` applies
  log normalization (`log(1+count)/log(1+max)`) to [0,1].
- `internal/repo/churn_test.go` (new): 13 table-driven tests for parsing,
  normalization, and config defaults.
- `internal/city/builder.go`: `AssembleState` gains a `churn` parameter.
  `BuildState` calls `ComputeChurn` + `NormalizeChurn` (best-effort).
  `MergeBuildings` preserves `Churn` on incremental updates.
- `internal/city/builder_test.go`: 3 new tests (`ChurnPopulated`,
  `ChurnNil`, `PreservesChurn`), all existing `AssembleState` calls updated.
- `cmd/agentic-city/main.go`: demo buildings get random churn values.

**Frontend:**
- `web/src/store/cityStore.ts`: `churn: number` on Building interface.
- `web/src/hud/palette.ts`: `churnColor()` — 5-stop ramp
  (cyan → blue → yellow → orange → red).
- `web/src/canvas/BuildingRenderer.ts`: `LANG_COLORS` removed, replaced
  with `churnColor(b.churn)`. Header comment updated.
- `web/src/hud/RightRail.tsx`: churn row in BuildingPanel (shows percentage
  or "cold").
- 4 test files: `churn: 0` added to mock buildings.

**Docs:**
- `README.md`: Phase 3 marked Done, churn description updated, known issue
  added for agent-driven churn noise.
- `SESSION_STATE.md`: this file.
- `encoding-redesign.md`: not yet updated for Phase 3 (do next).
- `.gitignore`: screenshot image patterns added.

**Tests:** `go test ./internal/...` all green. `go vet` clean.
TypeScript clean, ESLint clean, all 116 frontend tests pass.

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
| 1 | Commit Phase 3 + all uncommitted work | Working tree has significant changes. |
| 2 | Update `encoding-redesign.md` for Phase 3 | Mark churn decisions as resolved. |
| — | `encoding-redesign.md` §6 | Companion doc tasks (DESIGN.md pointer). Low priority. |

## Known issues

- **Agent overlay line cap:** Backend logs `parse failed: line exceeds size
  cap: 1368464 > 1048576` warnings from agentwatch hitting an upstream JSONL
  line cap. Agent tracking still works but some session updates are dropped.
- **Agent-driven churn noise:** Agents are much of the churn, so the signal
  partly re-encodes the live UFO layer. Filtering would require correlating
  agentwatch sessions with git authorship — flagged, not yet addressed.

## How to verify visually

Reload <http://localhost:5173> after the backend is up.

- Buildings should be **tinted by churn**: cyan (cold) → red (hot).
- Most buildings in a young fork will be cyan/blue; recently touched files
  (builder.go, engine.go, packer.go) should be warmer.
- Districts should be **different sizes** (treemap by footprint area).
- Selecting a building shows **blast radius**, **churn**, and **LOC** in
  the RightRail.
