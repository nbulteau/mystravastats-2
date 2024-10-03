# mystravastats-2

## Launch mystravastats

### Strava access

All calls to the Strava API require an access_token defining the athlete and application making the call.
Any registered Strava user can obtain an access_token by first creating an application
at <https://www.strava.com/settings/api>.

The Strava API application settings page provides *mandatory parameters* for My Strava Stats:

* clientId: your application’s ID.
* clientSecret: your client secret.

Create a directory 'strava-cache' with a '.stava' file in. Put your clientId and your clientSecret.

```shell
mkdir strava-cache
cd strava-cache
echo "clientId=YOUR_CLIENT_ID" > .strava
echo "clientSecret=YOUR_CLIENT_SECRET" >> .strava
export STRAVA_CACHE_PATH=$(pwd)
```

### Get activities from Strava

Activities are download in a local directory (strava-cache), in that way only new and missing ones are downloaded from Strava.
The first time you use My Strava Stats it will attempt to collect activities from 2010 to now.
Due to rate limitations (100 requests every 15 minutes, with up to 1,000 requests per day) it may be necessary to do it
in several attempts. (<https://developers.strava.com/docs/rate-limits/>)

Note : If you do not provide your Client Secret MyStravaStats will use locally downloaded activities.

A browser will open a browser on the Strava consent screen.
If browser does not open, copy/past URL from your terminal in a browser to allow mystravastats to access your Strava
data.
This URL will look like :

```link
https://www.strava.com/api/v3/oauth/authorize?client_id=[YOUR_CLIENT_ID]&response_type=code&redirect_uri=http://localhost/exchange_token&approval_prompt=force&scope=read_all,profile:read_all,activity:read_all
```

Login to Strava then click 'Authorize' and tick the required permissions if needed.

## Setting Up Environment Variables

Before running the `docker-compose` command, you need to set the `STRAVA_CACHE_PATH` environment variable. You can do this by creating a `.env` file in the root directory of your project with the following content:

```shell
export STRAVA_CACHE_PATH=[path to the strava-cache directory]
```

## build command

```shell
docker-compose build
```

* "back end" container will be built.
* "front end" container will be built.

## run command

```shell
export STRAVA_CACHE_PATH=[path to the strava-cache directory]
docker-compose up
```

Open link in a browser : <http://localhost/>
