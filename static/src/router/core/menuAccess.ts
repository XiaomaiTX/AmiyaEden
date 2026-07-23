import type { AppRouteRecord } from '../../types/router'
import { RoutesAlias } from '../routesAlias'

export function hasNonGuestRole(roles: string[]): boolean {
  return roles.some((role) => role !== 'guest')
}

/**
 * 评估当前用户的军团 capability 是否满足路由要求。
 *
 * 语义：
 * - `corpCapabilitiesAll`：全部命中才放行（与后端 `RequireCorpCapability`
 *   连续中间件链的 AND 行为一致）。
 * - `corpCapabilitiesAny`：至少一项命中即放行。空数组或缺失表示无此约束。
 *
 * Stage 0A 之后旧的 `corpCapabilities` 字段已废弃；新代码必须显式选择
 * `corpCapabilitiesAll` 或 `corpCapabilitiesAny`，避免再让一个模糊数组
 * 在不同路由上同时承担 OR / AND。
 */
export function hasCorpCapabilityPermission(
  roles: string[],
  corpCapabilities: string[],
  meta: {
    corpCapabilitiesAll?: string[]
    corpCapabilitiesAny?: string[]
  }
): boolean {
  if (roles.includes('super_admin')) {
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

export function applyMenuAccessFilter(
  menu: AppRouteRecord[],
  roles: string[],
  corpCapabilities: string[] = [],
  isCurrentlyNewbro?: boolean,
  isMentorMenteeEligible?: boolean
): AppRouteRecord[] {
  return menu.reduce((acc: AppRouteRecord[], item) => {
    const itemRoles = item.meta?.roles
    const requiresLogin = item.meta?.login === true
    const requiresNewbro = item.meta?.requiresNewbro === true
    const requiresMentorMenteeEligibility = item.meta?.requiresMentorMenteeEligibility === true
    const hasRolePermission = !itemRoles || itemRoles.some((role) => roles.includes(role))
    const hasCorpCapabilityPermissionForItem = hasCorpCapabilityPermission(
      roles,
      corpCapabilities,
      item.meta ?? {}
    )
    const hasLoginPermission = !requiresLogin || hasNonGuestRole(roles)
    const hasNewbroPermission = !requiresNewbro || isCurrentlyNewbro === true
    const hasMentorMenteePermission =
      !requiresMentorMenteeEligibility || isMentorMenteeEligible === true
    const hasPermission =
      hasRolePermission &&
      hasCorpCapabilityPermissionForItem &&
      hasLoginPermission &&
      hasNewbroPermission &&
      hasMentorMenteePermission

    if (!hasPermission) {
      return acc
    }

    const filteredItem = { ...item }
    if (filteredItem.children?.length) {
      filteredItem.children = applyMenuAccessFilter(
        filteredItem.children,
        roles,
        corpCapabilities,
        isCurrentlyNewbro,
        isMentorMenteeEligible
      )
    }

    acc.push(filteredItem)
    return acc
  }, [])
}

export function pruneEmptyMenus(menuList: AppRouteRecord[]): AppRouteRecord[] {
  return menuList
    .map((item) => {
      if (item.children && item.children.length > 0) {
        return {
          ...item,
          children: pruneEmptyMenus(item.children)
        }
      }

      return item
    })
    .filter((item) => {
      // Directory menus: keep only if they still have children after pruning
      if ('children' in item) {
        return item.children !== undefined && item.children.length > 0
      }

      // Leaf nodes: keep iframes, external links, or real components
      if (item.meta?.isIframe === true || item.meta?.link) {
        return true
      }

      return !!item.component && item.component !== '' && item.component !== RoutesAlias.Layout
    })
}
