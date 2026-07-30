import type { PropsWithChildren, ReactNode } from 'react'
import { RoutePermissionContext } from '@/auth/permission-context'
import { usePermission, useRole } from '@/hooks/use-permission'

export function RoutePermissionProvider({
  permissions,
  children,
}: PropsWithChildren<{ permissions: readonly string[] }>) {
  return (
    <RoutePermissionContext.Provider value={permissions}>
      {children}
    </RoutePermissionContext.Provider>
  )
}

export function PermissionGate({
  permission,
  children,
  fallback = null,
}: PropsWithChildren<{ permission: string; fallback?: ReactNode }>) {
  return usePermission(permission) ? children : fallback
}

export function RoleGate({
  roles,
  children,
  fallback = null,
}: PropsWithChildren<{ roles: string | readonly string[]; fallback?: ReactNode }>) {
  return useRole(roles) ? children : fallback
}
