package deps

import (
	"github.com/mferree/agent-city/internal/model"
)

// Compute returns the blast radius of each building — the count of distinct
// files that transitively depend on it — keyed by building ID.
//
// Given the edge convention that FromID imports ToID, the blast radius of X
// is the count of distinct nodes reachable from X in the reverse graph,
// excluding X itself. "If an agent edits X, how much of the system is
// downstream of it."
//
// Cycles are handled — each node is visited at most once per starting point.
// Buildings with no incoming dependencies have blast radius 0. Every ID in
// buildingIDs appears in the result.
//
// Phase 1 uses a binary count: all edges contribute equally regardless of
// model.Road.Confidence. Confidence-weighted blast radius is intentionally
// deferred (decimals do not read off a skyline, and the semantic gain is
// muddier than the analyzer-quality improvements that would feed it).
func Compute(buildingIDs []string, roads []model.Road) map[string]int {
	out := make(map[string]int, len(buildingIDs))
	for _, id := range buildingIDs {
		out[id] = 0
	}
	if len(roads) == 0 {
		return out
	}

	reverse := make(map[string][]string)
	for _, r := range roads {
		reverse[r.ToID] = append(reverse[r.ToID], r.FromID)
	}

	for _, id := range buildingIDs {
		out[id] = countReverseReachable(id, reverse)
	}
	return out
}

// countReverseReachable performs BFS in the reverse-import graph starting at
// `start`, returning the count of distinct nodes reached (excluding start).
func countReverseReachable(start string, reverse map[string][]string) int {
	visited := map[string]bool{start: true}
	queue := []string{start}
	count := 0
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		for _, prev := range reverse[node] {
			if visited[prev] {
				continue
			}
			visited[prev] = true
			count++
			queue = append(queue, prev)
		}
	}
	return count
}
