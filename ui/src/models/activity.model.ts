
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
export class DetailedActivity {
    averageSpeed: number;
    averageCadence: number;
    averageHeartrate: number;
    maxHeartrate: number;
    averageWatts: number;
    commute: boolean;
    distance: number;
    deviceWatts: boolean;
    elapsedTime: number;
    elevHigh: number;
    id: number;
    kilojoules: number;
    maxSpeed: number;
    movingTime: number;
    name: string;
    startDate: string;
    startDateLocal: string;
    startLatlng: number[] | null;
    totalDescent: number;
    totalElevationGain: number;
    type: string;
    weightedAverageWatts: number;
    stream: Stream;
    activityEfforts: ActivityEffort[] = [];

    constructor(
        averageSpeed: number,
        averageCadence: number,
        averageHeartrate: number,
        maxHeartrate: number,
        averageWatts: number,
        commute: boolean,
        distance: number,
        deviceWatts: boolean,
        elapsedTime: number,
        elevHigh: number,
        id: number,
        kilojoules: number,
        maxSpeed: number,
        movingTime: number,
        name: string,
        startDate: string,
        startDateLocal: string,
        startLatlng: number[] | null,
        totalDescent: number,
        totalElevationGain: number,
        type: string,
        weightedAverageWatts: number,
        stream: Stream,
        activityEfforts:  ActivityEffort[]
    ) {
        this.averageSpeed = averageSpeed;
        this.averageCadence = averageCadence;
        this.averageHeartrate = averageHeartrate;
        this.maxHeartrate = maxHeartrate;
        this.averageWatts = averageWatts;
        this.commute = commute;
        this.distance = distance;
        this.deviceWatts = deviceWatts;
        this.elapsedTime = elapsedTime;
        this.elevHigh = elevHigh;
        this.id = id;
        this.kilojoules = kilojoules;
        this.maxSpeed = maxSpeed;
        this.movingTime = movingTime;
        this.name = name;
        this.startDate = startDate;
        this.startDateLocal = startDateLocal;
        this.startLatlng = startLatlng;
        this.totalDescent = totalDescent;
        this.totalElevationGain = totalElevationGain;
        this.type = type;
        this.weightedAverageWatts = weightedAverageWatts;
        this.stream = stream;
        this.activityEfforts = activityEfforts;
    }
}

class Stream {
    distance: number[];
    time: number[];
    moving: boolean[] | null;
    altitude: number[] | null;
    latlng: number[][] | null;
    watts: number[] | null;
    velocitySmooth?: number[] | null;

    constructor(
        distance: number[],
        time: number[],
        moving: boolean[] | null,
        altitude: number[] | null,
        latlng: number[][] | null,
        watts: number[] | null,
        velocitySmooth?: number[]| null,
    ) {
        this.distance = distance;
        this.time = time;
        this.latlng = latlng;
        this.moving = moving;
        this.altitude = altitude;
        this.watts = watts;
        this.velocitySmooth = velocitySmooth;
    }
}

export class ActivityEffort {
    key: string;
    distance: number;
    seconds: number;
    deltaAltitude: number;
    idxStart: number;
    idxEnd: number;
    averagePower: number | null = null;
    description: string = '';

    constructor(
        key: string,
        distance: number,
        seconds: number,
        deltaAltitude: number,
        idxStart: number,
        idxEnd: number,
        averagePower: number | null,
        description: string,
    ) {
        this.key = key;
        this.distance = distance;
        this.seconds = seconds;
        this.deltaAltitude = deltaAltitude;
        this.idxStart = idxStart;
        this.idxEnd = idxEnd;
        this.averagePower = averagePower;
        this.description = description;
    }
}
