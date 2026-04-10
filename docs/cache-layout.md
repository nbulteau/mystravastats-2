# Cache Layout For Developers

This page explains how MyStravaStats stores Strava data on disk.

## Default Root Directory

By default, the root cache directory is:

```text
strava-cache/
```

It can be overridden with:

```text
STRAVA_CACHE_PATH
```

## Authentication File

At the root of the cache directory:

```text
strava-cache/.strava
```

This file contains:
- `clientId`
- `clientSecret`
- `useCache`

## Main Directory Structure

Typical layout:

```text
strava-cache/
  .strava
  strava-<clientId>/
    athlete-<clientId>.json
    strava-<clientId>-2010/
      activities-<clientId>-2010.json
      stream-<activityId>
      stravaActivity-<activityId>
    strava-<clientId>-2011/
      activities-<clientId>-2011.json
      stream-<activityId>
      stravaActivity-<activityId>
    ...
```

## File Types

### Athlete file

Path:

```text
strava-<clientId>/athlete-<clientId>.json
```

Purpose:
- stores athlete profile data

### Yearly activities file

Path:

```text
strava-<clientId>/strava-<clientId>-<year>/activities-<clientId>-<year>.json
```

Purpose:
- stores the list of activities for a given year

### Stream file

Path:

```text
strava-<clientId>/strava-<clientId>-<year>/stream-<activityId>
```

Purpose:
- stores the stream payload for one activity
- used for detailed charts and best-effort calculations

### Detailed activity file

Path:

```text
strava-<clientId>/strava-<clientId>-<year>/stravaActivity-<activityId>
```

Purpose:
- stores detailed activity data fetched separately from the main activity list

## How The Cache Is Used

Typical usage flow:
1. read `.strava`
2. locate the athlete directory
3. load the athlete file if present
4. load yearly activity files
5. load detailed activity or stream files on demand or during cache warming

## Why The Layout Looks Like This

The cache is split by:
- client
- year
- activity

That helps:
- partial refreshes
- progressive synchronization
- reduced repeated API calls
- easier debugging when a specific year or activity is problematic

## Developer Tips

- If one activity looks wrong, inspect its yearly directory first.
- If yearly data exists but detailed charts do not, stream files may still be missing.
- If the app behaves as if no cache exists, verify `STRAVA_CACHE_PATH`.
- If you want a clean import, keep a backup before removing cache content manually.
