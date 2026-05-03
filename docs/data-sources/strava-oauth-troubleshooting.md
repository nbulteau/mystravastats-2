# OAuth and Cache Troubleshooting

This page explains the most common issues when MyStravaStats tries to connect to Strava or read local cached data.

## Problem: The browser does not open

What happens:
- MyStravaStats tries to open the Strava authorization page automatically

What to do:
- copy the authorization URL printed in the terminal
- paste it into your browser manually
- log into Strava
- approve the requested permissions

## Problem: Strava authorization page opens but login does not complete

Check:
- the callback port is not already used by another process
- the Strava application `Authorization Callback Domain` matches the local flow (`127.0.0.1` for the setup assistant, `localhost` for the legacy backend default)
- no browser extension or local security tool is blocking the redirect
- Docker or the local application is really running

Symptoms:
- you approve access, but the app keeps waiting
- the local callback page does not load

## Problem: `.strava` file is missing

The application expects a `.strava` file in the cache directory.

Default location:
- `strava-cache/.strava`

If you use `STRAVA_CACHE_PATH`, the file must exist in that directory instead.

## Problem: `clientId` or `clientSecret` is wrong

Symptoms:
- OAuth fails
- Strava API calls fail
- the app cannot refresh activities

What to check:
- the values were copied exactly from the Strava API settings page
- there are no extra spaces around the values
- the file uses `key=value` lines

## Problem: OAuth keeps reopening on every launch

What it usually means:
- `.strava-token.json` is missing, unreadable or expired without a refresh token
- Strava returned a new refresh token but the application could not save it
- the app was revoked from Strava settings

What to do:
- run `node scripts/setup-strava-oauth.mjs`
- verify the cache directory is writable
- keep `.strava-token.json` private and outside git

## Problem: `useCache=true` but data is incomplete

What it means:
- the app is being told to prefer local cached data
- it may skip live Strava bootstrap logic

What to do:
- if you want fresh Strava downloads, set `useCache=false`
- if the cache is incomplete, run again with valid `clientId` and `clientSecret`

## Problem: First synchronization is slow

This is normal when:
- the athlete has many years of activities
- streams and details are still missing from the local cache
- Strava rate limits are close

The first synchronization is the slowest one.
Later launches are usually much faster because the cache is reused.

## Problem: Strava rate limit reached

Strava imposes API limits.

Typical consequences:
- some years are imported but not all
- some streams are still missing
- the app needs another run later

Reference:
- [Strava rate limits](https://developers.strava.com/docs/rate-limits/)

## Problem: Cache directory is wrong

If the app seems empty or keeps reimporting everything, verify:
- the right cache directory is being used
- `STRAVA_CACHE_PATH` is not pointing to another location
- the files are readable by your current user

## Problem: Cached data exists but some details are missing

That can happen when:
- the main yearly activity files exist
- but detailed activity files or stream files were never downloaded

This is not necessarily corruption.
It often means the cache is partially warm.

## Quick Checklist

Before looking deeper, verify:
- Docker is installed and running
- the app is really running
- `.strava` exists in the correct directory
- `.strava-token.json` exists after a successful OAuth setup
- `clientId` and `clientSecret` are correct
- `useCache` matches the behavior you expect
- the OAuth URL can be opened manually
- you are not currently blocked by Strava limits
