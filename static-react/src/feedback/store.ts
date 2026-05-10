import { create } from 'zustand'

export type ToastType = 'success' | 'error' | 'warning' | 'info'

export interface ToastItem {
  id: string
  message: string
  type: ToastType
}

export interface ConfirmOptions {
  title: string
  message: string
  confirmText: string
  cancelText: string
}

interface ConfirmState extends ConfirmOptions {
  open: boolean
}

interface FeedbackStoreState {
  toasts: ToastItem[]
  confirm: ConfirmState
  confirmResolver: ((result: boolean) => void) | null
  pushToast: (toast: Omit<ToastItem, 'id'>) => string
  removeToast: (id: string) => void
  openConfirm: (options: ConfirmOptions, resolver: (result: boolean) => void) => void
  resolveConfirm: (result: boolean) => void
}

const defaultConfirmState: ConfirmState = {
  open: false,
  title: '',
  message: '',
  confirmText: 'Confirm',
  cancelText: 'Cancel',
}

function nextToastId() {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

export const useFeedbackStore = create<FeedbackStoreState>()((set, get) => ({
  toasts: [],
  confirm: defaultConfirmState,
  confirmResolver: null,
  pushToast: (toast) => {
    const id = nextToastId()

    set((state) => ({
      toasts: [...state.toasts, { ...toast, id }],
    }))

    return id
  },
  removeToast: (id) => {
    set((state) => ({
      toasts: state.toasts.filter((toast) => toast.id !== id),
    }))
  },
  openConfirm: (options, resolver) => {
    const { confirmResolver } = get()
    if (confirmResolver) {
      confirmResolver(false)
    }

    set({
      confirm: {
        open: true,
        title: options.title,
        message: options.message,
        confirmText: options.confirmText,
        cancelText: options.cancelText,
      },
      confirmResolver: resolver,
    })
  },
  resolveConfirm: (result) => {
    const { confirmResolver } = get()

    if (confirmResolver) {
      confirmResolver(result)
    }

    set({
      confirm: defaultConfirmState,
      confirmResolver: null,
    })
  },
}))
