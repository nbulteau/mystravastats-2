<script setup lang="ts">
import L from "leaflet";
import "leaflet/dist/leaflet.css";
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import type { MapPassageSegment, MapPassages, MapRenderMode, MapTrack } from "@/models/map.model";
import type { MapViewport } from "@/stores/map";
import { getActivityTypeColor } from "@/utils/mapTrackColors";
import { useRouter } from "vue-router";

const props = defineProps<{
  mapTracks: MapTrack[];
  mapPassages?: MapPassages | null;
  datasetKey: string;
  loading?: boolean;
  error?: string | null;
  initialViewport?: MapViewport | null;
  recenterToken?: number;
  renderMode?: MapRenderMode;
}>();

const emit = defineEmits<{
  (event: "viewport-changed", value: MapViewport): void;
}>();

const DEFAULT_VIEW: [number, number] = [46.2276, 2.2137];
const DEFAULT_ZOOM = 6;
const MAX_POINTS_PER_TRACK = 1500;
const AGGREGATION_ZOOM_THRESHOLD = 9;
const AGGREGATION_MIN_TRACKS = 24;
const AGGREGATION_CELL_SIZE_PX = 64;
const HEATMAP_SAMPLE_CELL_SIZE_PX = 8;
const HEATMAP_MAX_POINTS_PER_TRACK = 360;
const HEATMAP_RADIUS_PX = 24;
const HEATMAP_MAX_DEVICE_PIXEL_RATIO = 2;

const map = ref<L.Map>();
const mapContainer = ref<HTMLDivElement | null>(null);
const tracksLayerGroup = ref<L.LayerGroup>();
const heatmapCanvas = ref<HTMLCanvasElement>();
const latestBounds = ref<L.LatLngBounds | null>(null);
const lastDatasetKey = ref<string | null>(null);
const canvasRenderer = L.canvas({ padding: 0.25 });
const router = useRouter();
const aggregatedClusterCount = ref(0);
const aggregatedTrackCount = ref(0);
const aggregatedDistanceKm = ref(0);
const aggregatedElevationGainM = ref(0);
const heatmapCellCount = ref(0);
const heatmapPointCount = ref(0);
const passageCorridorCount = ref(0);
const passageMaxCount = ref(0);
const passageResolutionMeters = ref(0);

const isValidCoordinate = (coord: number[]) =>
  Number.isFinite(coord[0]) && Number.isFinite(coord[1]);

const hasRenderableTracks = computed(() => latestBounds.value !== null);
const isHeatmapMode = computed(() => props.renderMode === "HEATMAP");
const isPassagesMode = computed(() => props.renderMode === "PASSAGES");
const isAggregatedMode = computed(() => !isHeatmapMode.value && !isPassagesMode.value && aggregatedClusterCount.value > 0);
const isHeatmapOverlayVisible = computed(() => isHeatmapMode.value && heatmapCellCount.value > 0);
const isPassagesOverlayVisible = computed(() => isPassagesMode.value && passageCorridorCount.value > 0);
const emptyMapMessage = computed(() => {
  if (isPassagesMode.value) {
    return "No passage corridors for the selected filters.";
  }
  return "No GPS tracks for the selected filters.";
});

type MapCluster = {
  center: L.LatLng;
  tracks: MapTrack[];
  points: L.LatLng[];
  totalDistanceKm: number;
  totalElevationGainM: number;
  weightedContribution: number;
};

type HeatmapCell = {
  point: L.Point;
  count: number;
};

function simplifyTrackCoordinates(trackCoordinates: number[][]): number[][] {
  if (trackCoordinates.length <= MAX_POINTS_PER_TRACK) {
    return trackCoordinates;
  }
  const step = Math.ceil(trackCoordinates.length / MAX_POINTS_PER_TRACK);
  const reduced: number[][] = [];
  for (let index = 0; index < trackCoordinates.length; index += step) {
    reduced.push(trackCoordinates[index]);
  }
  const lastPoint = trackCoordinates[trackCoordinates.length - 1];
  if (reduced.length === 0 || reduced[reduced.length - 1] !== lastPoint) {
    reduced.push(lastPoint);
  }
  return reduced;
}

