/**
 * CoverageAlarmOverlay — amber alarm overlay for coverage regressions.
 *
 * Shown independently of the build-failure alarm (red).  Dismissal is
 * independent: Escape or the dismiss button clears the coverage alarm
 * without touching the build-failure alarm state.
 *
 * Design spec: DESIGN.md P2.3 (variant for coverage regression)
 */

import { useEffect, useRef, useMemo, useCallback } from 'react';
import type { CSSProperties } from 'react';
import { useCityStore } from '../store/cityStore';
import type { Building, Road } from '../store/cityStore';
import { useUiStore } from '../store/uiStore';
import { useCoverageStore } from '../store/coverageStore';
import type { CoverageRegression } from '../store/coverageStore';
import { sol, FONT, TOP_BAR_H, BOTTOM_STRIP_H, tierColor } from '../hud/palette';
import { useFocusTrap, useFocusRestore } from '../hooks/useFocusTrap';
import { findDownstream } from './AlarmOverlay';

// ── Alarm info derivation ────────────────────────────────────────────────────

export interface CoverageAlarmInfo {
  regressions: CoverageRegression[];
  /** All buildings downstream from any regressed file (dependency blast radius). */
  blastRadius: Building[];
}

export function deriveCoverageAlarmInfo(
  regressions: CoverageRegression[],
  buildings: Building[],
  roads: Road[],
): CoverageAlarmInfo {
  const buildingMap = new Map(buildings.map((b) => [b.id, b]));

  const blastSet = new Set<string>();
  for (const r of regressions) {
    if (buildingMap.has(r.id)) {
      const downstream = findDownstream(r.id, roads, buildingMap);
      for (const id of downstream) blastSet.add(id);
    }
  }

  // Exclude the regressed files themselves from blast radius display
  for (const r of regressions) blastSet.delete(r.id);

  const blastRadius = buildings.filter((b) => blastSet.has(b.id));
  return { regressions, blastRadius };
}

// ── Styles ───────────────────────────────────────────────────────────────────

// Amber palette — RGB(181, 137, 0) — distinct from red build-failure alarm
const AMBER = sol.yellow; // #b58900
const PANEL_BG = 'rgba(30,22,0,0.93)';
const PANEL_BORDER = `1px solid ${AMBER}40`;

