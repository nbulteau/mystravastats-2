package application

import (
	"math"
	"mystravastats/internal/shared/domain/business"
	"mystravastats/internal/shared/domain/strava"
	"sort"
	"time"
)

// Reader uses the same exclusion-aware activity adapter as athlete statistics.
type Reader interface {
	FindActivitiesByYearAndTypes(*int, ...business.ActivityType) []*strava.Activity
	FindPerformanceSettings() business.AthletePerformanceSettings
}
type UseCase struct{ reader Reader }

func NewUseCase(reader Reader) *UseCase { return &UseCase{reader: reader} }
func (u *UseCase) Execute(week time.Time, types []business.ActivityType) Report {
	return BuildReport(week, u.reader.FindActivitiesByYearAndTypes(nil, types...), u.reader.FindPerformanceSettings())
}

type LoadActivity struct {
	ActivityID           int64    `json:"activityId"`
	Name                 string   `json:"name"`
	Date                 string   `json:"date"`
	Sport                string   `json:"sport"`
	Source               string   `json:"source"`
	Reason               string   `json:"reason"`
	Load                 *float64 `json:"load"`
	FTP                  *int     `json:"ftp"`
	FTPEffectiveFrom     *string  `json:"ftpEffectiveFrom"`
	NormalizedPower      *float64 `json:"normalizedPower"`
	IntensityFactor      *float64 `json:"intensityFactor"`
	Best5MinutePower     *float64 `json:"best5MinutePower"`
	MovingSeconds        int      `json:"movingSeconds"`
	ElapsedSeconds       int      `json:"elapsedSeconds"`
	RecordedSeconds      float64  `json:"recordedSeconds"`
	CoveredSeconds       float64  `json:"coveredSeconds"`
	DistanceMeters       float64  `json:"distanceMeters"`
	ElevationMeters      float64  `json:"elevationMeters"`
	AerobicSeconds       float64  `json:"aerobicSeconds"`
	ThresholdSeconds     float64  `json:"thresholdSeconds"`
	HighIntensitySeconds float64  `json:"highIntensitySeconds"`
}
type LoadPeriod struct {
	StartDate             string   `json:"startDate"`
	EndDate               string   `json:"endDate"`
	ActivityCount         int      `json:"activityCount"`
	ScoredCount           int      `json:"scoredCount"`
	MeasuredLoad          *float64 `json:"measuredLoad"`
	EstimatedLoad         *float64 `json:"estimatedLoad"`
	MovingSeconds         int      `json:"movingSeconds"`
	DistanceMeters        float64  `json:"distanceMeters"`
	ElevationMeters       float64  `json:"elevationMeters"`
	CoveredSeconds        float64  `json:"coveredSeconds"`
	RecordedSeconds       float64  `json:"recordedSeconds"`
	AerobicSeconds        float64  `json:"aerobicSeconds"`
	ThresholdSeconds      float64  `json:"thresholdSeconds"`
	HighIntensitySeconds  float64  `json:"highIntensitySeconds"`
	Best5MinutePower      *float64 `json:"best5MinutePower"`
	Best5MinuteActivityID *int64   `json:"best5MinuteActivityId"`
}
type LoadWeek struct {
	Summary LoadPeriod   `json:"summary"`
	Days    []LoadPeriod `json:"days"`
}
type Report struct {
	Method            string         `json:"method"`
	MaxGapSeconds     int            `json:"maxGapSeconds"`
	UndatedActivities int            `json:"undatedActivities"`
	Weeks             []LoadWeek     `json:"weeks"`
	Activities        []LoadActivity `json:"activities"`
}

