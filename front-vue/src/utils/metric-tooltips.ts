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
    "Easy time is Zone 1 plus Zone 2. Hard time is Zone 4 plus Zone 5. Displayed as Easy : Hard.",
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
  "Weighted Average Watts":
    "Intensity-weighted power estimate. It reflects variable efforts better than plain average watts.",
  Kilojoules:
    "Mechanical work from power over time. Useful to estimate training load and fueling demand.",
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
  "Most active month":
    "Month with the highest number of activities in the selected period.",
  "Eddington number":
    "Largest number E such that you have at least E days with distance greater than or equal to E.",
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
