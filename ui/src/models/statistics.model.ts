import type { ActivityShort } from "./activity.model";

export interface Statistics {
    label: string;
    value: string;
    activity?: ActivityShort;
}