function simplifyHeatmapCoordinates(trackCoordinates: number[][]): number[][] {
  if (trackCoordinates.length <= HEATMAP_MAX_POINTS_PER_TRACK) {
    return trackCoordinates;
  }
  const step = Math.ceil(trackCoordinates.length / HEATMAP_MAX_POINTS_PER_TRACK);
  const reduced: number[][] = [];
  for (let index = 0; index < trackCoordinates.length; index += step) {
    reduced.push(trackCoordinates[index]);
  }
  const lastPoint = trackCoordinates[trackCoordinates.length - 1];
  if (reduced.length === 0 || reduced[reduced.length - 1] !== lastPoint) {
    reduced.push(lastPoint);
  }
  return reduced;
}

function heatmapColorComponents(ratio: number): [number, number, number] {
  const clampedRatio = Math.max(0, Math.min(1, ratio));
  const stops: Array<{ stop: number; color: [number, number, number] }> = [
    { stop: 0, color: [27, 94, 196] },
    { stop: 0.25, color: [0, 188, 212] },
    { stop: 0.5, color: [67, 160, 71] },
    { stop: 0.75, color: [255, 193, 7] },
    { stop: 1, color: [224, 49, 49] },
  ];

  const rightIndex = Math.max(1, stops.findIndex((entry) => entry.stop >= clampedRatio));
  const right = stops[rightIndex];
  const left = stops[rightIndex - 1];
  const span = right.stop - left.stop || 1;
  const localRatio = (clampedRatio - left.stop) / span;

  return [
    Math.round(left.color[0] + (right.color[0] - left.color[0]) * localRatio),
    Math.round(left.color[1] + (right.color[1] - left.color[1]) * localRatio),
    Math.round(left.color[2] + (right.color[2] - left.color[2]) * localRatio),
  ];
}

function passageColor(count: number): string {
  if (count >= 10) {
    return "#d9480f";
  }
  if (count >= 5) {
    return "#f08c00";
  }
  if (count >= 2) {
    return "#2f9e44";
  }
  return "#4dabf7";
}

function passageWeight(count: number): number {
  return Math.min(9, 2.5 + Math.sqrt(Math.max(1, count)) * 1.4);
}

function resetTraceAggregationState() {
  aggregatedClusterCount.value = 0;
  aggregatedTrackCount.value = 0;
  aggregatedDistanceKm.value = 0;
  aggregatedElevationGainM.value = 0;
}

function resetHeatmapState() {
  heatmapCellCount.value = 0;
  heatmapPointCount.value = 0;
}

function resetPassagesState() {
  passageCorridorCount.value = 0;
  passageMaxCount.value = 0;
  passageResolutionMeters.value = props.mapPassages?.resolutionMeters ?? 0;
}

function escapeHtml(value: string): string {
  return value
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

function formatActivityDate(value: string): string {
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value.substring(0, 10);
  }
  return parsed.toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "2-digit",
  });
}

function popupContent(track: MapTrack): string {
  const activityName = escapeHtml(track.activityName);
  const activityType = escapeHtml(track.activityType || "-");
  const activityDate = escapeHtml(formatActivityDate(track.activityDate));
  const distance = Number.isFinite(track.distanceKm) ? `${track.distanceKm.toFixed(1)} km` : "-";
  const elevation = Number.isFinite(track.elevationGainM) ? `${Math.round(track.elevationGainM)} m` : "-";
  const localUrl = `/activities/${track.activityId}`;

  return `
    <div class="map-popup">
      <a class="map-popup__title map-popup__title-link" href="${localUrl}" data-internal-route="true">${activityName}</a>
      <div class="map-popup__meta">${activityDate} · ${activityType}</div>
      <div class="map-popup__meta">${distance} · D+ ${elevation}</div>
    </div>
  `;
}

function handlePopupOpen(event: L.PopupEvent) {
  const popupElement = event.popup.getElement();
  if (!popupElement) {
    return;
  }
  const internalLinks = popupElement.querySelectorAll<HTMLAnchorElement>("a[data-internal-route='true']");
  internalLinks.forEach((anchor) => {
    anchor.onclick = (clickEvent) => {
      clickEvent.preventDefault();
      const href = anchor.getAttribute("href");
      if (href) {
        void router.push(href);
      }
    };
  });
}

