/* eslint-disable no-unused-vars */
export enum ToastTypeEnum {
    NORMAL='NORMAL',
    WARN='WARN',
    ERROR='ERROR',
}

export interface Toast {
    id:string;
    type: ToastTypeEnum;
    message:string;
    apiErrorCode?:number;
    timeout?:number;
}
