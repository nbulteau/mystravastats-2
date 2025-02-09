# mystravastats-2

## Launch mystravastats backend

JDK 21 is needed to run mystravastats.

```shell
sdk install java 21.0.4-tem

git clone https://github.com/nbulteau/mystravastats.git
cd mystravastats
./gradlew bootRun    
```

Will download activities from 2010 to now from Strava, then display statistics and charts.

### Launch mystravastats using docker

#### build

First build the docker image.

```shell
git clone https://github.com/nbulteau/mystravastats.git
cd mystravastats
```

```shell
docker buildx build -t mystravastats:latest .
```

#### For mac

```shell
docker buildx build --platform linux/arm64 -t mystravastats:latest .
```

#### launch mystravastats using docker

```shell
docker run --rm -d -p 8080:8080 -p 8090:8090 -v [path to the strava cache]/strava-cache:/app/strava-cache --name mystravastats mystravastats:latest
```

```shell
docker run --rm --platform linux/arm64 -d -p 8080:8080 -p 8090:8090 -v [path to the strava cache]/strava-cache:/app/strava-cache --name mystravastats mystravastats:latest
```

### Using GPX files

mystravastats can work without Strava using the GPX files. Put GPX files in a directory structure 'gpx-xxxxx':

```shell
gpx-nicolas
   |- 2022
     |- XCVF234.gpx
     |- XCVF235.gpx
   |- 2021 
    |- XCVF236.gpx
``` 

Launch mystravastats with providing the GPX repository.

```shell
export GPX_FILES_PATH=[path to the GPX directory]
docker compose up back ui
```

### Using FIT files

mystravastats can work without Strava using the FIT files. Put FIT files in a directory structure 'fit-xxxxx':

```shell
fit-nicolas
   |- 2022
     |- XCVF234.FIT
     |- XCVF235.FIT
   |- 2021
     |- XCVF236.FIT
```

Launch mystravastats with providing the FIT repository.

```shell
export FIT_FILES_PATH=[path to the FIT directory]
docker compose up back ui
```

### Using SRTM Altitude Data

Here’s a breakdown of how to utilize STRM files for GPX or FIT files without altitude data

STRM files are used to store elevation data, specifically for Shuttle Radar Topography Mission (SRTM) data. 
SRTM data is a digital elevation model (DEM) created from radar altimetry measurements. 
It provides topographic information with a resolution of 30 meters (SRTM3) or 1 arc-second (approximately 30 meters) for global coverage. 
The data is stored in HGT (height) files, which contain elevation values in meters relative to the WGS84 datum.

Download the SRTM files from the following link: <https://dwtkns.com/srtm30m/>

create a directory 'srtm30m' in the root directory of the project and put the HGT files in it.

```shell
srtm30m
  |- N47E000.hgt
  |- N47E001.hgt
  |- N47E002.hgt 
  |- N47E003.hgt    
```

### Swagger

<http://localhost:8080/api/swagger-ui/index.html>

### Actuator

<http://localhost:8080/api/actuator>

### Health check

<http://localhost:8080/api/actuator/health>

### Memory usage : get the max heap memory

<http://localhost:8080/api/actuator/metrics/jvm.memory.max?tag=area:heap>
