# Route direction check

This check is retired for the public generation API.

Strava Art uses a single Draw art workflow through `POST /api/routes/generate/shape`. It no longer exposes a user-facing global heading constraint, so direction matrix validation is not part of the manual Strava Art checks.

Direction-related reasons may still appear when the internal routing engine reports a relaxation, and both backends keep those diagnostics mapped for parity.
