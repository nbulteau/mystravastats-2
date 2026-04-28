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

Kotlin supports GPX input through:

```text
GPX_FILES_PATH
```

The Go backend currently reports `GPX_FILES_PATH` in diagnostics but does not expose GPX activities from it.

Related docs:

- [Backend Capability Matrix](../architecture/backend-capability-matrix.md)
- [Runtime Configuration](../architecture/runtime-config.md)
