import { describe, it, expect, beforeEach } from 'vitest';
import {
  buildBaseline,
  detectRegressions,
  raiseBaseline,
  useCoverageStore,
  DEFAULT_REGRESSION_THRESHOLD,
} from './coverageStore';
import type { Building } from './cityStore';

// ── Fixtures ──────────────────────────────────────────────────────────────────

const makeBuilding = (overrides: Partial<Building>): Building => ({
  id: 'b1',
  districtId: 'd1',
  label: 'auth.go',
  language: 'go',
  loc: 100,
  coverage: 0.8,
  coverageWarn: false,
  status: 'ok',
  editing: false,
  exports: 3,
  blastRadius: 0,
  gx: 0,
  gy: 0,
  gw: 2,
  gh: 2,
  gz: 0,
  ...overrides,
});

// ── buildBaseline ─────────────────────────────────────────────────────────────

describe('buildBaseline', () => {
  it('returns empty object when no buildings', () => {
    expect(buildBaseline([])).toEqual({});
  });

  it('includes buildings with known coverage', () => {
    const buildings = [makeBuilding({ id: 'b1', coverage: 0.8 })];
    expect(buildBaseline(buildings)).toEqual({ b1: 0.8 });
  });

  it('excludes buildings with unknown coverage (-1)', () => {
    const buildings = [
      makeBuilding({ id: 'b1', coverage: 0.8 }),
      makeBuilding({ id: 'b2', coverage: -1 }),
    ];
    expect(buildBaseline(buildings)).toEqual({ b1: 0.8 });
  });

  it('includes buildings with 0.0 coverage (known, just zero)', () => {
    const buildings = [makeBuilding({ id: 'b1', coverage: 0.0 })];
    expect(buildBaseline(buildings)).toEqual({ b1: 0.0 });
  });
});

// ── detectRegressions ─────────────────────────────────────────────────────────

describe('detectRegressions', () => {
  it('returns empty when no regressions', () => {
    const baseline = { b1: 0.8 };
    const buildings = [makeBuilding({ id: 'b1', coverage: 0.8 })];
    expect(detectRegressions(buildings, baseline, 0.2)).toEqual([]);
  });

  it('detects a regression exceeding the threshold', () => {
    const baseline = { b1: 0.8 };
    const buildings = [makeBuilding({ id: 'b1', coverage: 0.5 })];
    const result = detectRegressions(buildings, baseline, 0.2);
    expect(result).toHaveLength(1);
    expect(result[0].id).toBe('b1');
    expect(result[0].drop).toBeCloseTo(0.3, 5);
    expect(result[0].previousCoverage).toBeCloseTo(0.8, 5);
    expect(result[0].currentCoverage).toBeCloseTo(0.5, 5);
  });

  it('does not flag a drop safely below the threshold', () => {
    const baseline = { b1: 0.8 };
    // 0.8 - 0.65 = 0.15, which is well below the 0.20 threshold
    const buildings = [makeBuilding({ id: 'b1', coverage: 0.65 })];
    expect(detectRegressions(buildings, baseline, 0.2)).toEqual([]);
  });

  it('flags a drop just above the threshold', () => {
    const baseline = { b1: 0.8 };
    const buildings = [makeBuilding({ id: 'b1', coverage: 0.59 })];
    // Drop ≈ 0.21 > 0.2 threshold
    const result = detectRegressions(buildings, baseline, 0.2);
    expect(result).toHaveLength(1);
  });

  it('skips buildings with unknown current coverage', () => {
    const baseline = { b1: 0.8 };
    const buildings = [makeBuilding({ id: 'b1', coverage: -1 })];
    expect(detectRegressions(buildings, baseline, 0.2)).toEqual([]);
  });

  it('skips buildings absent from baseline', () => {
    const baseline: Record<string, number> = {};
    const buildings = [makeBuilding({ id: 'b1', coverage: 0.5 })];
    expect(detectRegressions(buildings, baseline, 0.2)).toEqual([]);
  });

  it('skips buildings with unknown baseline coverage (-1 in baseline)', () => {
    const baseline = { b1: -1 };
    const buildings = [makeBuilding({ id: 'b1', coverage: 0.5 })];
    expect(detectRegressions(buildings, baseline, 0.2)).toEqual([]);
  });

  it('sorts by largest drop first', () => {
    const baseline = { b1: 0.9, b2: 0.7 };
    const buildings = [
      makeBuilding({ id: 'b1', coverage: 0.4 }), // drop 0.5
      makeBuilding({ id: 'b2', coverage: 0.2 }), // drop 0.5
    ];
    // b1 and b2 have equal drops — sorted deterministically
    const result = detectRegressions(buildings, baseline, 0.2);
    expect(result).toHaveLength(2);

    // Re-run with different drops to verify sort order
    const baseline2 = { b1: 0.9, b2: 0.9 };
    const buildings2 = [
      makeBuilding({ id: 'b1', coverage: 0.3 }), // drop 0.6
      makeBuilding({ id: 'b2', coverage: 0.5 }), // drop 0.4
    ];
    const result2 = detectRegressions(buildings2, baseline2, 0.2);
    expect(result2[0].id).toBe('b1');
    expect(result2[1].id).toBe('b2');
  });

  it('includes label and districtId in regression records', () => {
    const baseline = { b1: 0.9 };
    const buildings = [makeBuilding({ id: 'b1', label: 'auth.go', districtId: 'internal/auth', coverage: 0.5 })];
    const result = detectRegressions(buildings, baseline, 0.2);
    expect(result[0].label).toBe('auth.go');
    expect(result[0].districtId).toBe('internal/auth');
  });
});

