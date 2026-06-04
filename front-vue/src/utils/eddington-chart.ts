export const EDDINGTON_CURRENT_COLOR = "#e15759";
export const EDDINGTON_SATISFIED_COLOR = "#fc4c02";
export const EDDINGTON_PENDING_COLOR = "#c7ced8";
export const EDDINGTON_REFERENCE_COLOR = "#4e5058";

export type EddingtonChartInput = {
  eddingtonNumber?: number | null;
  eddingtonList?: readonly number[] | null;
  metric?: "distance" | "elevation" | null;
  basis?: "days" | "activities" | null;
  unit?: "km" | "m" | string | null;
  thresholdScale?: number | null;
  nextTarget?: number | null;
  qualifyingCount?: number | null;
  missingCount?: number | null;
  qualifyingDays?: number | null;
  missingDays?: number | null;
};

export type EddingtonChartPoint = {
  x: number;
  y: number;
  color: string;
  custom: {
    threshold: number;
    displayThreshold: number;
    unit: string;
    countSingular: string;
    metricLabel: string;
    isCurrent: boolean;
    isSatisfied: boolean;
    missingCount: number;
  };
};

export type EddingtonChartData = {
  currentNumber: number;
  points: EddingtonChartPoint[];
  referenceLine: Array<[number, number]>;
  maxThreshold: number;
  maxDays: number;
  axisMin: number;
  axisMax: number;
  yAxisMax: number;
  nextTarget: number;
  nextTargetCurrentCount: number;
  qualifyingCount: number;
  nextTargetMissingCount: number;
  unit: string;
  thresholdScale: number;
  countSingular: string;
  metricLabel: string;
  summary: string;
  hasData: boolean;
};

function normalizePositiveInteger(value: number | null | undefined): number {
  if (!Number.isFinite(value)) {
    return 0;
  }
  return Math.max(0, Math.floor(value ?? 0));
}

function pluralize(count: number, singular: string, plural = `${singular}s`): string {
  if (singular === "activity") {
    return count === 1 ? singular : "activities";
  }
  return count === 1 ? singular : plural;
}

export function pluralizeEddingtonCount(count: number, singular: string): string {
  return pluralize(count, singular);
}

function formatNextTargetSummary(data: {
  currentNumber: number;
  nextTarget: number;
  qualifyingCount: number;
  nextTargetMissingCount: number;
  unit: string;
  thresholdScale: number;
  countSingular: string;
  metricLabel: string;
  hasData: boolean;
}): string {
  const missingLabel = pluralize(data.nextTargetMissingCount, data.countSingular);

  if (!data.hasData) {
    return `No ${data.metricLabel}-qualified ${pluralize(2, data.countSingular)} yet.`;
  }

  if (data.currentNumber <= 0) {
    return `${data.nextTargetMissingCount} ${missingLabel} at ${data.nextTarget * data.thresholdScale}+ ${data.unit} starts the Eddington score.`;
  }

  return `E=${data.currentNumber} - ${data.qualifyingCount}/${data.nextTarget} ${pluralize(data.qualifyingCount, data.countSingular)} at ${data.nextTarget * data.thresholdScale}+ ${data.unit}; ${data.nextTargetMissingCount} more ${missingLabel} needed.`;
}

export function buildEddingtonChartData(input: EddingtonChartInput): EddingtonChartData {
  const currentNumber = normalizePositiveInteger(input.eddingtonNumber);
  const metric = input.metric === "elevation" ? "elevation" : "distance";
  const basis = input.basis === "activities" ? "activities" : "days";
  const unit = input.unit || (metric === "elevation" ? "m" : "km");
  const thresholdScale = normalizePositiveInteger(input.thresholdScale) || (metric === "elevation" ? 100 : 1);
  const countSingular = basis === "activities" ? "activity" : "day";
  const metricLabel = metric === "elevation" ? "elevation" : "distance";
  const counts = (input.eddingtonList ?? []).map(normalizePositiveInteger);
  const allPoints = counts.map((count, index): EddingtonChartPoint => {
    const threshold = index + 1;
    const isCurrent = threshold === currentNumber && count >= currentNumber;
    const isSatisfied = count >= threshold;
    return {
      x: threshold,
      y: count,
      color: isCurrent
        ? EDDINGTON_CURRENT_COLOR
        : isSatisfied
          ? EDDINGTON_SATISFIED_COLOR
          : EDDINGTON_PENDING_COLOR,
      custom: {
        threshold,
        displayThreshold: threshold * thresholdScale,
        unit,
        countSingular,
        metricLabel,
        isCurrent,
        isSatisfied,
        missingCount: Math.max(threshold - count, 0),
      },
    };
  });

  const maxThreshold = allPoints.length;
  const maxDays = allPoints.reduce((max, point) => Math.max(max, point.y), 0);
  const nextTarget = normalizePositiveInteger(input.nextTarget) || currentNumber + 1;
  const backendQualifyingCount = input.qualifyingCount ?? input.qualifyingDays;
  const backendMissingCount = input.missingCount ?? input.missingDays;
  const nextTargetCurrentCount = backendQualifyingCount == null
    ? counts[nextTarget - 1] ?? 0
    : normalizePositiveInteger(backendQualifyingCount);
  const nextTargetMissingCount = backendMissingCount == null
    ? Math.max(nextTarget - nextTargetCurrentCount, 0)
    : normalizePositiveInteger(backendMissingCount);
  const focusPadding = Math.max(12, Math.min(30, Math.ceil(Math.max(currentNumber, 1) * 0.4)));
  const focusMin = currentNumber > 0 ? Math.max(1, currentNumber - focusPadding) : 1;
  const focusMax = currentNumber > 0
    ? Math.min(maxThreshold, Math.max(nextTarget + 8, currentNumber + focusPadding))
    : Math.min(maxThreshold, 30);
  const points = allPoints.filter((point) => point.x >= focusMin && point.x <= focusMax);
  const axisMin = Math.max(0, focusMin - 1);
  const axisMax = Math.max(focusMax, nextTarget, 1);
  const visibleMaxDays = points.reduce((max, point) => Math.max(max, point.y), 0);
  const yAxisMax = Math.max(visibleMaxDays, axisMax, currentNumber, 1);
  const hasData = allPoints.length > 0;

  const dataWithoutSummary = {
    currentNumber,
    points,
    referenceLine: [[axisMin, axisMin], [axisMax, axisMax]] as Array<[number, number]>,
    maxThreshold,
    maxDays,
    axisMin,
    axisMax,
    yAxisMax,
    nextTarget,
    nextTargetCurrentCount,
    qualifyingCount: nextTargetCurrentCount,
    nextTargetMissingCount,
    unit,
    thresholdScale,
    countSingular,
    metricLabel,
    hasData,
  };

  return {
    ...dataWithoutSummary,
    summary: formatNextTargetSummary(dataWithoutSummary),
  };
}

export function formatEddingtonTooltip(point: EddingtonChartPoint): string {
  const countLabel = pluralize(point.y, point.custom.countSingular);
  const missingLabel = pluralize(point.custom.missingCount, point.custom.countSingular);
  const status = point.custom.isSatisfied
    ? "Threshold reached"
    : `${point.custom.missingCount} more ${missingLabel} needed`;
  const current = point.custom.isCurrent ? "<br/><b>Current Eddington number</b>" : "";

  return `<b>${point.custom.displayThreshold} ${point.custom.unit} threshold</b><br/>${point.y} ${countLabel} at ${point.custom.displayThreshold}+ ${point.custom.unit}<br/>${status}${current}`;
}
