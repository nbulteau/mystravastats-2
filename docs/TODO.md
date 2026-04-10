# Product Backlog

## P0 - High Impact

### 1. Activity Comparison
- Compare two activities side by side.
- Compare distance, elevation, moving time, average speed, heart rate, cadence, watts, and best efforts.
- Highlight improvements and regressions compared with the athlete's personal baseline.
- Useful for race analysis, climb attempts, and training follow-up.

### 2. Goal Tracking
- Let the athlete define yearly, monthly, or weekly goals.
- Examples: total distance, elevation, active days, ride count, running streak, climbing target.
- Show progress bars and estimated completion pace.
- Add goal history to see whether a goal was achieved or missed.

### 3. Training Load Dashboard
- Introduce a simple workload view based on distance, elevation, duration, and intensity signals.
- Provide short-term vs long-term load trends.
- Surface fatigue, form, and consistency indicators.
- Add alerts for overload or undertraining periods.

## P1 - Strong Product Extensions

### 4. Heart Rate Zone Analysis
- Show time spent in each heart rate zone per activity, month, and year.
- Support custom zones based on max HR, threshold HR, or reserve HR.
- Add charts for zone distribution and trends over time.
- Surface "easy vs hard" training balance.

### 5. Power and Efficiency Metrics
- Add normalized power, intensity factor, variability index, and power-to-weight ratio when data is available.
- Add cycling efficiency views combining heart rate, power, speed, and elevation.
- Show best power curve by duration.
- Support missing data gracefully when the athlete has no power meter.

### 6. Route Library and Route Reuse
- Detect recurring routes and group similar activities together.
- Show route frequency, best attempt, average attempt, and last attempt.
- Add route difficulty indicators and favorite route badges.
- Make it possible to start from a route page to analyze all attempts.

### 7. Segment and Climb Progression
- Create dedicated pages for favorite climbs and segments.
- Show all attempts, PR progression, consistency, pacing, and weather context if available later.
- Highlight "close to PR" efforts and best recent trends.

### 8. Multi-Sport Dashboard
- Build a single dashboard mixing ride, run, hike, ski, and skate activity families.
- Show distribution of training time across sports.
- Make transitions between sports visible over the year.

## P2 - Discovery and Engagement

### 9. Calendar Heatmap
- Add a GitHub-style training calendar.
- Support overlays for distance, elevation, duration, and intensity.
- Make it clickable to drill down into the activities of a given day.

### 10. Milestones and Badges 2.0
- Expand the badge system with streaks, seasonal challenges, route milestones, and PR milestones.
- Add "almost unlocked" badges for motivation.
- Make badge progress visible instead of binary.

### 11. Weather and Conditions Overlay
- Enrich activities with temperature, wind, and rain context when the data source is available.
- Show how conditions affect speed, power, or heart rate.
- Enable questions such as "best climbs in hot weather" or "pace drop in windy runs".

### 12. Nutrition and Fueling Notes
- Allow athletes to attach fueling notes to long activities.
- Correlate fueling strategy with performance and fatigue.
- Especially useful for endurance rides, races, and long runs.

## P3 - Quality of Life

### 13. Saved Filters and Custom Views
- Let users save common filters such as "All gravel rides in 2025" or "Long runs over 20 km".
- Support shareable URLs and named presets.
- Reduce friction for repeated analysis.

### 14. Export Center
- Centralize CSV and future export options in one place.
- Add JSON export and printable summary reports.
- Support exporting a selection of activities, not only one current filter.

### 15. Explain This Metric
- Add inline help for advanced metrics and formulas.
- Explain how each statistic is computed and when it is meaningful.
- Reduce the learning curve for non-expert users.

### 16. Mobile-Friendly Detail Experience
- Improve activity detail pages for smaller screens.
- Focus on quick scrolling between overview, charts, map, efforts, and segments.

## Technical Enablers for Future Features

### A. Background Data Refresh
- Refresh new activities and missing streams asynchronously.
- Reduce startup cost and improve perceived responsiveness.

### B. Feature Flag System
- Allow shipping experimental analytics progressively.
- Useful for metrics that need validation before becoming default.

### C. Metrics Instrumentation
- Measure endpoint latency, cache hit rate, and expensive statistic computations.
- Make performance work visible and trackable over time.
