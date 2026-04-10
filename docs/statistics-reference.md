# Statistics Reference

This page explains the main statistics exposed by MyStravaStats and the intuition behind them.

## Global Metrics

### Number of Activities

Definition:
- the total count of activities matching the current filter

### Number of Active Days

Definition:
- the number of distinct days with at least one activity matching the current filter

### Total Distance

Definition:
- the sum of all activity distances

Formula:

```text
Total Distance = sum(activity.distance)
```

### Total Elevation

Definition:
- the sum of all positive elevation gain values across matching activities

Formula:

```text
Total Elevation = sum(activity.totalElevationGain)
```

### Max Streak

Definition:
- the longest run of consecutive active days

## Eddington Number

The Eddington number is a consistency metric.

Definition:
- it is the largest number `E` such that you completed at least `E` different days with at least `E km` on each of those days

How MyStravaStats computes it:
1. group activities by day
2. sum the total distance of each day
3. convert each day total to kilometers
4. search for the largest `E` where at least `E` days reach `E km`

Formula:

```text
E = max value such that count(days where dayDistanceKm >= E) >= E
```

Example:
- 52 days at 52 km or more => Eddington number is at least 52
- 53 days are required to reach 53

Interpretation:
- it grows slowly
- it rewards repeatability
- it is harder to increase than total volume

## Best Efforts By Distance

Definition:
- the strongest continuous effort over a target distance

Examples:
- best 200 m
- best 1 km
- best 10 km

Method:
- MyStravaStats uses a sliding window over the stream data
- it scans the activity to find the segment with the best time for the target distance

## Best Efforts By Time

Definition:
- the strongest continuous effort over a target duration

Examples:
- best 30 min
- best 1 h
- best 2 h

Method:
- sliding window over stream data
- find the segment that covers the greatest distance in the target duration

## Best Average Power

Definition:
- the best power effort over a target duration when power data exists

Examples:
- best average power for 20 min
- best average power for 1 h

Method:
- use watts stream values
- compute the strongest rolling interval for the requested duration

## Best Gradient For Distance

Definition:
- the segment with the highest elevation gain on a target distance

Examples:
- best gradient for 500 m
- best gradient for 1 km
- best gradient for 5 km

Method:
- sliding window over distance and altitude streams
- maximize the altitude gain for the given distance

## Most Active Month

Definition:
- the month containing the highest number of matching activities

## Dashboard Metrics

Yearly dashboard metrics usually include:
- number of activities by year
- total distance by year
- average distance by year
- max distance by year
- total elevation by year
- average elevation by year
- max elevation by year
- average speed by year
- max speed by year
- average heart rate by year
- max heart rate by year
- average watts by year
- max watts by year

## Important Notes

- Some metrics require stream data, so they may not be available for every activity.
- Power metrics require power-enabled recordings.
- Some activity types expose richer metrics than others.
- Detailed results are influenced by Strava activity and stream quality.
