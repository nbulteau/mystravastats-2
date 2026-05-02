# Source Mode Smoke Fixtures

These fixtures are intentionally tiny and anonymous. They exercise the same
runtime source-mode paths as real Strava cache, FIT and GPX folders.

- `strava/` follows the shared local Strava cache layout with cache-only auth.
- `fit/` contains a generated FIT activity.
- `gpx/` contains one cycling GPX activity with elevation, heart rate, cadence
  and power samples.

The smoke script copies these folders to a temporary directory before launching
the backend, because providers may persist local diagnostics or detail caches.