const S: Record<string, CSSProperties> = {
  vignette: {
    position: 'fixed',
    inset: 0,
    pointerEvents: 'none',
    zIndex: 140,
    background: `radial-gradient(ellipse at center, transparent 30%, rgba(181,137,0,0.18) 70%, rgba(181,137,0,0.40) 100%)`,
    boxShadow: `inset 0 0 120px rgba(181,137,0,0.40), inset 0 0 240px rgba(181,137,0,0.20)`,
  },
  banner: {
    position: 'fixed',
    top: 0,
    left: 0,
    right: 0,
    height: TOP_BAR_H,
    borderBottom: `1px solid ${AMBER}`,
    display: 'flex',
    alignItems: 'center',
    padding: '0 12px',
    gap: 14,
    fontFamily: FONT,
    fontSize: 10,
    color: AMBER,
    background: PANEL_BG,
    zIndex: 160,
    pointerEvents: 'auto',
    userSelect: 'none',
  },
  leftPanel: {
    position: 'fixed',
    top: TOP_BAR_H + 8,
    left: 8,
    width: 200,
    maxWidth: 'calc(40vw - 16px)',
    zIndex: 155,
    display: 'flex',
    flexDirection: 'column',
    gap: 8,
    fontFamily: FONT,
    fontSize: 10,
    color: sol.base0,
    pointerEvents: 'auto',
    userSelect: 'none',
    maxHeight: `calc(100vh - ${TOP_BAR_H + BOTTOM_STRIP_H + 24}px)`,
    overflowY: 'auto',
  },
  rightPanel: {
    position: 'fixed',
    top: TOP_BAR_H + 8,
    right: 8,
    width: 200,
    maxWidth: 'calc(40vw - 16px)',
    zIndex: 155,
    display: 'flex',
    flexDirection: 'column',
    gap: 8,
    fontFamily: FONT,
    fontSize: 10,
    color: sol.base0,
    pointerEvents: 'auto',
    userSelect: 'none',
    maxHeight: `calc(100vh - ${TOP_BAR_H + BOTTOM_STRIP_H + 24}px)`,
    overflowY: 'auto',
  },
  panel: {
    background: PANEL_BG,
    border: PANEL_BORDER,
    borderRadius: 2,
    padding: '6px 8px',
  },
  panelTitle: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 6,
    fontSize: 9,
    textTransform: 'uppercase' as const,
    letterSpacing: '0.12em',
    fontWeight: 700,
  },
  panelSub: {
    opacity: 0.7,
    fontWeight: 400,
  },
  regressionRow: {
    display: 'flex',
    justifyContent: 'space-between',
    fontSize: 9,
    padding: '2px 0',
    borderBottom: `1px dotted ${AMBER}25`,
  },
  blastRow: {
    display: 'flex',
    justifyContent: 'space-between',
    fontSize: 9,
    padding: '2px 0',
    borderBottom: `1px dotted ${AMBER}20`,
  },
  statRow: {
    display: 'flex',
    justifyContent: 'space-between',
    fontSize: 9,
    padding: '2px 0',
  },
  dispatchBtn: {
    width: '100%',
    marginTop: 6,
    background: AMBER,
    color: sol.base03,
    border: 'none',
    fontFamily: FONT,
    fontSize: 10,
    fontWeight: 700,
    padding: '5px 0',
    cursor: 'pointer',
    letterSpacing: '0.1em',
    borderRadius: 2,
  },
  dismissBtn: {
    width: '100%',
    marginTop: 4,
    background: 'transparent',
    color: sol.base00,
    border: `1px solid ${AMBER}40`,
    fontFamily: FONT,
    fontSize: 9,
    padding: '3px 0',
    cursor: 'pointer',
    letterSpacing: '0.08em',
    borderRadius: 2,
  },
  hint: {
    fontSize: 8,
    color: sol.base00,
    marginTop: 4,
    textAlign: 'center' as const,
  },
};

// ── Helpers ──────────────────────────────────────────────────────────────────

function fmtPct(ratio: number): string {
  return `${Math.round(ratio * 100)}%`;
}

function fmtDrop(drop: number): string {
  return `-${Math.round(drop * 100)}pp`;
}

// ── Sub-components ───────────────────────────────────────────────────────────

function PanelBox({
  title,
  sub,
  children,
}: {
  title: string;
  sub?: string;
  children: React.ReactNode;
}): JSX.Element {
  return (
    <div style={S.panel}>
      <div style={{ ...S.panelTitle, color: AMBER }}>
        <span>{title}</span>
        {sub && <span style={S.panelSub}>{sub}</span>}
      </div>
      {children}
    </div>
  );
}

function StatLine({
  label,
  value,
  valueColor,
}: {
  label: string;
  value: string;
  valueColor?: string;
}): JSX.Element {
  return (
    <div style={S.statRow}>
      <span style={{ opacity: 0.65 }}>{label}</span>
      <span style={{ color: valueColor ?? sol.base1 }}>{value}</span>
    </div>
  );
}

// ── Rapid-response dispatch panel ────────────────────────────────────────────

