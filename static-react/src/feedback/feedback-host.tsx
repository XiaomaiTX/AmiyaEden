import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { useFeedbackStore } from '@/feedback/store'
import { cn } from '@/lib/utils'

const toastTypeClass: Record<string, string> = {
  success: 'border-success/20 bg-success/10 text-success-foreground',
  error: 'border-rose-200 bg-rose-50 text-rose-800',
  warning: 'border-amber-200 bg-amber-50 text-amber-800',
  info: 'border-sky-200 bg-sky-50 text-sky-800',
}

export function FeedbackHost() {
  const toasts = useFeedbackStore((state) => state.toasts)
  const removeToast = useFeedbackStore((state) => state.removeToast)
  const confirm = useFeedbackStore((state) => state.confirm)
  const resolveConfirm = useFeedbackStore((state) => state.resolveConfirm)

  return (
    <>
      <div className="pointer-events-none fixed right-4 bottom-4 z-[70] flex w-80 max-w-[calc(100vw-2rem)] flex-col gap-2">
        {toasts.map((toast) => (
          <div
            key={toast.id}
            className={cn(
              'pointer-events-auto rounded-md border px-3 py-2 text-sm shadow-md backdrop-blur',
              toastTypeClass[toast.type] ?? toastTypeClass.info
            )}
          >
            <div className="flex items-start justify-between gap-3">
              <span>{toast.message}</span>
              <button
                type="button"
                className="text-xs opacity-70 hover:opacity-100"
                onClick={() => removeToast(toast.id)}
                aria-label="dismiss-toast"
              >
                ×
              </button>
            </div>
          </div>
        ))}
      </div>

      <AlertDialog
        open={confirm.open}
        onOpenChange={(open) => {
          if (!open) resolveConfirm(false)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{confirm.title}</AlertDialogTitle>
            <AlertDialogDescription>{confirm.message}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => resolveConfirm(false)}>
              {confirm.cancelText}
            </AlertDialogCancel>
            <AlertDialogAction onClick={() => resolveConfirm(true)}>
              {confirm.confirmText}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
