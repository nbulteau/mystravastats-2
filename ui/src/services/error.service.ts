import { useContextStore } from '@/stores/context'
import { type Toast, ToastTypeEnum } from '@/models/toast.model'
import type { APIResponseError } from '@/models/error.model'

export class ErrorService {
    static async catchError(response: Response) {
        const contextStore = useContextStore()

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
        if (response.status < 500) {
            if (apiError.code) {
                const toast: Toast = {
                    id: `toast-${Date.now()}`,
                    message: `${apiError.description}`,
                    apiErrorCode: apiError.code,
                    type: ToastTypeEnum.ERROR,
                }
                contextStore.showToast(toast)
                // program toast disapear
                setTimeout(() => {
                    contextStore.removeToast(toast)
                }, 5000)
            } else {
                // error outside server

                const toast: Toast = {
                    id: `toast-${Date.now()}`,
                    message: 'Une erreur s\'est produite, veuillez réessayer ultérieurement',
                    type: ToastTypeEnum.ERROR,
                }
                contextStore.showToast(toast)

                // program toast disapear
                setTimeout(() => {
                    contextStore.removeToast(toast)
                }, 5000)
            }

            throw new Error(apiError.message)
        }
    }
}
