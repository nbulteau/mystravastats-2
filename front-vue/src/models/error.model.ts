export interface APIResponseError {
    code: number;
    name:string;
    message: string;
    path: string | null;
    timestamp: string;
    description: string;
}

