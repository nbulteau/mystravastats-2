# Route anti-retrace check

The legacy target-generation manual script is retired for the public API.

Anti-retrace is not a hard Strava Art invariant anymore. For public Draw art generation, the goal is to preserve the user model first; opposite-axis traversal and axis reuse are acceptable when they materially improve the visual match.

Keep validating strict anti-retrace behavior for classic sport-loop generation and the internal route explorer through backend route-engine tests. For Strava Art, validate that retrace/backtracking is surfaced as rideability context through `ART_FIT_RETRACE_ALLOWED` and does not override a better `Art fit` candidate.
