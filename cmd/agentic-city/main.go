package main

import (
	"context"
	"flag"
	"io/fs"
	"log/slog"
	"math"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/mferree/agent-city/internal/agents"
	"github.com/mferree/agent-city/internal/api"
	"github.com/mferree/agent-city/internal/city"
	"github.com/mferree/agent-city/internal/deps"
	"github.com/mferree/agent-city/internal/hub"
	"github.com/mferree/agent-city/internal/model"
	"github.com/mferree/agent-city/internal/repo"
	agentcityweb "github.com/mferree/agent-city/web"
)

func main() {
	demo := flag.Bool("demo", false, "Run in demo mode with synthetic city data")
	addr := flag.String("addr", ":8080", "HTTP listen address")
	repoPath := flag.String("repo", ".", "Path to the git repository to visualise")
	coveragePath := flag.String("coverage", "", "Path to coverage file (coverage.out, lcov.info, coverage.json); auto-detected if empty")
	testResultsPath := flag.String("test-results", "", "Path to test result file (JUnit XML or Go test JSON); auto-detected if empty")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	mux := http.NewServeMux()

	var cityState *hub.State
	var buildCfg city.BuildConfig

	if *demo {
		s := generateDemoState()
		s.Settings = model.DefaultSettings()
		s = city.MarkCoverageThresholds(s)
		cityState = hub.NewState(s)
		slog.Info("demo mode", "districts", len(s.Districts), "buildings", len(s.Buildings), "agents", len(s.Agents))
	} else {
		buildCfg = city.BuildConfig{
			DepsCfg: deps.Config{ModuleName: readModuleName(*repoPath)},
		}

		initial, err := city.BuildState(*repoPath, buildCfg)
		if err != nil {
			slog.Warn("live mode: initial scan failed — serving empty state", "err", err)
			initial = model.CityState{Timestamp: time.Now().UnixMilli()}
		} else {
			slog.Info("live mode: scanned", "buildings", len(initial.Buildings), "districts", len(initial.Districts))
		}
		initial.Settings = model.DefaultSettings()
		initial = city.MarkCoverageThresholds(initial)

		cityState = hub.NewState(initial)
	}

	h := hub.New(cityState)
	go h.Run(ctx)

	covHistory := model.NewCoverageHistory(model.HistoryCap)

	if *demo {
		// Demo mode has no real agent monitor, so mark the hub ready immediately.
		h.SetReady()
		go runDemoTicker(ctx, cityState, h)
	}

	if !*demo {
		tracker := agents.StartMonitor(ctx, *repoPath, cityState, h)
		go runWatcher(ctx, *repoPath, buildCfg, cityState, h, tracker)

		if mw := buildMetricsWatcher(*repoPath, *coveragePath, *testResultsPath, readModuleName(*repoPath)); mw != nil {
			if err := mw.Start(); err != nil {
				slog.Error("metrics watcher: start failed", "err", err)
				mw.Stop()
			} else {
				go runMetricsWatcher(ctx, mw, cityState, h, covHistory)
			}
		}
	}

	// Dev mode: --demo flag or --repo not explicitly set (default ".").
	repoExplicitlySet := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "repo" {
			repoExplicitlySet = true
		}
	})
	devMode := *demo || !repoExplicitlySet

	apiServer := api.New(cityState).
		WithDevMode(devMode).
		WithWSHandler(h.ServeWS).
		WithStateUpdater(cityState, cityState, h).
		WithHistory(covHistory)
	apiServer.Register(mux)

	distFS, err := fs.Sub(agentcityweb.Dist, "dist")
	if err != nil {
		slog.Error("static embed unavailable", "err", err)
	} else {
		mux.Handle("/", http.FileServer(http.FS(distFS)))
	}

	srv := &http.Server{
		Addr:    *addr,
		Handler: mux,
	}

	go func() {
		slog.Info("agent-city listening", "addr", *addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "err", err)
	}
}

