const exactMetricTooltips: Record<string, string> = {
  "Max HR":
    "Maximum heart rate in bpm. If provided, it is used directly for MAX method and as upper bound context for other methods.",
  "Threshold HR":
    "Threshold heart rate in bpm (roughly effort sustainable for about 40 to 60 minutes). If set, it has the highest priority for zone resolution.",
  "Reserve HR":
    "Heart rate reserve in bpm (max HR minus resting HR). Reserve method is used when Threshold HR is empty and Max HR + Reserve HR are valid.",
  "HR Zone Method":
    "How zones are resolved. Priority: Threshold HR, then Reserve HR (requires Max HR), then Max HR. If none is set, Max HR is derived from activities.",
  "Resolved Max HR":
    "The effective Max HR currently used by the zone engine. It can come from your settings or be derived automatically from activity data.",
  "HR Zone Source":
    "Source of the resolved zone settings: ATHLETE_SETTINGS (saved values) or DERIVED_FROM_DATA (computed from recorded activities).",
  "Heart Rate Zone Storage":
    "Zone settings are persisted per athlete in the local cache and reused on next startup.",
  "Tracked HR Time":
    "Total time where heart rate stream points are available. Missing stream sections are excluded.",
  "Easy / Hard Ratio":
    "Easy time is Z1 + Z2. Hard time is Z4 + Z5. Ratio = easy time / hard time. Zones come from your resolved HR settings: Threshold HR if set, otherwise Heart Rate Reserve, otherwise Max HR; if no setting exists, Max HR is derived from activity data.",
  "Easy/hard ratio":
    "Easy time is Z1 + Z2. Hard time is Z4 + Z5. Ratio = easy time / hard time. Zones come from your resolved HR settings: Threshold HR if set, otherwise Heart Rate Reserve, otherwise Max HR; if no setting exists, Max HR is derived from activity data.",
  "HR Data Availability":
    "Available when at least one activity in the current filters contains heart rate stream data.",
  "Average Speed":
    "Average speed over the activity duration or selected period.",
  "Max Speed":
    "Highest instantaneous speed recorded during the activity or selected period.",
  "Average Cadence":
    "For cycling: pedal revolutions per minute (rpm). For running: steps per minute (spm).",
  "Average Watts":
    "Arithmetic mean of power values across the activity.",
  "Average power":
    "Arithmetic mean of recorded power values. It is useful for total workload, but it can understate variable rides with descents or coasting.",
  "Average W/kg":
    "Average power divided by athlete weight. Uses manual weight when available. Useful for comparing effort relative to body mass, but less stable than FTP/kg.",
  "Max power":
    "Highest instantaneous recorded power value in the activity.",
  "Max avg power (20 min)":
    "Best rolling 20-minute average power in this activity. Often used as a rough FTP estimation input.",
  "Weighted Average Watts":
    "Intensity-weighted power estimate. It reflects variable efforts better than plain average watts.",
  "Weighted avg power":
    "Intensity-weighted power estimate provided by Strava. It reflects variable efforts better than plain average power.",
  "Normalized Power (NP)":
    "Intensity-weighted power estimate computed from a 30-second rolling average and fourth-power weighting. It is closer to physiological cost than average power.",
  "Intensity Factor (IF)":
    "Normalized Power divided by FTP. Around 1.00 means riding at FTP-equivalent intensity; values above 1.00 are only sustainable for shorter efforts.",
  "Training Stress Score (TSS)":
    "Training load estimate based on duration, Normalized Power, IF, and FTP. About 100 is roughly one hour at FTP.",
  "FTP setting":
    "Functional Threshold Power used for local calculations. Manual dated settings are preferred over Strava profile values and estimates.",
  "FTP effective date":
    "Date from which a manual FTP applies. Activities on or after this date use that FTP until a newer dated value exists.",
  "FTP priority":
    "FTP resolution order for activity analysis: manual dated FTP for the activity date, then Strava profile FTP, then estimate from the power stream.",
  "Estimated FTP":
    "Fallback FTP estimated from the activity power curve when no manual or Strava FTP is available.",
  "FTP / kg":
    "FTP divided by athlete weight. This is usually a better level indicator than Average W/kg from one ride.",
  "Aerobic power-zone time":
    "Estimated time at or below 90% of FTP. This mostly covers endurance and tempo work, but includes easy/coasting samples.",
  "Threshold / VO2 time":
    "Estimated time between 90% and 120% of FTP. This covers threshold and VO2max-like intensity, depending on duration.",
  "Anaerobic exposure":
    "Estimated time above 120% of FTP. This is a practical anaerobic-intensity signal, not a true W' or lactate-based measurement.",
  Kilojoules:
    "Mechanical work from power over time. Useful to estimate training load and fueling demand.",
  Work:
    "Mechanical work from power over time, expressed in kilojoules. It is not the same as food calories, but often tracks fueling demand.",
  "Average Heartrate":
    "Average heart rate in beats per minute (bpm).",
  "Max Heartrate":
    "Highest heart rate value reached in beats per minute (bpm).",
  "Tracked HR time":
    "Total time where heart rate points are available in the activity stream.",
  "Nb activities": "Number of recorded activities in the selected filters.",
  "Nb actives days":
    "Number of distinct calendar days with at least one activity in the selected filters.",
  "Max streak":
    "Longest consecutive sequence of active days in the selected period.",
  "Total distance":
    "Sum of all activity distances in the selected filters.",
  "Elapsed time":
    "Total elapsed time including pauses and stoppages.",
  "Total elevation":
    "Sum of positive elevation gain across selected activities.",
  "Km by activity":
    "Average distance per activity.",
  "Max distance":
    "Longest single activity distance.",
  "Max distance in a day":
    "Highest accumulated distance across activities done on the same calendar day.",
  "Max elevation": "Highest elevation gain achieved in one activity.",
  "Max elevation gain in a day":
    "Highest accumulated elevation gain across activities done on the same day.",
  "Highest point": "Highest altitude point reached.",
  "Max moving time":
    "Longest moving time in a single activity, excluding paused duration.",
  "Moving time by year":
    "Total moving time aggregated per year (in seconds in API, rendered as hours in charts).",
  "Elevation efficiency":
    "How much climbing you get for distance: total elevation gain divided by total distance, scaled to m per 10 km.",
  "Most active month":
    "Month with the highest number of activities in the selected period.",
  "Eddington number":
    "Largest number E such that at least E days or activities reach the selected threshold. Switch distance/elevation, days/activities, and the time scope.",
  "Heatmap Advanced Insights":
    "Advanced interpretation layer on top of daily heatmap data: consistency, streaks, weekly momentum, weekday signature, top days, activity mix, and best week.",
  "Heatmap Consistency":
    "Share of days with at least one activity in the considered period: active days / days in scope.",
  "Heatmap Longest Streak":
    "Longest run of consecutive active days.",
  "Heatmap Longest Break":
    "Longest run of consecutive inactive days in the same period.",
  "Heatmap Average Active Day":
    "Average selected metric value computed only across active days.",
  "Heatmap Weekly Momentum":
    "Difference between the recent weekly average and the previous weekly average for the selected metric.",
  "Heatmap Weekday Signature":
    "Distribution of the selected metric by weekday, useful to identify recurring training patterns.",
  "Heatmap Top Days":
    "Top individual calendar days ranked by the selected metric value.",
  "Heatmap Activity Mix":
    "Breakdown of sports recorded in the selected period.",
  "Heatmap Best Week":
    "Best calendar week for the selected metric, with active days and activity count.",
  YTD:
    "Year-To-Date: from January 1st of the selected year up to today. For past years, full-year values are used.",
  "YTD average":
    "Average computed on Year-To-Date data (from January 1st to today for current year, full year for past years).",
  "Charts granularity":
    "Monthly view groups values by month. Weekly view groups values by ISO week to reveal short-term variations.",
  "Charts refresh":
    "Reload chart data for current filters without changing year, sport filter, or current view.",
  "Weekly training load (TRIMP)":
    "Simplified training load: weekly sum of heart-rate zone time weighted by zone intensity (Z1..Z5). Useful to monitor fatigue trends.",
  "Distance distribution":
    "Histogram of activity distances. Helps identify whether your training is mostly short, medium, or long rides/runs.",
  "Long ride progression":
    "Weekly longest outing distance, with a 4-week moving average to visualize endurance progression.",
  "Easy / Hard ratio by month":
    "Monthly balance between easy time (Z1+Z2) and hard time (Z4+Z5), plus ratio trend.",
  "Weekly consistency":
    "Share of active weeks in the selected year (active weeks / total ISO weeks).",
};

const patternMetricTooltips: Array<{ pattern: RegExp; tooltip: string }> = [
  {
    pattern: /^Best /i,
    tooltip:
      "Best effort metric for a target distance or duration, computed from activity streams.",
  },
  {
    pattern: /^Max gradient for /i,
    tooltip:
      "Steepest average gradient reached over the specified distance window in one effort.",
  },
  {
    pattern: /^Average power$/i,
    tooltip: "Highest activity average power value (watts).",
  },
  {
    pattern: /^Weighted average power$/i,
    tooltip: "Highest activity weighted average power value (watts).",
  },
];

export function getMetricTooltip(label?: string | null): string | null {
  if (!label) return null;
  const direct = exactMetricTooltips[label];
  if (direct) return direct;

  for (const entry of patternMetricTooltips) {
    if (entry.pattern.test(label)) return entry.tooltip;
  }

  return null;
}
