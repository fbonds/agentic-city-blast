/**
 * RoadRenderer — rooftop-to-rooftop bezier arcs for dependency edges.
 *
 * Roads are hidden by default. When a building is selected, only its
 * inbound and outbound edges are drawn as arcs rising from the roof
 * center of each connected building. Cyan = outgoing (this file imports),
 * blue = incoming (depends on this file).
 */

import type { IsometricCamera } from './IsometricCamera';
import type { Building, Confidence, Road } from '../store/cityStore';
import { sol } from '../theme/colors';

// Arc height in screen pixels — how high the bezier control points rise
// above the rooftop endpoints.
const ROAD_ARC_H = 60;

// Per-confidence visual style.
const CONFIDENCE_STYLES: Record<Confidence, { dash: number[]; alpha: number; width: number }> = {
  exact:    { dash: [],     alpha: 0.75, width: 1.5 },
  inferred: { dash: [6, 4], alpha: 0.55, width: 1.2 },
  weak:     { dash: [3, 5], alpha: 0.35, width: 0.9 },
};
const FALLBACK_STYLE = CONFIDENCE_STYLES.weak;

const COLOR_OUTGOING = sol.cyan;  // selected building imports this
const COLOR_INCOMING = sol.blue;  // this file depends on selected building

export function drawRoads(
  ctx: CanvasRenderingContext2D,
  camera: IsometricCamera,
  roads: Road[],
  buildings: Building[],
  selectedBuildingId: string | null,
): void {
  // Only draw when a building is selected.
  if (!selectedBuildingId || roads.length === 0) return;

  // Pre-compute screen-space rooftop centers for all buildings.
  const roofCenters = new Map<string, [number, number]>();
  for (const b of buildings) {
    roofCenters.set(b.id, camera.project(b.gx + b.gw / 2, b.gy + b.gh / 2, b.gz));
  }

  ctx.save();
  ctx.lineCap = 'round';

  for (const road of roads) {
    const isOutgoing = road.fromId === selectedBuildingId;
    const isIncoming = road.toId === selectedBuildingId;
    if (!isOutgoing && !isIncoming) continue;

    const from = roofCenters.get(road.fromId);
    const to = roofCenters.get(road.toId);
    if (!from || !to) continue;

    const style = CONFIDENCE_STYLES[road.confidence] ?? FALLBACK_STYLE;
    const color = isOutgoing ? COLOR_OUTGOING : COLOR_INCOMING;

    ctx.setLineDash(style.dash);
    ctx.strokeStyle = color;
    ctx.lineWidth = style.width;
    ctx.globalAlpha = style.alpha;

    // Draw bezier arc from rooftop to rooftop with raised control points.
    ctx.beginPath();
    ctx.moveTo(from[0], from[1]);
    ctx.bezierCurveTo(
      from[0], from[1] - ROAD_ARC_H,
      to[0],   to[1]   - ROAD_ARC_H,
      to[0],   to[1],
    );
    ctx.stroke();

    // Draw a small dot at the far endpoint to show where the edge lands.
    const endpoint = isOutgoing ? to : from;
    ctx.setLineDash([]);
    ctx.globalAlpha = style.alpha * 1.2;
    ctx.beginPath();
    ctx.arc(endpoint[0], endpoint[1], 2.5, 0, Math.PI * 2);
    ctx.fillStyle = color;
    ctx.fill();
  }

  ctx.restore();
}