// ── raiseBaseline ─────────────────────────────────────────────────────────────

describe('raiseBaseline', () => {
  it('returns same reference when nothing improved', () => {
    const baseline = { b1: 0.8 };
    const buildings = [makeBuilding({ id: 'b1', coverage: 0.7 })];
    const result = raiseBaseline(buildings, baseline);
    expect(result).toBe(baseline); // same reference — no change
  });

  it('raises baseline when coverage improves', () => {
    const baseline = { b1: 0.8 };
    const buildings = [makeBuilding({ id: 'b1', coverage: 0.9 })];
    const result = raiseBaseline(buildings, baseline);
    expect(result.b1).toBeCloseTo(0.9, 5);
    expect(result).not.toBe(baseline);
  });

  it('adds new files not previously in baseline', () => {
    const baseline = { b1: 0.8 };
    const buildings = [
      makeBuilding({ id: 'b1', coverage: 0.8 }),
      makeBuilding({ id: 'b2', coverage: 0.6 }),
    ];
    const result = raiseBaseline(buildings, baseline);
    expect(result.b2).toBeCloseTo(0.6, 5);
  });

  it('skips buildings with unknown coverage', () => {
    const baseline = { b1: 0.8 };
    const buildings = [makeBuilding({ id: 'b1', coverage: -1 })];
    const result = raiseBaseline(buildings, baseline);
    expect(result).toBe(baseline);
  });
});

// ── useCoverageStore ──────────────────────────────────────────────────────────