function getDominantActivityType(tracks: MapTrack[]): string {
  const counts = new Map<string, number>();
  tracks.forEach((track) => {
    const type = track.activityType || "Unknown";
    counts.set(type, (counts.get(type) ?? 0) + 1);
  });
  let dominantType = "Unknown";
  let dominantCount = -1;
  counts.forEach((count, type) => {
    if (count > dominantCount) {
      dominantCount = count;
      dominantType = type;
    }
  });
  return dominantType;
}

function trackWeightContribution(track: MapTrack): number {
  const distanceScore = Number.isFinite(track.distanceKm) ? track.distanceKm : 0;
  const elevationScore = Number.isFinite(track.elevationGainM) ? track.elevationGainM / 100 : 0;
  return Math.max(0.5, distanceScore + elevationScore);
}

function computeClusterRadius(cluster: MapCluster): number {
  const activityScore = Math.log2(cluster.tracks.length + 1) * 1.8;
  const effortScore = Math.log2(cluster.weightedContribution + 1) * 3.4;
  return Math.min(28, Math.max(9, 6 + activityScore + effortScore));
}

function formatClusterSummary(cluster: MapCluster): string {
  const distanceLabel = `${cluster.totalDistanceKm.toFixed(0)} km`;
  const elevationLabel = `${Math.round(cluster.totalElevationGainM).toLocaleString()} m`;
  return `${cluster.tracks.length} activities · ${distanceLabel} · D+ ${elevationLabel}`;
}

function shouldUseAggregation(currentZoom: number): boolean {
  return currentZoom <= AGGREGATION_ZOOM_THRESHOLD && props.mapTracks.length >= AGGREGATION_MIN_TRACKS;
}

function trackRepresentativePoint(track: MapTrack): L.LatLng | null {
  const firstValid = track.coordinates.find((coord) => isValidCoordinate(coord));
  if (!firstValid) {
    return null;
  }
  return L.latLng(firstValid[0], firstValid[1]);
}

function buildClusters(currentZoom: number): MapCluster[] {
  if (!map.value) {
    return [];
  }
  const clusters = new Map<string, {
    weightedLatSum: number;
    weightedLngSum: number;
    weightSum: number;
    tracks: MapTrack[];
    points: L.LatLng[];
    totalDistanceKm: number;
    totalElevationGainM: number;
    weightedContribution: number;
  }>();
  props.mapTracks.forEach((track) => {
    const representativePoint = trackRepresentativePoint(track);
    if (!representativePoint) {
      return;
    }
    const contribution = trackWeightContribution(track);
    const projected = map.value!.project(representativePoint, currentZoom);
    const cellX = Math.floor(projected.x / AGGREGATION_CELL_SIZE_PX);
    const cellY = Math.floor(projected.y / AGGREGATION_CELL_SIZE_PX);
    const key = `${cellX}:${cellY}`;
    const existing = clusters.get(key);
    if (existing) {
      existing.weightedLatSum += representativePoint.lat * contribution;
      existing.weightedLngSum += representativePoint.lng * contribution;
      existing.weightSum += contribution;
      existing.tracks.push(track);
      existing.points.push(representativePoint);
      existing.totalDistanceKm += track.distanceKm;
      existing.totalElevationGainM += track.elevationGainM;
      existing.weightedContribution += contribution;
      return;
    }
    clusters.set(key, {
      weightedLatSum: representativePoint.lat * contribution,
      weightedLngSum: representativePoint.lng * contribution,
      weightSum: contribution,
      tracks: [track],
      points: [representativePoint],
      totalDistanceKm: track.distanceKm,
      totalElevationGainM: track.elevationGainM,
      weightedContribution: contribution,
    });
  });

  return Array.from(clusters.values()).map((cluster) => ({
    center: L.latLng(cluster.weightedLatSum / cluster.weightSum, cluster.weightedLngSum / cluster.weightSum),
    tracks: cluster.tracks,
    points: cluster.points,
    totalDistanceKm: cluster.totalDistanceKm,
    totalElevationGainM: cluster.totalElevationGainM,
    weightedContribution: cluster.weightedContribution,
  }));
}

