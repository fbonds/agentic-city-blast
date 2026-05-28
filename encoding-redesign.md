# Encoding Redesign — `agentic-city-blast`

**Status:** Proposed. Nothing in this document has been implemented yet.
**Fork of:** [`mrf/agentic-city`](https://github.com/mrf/agentic-city) (MIT, © 2026 Mark Ferree).
**Purpose of this doc:** Record the encoding change that defines this fork, the
reasoning behind it, exactly which code it touches, and the decisions/tradeoffs
made — so work can resume from here if a session is lost before implementation
is complete.

---

## 1. Premise & decision

The upstream project encodes **file size** as a building's dominant visual
properties: both its height and its footprint are derived from lines of code,
and its color encodes the file's language. (See `internal/layout/packer.go`,
function `footprint(loc)`, which returns width, depth, and height all as
functions of `LOC`; and `web/src/canvas/BuildingRenderer.ts`, which tints side
faces by `b.language`.)

File size is a cheap signal that correlates poorly with what matters when you
are orchestrating agents. The decision of this fork:

- **Height encodes blast radius** — the number of files that transitively
  depend on this file. "If an agent edits this, how much of the system is
  downstream of it." This claims the dominant visual channel.
- **Color encodes churn** — how frequently the file has changed recently (from
  git history). This is a volatile, recency-weighted "where is the action"
  signal.
- **Position is unchanged.** Buildings remain anchored to their directory
  (district), and directories remain laid out by the existing treemap/grid
  engine. This is the fork's one inviolable constraint — see §3.

Footprint (`GW`/`GH`) is the open sub-decision recorded in §5; the working
default is that footprint may retain a faint file-size read since it is the
cheapest channel and "how much code is here" is mild useful context, just not
worth the dominant channel.

---

## 2. Why

### Who the user is

This is not a code-comprehension tool for someone onboarding to an unfamiliar
codebase — that was the use case behind the original academic "code city" work
(Wettel & Lanza, 2007), where size or complexity encodings make sense. This is
an **agent-orchestration** tool. The user is a "mayor" watching AI coding
sessions edit their repo in real time, and the decisions the city must support
are: *where do I send an agent, and what is the risk if I do.* The skyline's
job is to answer "what matters here and what is dangerous to touch" at a glance.

### Why blast radius beats churn for the dominant (height) channel

The most expensive failure mode in multi-agent orchestration is an agent
editing a high-leverage file and silently breaking things far away — and that
danger is invisible in every chat-based tool, which is the entire reason a
spatial tool earns its existence. Encoding blast radius as height means the
tallest buildings are exactly the ones to think twice before dispatching an
agent into, or the ones where you most want a human review gate.

Blast radius is also **forward-looking** (risk you are about to incur) and
**stable** (architecture changes slowly, so the skyline — and your spatial
memory of it — holds). Churn, by contrast, is backward-looking and volatile: a
file that churned heavily last week may be settled now, but a churn-based
skyline would still show it as a tower, pointing attention at yesterday's
activity. Worse, in an agent-driven world the agents *are* much of the churn —
so a churn-based skyline partly re-encodes the live agent layer (UFOs, tractor
beams) that the city already shows, making the static structure redundant with
the live layer rather than complementary to it.

### Why the two encodings are complementary, not redundant

Height = blast radius and color = churn produce a useful two-variable read:

- **Tall + hot** (high blast radius, actively changing) → look here first; an
  agent is touching something a lot of the system depends on.
- **Tall + cool** (high blast radius, settled) → important but stable;
  relatively safe.
- **Short + hot** (low blast radius, lots of churn) → lots of activity, little
  downstream risk; usually fine to let an agent run.

That tri-state reading is more useful than either metric alone, and it is why
churn earns a channel — just not the dominant one.

### Honest caveat

