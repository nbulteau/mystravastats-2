// Explicit policy shared with Go/Kotlin. Never infer cadence or extend the last reading.
export const MAX_POWER_GAP_SECONDS = 10;
export class PowerTimeline {
  readonly valid: boolean;
  private readonly energy: number[];
  private readonly covered: number[];
  constructor(readonly watts: Array<number | null>, readonly times: number[]) {
    this.valid = times.length >= 2 && times.every((t, i) => Number.isFinite(t) && t >= 0 && (i === 0 || t > times[i - 1]!));
    this.energy = Array(times.length).fill(0);
    this.covered = Array(times.length).fill(0);
    if (!this.valid) return;
    for (let i = 0; i < times.length - 1; i++) {
      const dt = this.intervalValid(i) ? times[i + 1]! - times[i]! : 0;
      this.energy[i + 1] = this.energy[i]! + (dt ? watts[i]! * dt : 0);
      this.covered[i + 1] = this.covered[i]! + dt;
    }
  }
  intervalValid(i: number): boolean {
    return this.valid && i >= 0 && i + 1 < this.times.length && i + 1 < this.watts.length
      && this.times[i + 1]! - this.times[i]! <= MAX_POWER_GAP_SECONDS
      && [this.watts[i], this.watts[i + 1]].every(v => typeof v === 'number' && Number.isFinite(v) && v >= 0);
  }
  get duration(): number { return this.valid ? this.times.at(-1)! - this.times[0]! : 0; }
  get coveredSeconds(): number { return this.covered.at(-1) ?? 0; }
  private upper(t: number): number {
    let lo = 0, hi = this.times.length;
    while (lo < hi) { const mid = Math.floor((lo + hi) / 2); if (this.times[mid]! <= t) lo = mid + 1; else hi = mid; }
    return lo;
  }
  private integral(t: number): [number, number] {
    const i = this.upper(t) - 1;
    const dt = this.intervalValid(i) ? t - this.times[i]! : 0;
    return [this.energy[i]! + (dt ? this.watts[i]! * dt : 0), this.covered[i]! + dt];
  }
  average(start: number, end: number): number | null {
    if (!this.valid || !Number.isFinite(start) || !Number.isFinite(end) || start < this.times[0]! || end > this.times.at(-1)! || end <= start) return null;
    const [a, c] = this.integral(start), [b, d] = this.integral(end);
    return Math.abs(d - c - (end - start)) > 1e-7 ? null : (b - a) / (end - start);
  }
  best(seconds: number): number | null {
    if (!this.valid || !Number.isFinite(seconds) || seconds <= 0) return null;
    let best: number | null = null;
    for (const t of this.times) for (const start of [t, t - seconds]) {
      const avg = this.average(start, start + seconds);
      if (avg !== null && (best === null || avg > best)) best = avg;
    }
    return best;
  }
  // Exact time integral of the fourth power of the continuous 30 s rolling mean.
  // Between breakpoints this mean is linear; the five-term formula integrates its fourth power.
  normalized(): number | null {
    if (this.duration < 30 || this.coveredSeconds !== this.duration) return null;
    const start = this.times[0]! + 30, end = this.times.at(-1)!;
    if (start === end) return this.average(start - 30, start);
    const points = [...new Set([start, end, ...this.times.flatMap(t => [t, t + 30]).filter(t => t > start && t < end)])].sort((a,b) => a-b);
    let integral = 0;
    for (let i = 1; i < points.length; i++) {
      const a = this.average(points[i-1]! - 30, points[i-1]!)!, b = this.average(points[i]! - 30, points[i]!)!;
      const meanFourth = (a**4 + a**3*b + a*a*b*b + a*b**3 + b**4) / 5;
      integral += (points[i]! - points[i-1]!) * meanFourth;
    }
    return (integral / (end - start)) ** 0.25;
  }
}
