export class DashboardData {
    nbActivities: Record<string, number>;
    totalDistanceByYear: Record<string, number>;
    averageDistanceByYear: Record<string, number>;
    maxDistanceByYear: Record<string, number>;
    totalElevationByYear: Record<string, number>;
    averageElevationByYear: Record<string, number>;
    maxElevationByYear: Record<string, number>;
    averageSpeedByYear: Record<string, number>;
    maxSpeedByYear: Record<string, number>;
    averageHeartRateByYear: Record<string, number>;
    maxHeartRateByYear: Record<string, number>;
    averageWattsByYear: Record<string, number>;
    maxWattsByYear: Record<string, number>;
    averageCadence: Array<Array<number>>;

    constructor(
        nbActivities: Record<string, number>,
        totalDistanceByYear: Record<string, number>,
        averageDistanceByYear: Record<string, number>,
        maxDistanceByYear: Record<string, number>,
        totalElevationByYear: Record<string, number>,
        averageElevationByYear: Record<string, number>,
        maxElevationByYear: Record<string, number>,
        averageSpeedByYear: Record<string, number>,
        maxSpeedByYear: Record<string, number>,
        averageHeartRateByYear: Record<string, number>,
        maxHeartRateByYear: Record<string, number>,
        averageWattsByYear: Record<string, number>,
        maxWattsByYear: Record<string, number>,
        averageCadence: Array<Array<number>>
    ) {
        this.nbActivities = nbActivities;
        this.totalDistanceByYear = totalDistanceByYear;
        this.averageDistanceByYear = averageDistanceByYear;
        this.maxDistanceByYear = maxDistanceByYear;
        this.totalElevationByYear = totalElevationByYear;
        this.averageElevationByYear = averageElevationByYear;
        this.maxElevationByYear = maxElevationByYear;
        this.averageSpeedByYear = averageSpeedByYear;
        this.maxSpeedByYear = maxSpeedByYear;
        this.averageHeartRateByYear = averageHeartRateByYear;
        this.maxHeartRateByYear = maxHeartRateByYear;
        this.averageWattsByYear = averageWattsByYear;
        this.maxWattsByYear = maxWattsByYear;
        this.averageCadence = averageCadence;
    }
}