// ── Demo mode ─────────────────────────────────────────────────────────────────
//
// The demo creates a fictional-but-realistic city with scripted agent
// choreography and timed narration that builds toward a cow abduction on
// a high blast-radius file. The guided tour runs ~35 seconds before
// settling into a normal animation loop.

const cowTargetID = "pkg/core/config.go"

func generateDemoState() model.CityState {
	districts := makeDemoDistricts()
	buildings := makeDemoBuildings(districts)

	var totalLOC int
	var coveredCount int
	var coverageSum float64
	for _, b := range buildings {
		totalLOC += b.LOC
		if b.Coverage >= 0 {
			coverageSum += b.Coverage
			coveredCount++
		}
	}
	var avgCoverage float64
	if coveredCount > 0 {
		avgCoverage = coverageSum / float64(coveredCount)
	}

	return model.CityState{
		RepoInfo: model.RepoInfo{
			Name:       "acme-platform",
			Branch:     "main",
			HeadCommit: "e3a91b7",
			CIStatus:   "passing",
		},
		Districts:  districts,
		Buildings:  buildings,
		Roads:      makeDemoRoads(),
		Agents:     makeDemoAgents(),
		Activities: []model.ActivityEvent{}, // narration injected by ticker
		Stats: model.RepoStats{
			FileCount:    len(buildings),
			TotalLOC:     totalLOC,
			Coverage:     avgCoverage,
			OpenPRs:      3,
			BugCount:     1,
			TestsPassing: 89,
			TestsTotal:   94,
		},
		Timestamp: time.Now().UnixMilli(),
	}
}

func makeDemoDistricts() []model.District {
	return []model.District{
		{ID: "pkg/api", Label: "API/", ParentID: "", GX: 0, GY: 0, GW: 12, GH: 8},
		{ID: "pkg/core", Label: "CORE/", ParentID: "", GX: 12, GY: 0, GW: 12, GH: 10},
		{ID: "pkg/auth", Label: "AUTH/", ParentID: "", GX: 24, GY: 0, GW: 10, GH: 8},
		{ID: "pkg/ui", Label: "UI/", ParentID: "", GX: 0, GY: 10, GW: 16, GH: 10},
		{ID: "internal", Label: "INTERNAL/", ParentID: "", GX: 16, GY: 10, GW: 18, GH: 10},
	}
}

// demoBuilding defines a building with all encoding-relevant fields.
type demoBuilding struct {
	name       string
	districtID string
	lang       string
	loc        int
	br         int     // blast radius
	churn      float64 // normalized [0,1]
	coverage   float64 // -1=unknown, 0-1
	status     string
}

