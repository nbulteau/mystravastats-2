# OSM Routing Setup (OSRM)

This guide explains how to set up a local OSRM router for MyStravaStats route generation.

## Platform-Specific Setup

This guide provides commands for **Windows (PowerShell)**, **macOS**, and **Linux**.

Replace path examples with your actual project path:
- **Windows:** `D:\workspace\mystravastats-2`
- **macOS/Linux:** `/Users/nicolas/Workspace/mystravastats-2` (or your path)

## 1. Download an OSM extract (`.osm.pbf`)

Put the file here:

- **Windows:** `D:\workspace\mystravastats-2\osm\region.osm.pbf`
- **macOS/Linux:** `/Users/nicolas/Workspace/mystravastats-2/osm/region.osm.pbf`

### Option A - Geofabrik (recommended)

- Download portal: [https://download.geofabrik.de/](https://download.geofabrik.de/)
- France extract example: [https://download.geofabrik.de/europe/france-latest.osm.pbf](https://download.geofabrik.de/europe/france-latest.osm.pbf)
- Brittany extract example: [https://download.geofabrik.de/europe/france/bretagne-latest.osm.pbf](https://download.geofabrik.de/europe/france/bretagne-latest.osm.pbf)

**Windows (PowerShell):**

```powershell
$ProjectRoot = "D:\workspace\mystravastats-2"
New-Item -ItemType Directory -Path "$ProjectRoot\osm" -Force | Out-Null
Invoke-WebRequest -Uri "https://download.geofabrik.de/europe/france-latest.osm.pbf" `
  -OutFile "$ProjectRoot\osm\region.osm.pbf"
```

For Brittany only (recommended for lower RAM and faster preparation):

```powershell
$ProjectRoot = "D:\workspace\mystravastats-2"
New-Item -ItemType Directory -Path "$ProjectRoot\osm" -Force | Out-Null
Invoke-WebRequest -Uri "https://download.geofabrik.de/europe/france/bretagne-latest.osm.pbf" `
  -OutFile "$ProjectRoot\osm\region.osm.pbf"
```

**macOS/Linux:**

```sh
mkdir -p /Users/nicolas/Workspace/mystravastats-2/osm
curl -L "https://download.geofabrik.de/europe/france-latest.osm.pbf" \
  -o /Users/nicolas/Workspace/mystravastats-2/osm/region.osm.pbf
```

If you only want Brittany (recommended for lower RAM usage and faster OSRM preparation):

```sh
mkdir -p /Users/nicolas/Workspace/mystravastats-2/osm
curl -L "https://download.geofabrik.de/europe/france/bretagne-latest.osm.pbf" \
  -o /Users/nicolas/Workspace/mystravastats-2/osm/region.osm.pbf
```

### Option B - BBBike (custom area)

- Extract generator: [https://extract.bbbike.org/](https://extract.bbbike.org/)
- Download your custom `.osm.pbf`, then copy it to your project

Detailed steps:

1. Open [https://extract.bbbike.org/](https://extract.bbbike.org/).
2. In `City or area`, type a place close to your riding/running region (example: `Grenoble`, `Rennes`, `Lyon`).
3. Let the map center on that area, then adjust the extract rectangle:
   - drag the map to position it,
   - resize/move the selection box so it fully covers your target zone.
4. In `Formats`, select `PBF` (or `osm.pbf` depending on the label shown).
5. (Optional) Set a custom job name so you can identify your extract later.
6. Enter your email address and submit the extract request.
7. Wait for the BBBike email containing your download link.
8. Download the generated `.osm.pbf` file.
9. Copy it into your project:

**Windows (PowerShell):**

```powershell
$ProjectRoot = "D:\workspace\mystravastats-2"
New-Item -ItemType Directory -Path "$ProjectRoot\osm" -Force | Out-Null
Copy-Item -Path "C:\path\to\your-download.osm.pbf" -Destination "$ProjectRoot\osm\region.osm.pbf"
```

**macOS/Linux:**

```sh
mkdir -p /Users/nicolas/Workspace/mystravastats-2/osm
cp /path/to/your-download.osm.pbf /Users/nicolas/Workspace/mystravastats-2/osm/region.osm.pbf
```

Tips:

- Keep the area as small as possible for faster OSRM preparation and lower disk usage.
- If route generation is slow, reduce the extract size and regenerate.

## 2. Prepare OSRM data

Run extract + partition + customize:

**Windows (PowerShell):**

```powershell
$ProjectRoot = "D:\workspace\mystravastats-2"
docker compose -f "$ProjectRoot\docker-compose-routing-osrm.yml" --profile prepare run --rm osrm-prepare
```

**macOS/Linux:**

```sh
docker compose -f /Users/nicolas/Workspace/mystravastats-2/docker-compose-routing-osrm.yml --profile prepare run --rm osrm-prepare
```

By default, extraction uses `/opt/bicycle.lua` (cycling-oriented routing).
By default, preprocessing runs with `2` threads to reduce memory pressure.

If you want a different profile for extraction:

**Windows (PowerShell) - walking/hiking:**

```powershell
$ProjectRoot = "D:\workspace\mystravastats-2"
$env:OSRM_EXTRACT_PROFILE = "/opt/foot.lua"
docker compose -f "$ProjectRoot\docker-compose-routing-osrm.yml" --profile prepare run --rm osrm-prepare
```

**macOS/Linux - walking/hiking:**

```sh
OSRM_EXTRACT_PROFILE=/opt/foot.lua \
docker compose -f /Users/nicolas/Workspace/mystravastats-2/docker-compose-routing-osrm.yml --profile prepare run --rm osrm-prepare
```

**Windows (PowerShell) - car profile:**

```powershell
$ProjectRoot = "D:\workspace\mystravastats-2"
$env:OSRM_EXTRACT_PROFILE = "/opt/car.lua"
docker compose -f "$ProjectRoot\docker-compose-routing-osrm.yml" --profile prepare run --rm osrm-prepare
```

**macOS/Linux - car profile:**

```sh
OSRM_EXTRACT_PROFILE=/opt/car.lua \
docker compose -f /Users/nicolas/Workspace/mystravastats-2/docker-compose-routing-osrm.yml --profile prepare run --rm osrm-prepare
```

If your machine has enough RAM and you want faster preprocessing:

**Windows (PowerShell):**

```powershell
$ProjectRoot = "D:\workspace\mystravastats-2"
$env:OSRM_THREADS = "4"
docker compose -f "$ProjectRoot\docker-compose-routing-osrm.yml" --profile prepare run --rm osrm-prepare
```

**macOS/Linux:**

```sh
OSRM_THREADS=4 \
docker compose -f /Users/nicolas/Workspace/mystravastats-2/docker-compose-routing-osrm.yml --profile prepare run --rm osrm-prepare
```

## 3. Start the OSRM router

**Windows (PowerShell):**

```powershell
$ProjectRoot = "D:\workspace\mystravastats-2"
docker compose -f "$ProjectRoot\docker-compose-routing-osrm.yml" up -d osrm
```

**macOS/Linux:**

```sh
docker compose -f /Users/nicolas/Workspace/mystravastats-2/docker-compose-routing-osrm.yml up -d osrm
```

Default endpoint:

- `http://localhost:5000`

## 4. Verify backend health integration

Check:

- `http://localhost:8080/api/health/details`

Expected fields:

- `routing.engine = "osrm"`
- `routing.status = "up"` (or `disabled` if routing is disabled)
- `routing.reachable = true`

## 5. Routing environment variables

- `OSM_ROUTING_ENABLED` (default `true`)
- `OSM_ROUTING_BASE_URL` (default `http://localhost:5000`)
- `OSM_ROUTING_TIMEOUT_MS` (default `3000`)
- `OSM_ROUTING_PROFILE` (optional override, e.g. `cycling` or `walking`)
- `OSRM_EXTRACT_PROFILE` (default `/opt/bicycle.lua`, used at preprocess time)
- `OSRM_THREADS` (default `2`, used for extract/partition/customize)

## Troubleshooting: process ends with `Killed`

If you see `Killed` during `osrm-prepare`, it is typically an out-of-memory kill.

Try these fixes:

1. Use a smaller `.osm.pbf` extract (preferred).
2. Keep `OSRM_THREADS=1` or `OSRM_THREADS=2`.
3. Increase Docker Desktop memory (often at least 8 GB, ideally 12+ for larger extracts).

Example low-memory run:

**Windows (PowerShell):**

```powershell
$ProjectRoot = "D:\workspace\mystravastats-2"
$env:OSRM_THREADS = "1"
docker compose -f "$ProjectRoot\docker-compose-routing-osrm.yml" --profile prepare run --rm osrm-prepare
```

**macOS/Linux:**

```sh
OSRM_THREADS=1 \
docker compose -f /Users/nicolas/Workspace/mystravastats-2/docker-compose-routing-osrm.yml --profile prepare run --rm osrm-prepare
```
