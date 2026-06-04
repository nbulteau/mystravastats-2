export type CumulativeMetric = "distance" | "elevation";
export type CumulativeComparisonMode = "all-years" | "best-year" | "previous-year";
export type CumulativeSeriesRole = "history" | "current" | "projection";

export type CumulativeChartSeries = {
  name: string;
  year: string;
  role: CumulativeSeriesRole;
  data: Array<number | null>;
};

export type CumulativeChartData = {
  categories: string[];
  comparisonMode: CumulativeComparisonMode;
  currentYear: string;
  currentValue: number | null;
  comparisonValue: number | null;
  comparisonYear: string | null;
  hasData: boolean;
  metric: CumulativeMetric;
  projectedValue: number | null;
  series: CumulativeChartSeries[];
  summary: string;
  title: string;
  todayIndex: number;
  todayKey: string;
  unit: "km" | "m";
  yAxisTitle: string;
};

export type CumulativeChartInput = {
  comparisonMode?: CumulativeComparisonMode | null;
  distancePerYear: Map<string, Map<string, number>>;
  elevationPerYear: Map<string, Map<string, number>>;
  metric?: CumulativeMetric | null;
  now?: Date;
};

const MONTH_NAMES = [
  "January",
  "February",
  "March",
  "April",
  "May",
  "June",
  "July",
  "August",
  "September",
  "October",
  "November",
  "December",
];

function normalizeMetric(metric?: CumulativeMetric | null): CumulativeMetric {
  return metric === "elevation" ? "elevation" : "distance";
}

function normalizeComparisonMode(mode?: CumulativeComparisonMode | null): CumulativeComparisonMode {
  if (mode === "best-year" || mode === "previous-year") {
    return mode;
  }
  return "all-years";
}