function ensureHeatmapCanvas(): HTMLCanvasElement | null {
  if (!map.value) {
    return null;
  }

  if (!heatmapCanvas.value) {
    const canvas = L.DomUtil.create(
      "canvas",
      "activity-heatmap-canvas leaflet-zoom-animated",
      map.value.getPanes().overlayPane,
    ) as HTMLCanvasElement;
    canvas.style.pointerEvents = "none";
    canvas.style.display = "none";
    heatmapCanvas.value = canvas;
  }

  return heatmapCanvas.value;
}

function clearHeatmapCanvas() {
  const canvas = heatmapCanvas.value;
  if (!canvas) {
    return;
  }
  const context = canvas.getContext("2d");
  if (context) {
    context.setTransform(1, 0, 0, 1, 0, 0);
    context.clearRect(0, 0, canvas.width, canvas.height);
  }
  canvas.style.display = "none";
}

function prepareHeatmapCanvas(): {
  canvas: HTMLCanvasElement;
  context: CanvasRenderingContext2D;
  size: L.Point;
  topLeft: L.Point;
} | null {
  if (!map.value) {
    return null;
  }

  const canvas = ensureHeatmapCanvas();
  const context = canvas?.getContext("2d");
  if (!canvas || !context) {
    return null;
  }

  const size = map.value.getSize();
  const topLeft = map.value.containerPointToLayerPoint([0, 0]);
  const pixelRatio = Math.min(
    typeof window === "undefined" ? 1 : window.devicePixelRatio || 1,
    HEATMAP_MAX_DEVICE_PIXEL_RATIO,
  );
  const width = Math.max(1, Math.round(size.x * pixelRatio));
  const height = Math.max(1, Math.round(size.y * pixelRatio));

  if (canvas.width !== width || canvas.height !== height) {
    canvas.width = width;
    canvas.height = height;
  }
  canvas.style.width = `${size.x}px`;
  canvas.style.height = `${size.y}px`;
  canvas.style.display = "block";
  L.DomUtil.setPosition(canvas, topLeft);

  context.setTransform(1, 0, 0, 1, 0, 0);
  context.clearRect(0, 0, canvas.width, canvas.height);
  context.setTransform(pixelRatio, 0, 0, pixelRatio, 0, 0);

  return {
    canvas,
    context,
    size,
    topLeft,
  };
}

function buildHeatmapCells(topLeft: L.Point, size: L.Point): {
  cells: HeatmapCell[];
  pointsForBounds: L.LatLng[];
} {
  if (!map.value) {
    return { cells: [], pointsForBounds: [] };
  }

  const cells = new Map<string, {
    count: number;
    pointXSum: number;
    pointYSum: number;
  }>();
  const pointsForBounds: L.LatLng[] = [];
  const drawMargin = HEATMAP_RADIUS_PX * 2;

  props.mapTracks.forEach((track) => {
    const simplifiedCoordinates = simplifyHeatmapCoordinates(track.coordinates);
    const validCoordinates = simplifiedCoordinates.filter((coord) => isValidCoordinate(coord));
    validCoordinates.forEach((coord) => {
      const latLng = L.latLng(coord[0], coord[1]);
      pointsForBounds.push(latLng);

      const layerPoint = map.value!.latLngToLayerPoint(latLng);
      const point = layerPoint.subtract(topLeft);
      if (
        point.x < -drawMargin
        || point.y < -drawMargin
        || point.x > size.x + drawMargin
        || point.y > size.y + drawMargin
      ) {
        return;
      }

      const cellX = Math.floor(point.x / HEATMAP_SAMPLE_CELL_SIZE_PX);
      const cellY = Math.floor(point.y / HEATMAP_SAMPLE_CELL_SIZE_PX);
      const key = `${cellX}:${cellY}`;
      const existing = cells.get(key);
      if (existing) {
        existing.count += 1;
        existing.pointXSum += point.x;
        existing.pointYSum += point.y;
        return;
      }
      cells.set(key, {
        count: 1,
        pointXSum: point.x,
        pointYSum: point.y,
      });
    });
  });

  return {
    cells: Array.from(cells.values()).map((cell) => ({
      point: L.point(cell.pointXSum / cell.count, cell.pointYSum / cell.count),
      count: cell.count,
    })),
    pointsForBounds,
  };
}

