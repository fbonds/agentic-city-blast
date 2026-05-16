/**
 * OcclusionDetector — pure logic for finding which buildings occlude a focused building.
 *
 * A building B_front occludes B_back when:
 *   1. B_front has a strictly higher sort key (gx+gy) — it is drawn later / visually in front.
 *   2. B_front's grid footprint overlaps B_back's footprint — they share visual screen area.
 *
 * This is a conservative grid-space approximation. It detects the common case (footprint
 * overlap in top-down view) without requiring screen-space polygon intersection.
 */

import type { Building } from '../store/cityStore';

/**
 * Return the IDs of all buildings that visually occlude `focused`.
 * Used by the X-ray effect: when the keyboard cursor lands on a building that
 * is behind another building, the occluding buildings are faded so the user
 * can see the building they have focused.
 */
export function findOccluders(focused: Building, buildings: Building[]): Set<string> {
  const result = new Set<string>();
  const focusedKey = focused.gx + focused.gy;

  for (const b of buildings) {
    if (b.id === focused.id) continue;
    // Only consider buildings drawn after (in front of) the focused building.
    if (b.gx + b.gy <= focusedKey) continue;
    // Grid footprint overlap: both axes must overlap.
    const overlapX = b.gx < focused.gx + focused.gw && b.gx + b.gw > focused.gx;
    const overlapY = b.gy < focused.gy + focused.gh && b.gy + b.gh > focused.gy;
    if (overlapX && overlapY) {
      result.add(b.id);
    }
  }

  return result;
}
