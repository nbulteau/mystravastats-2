# FIT And GPX Sources

MyStravaStats can work from local activity files in addition to Strava.

## FIT

Both Go and Kotlin support FIT input through:

```text
FIT_FILES_PATH
```

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
