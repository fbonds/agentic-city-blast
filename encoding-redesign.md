# Encoding Redesign — `agentic-city-blast`

**Status:** In progress. Phases 1, 1.5, and 2a are implemented and committed.
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
  downstream of it." This claims the dominant visual channel. *(Implemented.)*
- **Footprint encodes blast radius** — coupled with height so a building's
  visual volume represents structural risk. `w = clamp(4 + √BR, 4, 12)`,
  `h = 0.8w`. *(Implemented — see §5 for the decision rationale.)*
- **Color encodes churn** — how frequently the file has changed recently (from
  git history). This is a volatile, recency-weighted "where is the action"
  signal. *(Not yet implemented.)*
- **District sizing uses a squarified treemap** weighted by total footprint
  area per district. This replaced the uniform grid so cross-district visual
  comparison is meaningful. *(Implemented.)*
- **Position is unchanged.** Buildings remain anchored to their directory
  (district). This is the fork's one inviolable constraint — see §3.

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

**Backend (Phase 1 — blast radius → height) — DONE:**

- `internal/deps/blastradius.go`: computes transitive-dependent count per
  building ID via reverse BFS. Handles cycles with a visited set.
- `internal/model/model.go`: `Building.BlastRadius int` field added.
- `internal/city/builder.go`: `AssembleState` calls `ComputeBlastRadius` after
  `BuildGraph`, populates the field. `MergeBuildings` preserves `BlastRadius`
  on incremental updates (bug fix).
- `internal/layout/packer.go`: `heightFromBlastRadius(br)` =
  `clamp(3 + 3·log₂(1+BR), 3, 30)`. Log scale handles the long-tailed
  distribution.

**Backend (Phase 1.5 — blast radius → footprint) — DONE:**

- `internal/layout/packer.go`: `footprint()` switched from `LOC` to
  `BlastRadius`. `w = clamp(4 + √BR, 4, 12)`, `h = 0.8w`. Square root pairs
  with the log₂ height: combined volume contrast is dramatic for high-BR files
  without making BR=1 files absurd.

**Backend (Phase 2a — treemap district sizing) — DONE:**

- `internal/layout/engine.go`: uniform grid replaced with `squarify()` call,
  weighted by total natural footprint area (Σ fw·fh) per district. No more
  `__pad_` placeholder districts. District GH expands if packed buildings
  overflow vertically.
- `internal/layout/layout_test.go`: grid tests replaced with treemap invariant
  tests (proportional area, no padding, count matches content).

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

- **The WebSocket hub** (`internal/hub/`).
- **agentwatch integration / agent tracking** (`internal/agents/`).
- **The agent (UFO) layer**, scanner, watcher core, coverage system.

**Note:** The original plan listed `internal/layout/engine.go` as not-touched.
In practice, the cross-district compression problem (see §5) required replacing
the uniform grid with a content-weighted treemap (Phase 2a). Position remains
directory-anchored; only district *sizing* changed.

---

## 4. Sequencing

Reading the code inverts the sequencing we initially assumed. Blast radius is
*cheaper* than churn, not harder, because the graph already exists and churn
data does not.

**Phase 1 — Blast radius as height. DONE.**
Backend-only. `blastradius.go` computes transitive dependents via reverse BFS.
`heightFromBlastRadius` uses log₂ scale, clamped to [3, 30].

**Phase 1.5 — Blast radius as footprint. DONE.**
`footprint()` switched from LOC to BR. Also fixed a `MergeBuildings` bug that
zeroed BR on incremental edits, and a frontend null-guard crash.

**Phase 2a — Treemap district sizing. DONE.**
Uniform grid replaced with `squarify()` weighted by total footprint area per
district. Fixes the cross-district compression problem where high-BR files in
dense districts looked smaller than low-BR files in sparse ones.

**Phase 2b — HUD legend update. TODO.**
Legend still reads "height = LOC" — now incorrect.

**Phase 3 — Churn as color. TODO.**
Requires a new git-history data pipeline plus frontend renderer and legend
changes. Independent of preceding phases.

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

### Blast-radius → height normalization (DECIDED — implemented)