function formatMonthDay(date: Date): string {
  return `${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;
}

function createYearCategories(year: number): string[] {
  const categories: string[] = [];
  const cursor = new Date(year, 0, 1, 12, 0, 0, 0);
  while (cursor.getFullYear() === year) {
    categories.push(formatMonthDay(cursor));
    cursor.setDate(cursor.getDate() + 1);
  }
  return categories;
}

function sortedYearKeys(data: Map<string, Map<string, number>>): string[] {
  return Array.from(data.keys())
    .filter((year) => Number.isFinite(Number.parseInt(year, 10)))
    .sort((left, right) => Number.parseInt(left, 10) - Number.parseInt(right, 10));
}

function normalizedNumber(value: number | undefined): number {
  const parsed = Number(value ?? 0);
  return Number.isFinite(parsed) ? parsed : 0;
}

function sortedDayKeys(data: Map<string, number>): string[] {
  return Array.from(data.keys()).sort();
}

function valueAtDay(data: Map<string, number> | undefined, dayKey: string): number {
  if (!data) {
    return 0;
  }
  const exactValue = data.get(dayKey);
  if (exactValue !== undefined) {
    return normalizedNumber(exactValue);
  }

  const fallbackKey = sortedDayKeys(data)
    .filter((key) => key <= dayKey)
    .pop();
  return normalizedNumber(fallbackKey ? data.get(fallbackKey) : 0);
}

function finalValue(data: Map<string, number> | undefined): number {
  if (!data || data.size === 0) {
    return 0;
  }
  const finalKey = sortedDayKeys(data).pop();
  return normalizedNumber(finalKey ? data.get(finalKey) : 0);
}

function bestHistoricalYear(years: string[], data: Map<string, Map<string, number>>, currentYear: string): string | null {
  const historicalYears = years.filter((year) => year !== currentYear);
  if (historicalYears.length === 0) {
    return null;
  }

  return historicalYears.reduce((bestYear, year) => {
    const bestValue = finalValue(data.get(bestYear));
    const yearValue = finalValue(data.get(year));
    if (yearValue > bestValue) {
      return year;
    }
    return bestYear;
  });
}

function previousAvailableYear(years: string[], currentYear: string): string | null {
  const currentYearNumber = Number.parseInt(currentYear, 10);
  const previousYears = years
    .filter((year) => Number.parseInt(year, 10) < currentYearNumber)
    .sort((left, right) => Number.parseInt(right, 10) - Number.parseInt(left, 10));
  return previousYears[0] ?? null;
}

function selectedComparisonYear(
  years: string[],
  data: Map<string, Map<string, number>>,
  currentYear: string,
  mode: CumulativeComparisonMode,
): string | null {
  if (mode === "best-year") {
    return bestHistoricalYear(years, data, currentYear);
  }
  return previousAvailableYear(years, currentYear);
}

function historicalYearsForMode(
  years: string[],
  data: Map<string, Map<string, number>>,
  currentYear: string,
  mode: CumulativeComparisonMode,
): string[] {
  if (mode === "all-years") {
    return years.filter((year) => year !== currentYear);
  }

  const comparisonYear = selectedComparisonYear(years, data, currentYear, mode);
  return comparisonYear ? [comparisonYear] : [];
}

function formatValue(value: number, unit: "km" | "m"): string {
  return `${Math.round(value).toLocaleString()} ${unit}`;
}

function formatSignedDelta(delta: number, unit: "km" | "m"): string {
  if (Math.abs(delta) < 0.5) {
    return `0 ${unit}`;
  }
  const sign = delta > 0 ? "+" : "-";
  return `${sign}${Math.round(Math.abs(delta)).toLocaleString()} ${unit}`;
}

function buildSummary(
  currentValue: number | null,
  comparisonValue: number | null,
  comparisonYear: string | null,
  projectedValue: number | null,
  currentYear: string,
  unit: "km" | "m",
): string {
  if (currentValue === null) {
    return `No ${currentYear} cumulative data yet.`;
  }

  const parts = [`Today: ${formatValue(currentValue, unit)}`];
  if (comparisonYear !== null && comparisonValue !== null) {
    parts.push(`${formatSignedDelta(currentValue - comparisonValue, unit)} vs ${comparisonYear}`);
  }
  if (projectedValue !== null) {
    parts.push(`projected: ${formatValue(projectedValue, unit)}`);
  }
  return parts.join(" - ");
}

function buildProjection(
  categories: string[],
  todayIndex: number,
  currentValue: number,
): { data: Array<number | null>; projectedValue: number | null } {
  if (todayIndex < 0 || todayIndex >= categories.length - 1) {
    return { data: [], projectedValue: null };
  }

  const daysElapsed = todayIndex + 1;
  const projectedValue = currentValue * (categories.length / daysElapsed);
  const remainingDays = categories.length - 1 - todayIndex;
  const data = categories.map((_, index) => {
    if (index < todayIndex) {
      return null;
    }
    const progress = remainingDays === 0 ? 1 : (index - todayIndex) / remainingDays;
    return currentValue + ((projectedValue - currentValue) * progress);
  });

  return { data, projectedValue };
}

function buildHistoricalSeries(
  year: string,
  yearData: Map<string, number>,
  categories: string[],
): CumulativeChartSeries {
  return {
    name: year,
    year,
    role: "history",
    data: categories.map((category) => valueAtDay(yearData, category)),
  };
}

function buildCurrentSeries(
  currentYear: string,
  yearData: Map<string, number>,
  categories: string[],
  todayIndex: number,
): CumulativeChartSeries {
  return {
    name: currentYear,
    year: currentYear,
    role: "current",
    data: categories.map((category, index) => (index <= todayIndex ? valueAtDay(yearData, category) : null)),
  };
}

export function formatCumulativeDateLabel(dateString: string): string {
  const [month, day] = dateString.split("-").map(Number);
  return month && day
    ? `${day} ${MONTH_NAMES[month - 1]}`
    : dateString;
}

export function buildCumulativeYearChartData(input: CumulativeChartInput): CumulativeChartData {
  const metric = normalizeMetric(input.metric);
  const comparisonMode = normalizeComparisonMode(input.comparisonMode);
  const data = metric === "distance" ? input.distancePerYear : input.elevationPerYear;
  const now = input.now ?? new Date();
  const currentYear = String(now.getFullYear());
  const todayKey = formatMonthDay(now);
  const categories = createYearCategories(now.getFullYear());
  const todayIndex = categories.indexOf(todayKey);
  const unit = metric === "distance" ? "km" : "m";
  const metricLabel = metric === "distance" ? "distance" : "elevation";
  const title = `Cumulative ${metricLabel} per year`;
  const yAxisTitle = metric === "distance" ? "Distance (km)" : "Elevation (m)";
  const years = sortedYearKeys(data);
  const currentYearData = data.get(currentYear);
  const currentValue = currentYearData ? valueAtDay(currentYearData, todayKey) : null;
  const comparisonYear = selectedComparisonYear(years, data, currentYear, comparisonMode);
  const comparisonValue = comparisonYear ? valueAtDay(data.get(comparisonYear), todayKey) : null;
  const historicalYears = historicalYearsForMode(years, data, currentYear, comparisonMode);
  const series = historicalYears
    .map((year) => {
      const yearData = data.get(year);
      return yearData ? buildHistoricalSeries(year, yearData, categories) : null;
    })
    .filter((item): item is CumulativeChartSeries => item !== null);
  let projectedValue: number | null = null;

  if (currentYearData) {
    series.push(buildCurrentSeries(currentYear, currentYearData, categories, todayIndex));
    const projection = buildProjection(categories, todayIndex, currentValue ?? 0);
    projectedValue = projection.projectedValue;
    if (projection.projectedValue !== null) {
      series.push({
        name: `${currentYear} projected`,
        year: currentYear,
        role: "projection",
        data: projection.data,
      });
    }
  }

  return {
    categories,
    comparisonMode,
    currentYear,
    currentValue,
    comparisonValue,
    comparisonYear,
    hasData: series.length > 0,
    metric,
    projectedValue,
    series,
    summary: buildSummary(currentValue, comparisonValue, comparisonYear, projectedValue, currentYear, unit),
    title,
    todayIndex,
    todayKey,
    unit,
    yAxisTitle,
  };
}