function colorizeHeatmapCanvas(context: CanvasRenderingContext2D, width: number, height: number) {
  const image = context.getImageData(0, 0, width, height);
  const pixels = image.data;

  for (let index = 0; index < pixels.length; index += 4) {
    const alpha = pixels[index + 3];
    if (alpha === 0) {
      continue;
    }

    const ratio = Math.min(1, alpha / 170);
    const [red, green, blue] = heatmapColorComponents(ratio);
    pixels[index] = red;
    pixels[index + 1] = green;
    pixels[index + 2] = blue;
    pixels[index + 3] = Math.min(235, Math.round(alpha * 1.45));
  }

  context.putImageData(image, 0, 0);
}

function renderHeatmapLayer(): L.LatLng[] {
  const prepared = prepareHeatmapCanvas();
  if (!prepared) {
    return [];
  }

  const { canvas, context, size, topLeft } = prepared;
  const { cells, pointsForBounds } = buildHeatmapCells(topLeft, size);
  const maxCellCount = cells.reduce((max, cell) => Math.max(max, cell.count), 0);

  heatmapCellCount.value = cells.length;
  heatmapPointCount.value = cells.reduce((sum, cell) => sum + cell.count, 0);

  if (cells.length === 0 || maxCellCount <= 0) {
    clearHeatmapCanvas();
    return pointsForBounds;
  }

  cells.forEach((cell) => {
    const normalized = cell.count / maxCellCount;
    const radius = Math.min(44, HEATMAP_RADIUS_PX + Math.sqrt(cell.count) * 2);
    const alpha = Math.min(0.52, 0.07 + Math.log2(cell.count + 1) * 0.055);
    const gradient = context.createRadialGradient(
      cell.point.x,
      cell.point.y,
      0,
      cell.point.x,
      cell.point.y,
      radius,
    );
    gradient.addColorStop(0, `rgba(0, 0, 0, ${Math.min(0.64, alpha + normalized * 0.12)})`);
    gradient.addColorStop(0.45, `rgba(0, 0, 0, ${alpha * 0.48})`);
    gradient.addColorStop(1, "rgba(0, 0, 0, 0)");

    context.fillStyle = gradient;
    context.beginPath();
    context.arc(cell.point.x, cell.point.y, radius, 0, Math.PI * 2);
    context.fill();
  });

  context.setTransform(1, 0, 0, 1, 0, 0);
  colorizeHeatmapCanvas(context, canvas.width, canvas.height);

  return pointsForBounds;
}

function passageTooltip(segment: MapPassageSegment): string {
  const passageLabel = segment.passageCount === 1 ? "passage" : "passages";
  return `${segment.passageCount} ${passageLabel} · ${segment.activityCount} activities · ${segment.distanceKm.toFixed(1)} km`;
}

function renderPassageLayer(): L.LatLng[] {
  if (!tracksLayerGroup.value) {
    return [];
  }

  const segments = props.mapPassages?.segments ?? [];
  passageCorridorCount.value = 0;
  passageMaxCount.value = segments.reduce((max, segment) => Math.max(max, segment.passageCount), 0);
  passageResolutionMeters.value = props.mapPassages?.resolutionMeters ?? 0;

  const pointsForBounds: L.LatLng[] = [];
  segments.forEach((segment) => {
    const latLngs = segment.coordinates
      .filter((coord) => isValidCoordinate(coord))
      .map((coord) => L.latLng(coord[0], coord[1]));
    if (latLngs.length < 2) {
      return;
    }

    passageCorridorCount.value++;
    pointsForBounds.push(...latLngs);
    const polyline = L.polyline(latLngs, {
      color: passageColor(segment.passageCount),
      weight: passageWeight(segment.passageCount),
      opacity: 0.82,
      renderer: canvasRenderer,
      smoothFactor: 0.6,
    }).addTo(tracksLayerGroup.value!);
    polyline.bindTooltip(passageTooltip(segment), { direction: "top", opacity: 0.95 });
  });

  return pointsForBounds;
}

function zoomToCluster(cluster: MapCluster) {
  if (!map.value) {
    return;
  }
  if (cluster.points.length === 1) {
    map.value.setView(cluster.points[0], Math.max(map.value.getZoom() + 2, 13));
    return;
  }
  const bounds = L.latLngBounds(cluster.points);
  if (bounds.isValid()) {
    map.value.fitBounds(bounds, { padding: [28, 28], maxZoom: 14 });
  }
}