**Decision:** Log₂ scale. `z = clamp(3 + 3·log₂(1 + BR), 3, 30)`. Each
doubling of dependents adds a constant slice of height. The [3, 30] window is a
hard requirement (per CLI review item #4): `packDistrict` rescales footprints
but not `z`, so unclamped values would clip the camera frustum.

### Footprint (GW/GH) meaning (DECIDED — implemented)

**Decision:** Footprint encodes blast radius, not file size. The original
working default ("footprint retains a faint file-size read") was reversed during
Phase 1.5 implementation. Empirically, LOC-driven footprint in densely packed
districts made high-LOC / low-BR files visually dominate over low-LOC / high-BR
files — the inverse of the encoding's intent. Coupling footprint to BR makes
visual volume consistently represent structural risk.

Formula: `w = clamp(4 + √BR, 4, 12)`, `h = 0.8w`. Square root pairs with
log₂ height: height grows fast at the low end (discriminates small BR values),
while width grows gradually, so combined volume contrast is dramatic for
genuine skyscrapers without making BR=1 buildings absurd.

### Cross-district compression (DECIDED — implemented)

Emerged during Phase 1.5 testing. Within a district, BR encoding was correct,
but `packDistrict` shrinks footprints uniformly to fit a district rectangle,
and the uniform grid gave all districts equal canvas area regardless of content.
Result: the highest-BR file in the city (`cityStore.ts`, BR=37) was visually
smaller than lower-BR files in sparser districts.

**Decision:** Option 2 — weight district sizes by content. The `squarify`
function (already in `internal/layout/treemap.go`, previously unused for
districts) now sizes districts proportionally to their total natural footprint
area. This preserves cross-district visual comparison.

### Churn noise (OPEN — Phase 2)

Git churn naively counts generated files, lockfiles, formatting passes, and mass
renames as "hot." Phase 2 must filter these (e.g. respect `.gitignore`-style
exclusions, ignore lockfiles/generated paths) or the hottest colors will be
meaningless.

### Confidence weighting (DECIDED)

**Decision:** Binary count — all edges treated equal regardless of confidence.
Per CLI review item #5: confidence-weighting muddies the "N files depend on
this" semantics, decimals don't read off a skyline, and it decouples
analyzer-quality work from height-encoding work. Implemented as-is in
`blastradius.go`.

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

---

## 7. CLI review (2026-05-28)

This review was added by Claude Code (CLI) after reading the doc against the
current source. The doc's web-counterpart authorship is otherwise preserved.
All load-bearing claims were spot-checked: edge direction verified in
`internal/deps/analyzer_test.go:237`; position-by-directory verified in
`internal/layout/engine.go`; the `LOC→GZ` assignment verified in
`internal/layout/packer.go:122`.

### What holds up

- **Premise.** File size is a poor signal for the agent-orchestration
  decision ("where do I send an agent, and what's the risk"). The dominant
  visual channel should carry structural risk, not size.
- **Phasing inversion.** Blast radius really is cheaper than churn. The graph
  exists; churn data does not. Ship in the order proposed.
- **Edge direction.** "Blast radius = files that reach X by reversing
  `FromID → ToID`" matches the code.
- **Tri-state read.** Tall+hot / tall+cool / short+hot is a stronger
  justification for color = churn than churn alone would be.
- **Honest caveat.** The edge-confidence inheritance disclaimer is
  appropriately humble; the relative-ordering escape valve defangs the
  strongest reviewer objection.

### Issues worth resolving before implementation

1. **§5.1 position-stability wording is slightly loose.** District assignment
   is directory-anchored, but *position within a district* is set by
   `packDistrict`, which sorts on `LOC desc, ID asc`
   (`internal/layout/packer.go:27`). The conclusion still holds — blast
   radius doesn't feed the packer's sort key — but the phrasing "position is
   directory-anchored, not blast-radius-derived" papers over the LOC sort.
   Recommend: "blast radius does not feed the packer's sort key, so
   within-district position is unaffected."

2. **§5.1 doesn't say what GZ a brand-new file gets between rescans.**
   `MergeBuildings` preserves `GZ` for existing entries
   (`internal/city/builder.go:132`), but for a brand-new file `u.GZ` is
   whatever the incremental path produces. In the LOC world this was
   trivially `LOC/30`; in the blast-radius world there is no answer without
   re-running `BuildGraph`. Phase 1 needs an explicit rule — recommend new
   buildings get the minimum height (3.0) until the next full rescan
   promotes them. Otherwise they render flat (z=0) and look like a bug.

3. **The `footprint(loc)` change isolates better than the doc says — say
   so.** `measurePackHeight` and `footprintScale` only consume `fw, fh` from
   `footprint()`; they discard `z` (`internal/layout/packer.go:96`, `:72`).
   So keeping `footprint(loc)` for w/h and adding a separate
   `height(blastRadius)` leaves the entire scale/pack pipeline untouched.
   Worth committing to that split in §3 rather than leaving footprint as
   "open" — it converts a worry into a small Phase 1 plus.

4. **The 30 ceiling is load-bearing, not just a guideline.** `packDistrict`
   rescales footprints to fit a district but does not rescale `z`. A
   long-tail blast-radius value with no clamp would clip the camera frustum.
   §5 should make "clamp to [3,30]" a hard requirement of the normalization
   decision, not a "candidate approach."

5. **Confidence weighting is an unanswered knob.** Edges carry
   `Confidence ∈ {exact, inferred, weak}` (`internal/model/model.go:101`).
   Should `weak` transitive paths pump blast radius as hard as `exact` ones?
   Recommend pre-deciding *binary count* for Phase 1: treat all edges equal.
   Confidence-weighting muddies the "N files depend on this" semantics, and
   decimals don't read off a skyline. This also decouples analyzer-quality
   work from the height-encoding work.

6. **§2 identifies the redundancy trap; §5 churn-noise doesn't close it.**
   §2 rules out churn-as-height because "agents *are* much of the churn."
   Color carries the same risk: agent-driven commits will dominate the churn
   signal and re-encode the UFO layer. §5's lockfile/generated filters don't
   address this — exclusion would require correlating agentwatch session
   windows with `git log` authorship. Flag in §5 as an open Phase 2 problem,
   not just noise filtering.

### Smaller items

- The fourth quadrant — short + cool — goes unmentioned in §2. The
  implication (most of the city stays quiet — a feature, not a bug) is
  worth one sentence so readers don't expect a uniformly-vivid skyline.
- §6 README rewrite list is correct. The DESIGN.md "light touch" pointer is
  the right call.

### Net

Structurally sound; the items above are tightening, not redirection. The
biggest gap is #2 (new-file GZ between rescans) — that question will surface
on day one of Phase 1.
