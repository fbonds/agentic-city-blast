# Session handoff — Phase 1.5 in progress

**Last touched:** 2026-05-29. Prior session was Claude Code (desktop app), being
handed off to terminal CC running in this directory so the working-directory
mismatch goes away (and UFO/agent overlays start working).

---

## Committed and pushed

- **Phase 1:** blast radius → building height. Backend-only. See
  `internal/deps/blastradius.go`, `internal/model/model.go` (`BlastRadius`
  field), `internal/city/builder.go` (`AssembleState` reorder), and
  `internal/layout/packer.go` (`heightFromBlastRadius`).
- **LICENSE:** Fletcher Bonds (fbonds) copyright line added alongside Mark
  Ferree's.
- **README + encoding-redesign.md:** §7 CLI review appended, README rewritten
  as a "thought experiment, not a project" framing.

## Uncommitted on disk right now

Four distinct change groups:

### 1. Phase 1.5 footprint refactor (the design decision "go with #1")
- `internal/layout/packer.go`: `footprint()` now takes `BlastRadius` instead
  of `LOC`. New math: `w = clamp(4 + √BR, 4, 12)`, `h = 0.8w`. Reasoning
  documented in the function comment.
- `internal/layout/engine.go`: canvas-size calc updated for the new
  signature.
- `internal/layout/layout_test.go`: `TestFootprint` rewritten with BR cases.

### 2. `MergeBuildings` BlastRadius preservation (bug fix discovered live)
- `internal/city/builder.go`: `MergeBuildings` now preserves `BlastRadius`
  from the existing entry, same way it preserves `GZ`. Without this, an
  incremental edit (watcher fire) zeroed out the BR of the edited file while
  the renderer kept the old `GZ`, causing the encoding to disagree with
  itself.
- `internal/city/builder_test.go`: added
  `TestMergeBuildings_PreservesBlastRadius` regression test.

### 3. Frontend defensive guard (pre-existing upstream bug, exposed at runtime)
- `web/src/store/cityStore.ts`: `settings.districtThresholds[d.id]` →
  `settings.districtThresholds?.[d.id]`. The backend serializes
  `districtThresholds: null` rather than `{}`, and the frontend crashed at
  startup before the city ever rendered.

### 4. Build stub (not for commit)
- `web/dist/placeholder.txt`: satisfies `//go:embed dist` in `web/static.go`
  so `go run ./cmd/agentic-city` compiles before `npm run build` has been
  executed. `web/dist/` is gitignored.

**Tests:** `go test ./internal/{model,deps,city,layout}/...` — all green.
`go vet ./internal/...` — clean.

---

## What's currently running (probably)

Two background processes started during the previous session. They may or may
not survive the desktop-app shutdown; check with `lsof -i :8080,5173` and kill
if needed:

- **Backend:** `go run ./cmd/agentic-city -repo /Users/fletcherbonds/code/agentic-city-blast -addr :8080`
- **Vite dev server:** `npm run dev` from `web/`

To restart cleanly from terminal CC in this directory:

```bash
# Backend (in one terminal or background):
/opt/homebrew/bin/go run ./cmd/agentic-city -repo .

# Frontend dev server (in another):
PATH="/Users/fletcherbonds/.nvm/versions/node/v22.20.0/bin:$PATH" npm --prefix web run dev
```

System node is 14.15.0 (too old for Vite 5); v22.20.0 is the one used.
Go was installed via Homebrew yesterday (2026-05-28): `/opt/homebrew/bin/go`.

Browser: <http://localhost:5173>.

---

## Where the visual currently lands

The encoding is **mathematically correct** (blast radius drives both height
and footprint) but **only reads well within a district**. Cross-district
comparison is broken because `packDistrict` shrinks footprints uniformly to
fit the district rectangle, and the upstream layout uses a **uniform grid**
for districts regardless of content.

Concrete: `web/src/store/cityStore.ts` has the highest BR in the city (37).
Natural width: 10. After dense-district packing: **GW=1.22**. Meanwhile
`internal/model/coverage_history.go` (BR=28, sparser district) keeps full
**GW=9.29** and visually dominates. The single highest-BR file looks smaller
than the second-highest file because of district-packing density. Not what
the design doc imagined.

## Open decision (this is where the next session picks up)

User leans toward option 2 but hasn't committed. Three paths:

1. **Accept it.** Within-district BR encoding is honest and useful; cross-
   district isn't. Document it, move on.
2. **Weight district sizes by content.** Switch `internal/layout/engine.go`
   from uniform grid to the `squarify` function that already exists in
   `internal/layout/treemap.go` (currently unused for districts). Districts
   holding more or higher-BR buildings would get more canvas area, removing
   the cross-district compression. Biggest change but cleanly fixes the
   visibility problem.
3. **Hybrid:** keep the grid for memorizability but drop the within-district
   footprint scaling. Districts grow vertically as needed (precedent:
   `TestLayout_BuildingsWithinDistrictBounds`). Risk: visual chaos as some
   districts spill far below others.

## Still deferred (called out in design doc)

- **Frontend legend** still reads "height = LOC" — now a lie. Phase 1.5 HUD
  cleanup.
- **Color** still encodes language. Phase 2 (churn pipeline).
- **encoding-redesign.md §5** still lists footprint as `OPEN`. Resolve to
  reflect the Phase 1.5 decision.
- **Agent overlays (UFOs):** invisible in the desktop-CC session because
  that session's CWD was the test-plan project, not this repo. Terminal CC
  here fixes that for free. There are also recurring `parse failed: line
  exceeds size cap: 1368464 > 1048576` warnings in the backend log from
  agentwatch hitting an upstream JSONL line cap — separate issue.

## How to verify the current state visually

Reload <http://localhost:5173> after the backend is up.

- Find `internal/model/coverage_history.go` in the `MODEL/` district — should
  be obviously the tallest and biggest building in the city.
- Find `web/src/store/cityStore.ts` in the `STORE/` district — should be
  visibly ~2.5× wider than its neighbors (the test files), but absolutely
  still small compared to the Go internals. This is the cross-district
  problem above.
