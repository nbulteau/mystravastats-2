# Route surface scoring check

This manual target-generation check is retired for the public API.

Strava Art still exposes surface-related reasons and `roadFitness` scores when the routing engine can compute them. The active validation path is now shape generation (`POST /api/routes/generate/shape`) plus backend parity tests for route-type and surface reason mapping.
