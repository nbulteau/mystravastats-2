# MyStravaStats

MyStravaStats is a personal Strava and local activity analytics application.

Start with the documentation index:

- [Documentation](./docs/README.md)
- [Quick start](./docs/getting-started/quick-start.md)
- [Strava OAuth setup](./docs/data-sources/strava-oauth.md)
- [Developer setup](./docs/getting-started/developer-setup.md)

## Strava Enrollment

Strava still requires creating the API application manually from:

```text
https://www.strava.com/settings/api
```

After that one manual step, MyStravaStats can automate the local OAuth flow from
`Diagnostics` > `Data Source` > `Connect Strava`, or from the CLI:

```shell
node scripts/setup-strava-oauth.mjs
```

The assistant creates `strava-cache/.strava`, opens the Strava authorization page,
receives the local callback, validates the athlete endpoint and stores a local
`.strava-token.json` refresh-token cache.
