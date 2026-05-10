export interface UnauthorizedEvent {
  reason: 'http_401' | 'manual'
  redirectTo?: string
}

type UnauthorizedListener = (event: UnauthorizedEvent) => void

const listeners = new Set<UnauthorizedListener>()

export function subscribeUnauthorized(listener: UnauthorizedListener) {
  listeners.add(listener)
  return () => {
    listeners.delete(listener)
  }
}

export function dispatchUnauthorized(event: UnauthorizedEvent) {
  for (const listener of listeners) {
    listener(event)
  }
}
