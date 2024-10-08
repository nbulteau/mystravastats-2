class DetailedActivityDto {
    achievementCount: number;
    athlete: number;
    athleteCount: number;
    averageCadence: number;
    averageSpeed: number;
    averageTemp: number;
    averageWatts: number;
    calories: number;
    commentCount: number;
    commute: boolean;
    description: string;
    deviceName: string | null;
    deviceWatts: boolean;
    distance: number;
    elapsedTime: number;
    elevHigh: number;
    elevLow: number;
    embedToken: string;
    endLatLng: number[];
    externalId: string;
    flagged: boolean;
    fromAcceptedTag: boolean;
    gear: Gear;
    gearId: string;
    hasHeartRate: boolean;
    hasKudoed: boolean;
    hideFromHome: boolean;
    id: number;
    kilojoules: number;
    kudosCount: number;
    leaderboardOptOut: boolean;
    map: GeoMap | null;
    manual: boolean;
    maxSpeed: number;
    maxWatts: number;
    movingTime: number;
    name: string;
    prCount: number;
    isPrivate: boolean;
    resourceState: number;
    segmentEfforts: SegmentEffortDto[];
    segmentLeaderboardOptOut: boolean;
    splitsMetric: SplitsMetricDto[];
    sportType: string;
    startDate: string;
    startSateLocal: string;
    startLatLng: number[];
    sufferScore: number | null;
    timezone: string;
    totalElevationGain: number;
    totalPhotoCount: number;
    trainer: boolean;
    type: string;
    uploadId: number;
    utcOffset: number;
    weightedAverageWatts: number;
    workoutType: number;

    constructor(
        achievementCount: number,
        athlete: number,
        athleteCount: number,
        averageCadence: number,
        averageSpeed: number,
        averageTemp: number,
        averageWatts: number,
        calories: number,
        commentCount: number,
        commute: boolean,
        description: string,
        deviceName: string | null,
        deviceWatts: boolean,
        distance: number,
        elapsedTime: number,
        elevHigh: number,
        elevLow: number,
        embedToken: string,
        endLatLng: number[],
        externalId: string,
        flagged: boolean,
        fromAcceptedTag: boolean,
        gear: Gear,
        gearId: string,
        hasHeartRate: boolean,
        hasKudoed: boolean,
        hideFromHome: boolean,
        id: number,
        kilojoules: number,
        kudosCount: number,
        leaderboardOptOut: boolean,
        map: GeoMap | null,
        manual: boolean,
        maxSpeed: number,
        maxWatts: number,
        movingTime: number,
        name: string,
        prCount: number,
        isPrivate: boolean,
        resourceState: number,
        segmentEfforts: SegmentEffortDto[],
        segmentLeaderboardOptOut: boolean,
        splitsMetric: SplitsMetricDto[],
        sportType: string,
        startDate: string,
        startSateLocal: string,
        startLatLng: number[],
        sufferScore: number | null,
        timezone: string,
        totalElevationGain: number,
        totalPhotoCount: number,
        trainer: boolean,
        type: string,
        uploadId: number,
        utcOffset: number,
        weightedAverageWatts: number,
        workoutType: number
    ) {
        this.achievementCount = achievementCount;
        this.athlete = athlete;
        this.athleteCount = athleteCount;
        this.averageCadence = averageCadence;
        this.averageSpeed = averageSpeed;
        this.averageTemp = averageTemp;
        this.averageWatts = averageWatts;
        this.calories = calories;
        this.commentCount = commentCount;
        this.commute = commute;
        this.description = description;
        this.deviceName = deviceName;
        this.deviceWatts = deviceWatts;
        this.distance = distance;
        this.elapsedTime = elapsedTime;
        this.elevHigh = elevHigh;
        this.elevLow = elevLow;
        this.embedToken = embedToken;
        this.endLatLng = endLatLng;
        this.externalId = externalId;
        this.flagged = flagged;
        this.fromAcceptedTag = fromAcceptedTag;
        this.gear = gear;
        this.gearId = gearId;
        this.hasHeartRate = hasHeartRate;
        this.hasKudoed = hasKudoed;
        this.hideFromHome = hideFromHome;
        this.id = id;
        this.kilojoules = kilojoules;
        this.kudosCount = kudosCount;
        this.leaderboardOptOut = leaderboardOptOut;
        this.map = map;
        this.manual = manual;
        this.maxSpeed = maxSpeed;
        this.maxWatts = maxWatts;
        this.movingTime = movingTime;
        this.name = name;
        this.prCount = prCount;
        this.isPrivate = isPrivate;
        this.resourceState = resourceState;
        this.segmentEfforts = segmentEfforts;
        this.segmentLeaderboardOptOut = segmentLeaderboardOptOut;
        this.splitsMetric = splitsMetric;
        this.sportType = sportType;
        this.startDate = startDate;
        this.startSateLocal = startSateLocal;
        this.startLatLng = startLatLng;
        this.sufferScore = sufferScore;
        this.timezone = timezone;
        this.totalElevationGain = totalElevationGain;
        this.totalPhotoCount = totalPhotoCount;
        this.trainer = trainer;
        this.type = type;
        this.uploadId = uploadId;
        this.utcOffset = utcOffset;
        this.weightedAverageWatts = weightedAverageWatts;
        this.workoutType = workoutType;
    }
}

