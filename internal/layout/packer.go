package layout

import (
	"math"
	"sort"

	"github.com/mferree/agent-city/internal/model"
)

const gutterSize = 2.0

// packDistrict computes layout fields (GX, GY, GW, GH, GZ) for buildings within
// the given district rectangle. Buildings are sorted by LOC descending (path
// ascending as tiebreaker) before packing, producing a deterministic shelf layout.
// Footprints are scaled down uniformly if necessary so all buildings stay within
// bw × bh. Returns the buildings with layout fields filled; original slice is not
// modified.
func packDistrict(buildings []model.Building, bx, by, bw, bh float64) []model.Building {
	out := make([]model.Building, len(buildings))
	copy(out, buildings)

	if len(out) == 0 {
		return out
	}

	// Deterministic order: LOC desc, path asc.
	sort.Slice(out, func(i, j int) bool {
		if out[i].LOC != out[j].LOC {
			return out[i].LOC > out[j].LOC
		}
		return out[i].ID < out[j].ID
	})

	// Scale footprints down if needed so buildings fit within bh.
	scale := footprintScale(out, bw, bh)

	// Shelf-pack: place buildings left-to-right; start a new shelf when width overflows.
	curX := bx + gutterSize
	curY := by + gutterSize
	shelfH := 0.0

	for i := range out {
		fw, fh := footprint(out[i].BlastRadius)
		fw *= scale
		fh *= scale

		// Start a new shelf if the building doesn't fit horizontally.
		if curX+fw+gutterSize > bx+bw && curX > bx+gutterSize {
			curY += shelfH + gutterSize
			curX = bx + gutterSize
			shelfH = 0
		}

		out[i].GX = curX
		out[i].GY = curY
		out[i].GW = fw
		out[i].GH = fh
		out[i].GZ = heightFromBlastRadius(out[i].BlastRadius)

		curX += fw + gutterSize
		if fh > shelfH {
			shelfH = fh
		}
	}

	return out
}

// footprintScale returns a uniform scale factor (≤ 1.0) so that shelf-packed
// buildings fit within bw × bh. Uses a two-pass estimate to account for the
// fixed gutter overhead (which does not scale with the footprints).
func footprintScale(buildings []model.Building, bw, bh float64) float64 {
	if bh <= 0 {
		return 1.0
	}
	needed := measurePackHeight(buildings, bw, 1.0)
	if needed <= bh {
		return 1.0
	}

	const minScale = 0.05

	// First estimate.
	s := math.Max(bh/needed, minScale)

	// Second pass: measure at scale s to correct for gutter overhead.
	needed2 := measurePackHeight(buildings, bw, s)
	if needed2 > bh && needed2 > 0 {
		s = math.Max(s*bh/needed2, minScale)
	}
	return s
}

// measurePackHeight simulates shelf packing at the given scale and returns the
// total height consumed (relative to origin 0).
func measurePackHeight(buildings []model.Building, bw, scale float64) float64 {
	curX := gutterSize
	curY := gutterSize
	shelfH := 0.0
	for _, b := range buildings {
		fw, fh := footprint(b.BlastRadius)
		fw *= scale
		fh *= scale
		if curX+fw+gutterSize > bw && curX > gutterSize {
			curY += shelfH + gutterSize
			curX = gutterSize
			shelfH = 0
		}
		curX += fw + gutterSize
		if fh > shelfH {
			shelfH = fh
		}
	}
	return curY + shelfH + gutterSize
}

// footprint returns the (width, depth) for a building with the given blast
// radius.
//
//	w = clamp(4 + √BR, 4, 12)
//	h = w × 0.8   (footprint depth)
//
// Both width and height (GZ) now encode blast radius. This is a deliberate
// reversal of the design doc's §5 working default ("footprint retains a faint
// file-size read"). Empirically the file-size read was not faint: in densely
// packed districts the packer scales footprints down, which combined with
// LOC-driven width made high-LOC / low-BR Go files visually dominate over
// low-LOC / high-BR TS stores — the inverse of what the encoding is for.
// Coupling footprint to blast radius makes structural risk own visual volume:
// a building's "size" matches what its name says — how much breaks if you
// touch it.
//
// Square root pairs with the log₂ height function (heightFromBlastRadius):
// height grows fast at the low end and saturates near the top of the long
// tail, while width grows more gradually, so combined volume contrast is
// dramatic for genuine skyscrapers without making BR=1 buildings absurd.
func footprint(blastRadius int) (float64, float64) {
	if blastRadius < 0 {
		blastRadius = 0
	}
	w := clamp(4.0+math.Sqrt(float64(blastRadius)), 4.0, 12.0)
	h := w * 0.8
	return w, h
}

// heightFromBlastRadius returns a building's visual height (GZ) from its blast
// radius — the count of files that transitively depend on it.
//
//	z = clamp(3 + 3·log₂(1 + br), 3, 30)
//
// Log scale handles the long-tailed dependent-count distribution: each
// doubling of dependents adds a constant slice of height. The [3, 30] window
// is required by the camera/pack pipeline (packDistrict scales footprints to
// fit a district rectangle, but does not scale z; values above 30 would clip
// the camera frustum).
func heightFromBlastRadius(br int) float64 {
	if br < 0 {
		br = 0
	}
	z := 3.0 + 3.0*math.Log2(1.0+float64(br))
	return clamp(z, 3.0, 30.0)
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
