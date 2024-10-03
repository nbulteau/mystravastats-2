import type { Activity } from "./activity.model";

export interface Statistics {
    label: string;
    value: string;
    activity?: Activity;
}