func makeDemoBuildings(districts []model.District) []model.Building {
	specs := []demoBuilding{
		// pkg/api — HTTP handlers, leaf-ish
		{"router.go", "pkg/api", "go", 180, 3, 0.20, 0.85, "ok"},
		{"handlers.go", "pkg/api", "go", 220, 2, 0.40, 0.72, "ok"},
		{"middleware.go", "pkg/api", "go", 95, 1, 0.10, 0.90, "ok"},
		{"errors.go", "pkg/api", "go", 60, 5, 0.15, 0.80, "ok"},

		// pkg/core — domain logic, contains the cow target
		{"engine.go", "pkg/core", "go", 310, 8, 0.50, 0.65, "ok"},
		{"config.go", "pkg/core", "go", 145, 25, 0.70, 0.55, "warn"}, // COW TARGET
		{"types.go", "pkg/core", "go", 85, 15, 0.30, 0.78, "ok"},
		{"validate.go", "pkg/core", "go", 120, 4, 0.20, 0.88, "ok"},
		{"transform.go", "pkg/core", "go", 175, 6, 0.45, 0.70, "ok"},

		// pkg/auth
		{"jwt.go", "pkg/auth", "go", 130, 3, 0.10, 0.92, "ok"},
		{"oauth.go", "pkg/auth", "go", 195, 2, 0.25, 0.68, "ok"},
		{"session.go", "pkg/auth", "go", 160, 7, 0.35, 0.75, "ok"},
		{"rbac.go", "pkg/auth", "go", 110, 4, 0.15, 0.82, "ok"},

		// pkg/ui — frontend
		{"App.tsx", "pkg/ui", "tsx", 380, 5, 0.60, 0.50, "ok"},
		{"Dashboard.tsx", "pkg/ui", "tsx", 290, 3, 0.55, 0.45, "ok"},
		{"Settings.tsx", "pkg/ui", "tsx", 175, 1, 0.80, 0.60, "ok"},
		{"Table.tsx", "pkg/ui", "tsx", 210, 2, 0.30, 0.72, "ok"},
		{"Chart.tsx", "pkg/ui", "tsx", 245, 1, 0.40, 0.58, "ok"},
		{"store.ts", "pkg/ui", "ts", 95, 10, 0.50, 0.65, "ok"},
		{"api.ts", "pkg/ui", "ts", 120, 3, 0.35, 0.80, "ok"},

		// internal — infra
		{"db.go", "internal", "go", 280, 12, 0.20, 0.85, "ok"},
		{"logger.go", "internal", "go", 75, 18, 0.05, 0.95, "ok"},
		{"errors.go", "internal", "go", 90, 14, 0.10, 0.90, "ok"},
		{"testutil.go", "internal", "go", 155, 0, 0.15, -1, "unknown"},
		{"migrate.go", "internal", "go", 190, 2, 0.60, 0.70, "warn"},
	}

	districtByID := map[string]model.District{}
	districtCursor := map[string][2]float64{}
	for _, d := range districts {
		districtByID[d.ID] = d
		districtCursor[d.ID] = [2]float64{d.GX + 1, d.GY + 1}
	}

	buildings := make([]model.Building, 0, len(specs))
	for _, s := range specs {
		w := clamp(4.0+math.Sqrt(float64(s.br)), 4, 12)
		h := w * 0.8
		z := clamp(3.0+3.0*math.Log2(1.0+float64(s.br)), 3, 30)

		cursor := districtCursor[s.districtID]
		gx := cursor[0]
		gy := cursor[1]

		districtCursor[s.districtID] = [2]float64{gx + w + 1, gy}
		if d := districtByID[s.districtID]; gx+w+1 > d.GX+d.GW-1 {
			districtCursor[s.districtID] = [2]float64{d.GX + 1, gy + h + 1}
		}

		buildings = append(buildings, model.Building{
			ID:          s.districtID + "/" + s.name,
			DistrictID:  s.districtID,
			Label:       s.name,
			Language:    s.lang,
			LOC:         s.loc,
			BlastRadius: s.br,
			Churn:       s.churn,
			Coverage:    s.coverage,
			Status:      s.status,
			Exports:     s.br, // approximate
			GX:          gx,
			GY:          gy,
			GW:          w,
			GH:          h,
			GZ:          z,
		})
	}

	return buildings
}

func makeDemoAgents() []model.Agent {
	return []model.Agent{
		{
			ID: "claude:demo-001", Color: "blue", Mode: "work",
			Task: "Refactoring error handling", Progress: 45,
			TargetID: "pkg/api/handlers.go", LocationConfidence: "exact",
		},
		{
			ID: "claude:demo-002", Color: "green", Mode: "work",
			Task: "Writing migration tests", Progress: 60,
			TargetID: "internal/migrate.go", LocationConfidence: "inferred",
		},
		{
			ID: "claude:demo-004", Color: "blue", Mode: "fly",
			Task: "Reviewing auth flow", Progress: 20,
			FromID: "pkg/auth/jwt.go", ToID: "pkg/auth/session.go",
			FlyProgress: 0.0,
		},
	}
}

