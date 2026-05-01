# Route anti-retrace check

The legacy target-generation manual script is retired for the public API.

Anti-retrace remains a routing-engine invariant for Strava Art: outside the explicit start/finish tolerance zone, opposite-axis traversal and excessive axis reuse must stay rejected. Validate this through backend route-engine tests and Strava Art shape-generation scenarios.