class SegmentEffortDto {
    achievements: AchievementDto[];
    activity: number;
    athlete: number;
    averageCadence: number;
    averageHeartRate: number;
    averageWatts: number;
    deviceWatts: boolean;
    distance: number;
    elapsedTime: number;
    endIndex: number;
    hidden: boolean;
    id: number;
    komRank: number | null;
    maxHeartRate: number;
    movingTime: number;
    name: string;
    prRank: number | null;
    resourceState: number;
    segment: SegmentDto;
    startDate: string;
    startDateLocal: string;
    startIndex: number;
    visibility: string | null;

    constructor(
        achievements: AchievementDto[],
        activity: number,
        athlete: number,
        averageCadence: number,
        averageHeartRate: number,
        averageWatts: number,
        deviceWatts: boolean,
        distance: number,
        elapsedTime: number,
        endIndex: number,
        hidden: boolean,
        id: number,
        komRank: number | null,
        maxHeartRate: number,
        movingTime: number,
        name: string,
        prRank: number | null,
        resourceState: number,
        segment: SegmentDto,
        startDate: string,
        startDateLocal: string,
        startIndex: number,
        visibility: string | null
    ) {
        this.achievements = achievements;
        this.activity = activity;
        this.athlete = athlete;
        this.averageCadence = averageCadence;
        this.averageHeartRate = averageHeartRate;
        this.averageWatts = averageWatts;
        this.deviceWatts = deviceWatts;
        this.distance = distance;
        this.elapsedTime = elapsedTime;
        this.endIndex = endIndex;
        this.hidden = hidden;
        this.id = id;
        this.komRank = komRank;
        this.maxHeartRate = maxHeartRate;
        this.movingTime = movingTime;
        this.name = name;
        this.prRank = prRank;
        this.resourceState = resourceState;
        this.segment = segment;
        this.startDate = startDate;
        this.startDateLocal = startDateLocal;
        this.startIndex = startIndex;
        this.visibility = visibility;
    }
}

class SegmentDto {
    activityType: string;
    averageGrade: number;
    city: string | null;
    climbCategory: number;
    country: string | null;
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
    state: string | null;

    constructor(
        activityType: string,
        averageGrade: number,
        city: string | null,
        climbCategory: number,
        country: string | null,
        distance: number,
        elevationHigh: number,
        elevationLow: number,
        endLatLng: number[],
        hazardous: boolean,
        id: number,
        maximumGrade: number,
        name: string,
        isPrivate: boolean,
        resourceState: number,
        starred: boolean,
        startLatLng: number[],
        state: string | null
    ) {
        this.activityType = activityType;
        this.averageGrade = averageGrade;
        this.city = city;
        this.climbCategory = climbCategory;
        this.country = country;
        this.distance = distance;
        this.elevationHigh = elevationHigh;
        this.elevationLow = elevationLow;
        this.endLatLng = endLatLng;
        this.hazardous = hazardous;
        this.id = id;
        this.maximumGrade = maximumGrade;
        this.name = name;
        this.isPrivate = isPrivate;
        this.resourceState = resourceState;
        this.starred = starred;
        this.startLatLng = startLatLng;
        this.state = state;
    }
}