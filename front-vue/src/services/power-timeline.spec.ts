import { describe, it, expect } from 'vitest';
import cases from '../../../test-fixtures/api/power-timing.json';
import { PowerTimeline } from './power-timeline';
import { buildPowerZoneEstimate } from './activity-power-analysis';
describe('shared time weighted power', () => {
  for (const fixture of cases) it(fixture.name, () => {
    const timeline = new PowerTimeline(fixture.watts, fixture.times);
    const average = timeline.average(fixture.times[0]!, fixture.times.at(-1)!);
    const best = timeline.best(fixture.seconds);
    if (fixture.average === null) expect(average).toBeNull(); else expect(average).toBeCloseTo(fixture.average, 7);
    if (fixture.best === null) expect(best).toBeNull(); else expect(best).toBeCloseTo(fixture.best, 7);
    expect(timeline.coveredSeconds).toBe(fixture.covered);
    if (fixture.normalized !== undefined) expect(timeline.normalized()).toBeCloseTo(fixture.normalized, 7);
  });
  it('zones count time rather than measurements', () => {
    expect(buildPowerZoneEstimate([0, 200, 300, 300], 200, [0,2,7,10])).toEqual({trackedSeconds:10,aerobicSeconds:2,thresholdVo2Seconds:5,anaerobicSeconds:3});
  });
  it('normalized power is invariant under subdivision', () => {
    const sparse = new PowerTimeline([100,100,100,300,300,300,300], [0,10,20,30,40,50,60]);
    const dense = new PowerTimeline(Array.from({length:61},(_,i)=>i<30?100:300),Array.from({length:61},(_,i)=>i));
    expect(sparse.normalized()).toBeCloseTo(dense.normalized()!, 7);
    expect(sparse.normalized()).toBeCloseTo(((300 ** 5 - 100 ** 5) / (5 * (300 - 100))) ** 0.25, 6);
  });
});