func Monday(date time.Time) time.Time { return date.AddDate(0, 0, -(int(date.Weekday())+6)%7) }
func localDate(activity *strava.Activity) (time.Time, bool) {
	raw := activity.StartDateLocal
	if raw == "" {
		raw = activity.StartDate
	}
	if len(raw) < 10 {
		return time.Time{}, false
	}
	d, err := time.Parse("2006-01-02", raw[:10])
	return d, err == nil
}
func clean(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
		return 0
	}
	return value
}
func BuildReport(date time.Time, activities []*strava.Activity, settings business.AthletePerformanceSettings) Report {
	start := Monday(date)
	from := start.AddDate(0, 0, -28)
	until := start.AddDate(0, 0, 7)
	report := Report{Method: "power-duration-v1", MaxGapSeconds: strava.MaxPowerGapSeconds, Weeks: []LoadWeek{}, Activities: []LoadActivity{}}
	for _, activity := range activities {
		if activity == nil {
			continue
		}
		day, ok := localDate(activity)
		if !ok {
			report.UndatedActivities++
			continue
		}
		if day.Before(from) || !day.Before(until) {
			continue
		}
		report.Activities = append(report.Activities, calculateActivity(activity, day.Format("2006-01-02"), settings))
	}
	sort.SliceStable(report.Activities, func(i, j int) bool {
		a, b := report.Activities[i], report.Activities[j]
		if a.Date == b.Date {
			return a.ActivityID > b.ActivityID
		}
		return a.Date > b.Date
	})
	for w := 0; w < 5; w++ {
		weekStart := start.AddDate(0, 0, -w*7)
		week := LoadWeek{Summary: period(weekStart, weekStart.AddDate(0, 0, 6), report.Activities), Days: []LoadPeriod{}}
		for d := 0; d < 7; d++ {
			day := weekStart.AddDate(0, 0, d)
			week.Days = append(week.Days, period(day, day, report.Activities))
		}
		report.Weeks = append(report.Weeks, week)
	}
	return report
}
func calculateActivity(a *strava.Activity, date string, settings business.AthletePerformanceSettings) LoadActivity {
	row := LoadActivity{ActivityID: a.Id, Name: a.Name, Date: date, Sport: a.Type, Source: "estimated", MovingSeconds: max(0, a.MovingTime), ElapsedSeconds: max(0, a.ElapsedTime), DistanceMeters: clean(a.Distance), ElevationMeters: clean(a.TotalElevationGain)}
	if a.DeviceWatts {
		row.Source = "measured"
	}
	for _, entry := range settings.FtpHistory {
		_, err := time.Parse("2006-01-02", entry.EffectiveFrom)
		if err == nil && entry.Ftp > 0 && entry.EffectiveFrom <= date && (row.FTPEffectiveFrom == nil || entry.EffectiveFrom >= *row.FTPEffectiveFrom) {
			value, day := entry.Ftp, entry.EffectiveFrom
			row.FTP = &value
			row.FTPEffectiveFrom = &day
		}
	}
	switch a.Type {
	case "Ride", "VirtualRide", "MountainBikeRide", "GravelRide", "Commute":
	default:
		row.Reason = "unsupported-sport"
		return row
	}
	if a.Stream == nil || a.Stream.Watts == nil || len(a.Stream.Watts.Data) == 0 {
		row.Source = "unavailable"
		row.Reason = "missing-power"
		return row
	}
	timeline := strava.NewPowerTimeline(a.Stream.Watts.Data, a.Stream.Time.Data)
	if !timeline.Valid {
		row.Reason = "invalid-time"
		return row
	}
	times := timeline.Times
	row.RecordedSeconds = float64(times[len(times)-1] - times[0])
	for i := 0; i < len(times)-1; i++ {
		if timeline.IntervalValid(i) {
			seconds := float64(times[i+1] - times[i])
			row.CoveredSeconds += seconds
			if row.FTP != nil {
				power, ftp := timeline.Watts[i], float64(*row.FTP)
				if power <= ftp*0.9 {
					row.AerobicSeconds += seconds
				} else if power <= ftp*1.2 {
					row.ThresholdSeconds += seconds
				} else {
					row.HighIntensitySeconds += seconds
				}
			}
		}
	}
	if best := timeline.Best(300); best != nil {
		row.Best5MinutePower = &best.Average
	}
	row.NormalizedPower = timeline.Normalized()
	switch {
	case row.CoveredSeconds < row.RecordedSeconds:
		row.Reason = "power-gaps"
	case row.RecordedSeconds < 30:
		row.Reason = "too-short"
	case times[0] != 0 || a.ElapsedTime <= 0 || math.Abs(float64(times[len(times)-1]-a.ElapsedTime)) > 1:
		row.Reason = "incomplete-activity"
	case row.FTP == nil:
		row.Reason = "missing-dated-ftp"
	case row.NormalizedPower == nil:
		row.Reason = "missing-power"
	default:
		intensity := *row.NormalizedPower / float64(*row.FTP)
		load := row.CoveredSeconds / 3600 * intensity * intensity * 100
		row.IntensityFactor = &intensity
		row.Load = &load
		row.Reason = "available"
	}
	return row
}
func period(start, end time.Time, rows []LoadActivity) LoadPeriod {
	p := LoadPeriod{StartDate: start.Format("2006-01-02"), EndDate: end.Format("2006-01-02")}
	add := func(target **float64, value float64) {
		if *target == nil {
			v := 0.0
			*target = &v
		}
		**target += value
	}
	for _, row := range rows {
		if row.Date < p.StartDate || row.Date > p.EndDate {
			continue
		}
		p.ActivityCount++
		p.MovingSeconds += row.MovingSeconds
		p.DistanceMeters += row.DistanceMeters
		p.ElevationMeters += row.ElevationMeters
		p.CoveredSeconds += row.CoveredSeconds
		p.RecordedSeconds += row.RecordedSeconds
		if row.Load != nil {
			p.ScoredCount++
			if row.Source == "measured" {
				add(&p.MeasuredLoad, *row.Load)
			} else {
				add(&p.EstimatedLoad, *row.Load)
			}
		}
		if row.Source == "measured" {
			p.AerobicSeconds += row.AerobicSeconds
			p.ThresholdSeconds += row.ThresholdSeconds
			p.HighIntensitySeconds += row.HighIntensitySeconds
			if row.Best5MinutePower != nil && (p.Best5MinutePower == nil || *row.Best5MinutePower > *p.Best5MinutePower) {
				v, id := *row.Best5MinutePower, row.ActivityID
				p.Best5MinutePower = &v
				p.Best5MinuteActivityID = &id
			}
		}
	}
	return p
}
