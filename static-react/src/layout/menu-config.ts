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
  UsersRound,
} from 'lucide-react'
import { appRouteSpecs } from '@/app/migration-routes'
import { evaluateRouteAccess } from '@/app/route-access'
import type { SessionSnapshot } from '@/stores'
import type { BadgeCounts } from '@/types/api/badge'

export interface ShellMenuItem {
  to: string
  labelKey: string
  badge?: number
}

export interface ShellMenuGroup {
  labelKey: string
  icon: LucideIcon
  items: ShellMenuItem[]
  badge?: number
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
  fuxiHall: UsersRound,
}

type SessionAccess = Pick<
  SessionSnapshot,
  'isLoggedIn' | 'roles' | 'corpCapabilities' | 'isCurrentlyNewbro' | 'isMentorMenteeEligible'
>

function canAccessRoute(route: (typeof appRouteSpecs)[number], session: SessionAccess) {
  return evaluateRouteAccess(route.meta, session) === 'allow'
}

export function buildShellMenuGroups(session: SessionAccess, badgeCounts: BadgeCounts = {}) {
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
      badge: route.badgeKey ? badgeCounts[route.badgeKey] : undefined,
    })
  }

  return Array.from(grouped.values())
    .filter((group) => group.items.length > 0)
    .map((group) => ({
      ...group,
      badge: group.items.reduce((total, item) => total + (item.badge ?? 0), 0) || undefined,
    }))
}
