# Route anti-retrace check

The legacy target-generation manual script is retired for the public API.

Anti-retrace is not a hard Strava Art invariant anymore. For public Draw art generation, the goal is to preserve the user model first; opposite-axis traversal and axis reuse are acceptable when they materially improve the visual match.

Keep validating strict anti-retrace behavior for classic sport-loop generation and the internal route explorer through backend route-engine tests. For Strava Art, validate that retrace/backtracking is surfaced as a rideability diagnostic and does not override a better `Art fit` candidate.
