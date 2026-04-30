import { beforeEach, describe, expect, test, vi } from 'vitest'
import { confirmAction, notifySuccess } from '@/feedback'
import { useFeedbackStore } from '@/feedback/store'

describe('feedback service', () => {
  beforeEach(() => {
    useFeedbackStore.setState({
      toasts: [],
      confirm: {
        open: false,
        title: '',
        message: '',
        confirmText: 'Confirm',
        cancelText: 'Cancel',
      },
      confirmResolver: null,
    })
  })

  test('notifySuccess pushes toast and auto-removes after timeout', () => {
    vi.useFakeTimers()

    notifySuccess('ok', 10)
    expect(useFeedbackStore.getState().toasts).toHaveLength(1)

    vi.advanceTimersByTime(11)
    expect(useFeedbackStore.getState().toasts).toHaveLength(0)

    vi.useRealTimers()
  })

  test('confirmAction resolves true when confirmed', async () => {
    const promise = confirmAction({
      title: 't',
      message: 'm',
      confirmText: 'yes',
      cancelText: 'no',
    })

    expect(useFeedbackStore.getState().confirm.open).toBe(true)

    useFeedbackStore.getState().resolveConfirm(true)

    await expect(promise).resolves.toBe(true)
    expect(useFeedbackStore.getState().confirm.open).toBe(false)
  })
})
