export function drawRouteSketchPng(canvas: HTMLCanvasElement, points: number[][], title: string): boolean {
  const usablePoints = points.filter((point) => point.length >= 2);
  const context = canvas.getContext("2d");
  if (usablePoints.length < 2 || !context) return false;

  canvas.width = 900;
  canvas.height = 600;
  const padding = 52;
  const latitudes = usablePoints.map((point) => point[0]);
  const longitudes = usablePoints.map((point) => point[1]);
  const minLat = Math.min(...latitudes);
  const maxLat = Math.max(...latitudes);
  const minLng = Math.min(...longitudes);
  const maxLng = Math.max(...longitudes);
  const latRange = Math.max(0.00001, maxLat - minLat);
  const lngRange = Math.max(0.00001, maxLng - minLng);
  const scale = Math.min((canvas.width - padding * 2) / lngRange, (canvas.height - padding * 2) / latRange);
  const offsetX = (canvas.width - lngRange * scale) / 2;
  const offsetY = (canvas.height - latRange * scale) / 2;

  context.fillStyle = "#ffffff";
  context.fillRect(0, 0, canvas.width, canvas.height);
  context.strokeStyle = "#dfe6f1";
  context.lineWidth = 2;
  context.strokeRect(16, 16, canvas.width - 32, canvas.height - 32);
  context.setLineDash([12, 10]);
  context.lineCap = "round";
  context.lineJoin = "round";
  context.strokeStyle = "#6f51ff";
  context.lineWidth = 6;
  context.beginPath();
  usablePoints.forEach((point, index) => {
    const x = offsetX + (point[1] - minLng) * scale;
    const y = offsetY + (maxLat - point[0]) * scale;
    if (index === 0) context.moveTo(x, y);
    else context.lineTo(x, y);
  });
  context.stroke();
  context.setLineDash([]);
  context.fillStyle = "#242933";
  context.font = "700 22px system-ui, -apple-system, BlinkMacSystemFont, sans-serif";
  context.fillText(title.trim() || "GPS Art sketch", 30, canvas.height - 28);
  return true;
}

export function safeSketchFilename(value: string): string {
  return value.trim().toLowerCase().replace(/[^a-z0-9-]+/g, "-").replace(/^-+|-+$/g, "") || "strava-art-sketch";
}
