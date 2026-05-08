package layout

import (
	"math"
	"sort"
)

// treemapNode is a weighted leaf for the squarified treemap algorithm.
type treemapNode struct {
	id     string
	weight float64
}

// treemapRect is the placed rectangle for a treemapNode.
type treemapRect struct {
	id         string
	x, y, w, h float64
}

// squarify computes a squarified treemap layout for nodes within (x, y, w, h).
// Nodes are sorted by weight descending before layout for best aspect ratios.
// Returns one treemapRect per node in input order.
func squarify(nodes []treemapNode, x, y, w, h float64) []treemapRect {
	n := len(nodes)
	out := make([]treemapRect, n)
	if n == 0 || w <= 0 || h <= 0 {
		for i, nd := range nodes {
			out[i].id = nd.id
		}
		return out
	}

	// Normalize weights to fill total area.
	totalW := 0.0
	for _, nd := range nodes {
		totalW += nd.weight
	}
	if totalW == 0 {
		totalW = float64(n)
	}
	area := w * h
	areas := make([]float64, n)
	for i, nd := range nodes {
		areas[i] = nd.weight / totalW * area
		if areas[i] <= 0 {
			areas[i] = area / float64(n)
		}
	}

	// Build sorted index by weight desc, path asc as tiebreaker.
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	sort.Slice(idx, func(a, b int) bool {
		if nodes[idx[a]].weight != nodes[idx[b]].weight {
			return nodes[idx[a]].weight > nodes[idx[b]].weight
		}
		return nodes[idx[a]].id < nodes[idx[b]].id
	})

	sortedAreas := make([]float64, n)
	for i, orig := range idx {
		sortedAreas[i] = areas[orig]
	}

	placedRects := make([]treemapRect, n)
	squarifyLayout(sortedAreas, x, y, w, h, placedRects)

	// Map placed rectangles back to original input order using the sort index.
	for sortedPos, origPos := range idx {
		r := placedRects[sortedPos]
		r.id = nodes[origPos].id
		out[origPos] = r
	}
	return out
}

// squarifyLayout is the iterative core of the squarified treemap algorithm.
// It fills out[] with placed rectangles for the given areas.
func squarifyLayout(areas []float64, x, y, w, h float64, out []treemapRect) {
	n := len(areas)
	i := 0
	for i < n {
		if w <= 0 || h <= 0 {
			// No space left; place remaining items at a degenerate point.
			for k := i; k < n; k++ {
				out[k].x, out[k].y = x, y
				out[k].w, out[k].h = 0, 0
			}
			break
		}

		// Lay items along the longer side.
		horizontal := w >= h
		var L float64
		if horizontal {
			L = w
		} else {
			L = h
		}

		// Build the row: keep adding items while aspect ratio improves (or stays equal).
		rowSum := areas[i]
		j := i + 1
		for j < n {
			newSum := rowSum + areas[j]
			if worstAR(areas[i:j], rowSum, L) >= worstAR(areas[i:j+1], newSum, L) {
				rowSum = newSum
				j++
			} else {
				break
			}
		}

		// Place the row items.
		placeRow(areas[i:j], rowSum, x, y, w, h, horizontal, out[i:j])

		// Advance the remaining bounding rectangle.
		if horizontal {
			stripH := rowSum / w
			y += stripH
			h -= stripH
		} else {
			stripW := rowSum / h
			x += stripW
			w -= stripW
		}
		i = j
	}
}

// worstAR returns the worst (largest) aspect ratio among items in a row.
// L is the length of the dimension along which items are laid out.
// rowSum is the sum of all areas in the row.
func worstAR(row []float64, rowSum, L float64) float64 {
	if len(row) == 0 || L == 0 || rowSum == 0 {
		return math.MaxFloat64
	}
	// Each item has one side = a*L/rowSum and the other = rowSum/L.
	// Aspect ratio = max(a*L²/rowSum², rowSum²/(a*L²)).
	L2 := L * L
	s2 := rowSum * rowSum
	worst := 0.0
	for _, a := range row {
		if a <= 0 {
			continue
		}
		ar := math.Max(a*L2/s2, s2/(a*L2))
		if ar > worst {
			worst = ar
		}
	}
	return worst
}

// placeRow positions items within the current strip.
// horizontal=true: strip is a horizontal band; items placed left-to-right.
// horizontal=false: strip is a vertical band; items placed top-to-bottom.
func placeRow(row []float64, rowSum, x, y, w, h float64, horizontal bool, out []treemapRect) {
	if horizontal {
		if w == 0 {
			return
		}
		stripH := rowSum / w
		curX := x
		for i, a := range row {
			itemW := a / stripH
			out[i].x = curX
			out[i].y = y
			out[i].w = itemW
			out[i].h = stripH
			curX += itemW
		}
	} else {
		if h == 0 {
			return
		}
		stripW := rowSum / h
		curY := y
		for i, a := range row {
			itemH := a / stripW
			out[i].x = x
			out[i].y = curY
			out[i].w = stripW
			out[i].h = itemH
			curY += itemH
		}
	}
}
