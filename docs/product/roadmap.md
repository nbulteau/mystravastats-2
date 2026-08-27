# Product Roadmap

This roadmap orders product ideas by user value, architectural readiness and
delivery risk. Dates are intentionally omitted; each item should move only when
its outcome and parity checks are testable.

## Now — reliability and useful guidance

### Athlete goals and progress

Let athletes define annual or rolling goals for distance, elevation, time and
activity count, then compare actual pace with the required pace.

Success means goals survive restarts, work with every source mode and clearly
identify whether the athlete is ahead or behind schedule.

### Training load and recovery trends

Build transparent weekly load, monotony and recovery indicators from available
heart-rate, power and perceived-effort signals. Always expose missing-data and
calculation diagnostics rather than presenting false precision.

Success means equivalent Go/Kotlin results on shared fixtures and explanations
for every displayed score.

### Predictive gear maintenance

Extend maintenance history with configurable distance/time thresholds and a
forecast of the next service date.

Success means alerts can be dismissed or completed, remain local and work when
gear metadata is incomplete.

## Next — deeper exploration

### Climb progression workspace

Combine the climb catalog, segment efforts and personal-record timeline into a
single progression view with comparable attempts and data-quality warnings.

### Asynchronous GPS Art generation

Move expensive route exploration to cancellable background jobs with progress,
stable request identifiers and downloadable diagnostics. Preserve strict
anti-retrace and the explicit start/finish hub rules.

### Offline-first installable frontend

Add PWA installation and read-only offline access to previously synchronized
dashboards and activities. Sensitive caches must remain scoped to the local
device and mutation queues must never replay ambiguously.

## Later — reach and enrichment

### Internationalization

Externalize frontend copy and diagnostics labels, starting with French and
English while keeping backend diagnostic codes stable.

### Broader European climb catalog

Expand the catalog country by country using sourced, dated records and strict
Go/Kotlin catalog parity. Follow the tracked work in `docs/TODO.md`.

### Comparative seasons and cohorts

Offer privacy-preserving comparison between personal seasons and optional,
explicitly imported reference datasets. No social or cloud upload should be
implicit.

## Delivery rules

Every roadmap feature needs a small vertical slice, an explicit API contract,
Go/Kotlin parity where applicable, representative fixtures, accessibility checks
and user-visible diagnostics for partial data.
