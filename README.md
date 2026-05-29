# agentic-city-blast

A fork of [mrf/agentic-city](https://github.com/mrf/agentic-city) by Mark Ferree.

**Status: Work in progress.** The encoding redesign described below is actively being implemented. See the progress table for current state.

## Why this exists

The upstream project is remarkable. It turns a codebase into a living isometric city — files become buildings, directories become districts, AI coding sessions appear as UFOs flying overhead, and everything updates in real time over WebSockets. It is a genuinely impressive piece of work, generously published under MIT, and I would not have thought to build it.

I came across it through a coworker. While chatting with Claude Code about the premise, one thing kept circling: the dominant visual channel — building height — is driven by file size. File size is the cheapest signal to compute, but it may not be the most useful one for the use case the city is pitched at (orchestrating AI agents). And so: *this is awesome — I wonder what it would be like if the metric were something else though.*

The redesign:

- **Height** encodes **blast radius** — how many files transitively depend on this one.
- **Footprint** encodes **blast radius** — coupled with height so a building's visual volume represents structural risk.
- **District sizing** uses a **squarified treemap** weighted by content, so cross-district comparison is meaningful.
- **Color** encodes **churn** — how often the file has changed in the last 90 days (cyan = cold, red = hot).

The full reasoning, code it touches, and open decisions are in [encoding-redesign.md](encoding-redesign.md).

## Progress

| Phase | What | Status | Notes |
|-------|------|--------|-------|
| 1 | Blast radius → building height | **Done** | `blastradius.go` computes transitive dependents via reverse BFS. `heightFromBlastRadius` uses `3 + 3·log₂(1+BR)`, clamped to [3, 30]. |
| 1.5a | Blast radius → building footprint | **Done** | `footprint()` switched from LOC to BR. `w = clamp(4 + √BR, 4, 12)`, `h = 0.8w`. |
| 1.5b | `MergeBuildings` BR preservation | **Done** | Bug fix: incremental edits no longer zero out blast radius on the edited file. |
| 1.5c | Frontend null guard | **Done** | `districtThresholds` null crash fixed in `cityStore.ts`. |
| 2a | Treemap district sizing | **Done** | Districts sized by total footprint area via `squarify()`. Fixes the cross-district compression problem where high-BR files in dense districts looked smaller than low-BR files in sparse ones. |
| 2b | HUD legend update | **Done** | `Building` interface now includes `blastRadius`. RightRail shows blast radius when a building is selected. |
| 3 | Color → churn pipeline | **Done** | `git log --since=90days` counts commits per file. Log-normalized to [0,1]. 5-stop color ramp: cyan → blue → yellow → orange → red. Replaces language-based building tint. |

## Known issues

- **Agent overlay line cap:** Backend logs `parse failed: line exceeds size cap` warnings from agentwatch hitting an upstream JSONL size limit. Agent (UFO) tracking still works but some session state updates are dropped.
- **Agent-driven churn noise:** Agents are much of the churn in an agent-driven workflow, so the churn signal partly re-encodes the live UFO layer. Filtering agent commits would require correlating agentwatch sessions with git authorship — flagged as a known limitation, not yet addressed.

## Honest caveats

I genuinely do not know if this will work, or if it will be of value to anyone beyond satisfying my own curiosity. The dependency graph this builds on is import-extraction-based, and upstream itself flags it as approximate, so blast-radius numbers inherit that uncertainty. It is entirely possible the result reads worse than the original.

This is also not a critique of upstream, which I think is excellent. The reason it lives in a fork rather than a PR is that the encoding swap conflicts with upstream's stated thesis ("file sizes determine building heights"), and asking Mark to take that on would not be respectful of his vision. Any dependency-analyzer improvements that fall out of this are vision-neutral and I would be happy to offer them back.

## Running it

See [DESIGN.md](DESIGN.md) for full architecture and keyboard bindings:

```bash
# Development (two terminals)
go run ./cmd/agentic-city --repo=/path/to/repo   # backend on :8080
cd web && npm run dev                            # Vite dev server on :5173 (proxies /api /ws)

# Or via Makefile
make run            # build everything, start server
make dev            # frontend dev server only

# Demo mode (synthetic city, no real repo needed)
go run ./cmd/agentic-city --demo
```

## License

MIT, retaining Mark Ferree's original copyright. See [LICENSE](LICENSE).
