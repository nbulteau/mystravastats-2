# Strava OAuth Setup

To access live Strava data, MyStravaStats needs a Strava API application linked to your own Strava account.

Create it from [Strava API Settings](https://www.strava.com/settings/api).

Strava does not expose a public API endpoint to create this developer application
or retrieve its secret. That first application-creation step remains manual.
MyStravaStats can automate everything after that: local `.strava` creation,
OAuth authorization, callback handling, token exchange and refresh-token reuse.

## 1. Create A Strava Application

On the Strava API settings page, create an application and keep:

- `clientId`
- `clientSecret`

The descriptive fields can use any valid values for your local use.

For local onboarding, set `Authorization Callback Domain` to:

```text
127.0.0.1
```

If you use the legacy backend OAuth flow directly, `localhost` is also supported
by the existing backend default.

## 2. Recommended Assistant

From the repository root:

```shell
node scripts/setup-strava-oauth.mjs
```

The assistant:

1. asks for the Strava cache directory,
2. opens the Strava API settings page so you can create or inspect the app,
3. writes `.strava` with `clientId`, `clientSecret` and `useCache`,
4. opens the Strava OAuth authorization page,
5. receives the local callback,
6. exchanges the short-lived code for tokens,
7. validates `/api/v3/athlete`,
8. stores `.strava-token.json` for future token refresh.

Use a non-default cache directory with:

```shell
node scripts/setup-strava-oauth.mjs --cache /your/custom/cache
```

## 3. Locate The Cache Directory

By default, MyStravaStats uses:

```text
strava-cache
```

If `STRAVA_CACHE_PATH` is defined, the application uses that directory instead.

## 4. Create `.strava`

Inside the cache directory, create:

```text
strava-cache/.strava
```

With a custom cache path:

```text
/your/custom/cache/.strava
```

## 5. Configure Credentials

Typical `.strava` content:

```properties
clientId=YOUR_CLIENT_ID
clientSecret=YOUR_CLIENT_SECRET
useCache=false
```

`clientId` is the Strava application identifier.

`clientSecret` is required when the application must download fresh data from Strava.

`useCache=false` allows live Strava refresh. `useCache=true` prefers local cached data and avoids a live bootstrap.

The OAuth assistant also creates:

```text
strava-cache/.strava-token.json
```

This file contains the latest short-lived access token and refresh token. Keep it
private like `.strava`; it should never be committed.

## Recommended Values

First import:

```properties
clientId=YOUR_CLIENT_ID
clientSecret=YOUR_CLIENT_SECRET
useCache=false
```

Offline or cache-first usage after data has already been downloaded:

```properties
clientId=YOUR_CLIENT_ID
clientSecret=YOUR_CLIENT_SECRET
useCache=true
```

## First Launch

On the first Strava-enabled launch:

1. MyStravaStats reads `.strava`.
2. It reuses `.strava-token.json` when a valid token is available.
3. If needed, it opens the Strava authorization screen.
4. You approve access.
5. Strava redirects back to a local callback URL.
6. MyStravaStats exchanges the authorization code for an access token and refresh token.
7. Activities start downloading into the local cache.

If the browser does not open automatically, the authorization URL is printed in the terminal.

## Notes

- The first import may take time if you have many years of activities.
- Streams and detailed activities may be filled progressively.
- Because of Strava rate limits, a full import may require more than one run.
- If `clientSecret` is missing, MyStravaStats can only rely on already cached data.
- If `useCache=true` but the cache is empty, the app cannot perform a full live import.
- If OAuth fails immediately after approval, check the Strava app `Authorization Callback Domain`.
- If you revoke the app from Strava settings, delete `.strava-token.json` and run the assistant again.

Related docs:

- [OAuth and Cache Troubleshooting](./strava-oauth-troubleshooting.md)
- [Cache Layout](../architecture/cache-layout.md)
