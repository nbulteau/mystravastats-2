import { useUiStore } from '@/stores/ui'
import { type Toast, ToastTypeEnum } from '@/models/toast.model'
import type { APIResponseError } from '@/models/error.model'

export class ErrorService {
    static async catchError(response: Response) {
        const uiStore = useUiStore()

        if ((response.status === 401 || response.status === 403) && response.headers.get('X-Goog-IAP-Generated-Response') !== null) {
            window.location.reload()
            return
        }
        let apiError: APIResponseError | null 
        try {
            apiError = ((await response.json()) as unknown) as APIResponseError
        } catch (e) {
            const message = response.statusText ? `${response.status} - ${response.statusText}` : 'Uknown Error'
            apiError = {
                message,
                code: response.status,
                path: 'null',
                description: message,
                name: 'customError',
                timestamp: `${Date.now()}`,
            }
        }
        const toast: Toast = {
            id: `toast-${Date.now()}`,
            message: apiError.code
                ? `${apiError.description}`
                : 'Une erreur s\'est produite, veuillez réessayer ultérieurement',
            apiErrorCode: apiError.code,
            type: ToastTypeEnum.ERROR,
        }
        uiStore.showToast(toast)
        setTimeout(() => {
            uiStore.removeToast(toast)
        }, 5000)

        throw new Error(apiError.message)
    }
}
