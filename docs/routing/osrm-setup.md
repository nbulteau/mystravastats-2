# OSM Routing Setup (OSRM)

This guide explains how to set up a local OSRM router for MyStravaStats route generation.

## Platform-Specific Setup

This guide provides commands for **Windows (PowerShell)**, **macOS**, and **Linux**.

Examples below assume:
- **Windows:** `$ProjectRoot = "D:\workspace\mystravastats-2"`
- **macOS/Linux:** `PROJECT_ROOT=/path/to/mystravastats-2`

## 1. Download an OSM extract (`.osm.pbf`)

Put the file here:

- **Windows:** `$ProjectRoot\osm\region.osm.pbf`
- **macOS/Linux:** `$PROJECT_ROOT/osm/region.osm.pbf`

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
PROJECT_ROOT=/path/to/mystravastats-2
mkdir -p "$PROJECT_ROOT/osm"
curl -L "https://download.geofabrik.de/europe/france-latest.osm.pbf" \
  -o "$PROJECT_ROOT/osm/region.osm.pbf"
```

If you only want Brittany (recommended for lower RAM usage and faster OSRM preparation):

```sh
PROJECT_ROOT=/path/to/mystravastats-2
mkdir -p "$PROJECT_ROOT/osm"
curl -L "https://download.geofabrik.de/europe/france/bretagne-latest.osm.pbf" \
  -o "$PROJECT_ROOT/osm/region.osm.pbf"
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
PROJECT_ROOT=/path/to/mystravastats-2
mkdir -p "$PROJECT_ROOT/osm"
cp /path/to/your-download.osm.pbf "$PROJECT_ROOT/osm/region.osm.pbf"
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
PROJECT_ROOT=/path/to/mystravastats-2
docker compose -f "$PROJECT_ROOT/docker-compose-routing-osrm.yml" --profile prepare run --rm osrm-prepare
```

By default, extraction uses `/opt/bicycle.lua` (cycling-oriented routing).
By default, preprocessing runs with `2` threads to reduce memory pressure.
The prepare step also writes the selected extract profile to `osm/region.osrm.profile`
so MyStravaStats can expose profile-aware route type availability in the UI.

If you want a different profile for extraction:

**Windows (PowerShell) - walking/hiking:**

```powershell
$ProjectRoot = "D:\workspace\mystravastats-2"
$env:OSRM_EXTRACT_PROFILE = "/opt/foot.lua"
docker compose -f "$ProjectRoot\docker-compose-routing-osrm.yml" --profile prepare run --rm osrm-prepare
```

**macOS/Linux - walking/hiking:**

```sh
PROJECT_ROOT=/path/to/mystravastats-2
OSRM_EXTRACT_PROFILE=/opt/foot.lua \
docker compose -f "$PROJECT_ROOT/docker-compose-routing-osrm.yml" --profile prepare run --rm osrm-prepare
```

**Windows (PowerShell) - car profile:**

```powershell
$ProjectRoot = "D:\workspace\mystravastats-2"
$env:OSRM_EXTRACT_PROFILE = "/opt/car.lua"
docker compose -f "$ProjectRoot\docker-compose-routing-osrm.yml" --profile prepare run --rm osrm-prepare
```

**macOS/Linux - car profile:**

```sh
PROJECT_ROOT=/path/to/mystravastats-2
OSRM_EXTRACT_PROFILE=/opt/car.lua \
docker compose -f "$PROJECT_ROOT/docker-compose-routing-osrm.yml" --profile prepare run --rm osrm-prepare
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
PROJECT_ROOT=/path/to/mystravastats-2
OSRM_THREADS=4 \
docker compose -f "$PROJECT_ROOT/docker-compose-routing-osrm.yml" --profile prepare run --rm osrm-prepare
```

## 3. Start the OSRM router

**Windows (PowerShell):**

```powershell
$ProjectRoot = "D:\workspace\mystravastats-2"
docker compose -f "$ProjectRoot\docker-compose-routing-osrm.yml" up -d osrm
```

**macOS/Linux:**

```sh
PROJECT_ROOT=/path/to/mystravastats-2
docker compose -f "$PROJECT_ROOT/docker-compose-routing-osrm.yml" up -d osrm
```

Default endpoint:

- `http://localhost:5000`

The Diagnostics tab can also start the local OSRM container with **Start OSRM**.
It runs the fixed command:

```sh
docker compose -f docker-compose-routing-osrm.yml up -d osrm
```

This only starts already-prepared OSRM data. It does not run the heavier
extract/partition/customize preparation step.

## 4. Verify backend health integration

Check:

- `http://localhost:8080/api/health/details`

Expected fields:

- `routing.engine = "osrm"`
- `routing.status = "up"` (or `disabled` if routing is disabled)
- `routing.reachable = true`
- `routing.extractProfile` (for example `/opt/bicycle.lua`)
- `routing.supportedRouteTypes` (Route Type combo availability in Routes tab)

## 5. Routing environment variables

- `OSM_ROUTING_ENABLED` (default `true`)
- `OSM_ROUTING_BASE_URL` (default `http://localhost:5000`)
- `OSM_ROUTING_TIMEOUT_MS` (default `3000`)
- `OSM_ROUTING_PROFILE` (optional override, e.g. `cycling` or `walking`)
- `OSM_ROUTING_EXTRACT_PROFILE` (optional backend override for extract profile detection)
- `OSM_ROUTING_EXTRACT_PROFILE_FILE` (default `./osm/region.osrm.profile`)
- `OSRM_CONTROL_ENABLED` (default `true`, allows the Diagnostics tab to start OSRM)
- `OSRM_CONTROL_TIMEOUT_MS` (default `30000`, start command timeout)
- `OSRM_CONTROL_PROJECT_DIR` (optional project root override for the compose command)
- `OSRM_CONTROL_COMPOSE_FILE` (default `docker-compose-routing-osrm.yml`)
- `OSRM_CONTROL_DOCKER_BIN` (optional Docker CLI path override)
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
PROJECT_ROOT=/path/to/mystravastats-2
OSRM_THREADS=1 \
docker compose -f "$PROJECT_ROOT/docker-compose-routing-osrm.yml" --profile prepare run --rm osrm-prepare
```