function renderAggregatedClusters(currentZoom: number): L.LatLng[] {
  if (!tracksLayerGroup.value) {
    return [];
  }
  const clusters = buildClusters(currentZoom);
  aggregatedClusterCount.value = clusters.length;
  aggregatedTrackCount.value = props.mapTracks.length;
  aggregatedDistanceKm.value = clusters.reduce((sum, cluster) => sum + cluster.totalDistanceKm, 0);
  aggregatedElevationGainM.value = clusters.reduce((sum, cluster) => sum + cluster.totalElevationGainM, 0);

  const pointsForBounds: L.LatLng[] = [];
  clusters.forEach((cluster) => {
    pointsForBounds.push(...cluster.points);
    const activityCount = cluster.tracks.length;
    const dominantType = getDominantActivityType(cluster.tracks);
    const marker = L.circleMarker(cluster.center, {
      radius: computeClusterRadius(cluster),
      color: "#ffffff",
      weight: 2,
      fillColor: getActivityTypeColor(dominantType),
      fillOpacity: 0.82,
    }).addTo(tracksLayerGroup.value!);
    marker.bindTooltip(formatClusterSummary(cluster), { direction: "top", opacity: 0.95 });

    if (activityCount === 1) {
      marker.bindPopup(popupContent(cluster.tracks[0]), {
        maxWidth: 320,
        className: "ms-map-popup",
      });
    } else {
      marker.bindPopup(
        `<div class="map-popup"><div class="map-popup__title">${activityCount} activities</div><div class="map-popup__meta">${cluster.totalDistanceKm.toFixed(0)} km · D+ ${Math.round(cluster.totalElevationGainM).toLocaleString()} m</div><div class="map-popup__meta">Zoom in for detailed tracks</div></div>`,
        { maxWidth: 260, className: "ms-map-popup" },
      );
    }
    marker.on("click", () => {
      if (activityCount === 1) {
        marker.openPopup();
        return;
      }
      zoomToCluster(cluster);
    });
  });

  return pointsForBounds;
}

function publishViewport() {
  if (!map.value) {
    return;
  }
  const center = map.value.getCenter();
  emit("viewport-changed", {
    center: [center.lat, center.lng],
    zoom: map.value.getZoom(),
  });
}

function handleZoomEnd() {
  publishViewport();
  renderMapLayers();
}

function handleMoveEnd() {
  publishViewport();
  renderMapLayers();
}

function applyViewport(viewport: MapViewport | null | undefined): boolean {
  if (
    !map.value
    || !viewport
    || !Number.isFinite(viewport.center[0])
    || !Number.isFinite(viewport.center[1])
    || !Number.isFinite(viewport.zoom)
  ) {
    return false;
  }
  map.value.setView(viewport.center, viewport.zoom, { animate: false });
  return true;
}

const initMap = () => {
  if (mapContainer.value) {
    if (map.value) {
      map.value.remove();
      map.value = undefined;
    }

    map.value = L.map(mapContainer.value, {
      zoomControl: true,
      preferCanvas: true,
    });
    map.value.setView(DEFAULT_VIEW, DEFAULT_ZOOM);

    L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
      maxZoom: 19,
    }).addTo(map.value);

    tracksLayerGroup.value = L.layerGroup().addTo(map.value);
    map.value.on("moveend", handleMoveEnd);
    map.value.on("zoomend", handleZoomEnd);
    map.value.on("popupopen", handlePopupOpen);

    applyViewport(props.initialViewport);
  }
};

