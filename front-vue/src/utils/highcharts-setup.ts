import Highcharts from "highcharts";

let themeConfigured = false;
let heatmapSetup: Promise<void> | null = null;
let accessibilitySetup: Promise<void> | null = null;
const highchartsSeriesTypes = (Highcharts as typeof Highcharts & {
  seriesTypes: Record<string, unknown>;
}).seriesTypes;

export async function setupHighcharts(includeHeatmap = false): Promise<void> {
  if (!themeConfigured) {
    configureHighchartsTheme();
    themeConfigured = true;
  }
  accessibilitySetup ??= registerAccessibilityModule();
  await accessibilitySetup;
  if (includeHeatmap && !highchartsSeriesTypes.heatmap) {
    heatmapSetup ??= registerHeatmapModule();
    await heatmapSetup;
  }
}

async function registerAccessibilityModule(): Promise<void> {
  (window as typeof window & { _Highcharts?: typeof Highcharts })._Highcharts = Highcharts;
  await import("highcharts/modules/accessibility");
}

async function registerHeatmapModule(): Promise<void> {
  // Highcharts 13 modules self-register against window._Highcharts. Setting the
  // reference before the side-effect import avoids the CommonJS/Vite mismatch
  // that otherwise leaves heatmap unavailable and logs an Axis error.
  (window as typeof window & { _Highcharts?: typeof Highcharts })._Highcharts = Highcharts;
  await import("highcharts/modules/heatmap");
  if (!highchartsSeriesTypes.heatmap) {
    throw new Error("Highcharts heatmap module did not register.");
  }
}

function configureHighchartsTheme(): void {
  Highcharts.setOptions({
    colors: [
      "#fc4c02", "#4e79a7", "#59a14f", "#e15759", "#af7aa1", "#17becf",
      "#f28e2b", "#edc948", "#76b7b2", "#9c755f", "#6c7a89", "#b6992d",
    ],
    chart: {
      backgroundColor: "transparent",
      style: { fontFamily: '"Avenir Next", "SF Pro Text", "Segoe UI", "Helvetica Neue", sans-serif' },
    },
    title: { style: { color: "#242428", fontWeight: "700" } },
    subtitle: { style: { color: "#6a6c75" } },
    xAxis: {
      lineColor: "#e5e7ee",
      tickColor: "#e5e7ee",
      labels: { style: { color: "#6a6c75" } },
      title: { style: { color: "#4e5058" } },
    },
    yAxis: {
      gridLineColor: "#eceff4",
      labels: { style: { color: "#6a6c75" } },
      title: { style: { color: "#4e5058" } },
    },
    legend: {
      itemStyle: { color: "#4f525a", fontWeight: "600" },
      itemHoverStyle: { color: "#242428" },
    },
    tooltip: {
      backgroundColor: "#ffffff",
      borderColor: "#ffd3c1",
      style: { color: "#2d3037" },
    },
    plotOptions: {
      series: { states: { inactive: { opacity: 1 } } },
      line: { lineWidth: 2, marker: { enabled: false } },
      spline: { lineWidth: 2, marker: { enabled: false } },
      areaspline: { lineWidth: 2, marker: { enabled: false } },
      column: { borderWidth: 0, borderRadius: 2 },
    },
  });
}