function CoverageRapidResponse({
  alarm,
  onDismiss,
}: {
  alarm: CoverageAlarmInfo;
  onDismiss: () => void;
}): JSX.Element {
  const openDispatchScope = useUiStore((s) => s.openDispatchScope);
  const alarmActive = useCoverageStore((s) => s.coverageAlarmActive);
  const dispatchMode = useUiStore((s) => s.dispatchMode);
  const agents = useCityStore((s) => s.city.agents);

  const availableAgent = agents.find(
    (a) => a.mode === 'idle' || a.mode === 'waiting' || a.mode === 'done',
  );

  const handleDispatch = useCallback(() => {
    const ids = alarm.regressions.map((r) => r.id);
    onDismiss();
    openDispatchScope(ids, 'add-test');
  }, [alarm.regressions, openDispatchScope, onDismiss]);

  // Enter dispatches when coverage alarm is active and dispatch wizard is closed
  useEffect(() => {
    if (!alarmActive || dispatchMode) return;

    const handler = (e: KeyboardEvent) => {
      if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) return;
      if (e.key === 'Enter') {
        handleDispatch();
        e.preventDefault();
      }
    };

    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [alarmActive, dispatchMode, handleDispatch]);

  return (
    <div style={S.panel}>
      <div style={{ ...S.panelTitle, color: AMBER }}>DISPATCH</div>
      <div style={{ fontSize: 9, color: sol.base0, lineHeight: 1.4, marginBottom: 6 }}>
        recommend agent →{' '}
        <span style={{ color: AMBER, fontWeight: 700 }}>add-test</span>
      </div>
      {availableAgent && (
        <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 9, padding: '2px 0', color: sol.base0 }}>
          <span>agent: {availableAgent.id}</span>
          {availableAgent.modelTier && (
            <span style={{ color: tierColor(availableAgent.modelTier) }}>
              {availableAgent.modelTier}
            </span>
          )}
        </div>
      )}
      <button type="button" style={S.dispatchBtn} onClick={handleDispatch}>
        {'▶ DISPATCH NOW'}
      </button>
      <button type="button" style={S.dismissBtn} onClick={onDismiss}>
        dismiss
      </button>
      <div style={S.hint}>Enter to dispatch · Esc to dismiss</div>
    </div>
  );
}

// ── Main component ───────────────────────────────────────────────────────────

