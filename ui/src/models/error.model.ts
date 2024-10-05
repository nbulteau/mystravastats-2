export interface APIResponseError {
    code: number;
    name:string;
    message: string;
    path: string;
    timestamp: string;
    description: string;
}

export interface FindError extends Error {
    cause:APIResponseError
}