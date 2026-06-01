export interface Activity {
    id: number;
    name: string;
    type: string;
    commute: boolean;
    link: string;
    distance: number;
    elapsedTime: number;
    movingTime: number;
    totalElevationGain: number;
    totalDescent: number;
    averageSpeed: number;
    averageHeartrate: number;
    bestSpeedForDistanceFor1000m: number;
    bestElevationForDistanceFor500m: number;
    bestElevationForDistanceFor1000m: number;
    date: string;
    averageWatts: number;
    weightedAverageWatts: number;
    bestPowerFor20minutes: number;
    bestPowerFor60minutes: number;
    ftp: number;
    badgeEffortSeconds?: number;
}

export interface Stream {
    distance: number[];
    time: number[];
    heartrate: number[] | null;
    cadence: number[] | null;
    moving: boolean[] | null;
    altitude: number[] | null;
    latlng: number[][] | null;
    watts: number[] | null;
    velocitySmooth?: number[] | null;
}

export interface DetailedActivity {
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
    maxWatts: number;
    movingTime: number;
    name: string;
    startDate: string;
    startDateLocal: string;
    startLatlng: number[] | null;
    totalDescent: number;
    totalElevationGain: number;
    type: string;
    sportType: string;
    weightedAverageWatts: number;
    stream: Stream;
    activityEfforts: ActivityEffort[];
    stravaSegmentEfforts: StravaSegmentEffort[];
    activityComparison?: ActivityComparison | null;
    calories?: number;
    sufferScore?: number | null;
}

export interface ActivityComparison {
    status: "insufficient-data" | "typical" | "faster" | "slower" | "atypical" | string;
    label: string;
    criteria: ActivityComparisonCriteria;
    baseline: ActivityComparisonBaseline;
    deltas: ActivityComparisonDeltas;
    similarActivities: ActivityComparisonActivity[];
    commonSegments: ActivityComparisonSegment[];
}

export interface ActivityComparisonCriteria {
    activityType: string;
    year: number;
    sampleSize: number;
}

export interface ActivityComparisonBaseline {
    distance: number;
    elevationGain: number;
    movingTime: number;
    averageSpeed: number;
    averageHeartrate: number;
    averageWatts: number;
    averageCadence: number;
}

export interface ActivityComparisonDeltas {
    distance: number;
    elevationGain: number;
    movingTime: number;
    averageSpeed: number;
    averageSpeedPct: number;
    averageHeartrate: number;
    averageWatts: number;
    averageCadence: number;
}

export interface ActivityComparisonActivity {
    id: number;
    name: string;
    date: string;
    distance: number;
    elevationGain: number;
    movingTime: number;
    averageSpeed: number;
    averageHeartrate: number;
    averageWatts: number;
    averageCadence: number;
    similarityScore: number;
}

export interface ActivityComparisonSegment {
    id: number;
    name: string;
    matchCount: number;
    activityIds: number[];
    activityNames: string[];
}

export interface ActivityShort {
    id: number;
    name: string;
    type: string;
}

export interface ActivityEffort {
    id: string;
    label: string;
    distance: number;
    seconds: number;
    deltaAltitude: number;
    idxStart: number;
    idxEnd: number;
    averagePower: number | null;
    description: string;
}

export interface StravaSegmentEffort {
    averageCadence: number;
    averageHeartRate: number;
    averageWatts: number;
    deviceWatts: boolean;
    distance: number;
    elapsedTime: number;
    endIndex: number;
    hidden: boolean;
    id: number;
    komRank?: number | null;
    maxHeartRate: number;
    movingTime: number;
    name: string;
    prRank?: number | null;
    resourceState: number;
    segment: StravaSegment;
    startDate: string;
    startDateLocal: string;
    startIndex: number;
    visibility?: string | null;
}

export interface StravaSegment {
    activityType: string;
    averageGrade: number;
    city?: string | null;
    climbCategory: number;
    country?: string | null;
    distance: number;
    elevationHigh: number;
    elevationLow: number;
    endLatLng: number[];
    hazardous: boolean;
    id: number;
    maximumGrade: number;
    name: string;
    isPrivate: boolean;
    resourceState: number;
    starred: boolean;
    startLatLng: number[];
    state?: string | null;
}
