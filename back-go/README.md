## 
### athlete
http://localhost:8080/api/athlete/me

### activities
http://localhost:8080/api/activities

http://localhost:8080/api/activities?year=2025&activityType=VirtualRide

### statistics

http://localhost:8080/api/statistics

http://localhost:8080/api/statistics?year=2025&activityType=VirtualRide

### charts

http://localhost:8080/api/charts/distance-by-period?activityType=Ride&year=2025&period=MONTHS

http://localhost:8080/api/charts/elevation-by-period?activityType=Ride&year=2025&period=MONTHS

http://localhost:8080/api/charts/average-speed-by-period?activityType=Ride&year=2025&period=MONTHS

### dashboard

http://localhost:8080/api/dashboard/cumulative-data-per-year?activityType=Ride&year=2025



```shell
go get -u ./...
go mod tidy
```