Blast radius is only as good as the dependency graph it is computed from. The
graph already exists (see §3) but is import-extraction–based and upstream flags
its accuracy as approximate (edge confidence is tracked as
`exact`/`inferred`/`weak`). Blast radius inherits that uncertainty. For a first
version this is acceptable — the *relative* ordering of buildings by transitive
dependents is informative even when absolute counts are imperfect — but
improving the analyzer is the highest-value follow-on work and is vision-neutral
(it would benefit upstream too).

---

## 3. What this touches (grounded in current source)

### The good news: the dependency graph already exists

`internal/deps/graph.go` already builds a directed, weighted, confidence-scored
dependency graph:

- `BuildGraph(buildings, readContent, cfg) []model.Road` produces edges.
- A `model.Road` has `FromID`, `ToID`, `Weight`, `Confidence`.
- **Edge direction:** `FromID` *imports* `ToID`. So `ToID` is depended-upon by
  `FromID`.

Therefore **blast radius of file X = the count of distinct files that can reach
X by following `FromID → ToID` edges in reverse** (everyone who imports X,
directly or transitively). This is pure aggregation of data the pipeline already
produces. No analyzer rewrite is required for a first version.

### Files in scope

**Backend (Phase 1 — blast radius):**

- `internal/deps/graph.go` (or a new `internal/deps/blastradius.go`): add a
  function that, given `[]model.Road`, computes a transitive-dependent count per
  building ID. Reverse the edges, then BFS/DFS from each node (or compute via
  reverse reachability). Watch for cycles — the import graph can contain them;
  use a visited set.
- `internal/model/model.go`: `Building` gains a field to carry the metric, e.g.
  `BlastRadius int json:"blastRadius"`. (`Exports int` already exists as
  precedent for a non-layout numeric field.)
- `internal/city/builder.go`: `AssembleState` already calls
  `deps.BuildGraph(...)`; after roads are built, compute blast radius per
  building and populate the new field before/within layout.
- `internal/layout/packer.go`, function `footprint(loc)`: this is the single
  place height (`GZ`) is assigned, currently `z = clamp(LOC/30, 3, 30)`. Height
  must instead be a function of blast radius, normalized to a comparable visual
  range (see §5 for the normalization decision). Footprint width/depth handling
  depends on the §5 footprint decision.

**Frontend (Phase 2 — churn color):**

- `web/src/canvas/BuildingRenderer.ts`: side-face tint currently comes from
  `LANG_COLORS[b.language]` (line ~218). Repurpose to a churn-derived color
  ramp. Note existing precedent in the same file: coverage already maps to a
  green/yellow/red window-dot color, and status maps to a stroke color — so
  value-based coloring infrastructure and palette entries already exist
  (`web/src/theme/colors.ts`, Solarized-dark).
- Legend / HUD (`web/src/hud/`): whatever currently documents the
  language-color and height-LOC mappings must be updated to the new meanings.

**New data pipeline (Phase 2 — churn):**

- There is currently **no churn or git-history data anywhere** in the repo.
  `internal/repo/metrics.go` covers coverage and test status only; the only git
  usage is `GatherRepoInfo` reading branch/commit. Churn requires net-new
  collection: shell out to `git log` per file (or a single `git log --name-only`
  pass), count recent commits within a window, cache the result, and surface it
  on `Building` (e.g. `Churn int`). This is the reason churn is Phase 2 — it is
  the expensive half.

### Explicitly NOT touched

- **Layout/position logic** (`internal/layout/engine.go` district grid; the
  treemap). Position stays directory-anchored.
- **The WebSocket hub** (`internal/hub/`).
- **agentwatch integration / agent tracking** (`internal/agents/`).
- **The agent (UFO) layer**, scanner, watcher core, coverage system.

---

## 4. Sequencing

Reading the code inverts the sequencing we initially assumed. Blast radius is
*cheaper* than churn, not harder, because the graph already exists and churn
data does not.

