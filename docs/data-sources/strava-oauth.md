# Strava OAuth Setup

To access live Strava data, MyStravaStats needs a Strava API application linked to your own Strava account.

Create it from [Strava API Settings](https://www.strava.com/settings/api).

## 1. Create A Strava Application

On the Strava API settings page, create an application and keep:

- `clientId`
- `clientSecret`

The descriptive fields can use any valid values for your local use.

## 2. Locate The Cache Directory

By default, MyStravaStats uses:

```text
strava-cache
```

If `STRAVA_CACHE_PATH` is defined, the application uses that directory instead.

## 3. Create `.strava`

Inside the cache directory, create:

```text
strava-cache/.strava
```

With a custom cache path:

```text
/your/custom/cache/.strava
```

## 4. Configure Credentials

Typical `.strava` content:

```properties
clientId=YOUR_CLIENT_ID
clientSecret=YOUR_CLIENT_SECRET
useCache=false
```

`clientId` is the Strava application identifier.

`clientSecret` is required when the application must download fresh data from Strava.

`useCache=false` allows live Strava refresh. `useCache=true` prefers local cached data and avoids a live bootstrap.

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
2. It opens the Strava authorization screen.
3. You approve access.
4. Strava redirects back to a local callback URL.
5. MyStravaStats receives an access token.
6. Activities start downloading into the local cache.

If the browser does not open automatically, the authorization URL is printed in the terminal.

## Notes

- The first import may take time if you have many years of activities.
- Streams and detailed activities may be filled progressively.
- Because of Strava rate limits, a full import may require more than one run.
- If `clientSecret` is missing, MyStravaStats can only rely on already cached data.
- If `useCache=true` but the cache is empty, the app cannot perform a full live import.

Related docs:

- [OAuth and Cache Troubleshooting](./strava-oauth-troubleshooting.md)
- [Cache Layout](../architecture/cache-layout.md)