function renderTraceLayers() {
  latestBounds.value = null;
  if (!map.value || !tracksLayerGroup.value) {
    return;
  }
  tracksLayerGroup.value.clearLayers();
  clearHeatmapCanvas();

  const currentZoom = map.value.getZoom();
  if (shouldUseAggregation(currentZoom)) {
    const aggregatedPoints = renderAggregatedClusters(currentZoom);
    if (aggregatedPoints.length === 0) {
      return;
    }
    const bounds = L.latLngBounds(aggregatedPoints);
    if (bounds.isValid()) {
      latestBounds.value = bounds;
    }
    return;
  }

  resetTraceAggregationState();

  const allLatLngs: L.LatLng[] = [];
  props.mapTracks.forEach((track) => {
    const latLngs = simplifyTrackCoordinates(track.coordinates)
      .filter((coord) => isValidCoordinate(coord))
      .map((coord) => L.latLng(coord[0], coord[1]));

    if (latLngs.length < 2) {
      return;
    }

    const polyline = L.polyline(latLngs, {
      color: getActivityTypeColor(track.activityType),
      weight: 3,
      opacity: 0.88,
      renderer: canvasRenderer,
      smoothFactor: 1.5,
    }).addTo(tracksLayerGroup.value!);
    polyline.bindPopup(popupContent(track), {
      maxWidth: 320,
      className: "ms-map-popup",
    });
    allLatLngs.push(...latLngs);
  });

  if (allLatLngs.length === 0) {
    return;
  }
  const bounds = L.latLngBounds(allLatLngs);
  if (bounds.isValid()) {
    latestBounds.value = bounds;
  }
}

function renderHeatmapMode() {
  latestBounds.value = null;
  if (!map.value) {
    return;
  }
  resetTraceAggregationState();
  resetPassagesState();

  const heatmapPoints = renderHeatmapLayer();
  if (heatmapPoints.length === 0) {
    return;
  }
  const bounds = L.latLngBounds(heatmapPoints);
  if (bounds.isValid()) {
    latestBounds.value = bounds;
  }
}

function renderPassagesMode() {
  latestBounds.value = null;
  if (!map.value || !tracksLayerGroup.value) {
    return;
  }
  tracksLayerGroup.value.clearLayers();
  resetTraceAggregationState();
  resetHeatmapState();
  clearHeatmapCanvas();

  const passagePoints = renderPassageLayer();
  if (passagePoints.length === 0) {
    return;
  }
  const bounds = L.latLngBounds(passagePoints);
  if (bounds.isValid()) {
    latestBounds.value = bounds;
  }
}

function renderMapLayers() {
  tracksLayerGroup.value?.clearLayers();
  clearHeatmapCanvas();
  resetHeatmapState();
  resetPassagesState();

  if (isPassagesMode.value) {
    renderPassagesMode();
    return;
  }
  if (isHeatmapMode.value) {
    renderHeatmapMode();
    return;
  }
  renderTraceLayers();
}

function fitToDataOrDefault() {
  if (!map.value) {
    return;
  }
  if (latestBounds.value && latestBounds.value.isValid()) {
    map.value.fitBounds(latestBounds.value, { padding: [24, 24] });
    return;
  }
  if (applyViewport(props.initialViewport)) {
    return;
  }
  map.value.setView(DEFAULT_VIEW, DEFAULT_ZOOM);
}

function syncMapWithData() {
  renderMapLayers();
  if (!map.value) {
    return;
  }

  const datasetChanged = lastDatasetKey.value !== props.datasetKey;
  if (datasetChanged) {
    lastDatasetKey.value = props.datasetKey;

    if (!applyViewport(props.initialViewport)) {
      fitToDataOrDefault();
    }
  }
}

watch(
  () => props.mapTracks,
  syncMapWithData,
  { immediate: true },
);

watch(
  () => props.mapPassages,
  syncMapWithData,
);

watch(
  () => props.datasetKey,
  syncMapWithData,
);

watch(
  () => props.renderMode,
  () => {
    syncMapWithData();
  },
);

watch(
  () => props.recenterToken,
  (next, previous) => {
    if (next === previous) {
      return;
    }
    syncMapWithData();
    fitToDataOrDefault();
  },
);

onMounted(() => {
  initMap();
  syncMapWithData();
});

onBeforeUnmount(() => {
  if (map.value) {
    map.value.off("moveend", handleMoveEnd);
    map.value.off("zoomend", handleZoomEnd);
    map.value.off("popupopen", handlePopupOpen);
    map.value.remove();
    map.value = undefined;
  }
  tracksLayerGroup.value = undefined;
  heatmapCanvas.value = undefined;
  latestBounds.value = null;
  resetTraceAggregationState();
  resetHeatmapState();
  resetPassagesState();
});
</script>

