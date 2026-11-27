import type { LucideIcon } from 'lucide-react'
import {
  Ban,
  Bot,
  Brain,
  ChartNoAxesColumn,
  ClipboardList,
  Coins,
  Factory,
  Gift,
  Info,
  LayoutDashboard,
  ListChecks,
  Megaphone,
  Settings,
  ShieldCheck,
  ShoppingBag,
  Ship,
  UserRoundPlus,
  UsersRound,
  Webhook,
  Wrench,
} from 'lucide-react'
import { appRouteSpecs, type AppRoutePath } from '@/app/migration-routes'
import { evaluateRouteAccess } from '@/app/route-access'
import type { SessionSnapshot } from '@/stores'
import type { AdmissionContext, BadgeCounts } from '@/api/generated/client'
import { canAccessAdmission } from '@/lib/admissions'

interface ShellMenuItem {
  to: string
  labelKey: string
  badge?: number
}

export type ShellMenuEntry =
  | (ShellMenuItem & { kind: 'link'; icon: LucideIcon })
  | {
      kind: 'group'
      labelKey: string
      icon: LucideIcon
      items: ShellMenuItem[]
      badge?: number
    }

export interface ShellMenuSection {
  labelKey: string
  entries: ShellMenuEntry[]
}

type MenuEntrySpec =
  | { path: AppRoutePath; icon: LucideIcon }
  | { labelKey: string; icon: LucideIcon; paths: AppRoutePath[] }

const sidebarSections: { labelKey: string; entries: MenuEntrySpec[] }[] = [
  {
    labelKey: 'nav.home',
    entries: [
      { path: 'dashboard/console', icon: LayoutDashboard },
      { path: 'announcements', icon: Megaphone },
    ],
  },
  {
    labelKey: 'nav.section.fleet',
    entries: [
      {
        labelKey: 'nav.group.operation',
        icon: Ship,
        paths: [
          'operation/pap',
          'operation/corporation-pap',
          'operation/fleets',
          'operation/fleet-configs',
        ],
      },
      {
        labelKey: 'nav.group.srp',
        icon: ShieldCheck,
        paths: ['srp/srp-apply', 'srp/srp-manage', 'srp/srp-prices'],
      },
      {
        labelKey: 'nav.group.skillPlanning',
        icon: Brain,
        paths: [
          'skill-planning/personal-skill-plans',
          'skill-planning/skill-plans',
          'skill-planning/completion-check',
        ],
      },
    ],
  },
  {
    labelKey: 'nav.section.charactersAndTools',
    entries: [
      {
        labelKey: 'nav.group.info',
        icon: Info,
        paths: [
          'info/wallet',
          'info/skill',
          'info/npc-kills',
          'info/ships',
          'info/implants',
          'info/fittings',
          'info/assets',
          'info/contracts',
        ],
      },
      {
        labelKey: 'nav.group.tools',
        icon: Wrench,
        paths: ['mumble', 'info/esi-check', 'info/tool-bookmarks'],
      },
    ],
  },
  {
    labelKey: 'nav.section.memberServices',
    entries: [
      {
        labelKey: 'admissions.title',
        icon: UserRoundPlus,
        paths: ['admissions/new', 'admissions/alt', 'admissions/history', 'admissions/review'],
      },
      {
        labelKey: 'nav.group.newbro',
        icon: UserRoundPlus,
        paths: [
          'newbro/select-mentor',
          'newbro/captain',
          'newbro/mentor',
          'newbro/recruit-link',
          'newbro/manage',
          'newbro/mentor-manage',
          'system/newbro-settings',
        ],
      },
      {
        labelKey: 'nav.group.welfare',
        icon: Gift,
        paths: ['welfare/my', 'welfare/approval', 'welfare/settings'],
      },
      {
        labelKey: 'nav.group.shop',
        icon: ShoppingBag,
        paths: ['shop/browse', 'shop/wallet', 'shop/manage', 'shop/order-manage'],
      },
      {
        labelKey: 'nav.group.ticket',
        icon: ClipboardList,
        paths: [
          'ticket/my-tickets',
          'ticket/create',
          'ticket/management',
          'ticket/categories',
          'ticket/statistics',
        ],
      },
      { path: 'fuxi-hall', icon: UsersRound },
    ],
  },
  {
    labelKey: 'nav.section.corpOperations',
    entries: [
      {
        labelKey: 'nav.group.corpData',
        icon: ChartNoAxesColumn,
        paths: ['dashboard/npc-kills', 'dashboard/skill-readiness'],
      },
      {
        labelKey: 'nav.group.territoryFacilities',
        icon: Factory,
        paths: [
          'dashboard/galaxy-registry',
          'dashboard/corporation-structures',
          'dashboard/fuel-officer-structures',
        ],
      },
      {
        labelKey: 'nav.group.finance',
        icon: Coins,
        paths: ['system/salary', 'system/wallet', 'system/pap', 'system/pap-exchange'],
      },
    ],
  },
  {
    labelKey: 'nav.group.system',
    entries: [
      { path: 'system/user', icon: UsersRound },
      { path: 'system/blacklist', icon: Ban },
      { path: 'system/auto-role', icon: ShieldCheck },
      { path: 'system/task-manager', icon: ListChecks },
      { path: 'system/audit', icon: ClipboardList },
      { path: 'system/webhook', icon: Webhook },
      { path: 'system/qq-official', icon: Bot },
      { path: 'system/basic-config', icon: Settings },
    ],
  },
]

const routesByPath = new Map(appRouteSpecs.map((route) => [route.path, route]))

type SessionAccess = Pick<
  SessionSnapshot,
  'isLoggedIn' | 'roles' | 'corpCapabilities' | 'isCurrentlyNewbro' | 'isMentorMenteeEligible'
>

export function buildShellMenuSections(
  session: SessionAccess,
  badgeCounts: BadgeCounts = {},
  admissionContext: AdmissionContext | null = null
): ShellMenuSection[] {
  function getVisibleItem(path: AppRoutePath): ShellMenuItem | undefined {
    const route = routesByPath.get(path)
    if (!route || route.menuHidden || evaluateRouteAccess(route.meta, session) !== 'allow') return
    if (route.meta?.admission && !canAccessAdmission(admissionContext, route.meta.admission)) return

    return {
      to: `/${path}`,
      labelKey: route.titleKey,
      badge: route.badgeKey ? badgeCounts[route.badgeKey] : undefined,
    }
  }

  return sidebarSections.flatMap(({ labelKey, entries }) => {
    const visibleEntries = entries.flatMap<ShellMenuEntry>((entry) => {
      if ('path' in entry) {
        const item = getVisibleItem(entry.path)
        return item ? [{ ...item, kind: 'link', icon: entry.icon }] : []
      }

      const items = entry.paths.map(getVisibleItem).filter((item) => item !== undefined)
      if (!items.length) return []
      return [
        {
          kind: 'group',
          labelKey: entry.labelKey,
          icon: entry.icon,
          items,
          badge: items.reduce((total, item) => total + (item.badge ?? 0), 0) || undefined,
        },
      ]
    })
    return visibleEntries.length ? [{ labelKey, entries: visibleEntries }] : []
  })
}
