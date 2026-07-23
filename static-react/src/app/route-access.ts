export interface RouteAuthMetaItem {
  title: string
  authMark: string
}

export interface RouteAccessMeta {
  login?: boolean
  roles?: string[]
  corpCapabilitiesAll?: string[]
  corpCapabilitiesAny?: string[]
  authList?: RouteAuthMetaItem[]
  requiresNewbro?: boolean
  requiresMentorMenteeEligibility?: boolean
}

export function hasRouteRolePermission(userRoles: string[], requiredRoles: string[] = []) {
  if (requiredRoles.length === 0) {
    return true
  }

  return requiredRoles.some((role) => userRoles.includes(role))
}

/**
 * Evaluates whether the current user's corporation capabilities satisfy the
 * route requirement. Mirrors the Vue `hasCorpCapabilityPermission` so both
 * frontends apply identical gating logic:
 *
 * - `corpCapabilitiesAll`: every entry must be present (AND).
 * - `corpCapabilitiesAny`: at least one entry must be present (OR).
 * - `super_admin` short-circuits to true.
 */
export function hasCorpCapabilityPermission(
  userRoles: string[],
  corpCapabilities: string[],
  meta: Pick<RouteAccessMeta, 'corpCapabilitiesAll' | 'corpCapabilitiesAny'>
): boolean {
  if (userRoles.includes('super_admin')) {
    return true
  }

  const all = meta.corpCapabilitiesAll
  if (all && all.length > 0) {
    if (!all.every((capability) => corpCapabilities.includes(capability))) {
      return false
    }
  }

  const any = meta.corpCapabilitiesAny
  if (any && any.length > 0) {
    if (!any.some((capability) => corpCapabilities.includes(capability))) {
      return false
    }
  }

  return true
}