func makeDemoRoads() []model.Road {
	// Intentional dependency fan toward high-BR nodes.
	return []model.Road{
		// Everything depends on config.go (BR=25)
		{FromID: "pkg/api/router.go", ToID: cowTargetID, Weight: 3, Confidence: "exact"},
		{FromID: "pkg/api/handlers.go", ToID: cowTargetID, Weight: 2, Confidence: "exact"},
		{FromID: "pkg/core/engine.go", ToID: cowTargetID, Weight: 5, Confidence: "exact"},
		{FromID: "pkg/auth/session.go", ToID: cowTargetID, Weight: 2, Confidence: "exact"},
		{FromID: "pkg/ui/store.ts", ToID: cowTargetID, Weight: 1, Confidence: "inferred"},
		{FromID: "internal/db.go", ToID: cowTargetID, Weight: 3, Confidence: "exact"},
		{FromID: "pkg/core/transform.go", ToID: cowTargetID, Weight: 2, Confidence: "exact"},
		{FromID: "pkg/auth/rbac.go", ToID: cowTargetID, Weight: 1, Confidence: "inferred"},
		{FromID: "pkg/core/validate.go", ToID: cowTargetID, Weight: 2, Confidence: "exact"},

		// logger.go (BR=18) — many importers
		{FromID: "pkg/api/middleware.go", ToID: "internal/logger.go", Weight: 1, Confidence: "exact"},
		{FromID: "pkg/core/engine.go", ToID: "internal/logger.go", Weight: 2, Confidence: "exact"},
		{FromID: "pkg/auth/oauth.go", ToID: "internal/logger.go", Weight: 1, Confidence: "exact"},
		{FromID: "internal/db.go", ToID: "internal/logger.go", Weight: 3, Confidence: "exact"},
		{FromID: "internal/migrate.go", ToID: "internal/logger.go", Weight: 1, Confidence: "exact"},
		{FromID: "pkg/api/handlers.go", ToID: "internal/logger.go", Weight: 1, Confidence: "inferred"},

		// types.go (BR=15)
		{FromID: "pkg/core/engine.go", ToID: "pkg/core/types.go", Weight: 4, Confidence: "exact"},
		{FromID: "pkg/api/handlers.go", ToID: "pkg/core/types.go", Weight: 2, Confidence: "exact"},
		{FromID: "pkg/core/validate.go", ToID: "pkg/core/types.go", Weight: 3, Confidence: "exact"},
		{FromID: "pkg/ui/store.ts", ToID: "pkg/core/types.go", Weight: 1, Confidence: "inferred"},

		// errors.go (BR=14)
		{FromID: "pkg/api/handlers.go", ToID: "internal/errors.go", Weight: 2, Confidence: "exact"},
		{FromID: "pkg/core/engine.go", ToID: "internal/errors.go", Weight: 1, Confidence: "exact"},
		{FromID: "pkg/auth/session.go", ToID: "internal/errors.go", Weight: 1, Confidence: "exact"},

		// Misc realistic edges
		{FromID: "pkg/api/router.go", ToID: "pkg/api/handlers.go", Weight: 3, Confidence: "exact"},
		{FromID: "pkg/ui/App.tsx", ToID: "pkg/ui/store.ts", Weight: 2, Confidence: "exact"},
		{FromID: "pkg/ui/Dashboard.tsx", ToID: "pkg/ui/api.ts", Weight: 1, Confidence: "inferred"},
	}
}

// ── Demo narration and choreography ──────────────────────────────────────────

type demoEvent struct {
	tick   int
	action func(s *model.CityState)
}

func guideMsg(t time.Time, msg string) model.ActivityEvent {
	return model.ActivityEvent{
		Timestamp: t.Format(time.RFC3339),
		Who:       "GUIDE",
		Message:   msg,
		Color:     "#b58900", // solarized yellow
		Severity:  "info",
	}
}

