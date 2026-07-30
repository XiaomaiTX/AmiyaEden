export interface RouteAuthMetaItem {
  titleKey: string
  authMark: string
}

export interface RouteAccessMeta {
  jwt?: boolean
  login?: boolean
  roles?: string[]
  corpCapabilitiesAll?: string[]
  corpCapabilitiesAny?: string[]
  authList?: RouteAuthMetaItem[]
  requiresNewbro?: boolean
  requiresMentorMenteeEligibility?: boolean
}

export interface RouteSessionAccess {
  isLoggedIn: boolean
  roles: string[]
  corpCapabilities: string[]
  isCurrentlyNewbro: boolean
  isMentorMenteeEligible: boolean
}

export type RouteAccessDecision = 'allow' | 'login' | 'forbidden'

export function hasRouteRolePermission(userRoles: string[], requiredRoles: string[] = []) {
  if (requiredRoles.length === 0) {
    return true
  }

  return requiredRoles.some((role) => userRoles.includes(role))
}

/**
 * Evaluates whether the current user's corporation capabilities satisfy the
 * route requirement:
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

function hasNonGuestRole(roles: string[]) {
  return roles.some((role) => role !== 'guest')
}

export function evaluateRouteAccess(
  meta: RouteAccessMeta | undefined,
  session: RouteSessionAccess
): RouteAccessDecision {
  if (!meta) {
    return 'allow'
  }

  const requiresAuthentication =
    meta.jwt === true ||
    meta.login === true ||
    Boolean(meta.roles?.length) ||
    Boolean(meta.corpCapabilitiesAll?.length) ||
    Boolean(meta.corpCapabilitiesAny?.length) ||
    meta.requiresNewbro === true ||
    meta.requiresMentorMenteeEligibility === true

  if (requiresAuthentication && !session.isLoggedIn) {
    return 'login'
  }

  if (meta.login && !hasNonGuestRole(session.roles)) {
    return 'forbidden'
  }

  if (!hasRouteRolePermission(session.roles, meta.roles)) {
    return 'forbidden'
  }

  if (!hasCorpCapabilityPermission(session.roles, session.corpCapabilities, meta)) {
    return 'forbidden'
  }

  if (meta.requiresNewbro && !session.isCurrentlyNewbro) {
    return 'forbidden'
  }

  if (meta.requiresMentorMenteeEligibility && !session.isMentorMenteeEligible) {
    return 'forbidden'
  }

  return 'allow'
}
