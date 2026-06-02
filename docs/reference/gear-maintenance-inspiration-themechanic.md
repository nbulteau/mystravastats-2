# Gear maintenance inspiration - The Bike Mechanic

Date: 2026-05-13

Source reviewed:
- https://themechanic.bike/fr
- https://themechanic.bike/fr/features
- https://themechanic.bike/fr/pricing

## Summary

The Bike Mechanic positions gear maintenance as a proactive workshop rather than a passive gear mileage table. The strongest ideas for My Activity Stats are component-level lifecycle tracking, clear due/soon/ok alerts, maintenance history tied to odometer distance, and per-bike calendars that stay current from activity sync.

My Activity Stats already has a useful local base: bikes and shoes, distance coverage, local service records, component tasks, odometer distance, and GPX/FIT/Strava-friendly local storage. The next step is not to copy a SaaS garage, but to turn the Gear tab into a local decision surface: what is due, why, on which bike, and what action should be logged next.

## Inspirations to Keep

- Component lifecycle: track chain, cassette, tires, brake pads, sealant, bearings and drivetrain as individual service items with their own distance or time interval.
- Clear maintenance state: each item should surface `overdue`, `due`, `soon`, or `ok`, with distance/time remaining and last service evidence.
- One-click service logging: a due task should be markable as done at the current odometer, with editable date, note and operation.
- Replacement history: each maintenance event should preserve the component, date, odometer and note so future decisions explain themselves.
- Wheel-specific tracking: tires, sealant, valve cores and wheel truing need front/rear or wheelset-level tracking instead of only bike-level totals.
- Spare parts/inventory: useful, but should start as a lightweight local inventory for parts ready to install, not a shopping or subscription workflow.
- Multi-bike overview: the Gear tab should show the next risky component across all bikes, not force the user to inspect each bike manually.

## Inspirations to Defer or Reject

- Third-party dependency: do not require The Bike Mechanic, Intervals.icu, Strava SaaS, email delivery, or an account system for local maintenance.
- Family sharing and paid plan limits: not relevant to this single-user local analytics app.
- Battery alerts from Garmin/Wahoo via Intervals.icu: valuable, but only if local activity files expose device battery metadata reliably. Track it as future telemetry, not as a core Gear dependency.
- Full inventory management: useful later, but too broad for `FUNC-P1-10`; begin with spare component records linked to maintenance replacements.

## Existing My Activity Stats Coverage

- Gear tab already computes distance, moving time, elevation, activities, monthly distance and best activities per gear.
- Local service records already exist with component, operation, date, odometer distance and note.
- Bike maintenance rules already cover core components such as chain, cassette, brakes, tires, sealant, bottom bracket, bearings and drivetrain.
- The current weakness is prioritization: users can log maintenance, but the page should make the next action unmistakable.

## Proposed Backlog for `FUNC-P1-10`

1. Maintenance priority board:
   Show the top overdue/due/soon tasks across all bikes at the top of Gear, sorted by severity and remaining distance/time.

2. Task evidence:
   For each task, show last service date, odometer at service, distance since service, next due distance and the rule that triggered the status.

3. Component lifecycle cards:
   Add a per-bike component view grouping active components by drivetrain, braking, wheels/tires, suspension and bearings.

4. Wheel and tire split:
   Promote front/rear tire, front/rear sealant, valve cores and wheel truing as first-class maintenance tracks.

5. Replacement flow:
   Add an explicit replacement action separate from generic service, preserving the old component history and starting a new lifecycle at the current odometer.

6. Lightweight spare parts:
   Store local spare parts with component type, purchase date, note and optional target bike; allow a replacement record to consume a spare.

7. Usage forecast:
   Estimate when a task will become due using recent monthly distance for that bike, with labels such as urgent, soon and monitor.

8. Source-aware limits:
   Explain when predictions are weak because gear assignment coverage is low or because the selected year hides lifetime mileage.