func buildDemoTimeline() []demoEvent {
	return []demoEvent{
		// ── Introduction ──
		{tick: 1, action: func(s *model.CityState) {
			s.Activities = model.AppendActivity(s.Activities, guideMsg(time.Now(),
				"Welcome to Agent City. Each building is a file in the codebase."))
		}},
		{tick: 30, action: func(s *model.CityState) {
			s.Activities = model.AppendActivity(s.Activities, model.ActivityEvent{
				Timestamp: time.Now().Format(time.RFC3339),
				Who: "claude:demo-001", Message: "Refactoring error responses in handlers.go",
				Color: "#4a7a9c", Severity: "info",
			})
		}},
		{tick: 50, action: func(s *model.CityState) {
			s.Activities = model.AppendActivity(s.Activities, guideMsg(time.Now(),
				"Building height encodes blast radius \u2014 how many files break if this one changes."))
		}},
		{tick: 70, action: func(s *model.CityState) {
			s.Activities = model.AppendActivity(s.Activities, model.ActivityEvent{
				Timestamp: time.Now().Format(time.RFC3339),
				Who: "claude:demo-002", Message: "Writing migration rollback tests",
				Color: "#6a8a4a", Severity: "info",
			})
		}},
		{tick: 100, action: func(s *model.CityState) {
			s.Activities = model.AppendActivity(s.Activities, guideMsg(time.Now(),
				"Color encodes churn \u2014 cyan is stable, red is actively changing."))
		}},
		{tick: 130, action: func(s *model.CityState) {
			s.Activities = model.AppendActivity(s.Activities, guideMsg(time.Now(),
				"The tallest buildings are the most structurally dangerous to edit."))
		}},

		// ── Agent spawns and flies toward cow target ──
		{tick: 150, action: func(s *model.CityState) {
			s.Agents = append(s.Agents, model.Agent{
				ID: "claude:demo-005", Color: "orange", Mode: "fly",
				Task: "Updating feature flags in config.go",
				FromID: "pkg/api/router.go", ToID: cowTargetID,
				FlyProgress: 0.0,
			})
			s.Activities = model.AppendActivity(s.Activities, model.ActivityEvent{
				Timestamp: time.Now().Format(time.RFC3339),
				Who: "claude:demo-005", Message: "Dispatched to config.go \u2014 updating feature flags",
				Color: "#cb4b16", Severity: "warn",
			})
		}},
		{tick: 180, action: func(s *model.CityState) {
			s.Activities = model.AppendActivity(s.Activities, guideMsg(time.Now(),
				"An agent is approaching config.go \u2014 blast radius 25..."))
		}},

		// ── Agent lands on cow target (fly → work). Cow triggers. ──
		{tick: 220, action: func(s *model.CityState) {
			for i := range s.Agents {
				if s.Agents[i].ID == "claude:demo-005" {
					s.Agents[i].Mode = "work"
					s.Agents[i].TargetID = cowTargetID
					s.Agents[i].FromID = ""
					s.Agents[i].ToID = ""
					s.Agents[i].FlyProgress = 0
					s.Agents[i].Progress = 5
					s.Agents[i].LocationConfidence = "exact"
					break
				}
			}
		}},
		{tick: 225, action: func(s *model.CityState) {
			s.Activities = model.AppendActivity(s.Activities, guideMsg(time.Now(),
				"\U0001f404 An agent is editing a high-risk file! 25 downstream files at risk."))
		}},

		// ── Post-cow: progress and wrap-up ──
		{tick: 260, action: func(s *model.CityState) {
			for i := range s.Agents {
				if s.Agents[i].ID == "claude:demo-005" {
					s.Agents[i].Progress = 35
					break
				}
			}
			s.Activities = model.AppendActivity(s.Activities, model.ActivityEvent{
				Timestamp: time.Now().Format(time.RFC3339),
				Who: "claude:demo-005", Message: "Modifying config.go \u2014 12 downstream files affected",
				Color: "#cb4b16", Severity: "warn",
			})
		}},
		{tick: 300, action: func(s *model.CityState) {
			s.Activities = model.AppendActivity(s.Activities, guideMsg(time.Now(),
				"Select any building to see its blast radius, churn, and dependency arcs."))
		}},

		// ── Agent departs, resumes flight loop ──
		{tick: 350, action: func(s *model.CityState) {
			for i := range s.Agents {
				if s.Agents[i].ID == "claude:demo-005" {
					s.Agents[i].Mode = "fly"
					s.Agents[i].TargetID = ""
					s.Agents[i].FromID = cowTargetID
					s.Agents[i].ToID = "pkg/core/engine.go"
					s.Agents[i].FlyProgress = 0.0
					break
				}
			}
		}},
	}
}

