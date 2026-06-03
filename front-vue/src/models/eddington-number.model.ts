export class EddingtonNumber {
    eddingtonNumber!: number;
    eddingtonList!: number[];
    scope!: "lifetime" | "year" | "rolling-12-months";
    metric!: "distance" | "elevation";
    basis!: "days" | "activities";
    unit!: "km" | "m";
    nextTarget!: number;
    qualifyingCount!: number;
    missingCount!: number;
    qualifyingDays!: number;
    missingDays!: number;
}
