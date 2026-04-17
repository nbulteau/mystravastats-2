export class DashboardData {
    nbActivitiesByYear: Record<string, number>;
    activeDaysByYear: Record<string, number>;
    consistencyByYear: Record<string, number>;
    movingTimeByYear: Record<string, number>;
    totalDistanceByYear: Record<string, number>;
    averageDistanceByYear: Record<string, number>;
    maxDistanceByYear: Record<string, number>;
    totalElevationByYear: Record<string, number>;
    averageElevationByYear: Record<string, number>;
    maxElevationByYear: Record<string, number>;
    elevationEfficiencyByYear: Record<string, number>;
    averageSpeedByYear: Record<string, number>;
    maxSpeedByYear: Record<string, number>;
    averageHeartRateByYear: Record<string, number>;
    maxHeartRateByYear: Record<string, number>;
    averageWattsByYear: Record<string, number>;
    maxWattsByYear: Record<string, number>;
    averageCadenceByYear: Array<Array<number>>;

    constructor(
        nbActivitiesByYear: Record<string, number>,
        activeDaysByYear: Record<string, number>,
        consistencyByYear: Record<string, number>,
        movingTimeByYear: Record<string, number>,
        totalDistanceByYear: Record<string, number>,
        averageDistanceByYear: Record<string, number>,
        maxDistanceByYear: Record<string, number>,
        totalElevationByYear: Record<string, number>,
        averageElevationByYear: Record<string, number>,
        maxElevationByYear: Record<string, number>,
        elevationEfficiencyByYear: Record<string, number>,
        averageSpeedByYear: Record<string, number>,
        maxSpeedByYear: Record<string, number>,
        averageHeartRateByYear: Record<string, number>,
        maxHeartRateByYear: Record<string, number>,
        averageWattsByYear: Record<string, number>,
        maxWattsByYear: Record<string, number>,
        averageCadenceByYear: Array<Array<number>>
    ) {
        this.nbActivitiesByYear = nbActivitiesByYear;
        this.activeDaysByYear = activeDaysByYear;
        this.consistencyByYear = consistencyByYear;
        this.movingTimeByYear = movingTimeByYear;
        this.totalDistanceByYear = totalDistanceByYear;
        this.averageDistanceByYear = averageDistanceByYear;
        this.maxDistanceByYear = maxDistanceByYear;
        this.totalElevationByYear = totalElevationByYear;
        this.averageElevationByYear = averageElevationByYear;
        this.maxElevationByYear = maxElevationByYear;
        this.elevationEfficiencyByYear = elevationEfficiencyByYear;
        this.averageSpeedByYear = averageSpeedByYear;
        this.maxSpeedByYear = maxSpeedByYear;
        this.averageHeartRateByYear = averageHeartRateByYear;
        this.maxHeartRateByYear = maxHeartRateByYear;
        this.averageWattsByYear = averageWattsByYear;
        this.maxWattsByYear = maxWattsByYear;
        this.averageCadenceByYear = averageCadenceByYear;
    }
}
