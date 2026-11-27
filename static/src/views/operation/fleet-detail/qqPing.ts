import type { FleetQQPingReceipt } from '@/api/generated/client'
import { HttpError } from '@/utils/http/http-error'

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
    }
  }
}

function fleetQQErrorReason(error: HttpError) {
  const body = error.data
  return body && typeof body === 'object' && 'msg' in body ? body.msg : undefined
}

export function fleetQQPingErrorKey(error: unknown) {
  const status = error instanceof HttpError ? error.code : undefined
  const reason = error instanceof HttpError ? fleetQQErrorReason(error) : undefined
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
  if (error.code === 409 || error.code === 429) return true
  const reason = fleetQQErrorReason(error)
  return (
    error.code === 503 &&
    (reason === 'disabled' || reason === 'unavailable' || reason === 'invalid_input')
  )
}

export function isFleetCreationUnconfirmed(error: unknown) {
  return (
    !(error instanceof HttpError) || error.code < 400 || error.code === 408 || error.code >= 500
  )
}
