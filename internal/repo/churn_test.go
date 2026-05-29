package repo

import (
	"math"
	"testing"
)

func TestParseChurnOutput(t *testing.T) {
	cases := []struct {
		name   string
		output string
		want   ChurnCounts
	}{
		{
			"empty output",
			"",
			ChurnCounts{},
		},
		{
			"single commit single file",
			"src/main.go\n",
			ChurnCounts{"src/main.go": 1},
		},
		{
			"single commit multiple files",
			"src/main.go\nsrc/util.go\n",
			ChurnCounts{"src/main.go": 1, "src/util.go": 1},
		},
		{
			"two commits same file",
			"src/main.go\n\nsrc/main.go\n",
			ChurnCounts{"src/main.go": 2},
		},
		{
			"two commits different files",
			"src/main.go\n\nsrc/util.go\n",
			ChurnCounts{"src/main.go": 1, "src/util.go": 1},
		},
		{
			"mixed commits with blank line separators",
			"a.go\nb.go\n\nb.go\nc.go\n\na.go\n",
			ChurnCounts{"a.go": 2, "b.go": 2, "c.go": 1},
		},
		{
			"trailing blank lines ignored",
			"a.go\n\n\n",
			ChurnCounts{"a.go": 1},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseChurnOutput(tc.output)
			if len(got) != len(tc.want) {
				t.Fatalf("got %d entries, want %d: %v", len(got), len(tc.want), got)
			}
			for k, wantV := range tc.want {
				if got[k] != wantV {
					t.Errorf("got[%q] = %d, want %d", k, got[k], wantV)
				}
			}
		})
	}
}

func TestNormalizeChurn(t *testing.T) {
	const eps = 1e-6

	cases := []struct {
		name string
		raw  ChurnCounts
		want map[string]float64
	}{
		{
			"nil input",
			nil,
			map[string]float64{},
		},
		{
			"all zero",
			ChurnCounts{"a.go": 0, "b.go": 0},
			map[string]float64{"a.go": 0, "b.go": 0},
		},
		{
			"single file is always 1.0",
			ChurnCounts{"a.go": 5},
			map[string]float64{"a.go": 1.0},
		},
		{
			"max file is 1.0",
			ChurnCounts{"a.go": 10, "b.go": 100},
			map[string]float64{"a.go": -1, "b.go": 1.0}, // -1 = check separately
		},
		{
			"log distribution not linear",
			ChurnCounts{"low.go": 1, "high.go": 100},
			map[string]float64{"low.go": -1, "high.go": 1.0}, // -1 = check separately
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeChurn(tc.raw)
			if len(got) != len(tc.want) {
				t.Fatalf("got %d entries, want %d", len(got), len(tc.want))
			}
			for k, wantV := range tc.want {
				if wantV == -1 {
					// Just check it's in (0, 1]
					if got[k] <= 0 || got[k] > 1.0+eps {
						t.Errorf("got[%q] = %f, want in (0, 1]", k, got[k])
					}
					continue
				}
				if math.Abs(got[k]-wantV) > eps {
					t.Errorf("got[%q] = %f, want %f", k, got[k], wantV)
				}
			}
		})
	}

	// Verify log distribution: low.go (1 commit) should be much less than
	// half of high.go (100 commits). Linear would give 0.01; log gives ~0.15.
	t.Run("log is sublinear", func(t *testing.T) {
		got := NormalizeChurn(ChurnCounts{"low.go": 1, "high.go": 100})
		if got["low.go"] >= 0.5 {
			t.Errorf("low.go = %f, expected < 0.5 (sublinear)", got["low.go"])
		}
		if got["low.go"] < 0.05 {
			t.Errorf("low.go = %f, expected > 0.05 (not too compressed)", got["low.go"])
		}
	})
}

func TestChurnConfig_DefaultSinceDays(t *testing.T) {
	cfg := ChurnConfig{}
	if cfg.sinceDays() != 90 {
		t.Errorf("default sinceDays = %d, want 90", cfg.sinceDays())
	}
	cfg.SinceDays = 30
	if cfg.sinceDays() != 30 {
		t.Errorf("explicit sinceDays = %d, want 30", cfg.sinceDays())
	}
}
