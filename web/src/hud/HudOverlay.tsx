import { useUiStore } from '../store/uiStore';
import { useCoverageStore } from '../store/coverageStore';
import { TopBar } from './TopBar';
import { LeftRail } from './LeftRail';
import { RightRail } from './RightRail';
import { BottomStrip } from './BottomStrip';
import { ShortcutOverlay } from './ShortcutOverlay';
import { Minimap } from './Minimap';
import { DispatchWizard } from '../orchestration/DispatchWizard';
import { CommandPalette } from '../orchestration/CommandPalette';
import { AlarmOverlay } from '../orchestration/AlarmOverlay';
import { CoverageDropToast } from '../orchestration/CoverageDropToast';
import { useCoverageWatcher } from '../hooks/useCoverageWatcher';
import { CoverageAlarmOverlay } from '../orchestration/CoverageAlarmOverlay';

export function HudOverlay(): JSX.Element {
  const highContrast = useUiStore((s) => s.highContrast);
  const alarmActive = useUiStore((s) => s.alarmActive);
  useCoverageWatcher();
  const coverageAlarmActive = useCoverageStore((s) => s.coverageAlarmActive);

  // Either alarm replaces the normal top bar and rails
  const eitherAlarm = alarmActive || coverageAlarmActive;

  return (
    <>
      <div
        style={
          highContrast
            ? { filter: 'brightness(1.5) contrast(1.2)', isolation: 'isolate' }
            : undefined
        }
      >
        {/* Alarm overlays replace normal top bar / rails when active */}
        {!eitherAlarm && <TopBar />}
        {!eitherAlarm && <LeftRail />}
        {!eitherAlarm && <RightRail />}
        <BottomStrip />
      </div>
      <ShortcutOverlay />
      <Minimap />
      <AlarmOverlay />
      <CoverageAlarmOverlay />
      <DispatchWizard />
      <CommandPalette />
      <CoverageDropToast />
    </>
  );
}
