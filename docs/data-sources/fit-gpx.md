# FIT And GPX Sources

My Activity Stats can work from local activity files in addition to Strava.

## FIT

Both Go and Kotlin support FIT input through:

```text
FIT_FILES_PATH
```

The Go and Kotlin local backends can also import FIT files from a mounted
Garmin/OpenMTP directory. At startup and from the Status page `Synchronize`
action, they look for an activity directory in this order:

```text
GARMIN_FIT_SOURCE_PATH
/Volumes/*/GARMIN/ACTIVITY
/Volumes/*/GARMIN/Activity
/run/media/<user>/*/GARMIN/ACTIVITY
/media/<user>/*/GARMIN/ACTIVITY
/media/*/GARMIN/ACTIVITY
/mnt/*/GARMIN/ACTIVITY
Windows drive roots, e.g. E:\GARMIN\ACTIVITY
```

For Garmin watches that expose activities through MTP, such as a Fenix 7 on
macOS, the preferred workflow is:

```text
Garmin watch
  -> OpenMTP or mounted filesystem directory
  -> native Go/Kotlin source sync
  -> FIT_INBOX_PATH
  -> Status / Synchronize
  -> FIT_FILES_PATH/<year>/
```

The optional source override is:

```text
GARMIN_FIT_SOURCE_PATH=/path/to/Garmin-or-GARMIN/ACTIVITY
FIT_INBOX_PATH=/path/to/fit-inbox
```

If `FIT_INBOX_PATH` is unset but `FIT_FILES_PATH` is configured, both backends
use:

```text
<FIT_FILES_PATH>/_inbox
```

During `Synchronize`, the backend first copies new `.fit` files from the Garmin
source into the inbox. `FIT_INBOX_PATH` can also be used as a manual drop zone:
place `.fit` files there, then run `Synchronize`.

Imported files are copied into the configured FIT library under the activity
year:

```text
<FIT_FILES_PATH>/2026/example.fit
```

After new files are imported, the active FIT/composite provider is reloaded
without requiring a backend restart. Existing activities are detected by their
decoded start date, type, distance and elapsed time, so re-running
`Synchronize` should keep already imported files untouched.

FIT power metrics use the same fallback in both backends:

- if the FIT session provides `avgPower`, it remains the source for `averageWatts` and `weightedAverageWatts`
- if `avgPower` is missing or zero, `record.power` samples drive `averageWatts`, `weightedAverageWatts`, and kilojoules
- average power includes zero-power samples, so coasting is preserved
- invalid or negative samples are ignored
- weighted power uses a 30-sample rolling normalized-power approximation, with a plain average fallback for shorter streams
- kilojoules keeps the app convention: `0.8604 * averageWatts * elapsedSeconds / 1000`

Known limits:

- devices that do not record power keep these metrics at zero
- devices that record FIT power below or above 1 Hz can slightly shift weighted-power approximation because the rolling window is record-count based

## GPX

Both Go and Kotlin support GPX input through:

```text
GPX_FILES_PATH
```

GPX activities use the same year-folder layout as FIT files:

```text
<GPX_FILES_PATH>/2026/example.gpx
```

GPX parsing keeps the route trace, elevation and optional extension fields such
as heart rate, cadence and power when they are present.

## Saving A Local Source

From the Status page (`/diagnostics`), choose `FIT` or `GPX`, enter the
directory, run `Check directory`, then use `Use this source`. The backend writes
the chosen path to `.env` in its working directory. Restart the backend with the
usual command before expecting the provider or composite mode to change.

## Composite Mode

When two or more sources are explicitly configured, both Go and Kotlin switch to
the composite provider automatically. Examples:

```text
STRAVA_CACHE_PATH=strava-cache
FIT_FILES_PATH=/data/fit
```

```text
FIT_FILES_PATH=/data/fit
GPX_FILES_PATH=/data/gpx
```

The composite provider keeps the existing source caches unchanged. If a local
FIT/GPX activity matches a Strava activity, the Strava activity ID and metadata
remain canonical, while the local stream can enrich the composite view. Local
activities without a Strava match stay visible in union mode with their stable
local IDs.

`/api/health/details` reports `provider=composite`, lists `activeProviders`, and
adds merge diagnostics for matched activities, local-only activities and
conflicts. The Status page renders those details in the `Data Source` section.

## Smoke Test

The source-mode smoke test validates the complete critical API path for
`STRAVA`, `FIT` and `GPX` on either backend:

```shell
node scripts/smoke-source-modes.mjs --backend go
node scripts/smoke-source-modes.mjs --backend kotlin
```

Related docs:

- [Backend Capability Matrix](../architecture/backend-capability-matrix.md)
- [Runtime Configuration](../architecture/runtime-config.md)
