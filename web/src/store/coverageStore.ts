/**
 * coverageStore — tracks coverage regressions across WebSocket updates.
 *
 * On first meaningful coverage data the baseline is captured (no alarm).
 * Subsequent updates are compared against the baseline; files that drop
 * more than regressionThreshold trigger coverageAlarmActive.
 *
 * Dismissing the alarm acknowledges the regression: the baseline is
 * updated to the current (lower) coverage so the same drop does not
 * re-trigger immediately. Improvements raise the baseline automatically
 * so future drops compare against the best-seen value.
 */

import { create } from 'zustand';
import type { Building } from './cityStore';

// ── Public types ─────────────────────────────────────────────────────────────

export interface CoverageRegression {
  id: string;
  label: string;
  districtId: string;
  previousCoverage: number; // 0.0–1.0
  currentCoverage: number;  // 0.0–1.0
  drop: number;             // previousCoverage - currentCoverage (> 0)
}

/** Default threshold: a >20 percentage-point drop in a single file triggers alarm. */
export const DEFAULT_REGRESSION_THRESHOLD = 0.20;

// ── Store interface ──────────────────────────────────────────────────────────

interface CoverageStore {
  /** Per-file best-seen coverage ratios. Empty until first data arrives. */
  baseline: Record<string, number>;
  /** Files whose coverage dropped below regressionThreshold since last baseline. */
  regressions: CoverageRegression[];
  /** True when at least one regression was detected and not yet dismissed. */
  coverageAlarmActive: boolean;
  /** Drop magnitude that triggers an alarm (0.0–1.0; default 0.20). */
  regressionThreshold: number;

  /**
   * Compare buildings against the stored baseline.
   * First call with coverage data sets the baseline (no alarm).
   * Subsequent calls detect regressions.
   */
  updateCoverage: (buildings: Building[]) => void;
  /**
   * Dismiss the coverage alarm and update the baseline to the current
   * (lower) coverage values so the same regression does not immediately
   * re-trigger.
   */
  dismissCoverageAlarm: () => void;
  /** Override the regression threshold (0.0–1.0). */
  setRegressionThreshold: (threshold: number) => void;
}

// ── Pure helpers (exported for tests) ───────────────────────────────────────

/** Build a baseline map from buildings that have known coverage (>= 0). */
export function buildBaseline(buildings: Building[]): Record<string, number> {
  const baseline: Record<string, number> = {};
  for (const b of buildings) {
    if (b.coverage >= 0) {
      baseline[b.id] = b.coverage;
    }
  }
  return baseline;
}

/**
 * Detect regressions by comparing buildings against the baseline.
 * Only buildings with known coverage in both snapshots are compared.
 * Results are sorted by largest drop first.
 */
export function detectRegressions(
  buildings: Building[],
  baseline: Record<string, number>,
  threshold: number,
): CoverageRegression[] {
  const regressions: CoverageRegression[] = [];
  for (const b of buildings) {
    if (b.coverage < 0) continue;
    const prev = baseline[b.id];
    if (prev === undefined || prev < 0) continue;
    const drop = prev - b.coverage;
    if (drop > threshold) {
      regressions.push({
        id: b.id,
        label: b.label,
        districtId: b.districtId,
        previousCoverage: prev,
        currentCoverage: b.coverage,
        drop,
      });
    }
  }
  return regressions.sort((a, b) => b.drop - a.drop);
}

/**
 * Raise the baseline for any building whose current coverage exceeds the stored
 * value. This ensures future regressions are measured from the best-seen state.
 * Returns a new baseline object (does not mutate).
 */
export function raiseBaseline(
  buildings: Building[],
  baseline: Record<string, number>,
): Record<string, number> {
  let changed = false;
  const next = { ...baseline };
  for (const b of buildings) {
    if (b.coverage < 0) continue;
    const prev = baseline[b.id];
    if (prev === undefined || b.coverage > prev) {
      next[b.id] = b.coverage;
      changed = true;
    }
  }
  return changed ? next : baseline;
}

// ── Store ────────────────────────────────────────────────────────────────────

export const useCoverageStore = create<CoverageStore>((set, get) => ({
  baseline: {},
  regressions: [],
  coverageAlarmActive: false,
  regressionThreshold: DEFAULT_REGRESSION_THRESHOLD,

  updateCoverage: (buildings) => {
    const { baseline, regressionThreshold } = get();
    const hasBaseline = Object.keys(baseline).length > 0;

    if (!hasBaseline) {
      // First batch of coverage data — capture baseline, no alarm.
      set({ baseline: buildBaseline(buildings) });
      return;
    }

    const regressions = detectRegressions(buildings, baseline, regressionThreshold);
    // Raise baseline for files that improved so future drops measure from best-seen.
    const updatedBaseline = raiseBaseline(buildings, baseline);

    if (regressions.length > 0) {
      set({ regressions, coverageAlarmActive: true, baseline: updatedBaseline });
    } else {
      set({ regressions: [], coverageAlarmActive: false, baseline: updatedBaseline });
    }
  },

  dismissCoverageAlarm: () => {
    // Acknowledge regressions: update baseline to current (lower) values so the
    // same drop does not re-trigger immediately.
    const { regressions, baseline } = get();
    const newBaseline = { ...baseline };
    for (const r of regressions) {
      newBaseline[r.id] = r.currentCoverage;
    }
    set({ regressions: [], coverageAlarmActive: false, baseline: newBaseline });
  },

  setRegressionThreshold: (threshold) => set({ regressionThreshold: threshold }),
}));
