export { RequireAuth } from '@/auth/require-auth'
export { RouteAccessGate } from '@/auth/route-access-gate'
export { getProfileLockReasons } from '@/auth/profile-lock'
export type { ProfileLockReason } from '@/auth/profile-lock'
export { SessionBootstrap } from '@/auth/session-bootstrap'
export { toSessionSnapshot } from '@/auth/session-hydration'
export {
  PermissionGate,
  RoleGate,
  RoutePermissionProvider,
} from '@/auth/permission-gates'
export { usePermission, useRole } from '@/hooks/use-permission'
export { UnauthorizedBridge } from '@/auth/unauthorized-bridge'
export { dispatchUnauthorized, subscribeUnauthorized } from '@/auth/unauthorized'
export type { UnauthorizedEvent } from '@/auth/unauthorized'
