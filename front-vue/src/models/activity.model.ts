
export class Activity  {
    id: number;
    name: string;
    type: string;
    link: string;
    distance: number;
    elapsedTime: number;
    movingTime: number;
    totalElevationGain: number;
    totalDescent: number;
    averageSpeed: number;
    bestSpeedForDistanceFor1000m: number;
    bestElevationForDistanceFor500m: number;
    bestElevationForDistanceFor1000m: number;
    date: string;
    averageWatts: number;
    weightedAverageWatts: number;
    bestPowerFor20minutes: number;
    bestPowerFor60minutes: number;
    ftp: number;

    constructor(
        id: number,
        name: string,
        type: string,
        link: string,
        distance: number,
        elapsedTime: number,
        movingTime: number,
        totalElevationGain: number,
        totalDescent: number,
        averageSpeed: number,
        bestTimeForDistanceFor1000m: number,
        bestElevationForDistanceFor500m: number,
        bestElevationForDistanceFor1000m: number,
        date: string,
        averageWatts: number,
        weightedAverageWatts: number,
        bestPowerFor20minutes: number,
        bestPowerFor60minutes: number,
        ftp: number
    ) {
        this.id = id;
        this.name = name;
        this.type = type;
        this.link = link;
        this.distance = distance;
        this.elapsedTime = elapsedTime;
        this.movingTime = movingTime;
        this.totalElevationGain = totalElevationGain;
        this.totalDescent = totalDescent;
        this.averageSpeed = averageSpeed;
        this.bestSpeedForDistanceFor1000m = bestTimeForDistanceFor1000m;
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

export class ActivityShort {
    id: number;
    name: string;
    type: string;

    constructor(id: number, name: string, type: string) {
        this.id = id;
        this.name = name;
        this.type = type;
    }
}

class Stream {
    distance: number[];
    time: number[];
    moving: boolean[] | null;
    altitude: number[] | null;
    latlng: number[][] | null;
    watts: number[] | null;
    velocitySmooth?: number[] | null | undefined;

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
    id: string;
    label: string;
    distance: number;
    seconds: number;
    deltaAltitude: number;
    idxStart: number;
    idxEnd: number;
    averagePower: number | null = null;
    description: string = '';

    constructor(
        id: string,
        label: string,
        distance: number,
        seconds: number,
        deltaAltitude: number,
        idxStart: number,
        idxEnd: number,
        averagePower: number | null,
        description: string,
    ) {
        this.id = id;
        this.label = label;
        this.distance = distance;
        this.seconds = seconds;
        this.deltaAltitude = deltaAltitude;
        this.idxStart = idxStart;
        this.idxEnd = idxEnd;
        this.averagePower = averagePower;
        this.description = description;
    }
}
