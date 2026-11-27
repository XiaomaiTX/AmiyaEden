import type { FleetQQPingReceipt } from '@/api/generated/client'
import { HttpError } from '@/api/http-error'

export function createFleetQQPingIntent(newKey: () => string = () => crypto.randomUUID()) {
  let key: string | undefined
  let pending = false
  let accepted = false

  return {
    submit(send: (key: string) => Promise<FleetQQPingReceipt>) {
      if (pending || accepted) return
      pending = true
      return (async () => {
        try {
          key ??= newKey()
          const receipt = await send(key)
          accepted = true
          return receipt
        } finally {
          pending = false
        }
      })()
    },
    resetAfterAcceptance() {
      if (!accepted) return false
      accepted = false
      key = undefined
      return true
    },
  }
}

export function fleetQQPingErrorKey(error: unknown) {
  const status = error instanceof HttpError ? error.status : undefined
  const reason = error instanceof HttpError ? error.message : undefined
  switch (status) {
    case 403:
      return 'fleet.qqPing.forbidden'
    case 409:
      if (reason === 'no_targets' || reason === 'group_not_authorized')
        return 'fleet.qqPing.noTargets'
      if (reason === 'expired') return 'fleet.qqPing.expired'
      return 'qqOfficial.conflict'
    case 429:
      return 'qqOfficial.rateLimited'
    case 503:
      return reason === 'disabled' ? 'fleet.qqPing.disabled' : 'qqOfficial.unavailable'
    default:
      return 'fleet.qqPing.queueUnconfirmed'
  }
}

export function isFleetQQAdmissionError(error: unknown) {
  if (!(error instanceof HttpError)) return false
  if (error.status === 409 || error.status === 429) return true
  return (
    error.status === 503 &&
    ['disabled', 'unavailable', 'invalid_input'].includes(error.message)
  )
}

export function isFleetCreationUnconfirmed(error: unknown) {
  return (
    !(error instanceof HttpError) ||
    error.status < 400 ||
    error.status === 408 ||
    error.status >= 500
  )
}