export function CoverageAlarmOverlay(): JSX.Element | null {
  const phase2 = useUiStore((s) => s.phase2);
  const dispatchMode = useUiStore((s) => s.dispatchMode);

  const coverageAlarmActive = useCoverageStore((s) => s.coverageAlarmActive);
  const regressions = useCoverageStore((s) => s.regressions);
  const dismissCoverageAlarm = useCoverageStore((s) => s.dismissCoverageAlarm);
  const regressionThreshold = useCoverageStore((s) => s.regressionThreshold);

  const buildings = useCityStore((s) => s.city.buildings);
  const roads = useCityStore((s) => s.city.roads);
  const stats = useCityStore((s) => s.city.stats);

  const alarm = useMemo(
    () => deriveCoverageAlarmInfo(regressions, buildings, roads),
    [regressions, buildings, roads],
  );

  const overlayRef = useRef<HTMLDivElement>(null);
  useFocusTrap(overlayRef, coverageAlarmActive);
  useFocusRestore(coverageAlarmActive);

  // Move focus into overlay when alarm activates
  useEffect(() => {
    if (coverageAlarmActive) {
      requestAnimationFrame(() => {
        const el = overlayRef.current;
        if (!el) return;
        const first = el.querySelector<HTMLElement>('button:not([disabled])');
        (first ?? el).focus();
      });
    }
  }, [coverageAlarmActive]);

  // Escape dismisses alarm when dispatch wizard is closed
  const dispatchRef = useRef(dispatchMode);
  dispatchRef.current = dispatchMode;

  useEffect(() => {
    if (!coverageAlarmActive) return;

    const handler = (e: KeyboardEvent) => {
      if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) return;
      if (dispatchRef.current) return;

      if (e.key === 'Escape') {
        dismissCoverageAlarm();
        e.preventDefault();
      }
    };

    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [coverageAlarmActive, dismissCoverageAlarm]);

  if (!phase2 || !coverageAlarmActive || regressions.length === 0) return null;

  const thresholdPct = Math.round(regressionThreshold * 100);
  const cov = stats.coverage >= 0 ? fmtPct(stats.coverage) : '--';

  return (
    <div ref={overlayRef} tabIndex={-1} style={{ outline: 'none' }}>
      <style>{`
        @keyframes ac-cov-pulse {
          0%, 100% { opacity: 0.35; transform: scale(1); }
          50% { opacity: 0.75; transform: scale(1.04); }
        }
        @keyframes ac-cov-blink {
          0%, 49% { opacity: 1; }
          50%, 100% { opacity: 0.30; }
        }
        .ac-cov-pulse { animation: ac-cov-pulse 2s ease-in-out infinite; transform-origin: center; }
        .ac-cov-blink { animation: ac-cov-blink 1.4s steps(1) infinite; }
      `}</style>

      {/* Amber vignette */}
      <div className="ac-cov-pulse" style={S.vignette} />

      {/* Coverage regression banner */}
      <div style={S.banner}>
        <span
          className="ac-cov-blink"
          style={{ fontWeight: 700, letterSpacing: '0.2em', color: AMBER }}
        >
          {'▲ COVERAGE REGRESSION'}
        </span>
        <span style={{ opacity: 0.7 }}>
          threshold={thresholdPct}pp
        </span>
        <span style={{ opacity: 0.7 }}>
          files={alarm.regressions.length}
        </span>
        <span style={{ flex: 1 }} />
        <span>cov={cov}</span>
        <span
          className="ac-cov-blink"
          style={{ color: AMBER, fontWeight: 700 }}
        >
          {'● '}{alarm.regressions.length} regressed
        </span>
      </div>

      {/* Left panel — regression origins + blast radius + dispatch */}
      <div style={S.leftPanel}>
        {/* Regressed files */}
        <PanelBox
          title="COVERAGE DROP"
          sub={`>${thresholdPct}pp regression`}
        >
          {alarm.regressions.map((r) => (
            <div key={r.id} style={S.regressionRow}>
              <span style={{ color: AMBER, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', maxWidth: 120 }}>
                {'● '}{r.label}
              </span>
              <span style={{ opacity: 0.85, whiteSpace: 'nowrap' }}>
                {fmtPct(r.previousCoverage)} → {fmtPct(r.currentCoverage)}
                {'  '}
                <span style={{ color: AMBER, fontWeight: 700 }}>{fmtDrop(r.drop)}</span>
              </span>
            </div>
          ))}
        </PanelBox>

        {/* Downstream blast radius */}
        {alarm.blastRadius.length > 0 && (
          <PanelBox title="BLAST RADIUS" sub="downstream">
            {alarm.blastRadius.map((b) => (
              <div key={b.id} style={S.blastRow}>
                <span style={{ color: sol.base0 }}>{'● '}{b.label}</span>
                <span style={{ opacity: 0.50 }}>dep</span>
              </div>
            ))}
          </PanelBox>
        )}

        {/* Rapid-response dispatch */}
        <CoverageRapidResponse alarm={alarm} onDismiss={dismissCoverageAlarm} />
      </div>

      {/* Right panel — stats */}
      <div style={S.rightPanel}>
        <PanelBox title="COVERAGE HEALTH">
          <StatLine label="files regressed" value={String(alarm.regressions.length)} valueColor={AMBER} />
          <StatLine label="blast radius" value={String(alarm.blastRadius.length)} />
          <StatLine label="repo coverage" value={cov} />
          <StatLine
            label="threshold"
            value={`${thresholdPct}pp`}
            valueColor={sol.base0}
          />
        </PanelBox>

        {/* Per-regression details */}
        {alarm.regressions.map((r) => (
          <PanelBox key={r.id} title={r.label} sub={r.districtId}>
            <StatLine label="before" value={fmtPct(r.previousCoverage)} />
            <StatLine label="after" value={fmtPct(r.currentCoverage)} valueColor={AMBER} />
            <StatLine label="drop" value={fmtDrop(r.drop)} valueColor={AMBER} />
          </PanelBox>
        ))}
      </div>
    </div>
  );
}
