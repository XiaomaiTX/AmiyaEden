import { useContext } from 'react'
import { RoutePermissionContext } from '@/auth/permission-context'
import { useSessionStore } from '@/stores'

export function usePermission(permission: string) {
  const permissions = useContext(RoutePermissionContext)
  return permissions.includes(permission)
}

export function useRole(requiredRoles: string | readonly string[]) {
  const roles = useSessionStore((state) => state.roles)
  const required = typeof requiredRoles === 'string' ? [requiredRoles] : requiredRoles
  return required.some((role) => roles.includes(role))
}
