import { useEffect, useState, type CSSProperties } from 'react';
import { useCityStore } from '../store/cityStore';
import { sol, hudBase } from './palette';

/**
 * GuideOverlay — displays GUIDE narration messages as a prominent overlay
 * in the lower-center of the viewport. Messages fade in, hold, then fade out.
 * Only shown when the activity log contains entries from "GUIDE".
 * Position is fixed (not affected by zoom/pan).
 */

const FADE_IN_MS = 400;
const HOLD_MS = 4500;
const FADE_OUT_MS = 600;

const S: Record<string, CSSProperties> = {
  container: {
    ...hudBase,
    position: 'fixed',
    bottom: 48,
    left: '50%',
    transform: 'translateX(-50%)',
    maxWidth: 600,
    padding: '10px 20px',
    background: `${sol.base02}e8`,
    border: `1px solid ${sol.yellow}44`,
    borderRadius: 6,
    zIndex: 95,
    textAlign: 'center',
    pointerEvents: 'none',
    transition: `opacity ${FADE_IN_MS}ms ease-in`,
  },
  text: {
    color: sol.yellow,
    fontSize: 13,
    lineHeight: '1.5',
    letterSpacing: '0.02em',
  },
  label: {
    color: sol.base01,
    fontSize: 9,
    textTransform: 'uppercase' as const,
    letterSpacing: '0.1em',
    marginBottom: 4,
  },
};

export function GuideOverlay(): JSX.Element | null {
  const activities = useCityStore((s) => s.city.activities);
  const [visible, setVisible] = useState(false);
  const [message, setMessage] = useState('');

  // Track the last guide message we displayed to avoid re-triggering.
  const [lastShown, setLastShown] = useState('');

  useEffect(() => {
    // Find the most recent GUIDE message.
    const guideMessages = activities.filter((a) => a.who === 'GUIDE');
    if (guideMessages.length === 0) return;

    const latest = guideMessages[guideMessages.length - 1];
    if (latest.message === lastShown) return;

    // New guide message — show it.
    setMessage(latest.message);
    setLastShown(latest.message);
    setVisible(true);

    // Auto-hide after hold period.
    const timer = setTimeout(() => {
      setVisible(false);
    }, HOLD_MS);

    return () => clearTimeout(timer);
  }, [activities, lastShown]);

  if (!message) return null;

  return (
    <div
      style={{
        ...S.container,
        opacity: visible ? 1 : 0,
        transition: visible
          ? `opacity ${FADE_IN_MS}ms ease-in`
          : `opacity ${FADE_OUT_MS}ms ease-out`,
      }}
    >
      <div style={S.label}>guide</div>
      <div style={S.text}>{message}</div>
    </div>
  );
}
