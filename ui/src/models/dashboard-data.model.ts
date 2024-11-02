export class DashboardData {
    averageSpeedByYear: Record<string, number>;
    maxSpeedByYear: Record<string, number>;
    averageDistanceByYear: Record<string, number>;
    maxDistanceByYear: Record<string, number>;
    averageElevationByYear: Record<string, number>;
    maxElevationByYear: Record<string, number>;

    constructor(
        averageSpeedByYear: Record<string, number>,
        maxSpeedByYear: Record<string, number>,
        averageDistanceByYear: Record<string, number>,
        maxDistanceByYear: Record<string, number>,
        averageElevationByYear: Record<string, number>,
        maxElevationByYear: Record<string, number>
    ) {
        this.averageSpeedByYear = averageSpeedByYear;
        this.maxSpeedByYear = maxSpeedByYear;
        this.averageDistanceByYear = averageDistanceByYear;
        this.maxDistanceByYear = maxDistanceByYear;
        this.averageElevationByYear = averageElevationByYear;
        this.maxElevationByYear = maxElevationByYear;
    }
}