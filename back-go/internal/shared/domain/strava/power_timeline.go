package strava

import (
	"math"
	"sort"
)

// MaxPowerGapSeconds is an explicit coverage policy, not inferred from sample cadence.
const MaxPowerGapSeconds = 10

// PowerTimeline integrates left-held samples on [time[i], time[i+1]). Both
// endpoints must be valid; the last sample has no inferred duration.
type PowerTimeline struct {
	Times           []int
	Watts           []float64
	energy, covered []float64
	Valid           bool
}
type PowerWindow struct {
	Start, End, Average  float64
	StartIndex, EndIndex int
}

func NewPowerTimeline(watts []float64, times []int) *PowerTimeline {
	p := &PowerTimeline{Times: times, Watts: watts, energy: make([]float64, len(times)), covered: make([]float64, len(times)), Valid: len(times) >= 2}
	for i, t := range times {
		if t < 0 || (i > 0 && t <= times[i-1]) {
			p.Valid = false
		}
	}
	if !p.Valid {
		return p
	}
	for i := 0; i < len(times)-1; i++ {
		p.energy[i+1] = p.energy[i]
		p.covered[i+1] = p.covered[i]
		if p.IntervalValid(i) {
			dt := float64(times[i+1] - times[i])
			p.energy[i+1] += watts[i] * dt
			p.covered[i+1] += dt
		}
	}
	return p
}
func (p *PowerTimeline) IntervalValid(i int) bool {
	if !p.Valid || i < 0 || i+1 >= len(p.Times) || i+1 >= len(p.Watts) || p.Times[i+1]-p.Times[i] > MaxPowerGapSeconds {
		return false
	}
	for _, v := range p.Watts[i : i+2] {
		if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
			return false
		}
	}
	return true
}
func (p *PowerTimeline) integral(t float64) (float64, float64) {
	i := sort.Search(len(p.Times), func(i int) bool { return float64(p.Times[i]) > t }) - 1
	e, c := p.energy[i], p.covered[i]
	if p.IntervalValid(i) {
		dt := t - float64(p.Times[i])
		e += p.Watts[i] * dt
		c += dt
	}
	return e, c
}
func (p *PowerTimeline) Average(start, end float64) *float64 {
	if !p.Valid || math.IsNaN(start) || math.IsNaN(end) || start < float64(p.Times[0]) || end > float64(p.Times[len(p.Times)-1]) || end <= start {
		return nil
	}
	a, c := p.integral(start)
	b, d := p.integral(end)
	if math.Abs((d-c)-(end-start)) > 1e-7 {
		return nil
	}
	avg := (b - a) / (end - start)
	return &avg
}
func (p *PowerTimeline) AverageIndices(start, end int) *float64 {
	if start < 0 || end >= len(p.Times) || end <= start {
		return nil
	}
	return p.Average(float64(p.Times[start]), float64(p.Times[end]))
}
func (p *PowerTimeline) Best(seconds int) *PowerWindow {
	if !p.Valid || seconds <= 0 {
		return nil
	}
	var best *PowerWindow
	// An integral of piecewise-constant power changes slope only when either
	// boundary crosses a sample. Evaluate both sets, including partial intervals.
	for _, t := range p.Times {
		for _, start := range []float64{float64(t), float64(t - seconds)} {
			end := start + float64(seconds)
			avg := p.Average(start, end)
			if avg != nil && (best == nil || *avg > best.Average || (*avg == best.Average && start < best.Start)) {
				best = &PowerWindow{Start: start, End: end, Average: *avg, StartIndex: sort.Search(len(p.Times), func(i int) bool { return float64(p.Times[i]) > start }) - 1, EndIndex: sort.Search(len(p.Times), func(i int) bool { return float64(p.Times[i]) >= end })}
			}
		}
	}
	return best
}

// ValueAt linearly interpolates geometry only, never power or sensor gaps.
func (p *PowerTimeline) ValueAt(values []float64, t float64) float64 {
	i := sort.Search(len(p.Times), func(i int) bool { return float64(p.Times[i]) >= t })
	if i >= len(values) || i >= len(p.Times) {
		return 0
	}
	if i == 0 || float64(p.Times[i]) == t {
		return values[i]
	}
	f := (t - float64(p.Times[i-1])) / float64(p.Times[i]-p.Times[i-1])
	return values[i-1] + f*(values[i]-values[i-1])
}

// Normalized integrates the fourth power of the 30-second rolling mean over time.
// It is unavailable unless the whole recorded span is covered and lasts >=30 s.
func (p *PowerTimeline) Normalized() *float64 {
	if !p.Valid {
		return nil
	}
	start, end := float64(p.Times[0]+30), float64(p.Times[len(p.Times)-1])
	if start > end || p.Average(float64(p.Times[0]), end) == nil {
		return nil
	}
	if start == end {
		return p.Average(start-30, start)
	}
	points := []float64{start, end}
	for _, t := range p.Times {
		for _, v := range []float64{float64(t), float64(t) + 30} {
			if v > start && v < end {
				points = append(points, v)
			}
		}
	}
	sort.Float64s(points)
	integral := 0.0
	for i := 1; i < len(points); i++ {
		if points[i] == points[i-1] {
			continue
		}
		a, b := *p.Average(points[i-1]-30, points[i-1]), *p.Average(points[i]-30, points[i])
		fourth := (math.Pow(a, 4) + math.Pow(a, 3)*b + a*a*b*b + a*math.Pow(b, 3) + math.Pow(b, 4)) / 5
		integral += (points[i] - points[i-1]) * fourth
	}
	result := math.Pow(integral/(end-start), 0.25)
	return &result
}