describe('useCoverageStore', () => {
  beforeEach(() => {
    // Reset store to initial state between tests
    useCoverageStore.setState({
      baseline: {},
      regressions: [],
      coverageAlarmActive: false,
      regressionThreshold: DEFAULT_REGRESSION_THRESHOLD,
    });
  });

  it('sets baseline on first updateCoverage call, no alarm', () => {
    const buildings = [makeBuilding({ id: 'b1', coverage: 0.8 })];
    useCoverageStore.getState().updateCoverage(buildings);

    const state = useCoverageStore.getState();
    expect(state.baseline).toEqual({ b1: 0.8 });
    expect(state.coverageAlarmActive).toBe(false);
    expect(state.regressions).toEqual([]);
  });

  it('does not trigger alarm when coverage is stable', () => {
    const buildings = [makeBuilding({ id: 'b1', coverage: 0.8 })];
    useCoverageStore.getState().updateCoverage(buildings);
    useCoverageStore.getState().updateCoverage(buildings);

    expect(useCoverageStore.getState().coverageAlarmActive).toBe(false);
  });

  it('triggers alarm when a file drops more than threshold', () => {
    const initial = [makeBuilding({ id: 'b1', coverage: 0.8 })];
    useCoverageStore.getState().updateCoverage(initial);

    const regressed = [makeBuilding({ id: 'b1', coverage: 0.5 })];
    useCoverageStore.getState().updateCoverage(regressed);

    const state = useCoverageStore.getState();
    expect(state.coverageAlarmActive).toBe(true);
    expect(state.regressions).toHaveLength(1);
    expect(state.regressions[0].id).toBe('b1');
  });

  it('does not trigger alarm when drop is within threshold', () => {
    const initial = [makeBuilding({ id: 'b1', coverage: 0.8 })];
    useCoverageStore.getState().updateCoverage(initial);

    // Drop of exactly 0.1 — below default threshold of 0.20
    const stable = [makeBuilding({ id: 'b1', coverage: 0.7 })];
    useCoverageStore.getState().updateCoverage(stable);

    expect(useCoverageStore.getState().coverageAlarmActive).toBe(false);
  });

  it('dismissCoverageAlarm clears alarm and updates baseline', () => {
    const initial = [makeBuilding({ id: 'b1', coverage: 0.8 })];
    useCoverageStore.getState().updateCoverage(initial);

    const regressed = [makeBuilding({ id: 'b1', coverage: 0.5 })];
    useCoverageStore.getState().updateCoverage(regressed);
    expect(useCoverageStore.getState().coverageAlarmActive).toBe(true);

    useCoverageStore.getState().dismissCoverageAlarm();

    const state = useCoverageStore.getState();
    expect(state.coverageAlarmActive).toBe(false);
    expect(state.regressions).toEqual([]);
    // Baseline updated to post-regression value
    expect(state.baseline.b1).toBeCloseTo(0.5, 5);
  });

  it('does not re-trigger after dismiss if coverage stays at regressed level', () => {
    const initial = [makeBuilding({ id: 'b1', coverage: 0.8 })];
    useCoverageStore.getState().updateCoverage(initial);

    const regressed = [makeBuilding({ id: 'b1', coverage: 0.5 })];
    useCoverageStore.getState().updateCoverage(regressed);
    useCoverageStore.getState().dismissCoverageAlarm();

    // Same coverage — no new regression relative to acknowledged baseline
    useCoverageStore.getState().updateCoverage(regressed);
    expect(useCoverageStore.getState().coverageAlarmActive).toBe(false);
  });

  it('re-triggers if coverage drops further after dismiss', () => {
    const initial = [makeBuilding({ id: 'b1', coverage: 0.8 })];
    useCoverageStore.getState().updateCoverage(initial);

    const regressed = [makeBuilding({ id: 'b1', coverage: 0.5 })];
    useCoverageStore.getState().updateCoverage(regressed);
    useCoverageStore.getState().dismissCoverageAlarm();

    // Another big drop from the new baseline of 0.5
    const worse = [makeBuilding({ id: 'b1', coverage: 0.2 })];
    useCoverageStore.getState().updateCoverage(worse);
    expect(useCoverageStore.getState().coverageAlarmActive).toBe(true);
  });

  it('setRegressionThreshold changes the threshold', () => {
    useCoverageStore.getState().setRegressionThreshold(0.05);
    expect(useCoverageStore.getState().regressionThreshold).toBe(0.05);

    const initial = [makeBuilding({ id: 'b1', coverage: 0.8 })];
    useCoverageStore.getState().updateCoverage(initial);

    // Drop of 0.1 — above new threshold of 0.05
    const regressed = [makeBuilding({ id: 'b1', coverage: 0.7 })];
    useCoverageStore.getState().updateCoverage(regressed);
    expect(useCoverageStore.getState().coverageAlarmActive).toBe(true);
  });

  it('raises baseline when coverage improves, preventing false alarms', () => {
    const initial = [makeBuilding({ id: 'b1', coverage: 0.6 })];
    useCoverageStore.getState().updateCoverage(initial);

    // Coverage improves
    const improved = [makeBuilding({ id: 'b1', coverage: 0.9 })];
    useCoverageStore.getState().updateCoverage(improved);
    expect(useCoverageStore.getState().baseline.b1).toBeCloseTo(0.9, 5);

    // Now drop back to 0.6 — from baseline of 0.9, drop = 0.3 > 0.2 → alarm
    const dropped = [makeBuilding({ id: 'b1', coverage: 0.6 })];
    useCoverageStore.getState().updateCoverage(dropped);
    expect(useCoverageStore.getState().coverageAlarmActive).toBe(true);
  });

  it('clears alarm when regressions resolve without explicit dismiss', () => {
    const initial = [makeBuilding({ id: 'b1', coverage: 0.8 })];
    useCoverageStore.getState().updateCoverage(initial);

    // Trigger alarm with a big drop
    const regressed = [makeBuilding({ id: 'b1', coverage: 0.5 })];
    useCoverageStore.getState().updateCoverage(regressed);
    expect(useCoverageStore.getState().coverageAlarmActive).toBe(true);

    // Coverage recovers — alarm should clear without explicit dismiss
    const recovered = [makeBuilding({ id: 'b1', coverage: 0.85 })];
    useCoverageStore.getState().updateCoverage(recovered);
    expect(useCoverageStore.getState().coverageAlarmActive).toBe(false);
    expect(useCoverageStore.getState().regressions).toEqual([]);
  });
});
