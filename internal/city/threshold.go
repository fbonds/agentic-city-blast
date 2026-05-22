package city

import "github.com/mferree/agent-city/internal/model"

// MarkCoverageThresholds scans all buildings in state and sets CoverageWarn=true
// for any building whose known coverage ratio falls strictly below its applicable
// threshold. Buildings with unknown coverage (Coverage < 0) are left unwarn-ed.
// Returns a new CityState with updated buildings; does not modify state in place.
func MarkCoverageThresholds(state model.CityState) model.CityState {
	s := state.Settings
	updated := make([]model.Building, len(state.Buildings))
	for i, b := range state.Buildings {
		if b.Coverage >= 0 {
			b.CoverageWarn = b.Coverage < s.ThresholdFor(b.DistrictID)
		} else {
			b.CoverageWarn = false
		}
		updated[i] = b
	}
	state.Buildings = updated
	return state
}

// DetectThresholdCrossings returns the IDs of buildings that newly dropped below
// threshold — buildings where CoverageWarn was false in before but is true in after.
// Used to emit activity events on threshold violations.
func DetectThresholdCrossings(before, after model.CityState) []string {
	prevWarn := make(map[string]bool, len(before.Buildings))
	for _, b := range before.Buildings {
		prevWarn[b.ID] = b.CoverageWarn
	}

	var crossed []string
	for _, b := range after.Buildings {
		if b.CoverageWarn && !prevWarn[b.ID] {
			crossed = append(crossed, b.ID)
		}
	}
	return crossed
}
