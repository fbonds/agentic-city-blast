import { describe, it, expect } from 'vitest';
import { selectDistrictBuildings } from './cityStore';
import type { Building, District, Agent } from './cityStore';

const BASE_DISTRICT: District = {
  id: 'd1',
  label: 'pkg/auth',
  parentId: 'root',
  gx: 0,
  gy: 0,
  gw: 10,
  gh: 8,
};

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
  gx: 0,
  gy: 0,
  gw: 2,
  gh: 2,
  gz: 0,
  ...overrides,
});

describe('selectDistrictBuildings', () => {
  it('returns one district-building per district', () => {
    const districts: District[] = [BASE_DISTRICT];
    const buildings: Building[] = [
      makeBuilding({ id: 'b1', districtId: 'd1', loc: 200 }),
      makeBuilding({ id: 'b2', districtId: 'd1', loc: 300 }),
    ];
    const result = selectDistrictBuildings(districts, buildings, []);
    expect(result).toHaveLength(1);
    expect(result[0].id).toBe('d1');
  });

  it('sums LOC from all child buildings', () => {
    const districts: District[] = [BASE_DISTRICT];
    const buildings: Building[] = [
      makeBuilding({ id: 'b1', districtId: 'd1', loc: 150 }),
      makeBuilding({ id: 'b2', districtId: 'd1', loc: 250 }),
    ];
    const result = selectDistrictBuildings(districts, buildings, []);
    expect(result[0].loc).toBe(400);
  });

  it('counts files correctly', () => {
    const districts: District[] = [BASE_DISTRICT];
    const buildings: Building[] = [
      makeBuilding({ id: 'b1', districtId: 'd1' }),
      makeBuilding({ id: 'b2', districtId: 'd1' }),
      makeBuilding({ id: 'b3', districtId: 'd1' }),
    ];
    const result = selectDistrictBuildings(districts, buildings, []);
    expect(result[0].fileCount).toBe(3);
  });

  it('computes weighted average coverage from known children', () => {
    const districts: District[] = [BASE_DISTRICT];
    // b1: 100 LOC, 80% coverage → contributes 80
    // b2: 200 LOC, 50% coverage → contributes 100
    // total: 180 / 300 = 0.6
    const buildings: Building[] = [
      makeBuilding({ id: 'b1', districtId: 'd1', loc: 100, coverage: 0.8 }),
      makeBuilding({ id: 'b2', districtId: 'd1', loc: 200, coverage: 0.5 }),
    ];
    const result = selectDistrictBuildings(districts, buildings, []);
    expect(result[0].coverage).toBeCloseTo(0.6, 5);
  });

  it('returns -1 coverage when no children have known coverage', () => {
    const districts: District[] = [BASE_DISTRICT];
    const buildings: Building[] = [
      makeBuilding({ id: 'b1', districtId: 'd1', coverage: -1 }),
      makeBuilding({ id: 'b2', districtId: 'd1', coverage: -1 }),
    ];
    const result = selectDistrictBuildings(districts, buildings, []);
    expect(result[0].coverage).toBe(-1);
  });

  it('excludes children with unknown coverage (-1) from weighted average', () => {
    const districts: District[] = [BASE_DISTRICT];
    // b1: 100 LOC, 0.8 coverage (known)
    // b2: 200 LOC, -1 coverage (unknown — excluded)
    const buildings: Building[] = [
      makeBuilding({ id: 'b1', districtId: 'd1', loc: 100, coverage: 0.8 }),
      makeBuilding({ id: 'b2', districtId: 'd1', loc: 200, coverage: -1 }),
    ];
    const result = selectDistrictBuildings(districts, buildings, []);
    expect(result[0].coverage).toBeCloseTo(0.8, 5);
  });

  it('applies worst-case status: err beats everything', () => {
    const districts: District[] = [BASE_DISTRICT];
    const buildings: Building[] = [
      makeBuilding({ id: 'b1', districtId: 'd1', status: 'ok' }),
      makeBuilding({ id: 'b2', districtId: 'd1', status: 'err' }),
      makeBuilding({ id: 'b3', districtId: 'd1', status: 'warn' }),
    ];
    const result = selectDistrictBuildings(districts, buildings, []);
    expect(result[0].status).toBe('err');
  });

  it('applies worst-case status: warn beats ok and unknown', () => {
    const districts: District[] = [BASE_DISTRICT];
    const buildings: Building[] = [
      makeBuilding({ id: 'b1', districtId: 'd1', status: 'ok' }),
      makeBuilding({ id: 'b2', districtId: 'd1', status: 'warn' }),
      makeBuilding({ id: 'b3', districtId: 'd1', status: 'unknown' }),
    ];
    const result = selectDistrictBuildings(districts, buildings, []);
    expect(result[0].status).toBe('warn');
  });

  it('applies worst-case status: unknown beats ok', () => {
    const districts: District[] = [BASE_DISTRICT];
    const buildings: Building[] = [
      makeBuilding({ id: 'b1', districtId: 'd1', status: 'ok' }),
      makeBuilding({ id: 'b2', districtId: 'd1', status: 'unknown' }),
    ];
    const result = selectDistrictBuildings(districts, buildings, []);
    expect(result[0].status).toBe('unknown');
  });

  it('uses district geometry for gx/gy/gw/gh', () => {
    const district: District = { ...BASE_DISTRICT, gx: 5, gy: 10, gw: 20, gh: 15 };
    const buildings: Building[] = [makeBuilding({ districtId: 'd1' })];
    const result = selectDistrictBuildings([district], buildings, []);
    expect(result[0].gx).toBe(5);
    expect(result[0].gy).toBe(10);
    expect(result[0].gw).toBe(20);
    expect(result[0].gh).toBe(15);
  });

  it('counts agents targeting buildings in this district', () => {
    const districts: District[] = [BASE_DISTRICT];
    const buildings: Building[] = [
      makeBuilding({ id: 'b1', districtId: 'd1' }),
      makeBuilding({ id: 'b2', districtId: 'd1' }),
    ];
    const agents: Agent[] = [
      { id: 'a1', color: 'cyan', mode: 'work', task: 't', progress: 0.5, targetId: 'b1' },
      { id: 'a2', color: 'cyan', mode: 'work', task: 't', progress: 0.5, targetId: 'b2' },
      { id: 'a3', color: 'cyan', mode: 'work', task: 't', progress: 0.5, targetId: 'other' },
    ];
    const result = selectDistrictBuildings(districts, buildings, agents);
    expect(result[0].agentCount).toBe(2);
  });

  it('counts flying agents (toId in district) toward agent count', () => {
    const districts: District[] = [BASE_DISTRICT];
    const buildings: Building[] = [makeBuilding({ id: 'b1', districtId: 'd1' })];
    const agents: Agent[] = [
      {
        id: 'a1', color: 'cyan', mode: 'fly', task: 't', progress: 0.5,
        fromId: 'other', toId: 'b1', flyProgress: 0.5,
      },
    ];
    const result = selectDistrictBuildings(districts, buildings, agents);
    expect(result[0].agentCount).toBe(1);
  });

  it('returns empty array when no districts', () => {
    const result = selectDistrictBuildings([], [], []);
    expect(result).toHaveLength(0);
  });

  it('returns district with loc=0 and fileCount=0 for empty district', () => {
    const result = selectDistrictBuildings([BASE_DISTRICT], [], []);
    expect(result[0].loc).toBe(0);
    expect(result[0].fileCount).toBe(0);
    expect(result[0].agentCount).toBe(0);
  });

  it('handles multiple districts independently', () => {
    const d2: District = { ...BASE_DISTRICT, id: 'd2', label: 'pkg/db', gx: 20, gy: 0, gw: 10, gh: 8 };
    const buildings: Building[] = [
      makeBuilding({ id: 'b1', districtId: 'd1', loc: 100 }),
      makeBuilding({ id: 'b2', districtId: 'd2', loc: 500 }),
    ];
    const result = selectDistrictBuildings([BASE_DISTRICT, d2], buildings, []);
    const r1 = result.find((r) => r.id === 'd1')!;
    const r2 = result.find((r) => r.id === 'd2')!;
    expect(r1.loc).toBe(100);
    expect(r2.loc).toBe(500);
  });
});
