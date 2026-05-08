# Data Quality Fixtures

This directory contains small anonymized local-provider fixtures used by both
backends to keep data quality diagnostics aligned.

`local-activity-anomalies.json` stores normalized activity records rather than
raw user files. The cases represent FIT and GPX imports after parsing, with
stable synthetic coordinates and no personal data.

Covered categories:

- invalid scalar values and inconsistent timing
- missing local streams
- incomplete stream fields
- GPS outlier jumps
- altitude spikes
- missing sensor samples

`expected-local-activity-anomalies.snapshot.json` is the shared parity snapshot.
It intentionally excludes generated timestamps and compares stable diagnostics:
summary counts, issue category/severity/field triples, correction availability,
and before/after correction impact.
