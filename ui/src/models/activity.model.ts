export interface Activity {
    name: string;
    type: string;
    link: string;
    distance: number;
    elapsedTime: number;
    totalElevationGain: number;
    totalDescent: number;
    averageSpeed: number;
    bestTimeForDistanceFor1000m: string;
    bestElevationForDistanceFor500m: string;
    bestElevationForDistanceFor1000m: string;
    date: string;
    averageWatts: string;
    weightedAverageWatts: string;
    bestPowerFor20minutes: string;
    bestPowerFor60minutes: string;
    ftp: string;
}