
export class Activity  {
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

    constructor(
        name: string,
        type: string,
        link: string,
        distance: number,
        elapsedTime: number,
        totalElevationGain: number,
        totalDescent: number,
        averageSpeed: number,
        bestTimeForDistanceFor1000m: string,
        bestElevationForDistanceFor500m: string,
        bestElevationForDistanceFor1000m: string,
        date: string,
        averageWatts: string,
        weightedAverageWatts: string,
        bestPowerFor20minutes: string,
        bestPowerFor60minutes: string,
        ftp: string
    ) {
        this.name = name;
        this.type = type;
        this.link = link;
        this.distance = distance;
        this.elapsedTime = elapsedTime;
        this.totalElevationGain = totalElevationGain;
        this.totalDescent = totalDescent;
        this.averageSpeed = averageSpeed;
        this.bestTimeForDistanceFor1000m = bestTimeForDistanceFor1000m;
        this.bestElevationForDistanceFor500m = bestElevationForDistanceFor500m;
        this.bestElevationForDistanceFor1000m = bestElevationForDistanceFor1000m;
        this.date = date;
        this.averageWatts = averageWatts;
        this.weightedAverageWatts = weightedAverageWatts;
        this.bestPowerFor20minutes = bestPowerFor20minutes;
        this.bestPowerFor60minutes = bestPowerFor60minutes;
        this.ftp = ftp;
    }
}

// Define the DetailedActivity class that extends Activity class
export class DetailedActivity extends Activity {

    constructor(
        name: string,
        type: string,
        link: string,
        distance: number,
        elapsedTime: number,
        totalElevationGain: number,
        totalDescent: number,
        averageSpeed: number,
        bestTimeForDistanceFor1000m: string,
        bestElevationForDistanceFor500m: string,
        bestElevationForDistanceFor1000m: string,
        date: string,
        averageWatts: string,
        weightedAverageWatts: string,
        bestPowerFor20minutes: string,
        bestPowerFor60minutes: string,
        ftp: string,

        public stream: Stream,
    ) {
        super(
            name,
            type,
            link,
            distance,
            elapsedTime,
            totalElevationGain,
            totalDescent,
            averageSpeed,
            bestTimeForDistanceFor1000m,
            bestElevationForDistanceFor500m,
            bestElevationForDistanceFor1000m,
            date,
            averageWatts,
            weightedAverageWatts,
            bestPowerFor20minutes,
            bestPowerFor60minutes,
            ftp
        );
        this.stream = stream;
    }
}

class Stream {
    distance: number[];
    time: number[];
    moving?: number[];
    altitude?: number[];
    latitudeLongitude?: number[][];
    watts?: number[];

    constructor(
        distance: number[],
        time: number[],
        moving?: number[],
        altitude?: number[],
        latitudeLongitude?: number[][],
        watts?: number[]
    ) {
        this.distance = distance;
        this.time = time;
        this.moving = moving;
        this.altitude = altitude;
        this.latitudeLongitude = latitudeLongitude;
        this.watts = watts;
    }
}
