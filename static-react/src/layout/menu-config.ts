import type { LucideIcon } from 'lucide-react'
import {
  Brain,
  ClipboardList,
  Gift,
  Info,
  LayoutDashboard,
  Settings,
  ShieldCheck,
  ShoppingBag,
  Ship,
  UserRoundPlus,
} from 'lucide-react'
import { appRouteSpecs } from '@/app/migration-routes'
import type { SessionSnapshot } from '@/stores'

export interface ShellMenuItem {
  to: string
  labelKey: string
}

export interface ShellMenuGroup {
  labelKey: string
  icon: LucideIcon
  items: ShellMenuItem[]
}

const groupIconMap: Record<string, LucideIcon> = {
  dashboard: LayoutDashboard,
  info: Info,
  operation: Ship,
  skillPlanning: Brain,
  welfare: Gift,
  newbro: UserRoundPlus,
  shop: ShoppingBag,
  srp: ShieldCheck,
  ticket: ClipboardList,
  system: Settings,
}

function hasNonGuestRole(roles: string[]) {
  return roles.some((role) => role !== 'guest')
}

function canAccessRoute(
  route: (typeof appRouteSpecs)[number],
  session: Pick<
    SessionSnapshot,
    'isLoggedIn' | 'roles' | 'isCurrentlyNewbro' | 'isMentorMenteeEligible'
  >
) {
  const { meta } = route
  if (!meta) return true

  if (meta.login && !session.isLoggedIn) return false
  if (meta.login && !hasNonGuestRole(session.roles)) return false
  if (meta.roles && meta.roles.length > 0 && !meta.roles.some((role) => session.roles.includes(role))) {
    return false
  }
  if (meta.requiresNewbro && !session.isCurrentlyNewbro) return false
  if (meta.requiresMentorMenteeEligibility && !session.isMentorMenteeEligible) return false

  return true
}

export function buildShellMenuGroups(
  session: Pick<
    SessionSnapshot,
    'isLoggedIn' | 'roles' | 'isCurrentlyNewbro' | 'isMentorMenteeEligible'
  >
) {
  const grouped = new Map<string, ShellMenuGroup>()

  for (const route of appRouteSpecs) {
    if (!route.menuGroup || !route.menuIcon || route.menuHidden) continue
    if (!canAccessRoute(route, session)) continue

    const icon = groupIconMap[route.menuIcon]
    if (!icon) continue

    if (!grouped.has(route.menuGroup)) {
      grouped.set(route.menuGroup, { labelKey: route.menuGroup, icon, items: [] })
    }

    grouped.get(route.menuGroup)?.items.push({
      to: `/${route.path}`,
      labelKey: route.titleKey,
    })
  }

  return Array.from(grouped.values()).filter((group) => group.items.length > 0)
}
