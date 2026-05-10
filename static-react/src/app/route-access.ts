export interface RouteAuthMetaItem {
  title: string
  authMark: string
}

export interface RouteAccessMeta {
  login?: boolean
  roles?: string[]
  authList?: RouteAuthMetaItem[]
}

export function hasRouteRolePermission(userRoles: string[], requiredRoles: string[] = []) {
  if (requiredRoles.length === 0) {
    return true
  }

  return requiredRoles.some((role) => userRoles.includes(role))
}