// runDemoTicker drives the scripted demo timeline and continuous flight
// animation. Events fire at predetermined tick counts (100ms per tick).
// After all scripted events have fired (~35s), flying agents continue
// looping on their bezier arcs indefinitely.
func runDemoTicker(ctx context.Context, state *hub.State, h *hub.Hub) {
	const tickRate = 100 * time.Millisecond
	flySpeeds := [4]float64{0.014, 0.010, 0.017, 0.011}

	ticker := time.NewTicker(tickRate)
	defer ticker.Stop()

	events := buildDemoTimeline()
	eventIdx := 0
	tickCount := 0

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tickCount++
			changed := false

			state.Update(func(curr model.CityState) model.CityState {
				// Fire all scheduled events at or before this tick.
				for eventIdx < len(events) && events[eventIdx].tick <= tickCount {
					events[eventIdx].action(&curr)
					eventIdx++
					changed = true
				}

				// Advance all flying agents.
				fi := 0
				for i := range curr.Agents {
					a := &curr.Agents[i]
					if a.Mode == "fly" && a.FromID != "" && a.ToID != "" {
						a.FlyProgress += flySpeeds[fi%len(flySpeeds)]
						fi++
						if a.FlyProgress >= 1.0 {
							a.FlyProgress = 0.0
							a.FromID, a.ToID = a.ToID, a.FromID
						}
						changed = true
					}
				}

				return curr
			})

			if changed {
				h.Notify()
			}
		}
	}
}

func clamp(v, lo, hi float64) float64 {
	return max(lo, min(hi, v))
}

// runWatcher watches repoPath for file changes and applies them to cityState,
// then calls h.Notify() so the hub broadcasts a patch to all connected clients.
//
// Structural changes (creates/deletes/renames) trigger a full rescan to keep
// the layout consistent. Content-only changes are merged incrementally.
// Agents and Activities are always preserved across refreshes.
//
// When tracker is non-nil, each changed file is fed as a FileEvent so the
// tracker can correlate file activity with agent working directories.
func runWatcher(ctx context.Context, repoPath string, cfg city.BuildConfig, state *hub.State, h *hub.Hub, tracker *agents.Tracker) {
	w, err := repo.NewWatcher(repoPath, cfg.ScanCfg)
	if err != nil {
		slog.Error("watcher: init failed", "err", err)
		return
	}
	if err := w.Start(); err != nil {
		slog.Error("watcher: start failed", "err", err)
		return
	}
	defer w.Stop()

	absRepo, _ := filepath.Abs(repoPath)

	for {
		select {
		case <-ctx.Done():
			return

		case update, ok := <-w.Updates:
			if !ok {
				return
			}

			// Feed file events to the tracker for agent location inference.
			if tracker != nil {
				now := time.Now()
				for _, b := range update.Buildings {
					if b.ID == "" {
						continue
					}
					tracker.ObserveFileEvent(agents.FileEvent{
						AbsPath: filepath.Join(absRepo, filepath.FromSlash(b.ID)),
						At:      now,
					})
				}
			}

			if update.HasStructural {
				// A file was created, deleted, or renamed — full rescan to
				// recompute layout and dependency graph.
				next, err := city.BuildState(repoPath, cfg)
				if err != nil {
					slog.Error("watcher: full rescan failed", "err", err)
					continue
				}
				state.Update(func(curr model.CityState) model.CityState {
					next.Agents = curr.Agents
					next.Activities = curr.Activities
					return next
				})
				slog.Info("watcher: full rescan", "buildings", len(next.Buildings))
			} else {
				// Content-only changes — merge incrementally.
				state.Update(func(curr model.CityState) model.CityState {
					return city.MergeBuildings(curr, update.Buildings)
				})
			}

			if h != nil {
				h.Notify()
			}
		}
	}
}

