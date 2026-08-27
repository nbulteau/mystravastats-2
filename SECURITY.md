# Security Policy

My Activity Stats is a local-first application that processes personal activity,
health and location data. Its default deployment is intended for one user on a
trusted workstation, not as an Internet-facing service.

## Supported version

Security fixes are applied to the current default branch. Older tags are not
maintained separately.

## Deployment expectations

- Keep the frontend, Go/Kotlin backend and OSRM service bound to loopback unless
  an authenticated reverse proxy is deliberately added.
- Configure `CORS_ALLOWED_ORIGINS` explicitly when the browser UI is served from
  a non-default origin. Browser mutations from other origins are rejected.
- Treat `.env`, `strava-cache/.strava`, `.strava-token.json`, FIT/GPX files and
  generated backups as sensitive. Do not commit or publish them.
- Restrict filesystem permissions on mounted source directories and backups.
- Do not expose the OSRM control endpoints or Docker socket to untrusted users.

The application does not currently provide multi-user authentication or
authorization. Exposing it beyond the local machine requires an external access
control layer and TLS.

## Reporting a vulnerability

Report suspected vulnerabilities privately to the repository owner through a
private repository channel. Include the affected version, reproduction steps,
impact and any proposed mitigation. Avoid attaching real activity, token or
location data; use redacted fixtures instead.

Do not open a public issue before the report has been assessed and a coordinated
fix is available.