**Phase 1 — Blast radius as height (ship first).**
Self-contained, backend-only, aggregates existing graph data, touches one height
function plus a model field and the builder. High value on its own. This is the
minimum that makes the fork's thesis real and the README honest.

**Phase 2 — Churn as color (ship second).**
Requires a new git-history data pipeline plus frontend renderer and legend
changes. Independent of Phase 1 and can land later without blocking it.

Each phase is independently shippable and independently useful.

---

## 5. Decisions & tradeoffs recorded

### Height refresh cadence (DECIDED)

`internal/city/builder.go`'s `MergeBuildings` deliberately preserves
`GX/GY/GW/GH/GZ` across incremental updates so buildings don't jump on every
file edit. With height = blast radius, a *correct* live height would change when
dependency edges change — which the current merge logic freezes.

**Decision:** For Phase 1, height (blast radius) refreshes only on a **full
rescan**, not on incremental single-file merges. Height may therefore be
briefly stale between rescans. This keeps the change small and preserves the
anti-jump behavior. Live height recomputation on edge changes is deferred as a
possible later enhancement. (Position stability — the property that actually
matters — is unaffected either way, since position is directory-anchored, not
blast-radius-derived.)

### Blast-radius → height normalization (OPEN — resolve in implementation)

Current height is `clamp(LOC/30, 3, 30)`. Blast radius is an unbounded integer
count with a very different distribution (likely long-tailed: most files have
near-zero dependents, a few have many). A linear map will make almost every
building minimum-height with a handful of skyscrapers. Candidate approaches to
decide during implementation: log scale, percentile/rank-based mapping, or
clamped linear with an empirically chosen divisor. Keep the same visual range
(~3–30) so the camera/layout math is undisturbed.

### Footprint (GW/GH) meaning (OPEN — resolve in implementation)

Working default: footprint retains a faint file-size read (cheapest channel,
mild context). Alternative: free the channel for a third metric later. No commit
required for Phase 1 beyond "footprint is not the dominant signal."

### Churn noise (OPEN — Phase 2)

Git churn naively counts generated files, lockfiles, formatting passes, and mass
renames as "hot." Phase 2 must filter these (e.g. respect `.gitignore`-style
exclusions, ignore lockfiles/generated paths) or the hottest colors will be
meaningless.

### Fork vs. contribute (DECIDED)

**Fork.** The change alters upstream's stated thesis ("file sizes determine
building heights," front and center in its README and DESIGN.md), which a
maintainer cannot be assumed to want. The relationship stays friendly: any
dependency-analyzer improvements are vision-neutral and could be offered
upstream.

---

## 6. Companion documents (not this doc)

These are separate deliverables, to be done after this design doc is approved:

1. **README rewrite.** The current README contains two now-false sentences —
   the tagline "Buildings are files... File sizes determine building heights"
   and the body "File sizes determine building heights." These must be replaced.
   The README must also state that this is a fork of `mrf/agentic-city` (with
   link), the thesis difference (encodes structural risk, not file size), the
   new legend in words (height = blast radius, color = churn), and an honest
   implemented-vs-planned status (blast radius first, churn planned).
2. **LICENSE.** MIT requires retaining Mark Ferree's copyright and permission
   notice. Keep his copyright line; a fork-maintainer copyright line for new
   contributions may be added but his may not be removed.
3. **DESIGN.md pointer (light touch).** Upstream `DESIGN.md` (~54KB) also leads
   on the old metaphor, but most of it (layout, WebSocket protocol, keyboard
   bindings) remains accurate. Rather than rewrite it wholesale, add a short
   note at its top stating the encoding model changed in this fork and pointing
   to this document. A full rewrite is out of scope for now.

### Open questions deferred to the README phase

- Repo/short description wording (full-vision vs. honest-to-first-release
  phrasing for blast radius before churn ships).
- Whether the GitHub repo-description field should spell out "blast radius =
  how much breaks if you touch it" for readers with no project context.
