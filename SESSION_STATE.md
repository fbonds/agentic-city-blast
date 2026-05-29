# Session handoff — Phase 2a complete

**Last touched:** 2026-05-29. Terminal Claude Code session in this directory.

---

## Committed and pushed

Everything below is committed on `main`. Working tree is clean.

- **Phase 1:** Blast radius → building height. `internal/deps/blastradius.go`
  computes transitive dependents via reverse BFS.
  `internal/layout/packer.go:heightFromBlastRadius` uses
  `clamp(3 + 3·log₂(1+BR), 3, 30)`. `internal/model/model.go` has
  `BlastRadius` field. `internal/city/builder.go:AssembleState` wires it up.

- **Phase 1.5a — Footprint refactor:** `internal/layout/packer.go:footprint()`
  switched from `LOC` to `BlastRadius`. `w = clamp(4 + √BR, 4, 12)`,
  `h = 0.8w`.

- **Phase 1.5b — MergeBuildings bug fix:** `internal/city/builder.go`
  preserves `BlastRadius` on incremental edits (same as `GZ`).
  Regression test in `internal/city/builder_test.go`.

- **Phase 1.5c — Frontend null guard:** `web/src/store/cityStore.ts`
  `districtThresholds?.[]` optional chaining. Backend sends `null` not `{}`.

- **Phase 2a — Treemap district sizing:** `internal/layout/engine.go` uniform
  grid replaced with `squarify()` weighted by total footprint area per
  district. No more `__pad_` placeholder districts. District GH expands if
  packed buildings overflow. Tests updated in
  `internal/layout/layout_test.go` (proportional area, no padding, count
  matches content).

- **README.md:** Updated to WIP status with progress table and known issues.

- **encoding-redesign.md:** Updated — status, decisions resolved (height
  normalization, footprint, cross-district compression, confidence weighting),
  phase sequencing reflects actual progress.

- **LICENSE:** Fletcher Bonds copyright line added alongside Mark Ferree's.

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
| 1 | HUD legend update | Legend reads "height = LOC" — now a lie. Quick frontend fix. |
| 2 | Churn pipeline (Phase 3) | Git history analysis, recency-weighted score, color mapping. Net-new data pipeline. |
| — | `encoding-redesign.md` §6 | Companion doc tasks (DESIGN.md pointer). Low priority. |

## Known issues

- **Agent overlay line cap:** Backend logs `parse failed: line exceeds size
  cap: 1368464 > 1048576` warnings from agentwatch hitting an upstream JSONL
  line cap. Agent tracking still works but some session updates are dropped.
  Separate issue from this fork's work.

## How to verify visually

Reload <http://localhost:5173> after the backend is up.

- Districts should be **different sizes** — the treemap gives more canvas area
  to districts with higher total blast radius.
- `internal/model/coverage_history.go` should still be the tallest building.
- `web/src/store/cityStore.ts` should now be visually comparable across
  districts (no longer crushed by dense-district packing).
