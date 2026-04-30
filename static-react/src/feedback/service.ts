import type { ConfirmOptions, ToastType } from '@/feedback/store'
import { useFeedbackStore } from '@/feedback/store'

function emitToast(type: ToastType, message: string, timeoutMs: number = 3000) {
  const { pushToast, removeToast } = useFeedbackStore.getState()
  const id = pushToast({ type, message })

  setTimeout(() => {
    removeToast(id)
  }, timeoutMs)
}

export function notifySuccess(message: string, timeoutMs?: number) {
  emitToast('success', message, timeoutMs)
}

export function notifyError(message: string, timeoutMs?: number) {
  emitToast('error', message, timeoutMs)
}

export function notifyInfo(message: string, timeoutMs?: number) {
  emitToast('info', message, timeoutMs)
}

export function notifyWarning(message: string, timeoutMs?: number) {
  emitToast('warning', message, timeoutMs)
}

export function confirmAction(options: ConfirmOptions) {
  return new Promise<boolean>((resolve) => {
    useFeedbackStore.getState().openConfirm(options, resolve)
  })
}
