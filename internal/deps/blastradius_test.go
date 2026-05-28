package deps

import (
	"testing"

	"github.com/mferree/agent-city/internal/model"
)

func TestCompute_NoRoads(t *testing.T) {
	ids := []string{"a.go", "b.go"}
	got := Compute(ids, nil)
	if got["a.go"] != 0 || got["b.go"] != 0 {
		t.Errorf("expected zero blast radius for all, got %v", got)
	}
	if len(got) != 2 {
		t.Errorf("result should contain every input ID; got %d entries", len(got))
	}
}

func TestCompute_LinearChain(t *testing.T) {
	// a imports b imports c.
	// Reverse-reachability: a=0, b=1 (a), c=2 (b, a).
	ids := []string{"a.go", "b.go", "c.go"}
	roads := []model.Road{
		{FromID: "a.go", ToID: "b.go"},
		{FromID: "b.go", ToID: "c.go"},
	}
	got := Compute(ids, roads)
	if got["a.go"] != 0 {
		t.Errorf("a.go = %d, want 0", got["a.go"])
	}
	if got["b.go"] != 1 {
		t.Errorf("b.go = %d, want 1", got["b.go"])
	}
	if got["c.go"] != 2 {
		t.Errorf("c.go = %d, want 2", got["c.go"])
	}
}

func TestCompute_Diamond(t *testing.T) {
	// a→b, a→c, b→d, c→d. d is reached transitively by 3 distinct nodes (a, b, c).
	ids := []string{"a.go", "b.go", "c.go", "d.go"}
	roads := []model.Road{
		{FromID: "a.go", ToID: "b.go"},
		{FromID: "a.go", ToID: "c.go"},
		{FromID: "b.go", ToID: "d.go"},
		{FromID: "c.go", ToID: "d.go"},
	}
	got := Compute(ids, roads)
	if got["d.go"] != 3 {
		t.Errorf("d.go = %d, want 3 (a, b, c via diamond)", got["d.go"])
	}
	if got["a.go"] != 0 {
		t.Errorf("a.go = %d, want 0 (no incoming edges)", got["a.go"])
	}
}

func TestCompute_Cycle(t *testing.T) {
	// Mutual import: a→b and b→a. Each reaches the other exactly once.
	ids := []string{"a.go", "b.go"}
	roads := []model.Road{
		{FromID: "a.go", ToID: "b.go"},
		{FromID: "b.go", ToID: "a.go"},
	}
	got := Compute(ids, roads)
	if got["a.go"] != 1 {
		t.Errorf("a.go = %d, want 1", got["a.go"])
	}
	if got["b.go"] != 1 {
		t.Errorf("b.go = %d, want 1", got["b.go"])
	}
}

func TestCompute_ThreeCycle(t *testing.T) {
	// a→b→c→a. Each node reaches the other two.
	ids := []string{"a.go", "b.go", "c.go"}
	roads := []model.Road{
		{FromID: "a.go", ToID: "b.go"},
		{FromID: "b.go", ToID: "c.go"},
		{FromID: "c.go", ToID: "a.go"},
	}
	got := Compute(ids, roads)
	for _, id := range ids {
		if got[id] != 2 {
			t.Errorf("%s = %d, want 2 (3-cycle)", id, got[id])
		}
	}
}

func TestCompute_IgnoresConfidence(t *testing.T) {
	// Binary count for Phase 1: weak edges contribute the same as exact.
	ids := []string{"a.go", "b.go"}
	roads := []model.Road{
		{FromID: "a.go", ToID: "b.go", Confidence: ConfidenceWeak},
	}
	got := Compute(ids, roads)
	if got["b.go"] != 1 {
		t.Errorf("b.go = %d, want 1 (weak edge should still count)", got["b.go"])
	}
}

func TestCompute_UnknownIDsInRoadsIgnored(t *testing.T) {
	// Roads can reference IDs not present in buildingIDs (e.g. external
	// modules); they should not appear in the result but should not panic.
	ids := []string{"a.go"}
	roads := []model.Road{
		{FromID: "external/lib", ToID: "a.go"},
	}
	got := Compute(ids, roads)
	if got["a.go"] != 1 {
		t.Errorf("a.go = %d, want 1 (external dependent still counts)", got["a.go"])
	}
	if _, ok := got["external/lib"]; ok {
		t.Errorf("external/lib should not appear in result; got %v", got)
	}
}