// buildMetricsWatcher constructs a MetricsWatcher using explicit paths when
// provided, falling back to auto-detection of well-known filenames at repoPath.
// Returns nil when no coverage or test-result files are found.
func buildMetricsWatcher(repoPath, coveragePath, testResultsPath, modulePath string) *repo.MetricsWatcher {
	coverageFiles, testResultFiles := autoDetectMetricsFiles(repoPath)

	if coveragePath != "" {
		coverageFiles = []string{coveragePath}
	}
	if testResultsPath != "" {
		testResultFiles = []string{testResultsPath}
	}

	if len(coverageFiles) == 0 && len(testResultFiles) == 0 {
		return nil
	}

	slog.Info("metrics watcher", "coverage", coverageFiles, "test-results", testResultFiles)

	mw, err := repo.NewMetricsWatcher(repo.MetricsConfig{
		CoverageFiles:   coverageFiles,
		TestResultFiles: testResultFiles,
		RepoRoot:        repoPath,
		ModulePath:      modulePath,
	})
	if err != nil {
		slog.Error("metrics watcher: init failed", "err", err)
		return nil
	}
	return mw
}

// autoDetectMetricsFiles scans repoPath for well-known coverage and test-result
// filenames and returns their absolute paths grouped by type.
func autoDetectMetricsFiles(repoPath string) (coverageFiles, testResultFiles []string) {
	for _, name := range []string{"coverage.out", "lcov.info", "coverage.json"} {
		p := filepath.Join(repoPath, name)
		if _, err := os.Stat(p); err == nil {
			coverageFiles = append(coverageFiles, p)
		}
	}
	for _, name := range []string{"test-results.xml", "test-results.json"} {
		p := filepath.Join(repoPath, name)
		if _, err := os.Stat(p); err == nil {
			testResultFiles = append(testResultFiles, p)
		}
	}
	return coverageFiles, testResultFiles
}

// runMetricsWatcher consumes MetricsWatcher updates, applies coverage and test
// status to all buildings, marks threshold warnings, emits activity events
// for threshold crossings, and records coverage history if hist is non-nil.
func runMetricsWatcher(ctx context.Context, mw *repo.MetricsWatcher, state *hub.State, h *hub.Hub, hist *model.CoverageHistory) {
	defer mw.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case src, ok := <-mw.Updates:
			if !ok {
				return
			}
			var crossings []string
			state.Update(func(curr model.CityState) model.CityState {
				prev := curr
				next := city.ApplyMetrics(curr, src)
				next = city.MarkCoverageThresholds(next)
				crossings = city.DetectThresholdCrossings(prev, next)
				return next
			})
			for _, id := range crossings {
				label := filepath.Base(id)
				state.AddActivity(model.ActivityEvent{
					Timestamp: time.Now().Format(time.RFC3339),
					Who:       "COV",
					Message:   "Coverage below threshold: " + label,
					Color:     "#b58900",
					Severity:  "warn",
				})
				slog.Info("coverage threshold crossed", "file", id)
			}
			if hist != nil {
				hist.Record(model.SnapshotFromState(state.GetState()))
			}
			if h != nil {
				h.Notify()
			}
		}
	}
}

// readModuleName reads the Go module name from go.mod at repoRoot.
// Returns an empty string if go.mod is missing or malformed.
func readModuleName(repoRoot string) string {
	data, err := os.ReadFile(filepath.Join(repoRoot, "go.mod"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if after, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
			return strings.TrimSpace(after)
		}
	}
	return ""
}
