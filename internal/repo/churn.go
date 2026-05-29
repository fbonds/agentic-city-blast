package repo

import (
	"fmt"
	"math"
	"os/exec"
	"strings"
)

// ChurnConfig controls the git log window for churn computation.
type ChurnConfig struct {
	// SinceDays is the number of days of git history to consider.
	// Default (0) is treated as 90.
	SinceDays int
}

func (c ChurnConfig) sinceDays() int {
	if c.SinceDays <= 0 {
		return 90
	}
	return c.SinceDays
}

// ChurnCounts maps repo-relative file paths to the number of commits
// that touched each file within the configured recency window.
type ChurnCounts map[string]int

// ComputeChurn runs git log in repoDir and counts how many commits touched
// each file within the configured recency window. Returns raw counts.
// Errors are best-effort: if git is unavailable, returns nil and the error.
func ComputeChurn(repoDir string, cfg ChurnConfig) (ChurnCounts, error) {
	days := cfg.sinceDays()
	cmd := exec.Command("git", "log",
		fmt.Sprintf("--since=%ddays", days),
		"--name-only",
		"--format=",
		"--diff-filter=AMRC",
	)
	cmd.Dir = repoDir

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log: %w", err)
	}

	return parseChurnOutput(string(out)), nil
}

// parseChurnOutput parses the output of git log --name-only --format=""
// and returns per-file commit counts. Commits are separated by blank lines;
// each non-blank line within a block is a file path.
func parseChurnOutput(output string) ChurnCounts {
	counts := make(ChurnCounts)
	if output == "" {
		return counts
	}

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		counts[line]++
	}
	return counts
}

// NormalizeChurn converts raw commit counts to [0, 1] using log normalization:
//
//	normalized = log(1 + count) / log(1 + max)
//
// Returns 0.0 for files with zero commits. The file with the most commits
// normalizes to exactly 1.0. Returns an empty map for nil input.
func NormalizeChurn(raw ChurnCounts) map[string]float64 {
	result := make(map[string]float64, len(raw))
	if len(raw) == 0 {
		return result
	}

	maxCount := 0
	for _, c := range raw {
		if c > maxCount {
			maxCount = c
		}
	}

	if maxCount == 0 {
		for k := range raw {
			result[k] = 0
		}
		return result
	}

	logMax := math.Log(1 + float64(maxCount))
	for k, c := range raw {
		result[k] = math.Log(1+float64(c)) / logMax
	}
	return result
}
