import type {Ref} from "vue";

export interface ContextStoreState {
    currentYear: Ref<number>,
    currentActivityType: string,
    athleteDisplayName: string,
    toasts: any[];
}

export enum ActivityType {
    Ride, Run, Hike
}