<template>
  <main class="map-shell">
    <div
      ref="mapContainer"
      class="map-canvas"
    />
    <div
      v-if="loading"
      class="map-overlay map-overlay--loading"
    >
      Loading map data...
    </div>
    <div
      v-else-if="error"
      class="map-overlay map-overlay--error"
    >
      {{ error }}
    </div>
    <div
      v-else-if="!hasRenderableTracks"
      class="map-overlay map-overlay--empty"
    >
      {{ emptyMapMessage }}
    </div>
    <div
      v-if="isAggregatedMode"
      class="map-overlay map-overlay--aggregation"
    >
      Aggregated mode · {{ aggregatedTrackCount }} tracks in {{ aggregatedClusterCount }} clusters
      · {{ aggregatedDistanceKm.toFixed(0) }} km · D+ {{ Math.round(aggregatedElevationGainM).toLocaleString() }} m
    </div>
    <div
      v-if="isHeatmapOverlayVisible"
      class="map-overlay map-overlay--heatmap"
    >
      Heatmap · {{ heatmapCellCount }} cells · {{ heatmapPointCount.toLocaleString() }} sampled points
    </div>
    <div
      v-if="isPassagesOverlayVisible"
      class="map-overlay map-overlay--passages"
    >
      Route frequency · {{ passageCorridorCount }} corridors · max {{ passageMaxCount }} passes
      <span v-if="passageResolutionMeters > 0">· {{ passageResolutionMeters }} m</span>
      <span v-if="(mapPassages?.omittedSegments ?? 0) > 0">· {{ mapPassages?.omittedSegments.toLocaleString() }} hidden</span>
    </div>
  </main>
</template>

<style scoped>
.map-shell {
  position: relative;
  border: 1px solid var(--ms-border);
  border-radius: 14px;
  overflow: hidden;
  box-shadow: var(--ms-shadow-soft);
  background: var(--ms-surface-strong);
}

.map-canvas {
  width: 100%;
  height: calc(100vh - 190px);
}

.map-overlay {
  position: absolute;
  inset: auto 14px 14px auto;
  z-index: 550;
  border-radius: 10px;
  padding: 7px 10px;
  font-size: 0.86rem;
  font-weight: 600;
  backdrop-filter: blur(8px);
}

.map-overlay--loading {
  color: #5c4b00;
  border: 1px solid #f2dea1;
  background: rgba(255, 246, 219, 0.95);
}

.map-overlay--error {
  color: #7a2131;
  border: 1px solid #f4bac7;
  background: rgba(255, 234, 238, 0.96);
}

.map-overlay--empty {
  color: var(--ms-text-muted);
  border: 1px solid var(--ms-border);
  background: rgba(255, 255, 255, 0.95);
}

.map-overlay--aggregation {
  inset: 14px 14px auto auto;
  color: #174a2a;
  border: 1px solid #b8e0c7;
  background: rgba(236, 250, 241, 0.95);
  font-size: 0.8rem;
}

.map-overlay--heatmap {
  inset: 14px 14px auto auto;
  color: #471111;
  border: 1px solid #f0b4a8;
  background: rgba(255, 238, 235, 0.95);
  font-size: 0.8rem;
}

:deep(.activity-heatmap-canvas) {
  pointer-events: none;
  opacity: 0.9;
}

.map-overlay--passages {
  inset: 14px 14px auto auto;
  color: #5b2b00;
  border: 1px solid #f2c27b;
  background: rgba(255, 244, 226, 0.95);
  font-size: 0.8rem;
}

:deep(.ms-map-popup .leaflet-popup-content) {
  margin: 10px 12px;
}

:deep(.map-popup) {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 220px;
}

:deep(.map-popup__title) {
  font-weight: 800;
  color: var(--ms-text);
  line-height: 1.25;
}

:deep(.map-popup__title-link) {
  text-decoration: none;
  color: var(--ms-primary);
}

:deep(.map-popup__title-link:hover) {
  text-decoration: underline;
}

:deep(.map-popup__meta) {
  font-size: 0.82rem;
  color: var(--ms-text-muted);
}

@media (max-width: 992px) {
  .map-canvas {
    height: calc(100vh - 250px);
  }
}
</style>
