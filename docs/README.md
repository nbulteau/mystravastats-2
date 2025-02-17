# MyStravaStats-2

A tool to calculate and display various statistics on Strava activities.

## Features

This tool scans through activities and provides the following statistics:

* **Best Effort**: Finds the fastest consecutive 1 km, 5 km, 10 km you've run, or the fastest 2 hours, 3 hours in ride activities.
* **Eddington Number**: Calculates the Eddington number for rides, runs, inline skating, and hikes. The Eddington number is the largest number, E, such that you have ridden at least E km on at least E days. [Learn more](https://en.wikipedia.org/wiki/Arthur_Eddington#Eddington_number_for_cycling)
* **Best Cooper (12 min)**: Finds the best effort for the given time (12 minutes) on running activities using a 'sliding window'. [Learn more](https://fr.wikipedia.org/wiki/Test_de_Cooper)
* **Best vVO2max (6 min)**: Finds the best effort for the given time (6 minutes) on running activities using a 'sliding window'. [Learn more](https://en.wikipedia.org/wiki/VVO2max)
* **FTP (Functional Threshold Power)**: Finds the best effort for the given time (1 hour) on bike activities using a 'sliding window'. The easiest way to calculate your FTP is to test your best average power for 20 minutes and then subtract 5%.

And many other statistics.

All statistics can be exported as a CSV file.

## User Interface

<img src="https://github.com/nbulteau/mystravastats-2/blob/main/docs/screen_shoot-1.png?raw=true" width="40%" /> <img src="https://github.com/nbulteau/mystravastats-2/blob/main/docs/screen_shoot-2.png?raw=true" width="40%" />
<img src="https://github.com/nbulteau/mystravastats-2/blob/main/docs/screen_shoot-4.png?raw=true" width="40%" /> <img src="https://github.com/nbulteau/mystravastats-2/blob/main/docs/screen_shoot-7.png?raw=true" width="40%" />
<img src="https://github.com/nbulteau/mystravastats-2/blob/main/docs/screen_shoot-5.png?raw=true" width="40%" /><img src="https://github.com/nbulteau/mystravastats-2/blob/main/docs/screen_shoot-8.png?raw=true" width="40%" />
<img src="https://github.com/nbulteau/mystravastats-2/blob/main/docs/screen_shoot-6.png?raw=true" width="40%" /><img src="https://github.com/nbulteau/mystravastats-2/blob/main/docs/screen_shoot-7.png?raw=true" width="40%" />

Icons made by [Freepik](https://www.freepik.com) from [Flaticon](https://www.flaticon.com).

## Table of Contents

1. [Strava Access](#strava-access)
2. [Get Activities from Strava](#get-activities-from-strava)
3. [Setting Up Environment Variables](#setting-up-environment-variables)
4. [Build Command](#build-command)
5. [Run Command](#run-command)
6. [Provided Statistics](#provided-statistics)
    1. [Global Statistics](#global-statistics)
    2. [Rides (Commute)](#rides-commute)
    3. [Rides (Sport)](#rides-sport)
    4. [Runs](#runs)
    5. [InlineSkate](#inlineskate)
    6. [Hikes](#hikes)

### Strava Access

All calls to the Strava API require an `access_token` defining the athlete and application making the call. Any registered Strava user can obtain an `access_token` by first creating an application at [Strava API Settings](https://www.strava.com/settings/api).

The Strava API application settings page provides *mandatory parameters* for MyStravaStats:

* `clientId`: Your application’s ID.
* `clientSecret`: Your client secret.

Create a directory `strava-cache` with a `.strava` file in it. Put your `clientId` and `clientSecret` in the file.

```shell
mkdir strava-cache
cd strava-cache
echo "clientId=[YOUR_CLIENT_ID]" > .strava
echo "clientSecret=[YOUR_CLIENT_SECRET]" >> .strava
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

## build command

You need to have docker on your computer : it prevent you to install all the developpment environement stuffs.

Launch the proper script :

### MacOS

```shell
./build-macos.zsh
```

### Ubuntu

```shell
./build-ubuntu.sh
```

🔄 Change owner of mystravastats to current user: 'sudo chown $(whoami):$(whoami) mystravastats'"

```shell
sudo chown $(whoami):$(whoami) mystravastats
````

### Windows

```shell
./build-windows.ps1
```

## Setting Up Environment Variables

Before running the run command, you need to set the `STRAVA_CACHE_PATH` environment variable. You can do this by creating a `.env` file in the root directory of your project with the following content:

```shell
STRAVA_CACHE_PATH=[path to the strava-cache directory]
```

You can use .env file

```shell
echo STRAVA_CACHE_PATH=[path to the strava-cache directory] > .env
docker compose build
```

## run command

Launch the proper script :

### MacOS or Ubuntu

```shell
./mystravastats
```

### Windows

```shell
mystravastats.exe
```

Open link in a browser : <http://localhost:8080/>

## Provided Statistics

### Global Statistics

| Global Statistics  ||
|--------------------|---|
| Nb activities      | Total of all activities.|
| Nb actives days    | Number of active days for all activities.|
| Max streak         | Max streak of activities for consecutive days.|
| Most active month. | The most active month of the year.|

### Rides (commute)

| Rides (commute)   ||
|-------------------|---|
| Nb activities     | Total of all commute rides.|
| Nb actives days   | Number of active days for all commute rides.|
| Max streak        | Max streak of commute rides for consecutive days.|
| Total distance    | Total elevation accumulated on all commute rides.|
| Total elevation   | Total elevation accumulated on all commute rides.|
| Max distance      | Max distance calculated by Strava for commute rides.|
| Max elevation     | Max elevation calculated by Strava for commute rides.|
| Max moving time   | Max moving time for commute rides. Moving time, is a measure of how long you were active. Strava attempt to calculate this based on the GPS locations, distance, and speed of your activity.|
| Most active month | The most active month of the year for commute rides.|
| Eddington number  | The Eddington number in the context of cycling is defined as the maximum number E such that the cyclist has cycled E km on E days.|

### Rides (sport)

| Rides (sport)           ||
|-------------------------| --- |
| Nb activities           | Total of all bike rides.|
| Nb actives days         | Number of active days for all bike rides.|
| Max streak              | Max streak of bike rides for a consecutive days. |
| Total distance          | Total elevation accumulated on all bike rides. |
| Total elevation         | Total elevation accumulated on all bike rides. |
| Max distance            | Max distance calculated by Strava for bike rides.|
| Max elevation           | Max elevation calculated by Strava for bike rides.|
| Max moving time         | Max moving time for bike rides. Moving time, is a measure of how long you were active. Strava attempt to calculate this based on the GPS locations, distance, and speed of your activity.|
| Most active month       | The most active month of the year for bike rides. |
| Eddington number        | The Eddington number in the context of cycling is defined as the maximum number E such that the cyclist has cycled E km on E days.|
| Max speed               | Max speed calculated by Strava for bike rides.|
| Max moving time         | Max moving time calculated by Strava for bike rides|
| Best 250 m              | Sliding window best effort for a given distance.|
| Best 500 m              | Sliding window best effort for a given distance.|
| Best 1000 m             | Sliding window best effort for a given distance.|
| Best 5 km               | Sliding window best effort for a given distance.|
| Best 10 km              | Sliding window best effort for a given distance.|
| Best 20 km              | Sliding window best effort for a given distance.|  
| Best 50 km              | Sliding window best effort for a given distance.|  
| Best 100 km             | Sliding window best effort for a given distance.|  
| Best 30 min             | Sliding window best effort for a given time.|
| Best 1 h                | Sliding window best effort for a given time.|
| Best 2 h                | Sliding window best effort for a given time.|  
| Best 3 h                | Sliding window best effort for a given time.|
| Best 4 h                | Sliding window best effort for a given time.|  
| Best 5 h                | Sliding window best effort for a given time.|  
| Max gradient for 250 m  | Sliding window max gradient for a given distance.|
| Max gradient for 500 m  | Sliding window max gradient for a given distance.|
| Max gradient for 1000 m | Sliding window max gradient for a given distance.|
| Max gradient for 5 km   | Sliding window max gradient for a given distance.|
| Max gradient for 10 km  | Sliding window max gradient for a given distance.|
| Max gradient for 20 km  | Sliding window max gradient for a given distance.|

### Runs

| Runs ||
|---|--|
| Nb activities | Total of all bike rides.|
| Nb actives days | Number of active days for all running.|
| Max streak | Max streak of bike rides for a running days.|
| Total distance | Total elevation accumulated on all running.|
| Total elevation | Total elevation accumulated on all running.|
| Max distance | Max distance calculated by Strava for running.|
| Max elevation | Max elevation calculated by Strava for running.|
| Max moving time | Max moving time for running. Moving time, is a measure of how long you were active. Strava attempt to calculate this based on the GPS locations, distance, and speed of your activity.|
| Most active month | The most active month of the year for running.|
| Eddington number | The Eddington number in the context of running is defined as the maximum number E such that the runner has run E km on E days.|
| Best Cooper (12 min) | best effort for the given time (12 minutes) on running activities|
| Best vVO2max (6 min) | best effort for the given time (6 minutes) on running activities|
| Best 200 m | Sliding window best effort for a given distance.|
| Best 400 m | Sliding window best effort for a given distance.|
| Best 1000 m | Sliding window best effort for a given distance.|
| Best 5000 m | Sliding window best effort for a given distance.|
| Best 10000 m | Sliding window best effort for a given distance.|
| Best half Marathon | Sliding window best effort for a given distance.|
| Best Marathon | Sliding window best effort for a given distance.|
| Best 1 h | Sliding window best effort for a given time.|
| Best 2 h | Sliding window best effort for a given time.|
| Best 3 h | Sliding window best effort for a given time.|
| Best 4 h | Sliding window best effort for a given time.|
| Best 5 h | Sliding window best effort for a given time.|
| Best 6 h | Sliding window best effort for a given time.|

### InlineSkate

| InlineSkate        ||
|--------------------| --- |
| Nb activities      | Total of all InlineSkate rides.|
| Nb actives days    | Number of active days for all InlineSkate rides.|
| Max streak         | Max streak of InlineSkate rides for a consecutive days. |
| Total distance     | Total elevation accumulated on all InlineSkate rides. |
| Total elevation    | Total elevation accumulated on all InlineSkate rides. |
| Max distance       | Max distance calculated by Strava for InlineSkate rides.|
| Max elevation      | Max elevation calculated by Strava for InlineSkate rides.|
| Max moving time    | Max moving time for InlineSkate rides. Moving time, is a measure of how long you were active. Strava attempt to calculate this based on the GPS locations, distance, and speed of your activity.|
| Most active month  | The most active month of the year for InlineSkate rides. |
| Eddington number   | The Eddington number in the context of InlineSkate is defined as the maximum number E such that the cyclist has cycled E km on E days.|
| Max speed          | Max speed calculated by Strava for InlineSkate rides.|
| Max moving time    | Max moving time calculated by Strava for InlineSkate rides|
| Best 200 m         | Sliding window best effort for a given distance.|
| Best 400 m         | Sliding window best effort for a given distance.|
| Best 1000 m        | Sliding window best effort for a given distance.|
| Best 10000 m       | Sliding window best effort for a given distance.|
| Best half Marathon | Sliding window best effort for a given distance.|
| Best Marathon      | Sliding window best effort for a given distance.|
| Best 1 h           | Sliding window best effort for a given time.|
| Best 2 h           | Sliding window best effort for a given time.|  
| Best 3 h           | Sliding window best effort for a given time.|
| Best 4 h           | Sliding window best effort for a given time.|  

### Hikes

| Hikes ||
|---|--|
| Nb activities | Total of all hikes.|
| Nb actives days | Number of active days for all hikes.|
| Max streak | Max streak of hikes for consecutive days.|
| Total distance | Total elevation accumulated on all hikes.|
| Total elevation | Total elevation accumulated on all hikes.|
| Max distance | Max distance calculated by Strava for hikes.|
| Max elevation | Max elevation calculated by Strava for hikes.|
| Max moving time | Max moving time for hikes. Moving time, is a measure of how long you were active. Strava attempt to calculate this based on the GPS locations, distance, and speed of your activity.|
| Most active month | The most active month of the year for hikes.|
| Eddington number | The Eddington number in the context of cycling is defined as the maximum number E such that the cyclist has cycled E km on E days.|
| Max distance in a day | Max walked distance in a day for hikes.|
| Max elevation in a day | Max elevation in a day for hikes.|
