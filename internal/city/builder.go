// Package city assembles a model.CityState from a scanned repository.
// It wires together repo.ScanRepo, layout.Layout, and deps.BuildGraph,
// and exposes helpers for incremental state merges driven by repo.Watcher.
package city

import (
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mferree/agent-city/internal/deps"
	"github.com/mferree/agent-city/internal/layout"
	"github.com/mferree/agent-city/internal/model"
	"github.com/mferree/agent-city/internal/repo"
)

// minBuildingHeight is the floor for GZ on incremental updates where a true
// blast-radius-derived height is not yet available. Matches the lower bound
// of layout.heightFromBlastRadius — kept here as a literal to avoid a layout
// package import from the merge path.
const minBuildingHeight = 3.0

// ApplyMetrics overlays coverage ratios and test status from src onto the buildings
// in state. Only buildings with entries in src are updated; all others retain their
// existing Coverage and Status values. Stats and Timestamp are recomputed.
func ApplyMetrics(state model.CityState, src repo.MetricsSource) model.CityState {
	updated := make([]model.Building, len(state.Buildings))
	for i, b := range state.Buildings {
		if cov, ok := src.Coverage[b.ID]; ok {
			b.Coverage = cov
		}
		if st := src.StatusFor(b.ID); st != "unknown" {
			b.Status = st
		}
		updated[i] = b
	}
	state.Buildings = updated
	state.Stats = computeStats(updated)
	state.Timestamp = time.Now().UnixMilli()
	return state
}

// BuildConfig holds the parameters for a full city state build.
type BuildConfig struct {
	ScanCfg   repo.ScanConfig
	LayoutCfg layout.Config
	DepsCfg   deps.Config
}

// GatherRepoInfo extracts git metadata from the repository at repoPath.
// Uses the git CLI directly so it works in both normal repos and worktrees.
// Falls back to safe defaults on any error.
func GatherRepoInfo(repoPath string) (model.RepoInfo, error) {
	name := filepath.Base(repoPath)
	info := model.RepoInfo{Name: name, CIStatus: "unknown"}

	branch, err := gitOutput(repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return info, err // not a git repo
	}
	info.Branch = branch

	hash, err := gitOutput(repoPath, "rev-parse", "--short=7", "HEAD")
	if err == nil {
		info.HeadCommit = hash
	}

	return info, nil
}

// gitOutput runs a git command in dir and returns stdout trimmed.
func gitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// AssembleState builds a CityState from pre-scanned buildings plus git metadata.
// It runs the dependency analyzer to compute the import graph, derives each
// building's blast radius from it, then runs the layout engine (which uses
// blast radius to set building heights), and finally computes summary stats.
// readContent is called per-file by the dependency analyzer (best-effort; errors skipped).
func AssembleState(
	buildings []model.Building,
	repoInfo model.RepoInfo,
	readContent func(id string) ([]byte, error),
	layoutCfg layout.Config,
	depsCfg deps.Config,
) model.CityState {
	// Build the import graph before layout so blast radius can be populated
	// on each building and consumed by the packer as it assigns heights.
	roads := deps.BuildGraph(buildings, readContent, depsCfg)

	ids := make([]string, len(buildings))
	for i, b := range buildings {
		ids[i] = b.ID
	}
	blastRadius := deps.Compute(ids, roads)

	enriched := make([]model.Building, len(buildings))
	for i, b := range buildings {
		b.BlastRadius = blastRadius[b.ID]
		enriched[i] = b
	}

	laid := layout.Layout(enriched, layoutCfg)

	return model.CityState{
		RepoInfo:  repoInfo,
		Districts: laid.Districts,
		Buildings: laid.Buildings,
		Roads:     roads,
		Stats:     computeStats(laid.Buildings),
		Timestamp: time.Now().UnixMilli(),
	}
}

// BuildState performs a full scan → layout → deps pipeline and returns a CityState.
// Agents and Activities are left nil; callers should preserve them across refreshes.
func BuildState(repoPath string, cfg BuildConfig) (model.CityState, error) {
	buildings, err := repo.ScanRepo(repoPath, cfg.ScanCfg)
	if err != nil {
		return model.CityState{}, err
	}

	repoInfo, _ := GatherRepoInfo(repoPath) // best-effort; zero value is fine

	readContent := deps.DirReader(repoPath)
	return AssembleState(buildings, repoInfo, readContent, cfg.LayoutCfg, cfg.DepsCfg), nil
}

// MergeBuildings applies incremental building updates to the current state.
// A Building with LOC == 0 is a tombstone — it signals the file was deleted.
// Layout position fields (GX/GY/GW/GH/GZ) are preserved from the existing entry
// when the file already has a non-zero footprint, so the city map stays stable.
// Stats and Timestamp are recomputed on every call.
//
// Brand-new buildings (no prior entry) have no blast radius yet — that is
// only computed during a full rescan — so their GZ is floored to the minimum
// visible height to avoid rendering them flat (GZ=0) until the next rescan
// promotes them to their true height.
func MergeBuildings(current model.CityState, updates []model.Building) model.CityState {
	byID := make(map[string]model.Building, len(current.Buildings))
	for _, b := range current.Buildings {
		byID[b.ID] = b
	}

	for _, u := range updates {
		if u.LOC == 0 {
			// Tombstone: file was deleted or renamed away.
			delete(byID, u.ID)
			continue
		}
		if existing, ok := byID[u.ID]; ok && existing.GW > 0 {
			// Preserve layout position so the building doesn't jump.
			u.GX = existing.GX
			u.GY = existing.GY
			u.GW = existing.GW
			u.GH = existing.GH
			u.GZ = existing.GZ
		}
		// Floor GZ at the minimum visible height. Applies to brand-new
		// buildings (no prior layout) and to any update path that did not
		// populate GZ — without this, new files would render flat until the
		// next full rescan computes their blast radius.
		if u.GZ < minBuildingHeight {
			u.GZ = minBuildingHeight
		}
		byID[u.ID] = u
	}

	next := make([]model.Building, 0, len(byID))
	for _, b := range byID {
		next = append(next, b)
	}
	sort.Slice(next, func(i, j int) bool { return next[i].ID < next[j].ID })

	current.Buildings = next
	current.Stats = computeStats(next)
	current.Timestamp = time.Now().UnixMilli()
	return current
}

// computeStats derives RepoStats from a slice of buildings.
// Coverage is the mean of all buildings with known coverage (>= 0).
// TestsPassing counts buildings with Status == "ok".
// TestsTotal counts buildings with Status != "unknown".
func computeStats(buildings []model.Building) model.RepoStats {
	var totalLOC int
	var coverageSum float64
	coveredCount := 0
	testsPassing := 0
	testsTotal := 0

	for _, b := range buildings {
		totalLOC += b.LOC
		if b.Coverage >= 0 {
			coverageSum += b.Coverage
			coveredCount++
		}
		if b.Status != "unknown" {
			testsTotal++
			if b.Status == "ok" {
				testsPassing++
			}
		}
	}

	var avgCoverage float64
	if coveredCount > 0 {
		avgCoverage = coverageSum / float64(coveredCount)
	}

	return model.RepoStats{
		FileCount:    len(buildings),
		TotalLOC:     totalLOC,
		Coverage:     avgCoverage,
		TestsPassing: testsPassing,
		TestsTotal:   testsTotal,
	}
}
