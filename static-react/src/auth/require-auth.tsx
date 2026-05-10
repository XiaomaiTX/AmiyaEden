import type { PropsWithChildren } from 'react'
import { RouteAccessGate } from '@/auth/route-access-gate'

export function RequireAuth({ children }: PropsWithChildren) {
  return <RouteAccessGate meta={{ login: true }}>{children}</RouteAccessGate>
}
