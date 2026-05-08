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
// Returns the buildings with layout fields filled; original slice is not modified.
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

	// Shelf-pack: place buildings left-to-right; start a new shelf when width overflows.
	curX := bx + gutterSize
	curY := by + gutterSize
	shelfH := 0.0

	for i := range out {
		fw, fh, fz := footprint(out[i].LOC)

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
		out[i].GZ = fz

		curX += fw + gutterSize
		if fh > shelfH {
			shelfH = fh
		}
	}

	return out
}

// footprint returns the (width, depth, height) for a building with the given LOC.
//
//	w = clamp(√(LOC/20), 4, 12)
//	h = w × 0.8   (footprint depth)
//	z = clamp(LOC/30, 3, 30)  (visual height)
func footprint(loc int) (w, h, z float64) {
	w = clamp(math.Sqrt(float64(loc)/20.0), 4.0, 12.0)
	h = w * 0.8
	z = clamp(float64(loc)/30.0, 3.0, 30.0)
	return
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
