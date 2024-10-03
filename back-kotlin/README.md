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

### Swagger

<http://localhost:8080/api/swagger-ui/index.html>

### Actuator

<http://localhost:8080/api/actuator>

### Health check

<http://localhost:8080/api/actuator/health>

### Memory usage : get the max heap memory

<http://localhost:8080/api/actuator/metrics/jvm.memory.max?tag=area:heap>
