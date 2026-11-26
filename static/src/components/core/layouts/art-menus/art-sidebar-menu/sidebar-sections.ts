import type { AppRouteRecord } from '@/types/router'

interface SidebarSection {
  title: string
  items: AppRouteRecord[]
}

interface MenuPage {
  path: string
  title?: string
  icon?: string
}

interface MenuGroup {
  group: string
  title: string
  icon: string
  children: readonly (string | MenuPage)[]
}

const sections: readonly {
  title: string
  items: readonly (string | MenuPage | MenuGroup)[]
}[] = [
  {
    title: 'menus.sidebar.home',
    items: [
      { path: '/dashboard/console', title: 'menus.sidebar.console', icon: 'ri:dashboard-line' },
      '/announcements'
    ]
  },
  {
    title: 'menus.sidebar.fleetReadiness',
    items: [
      {
        group: 'operation',
        title: 'menus.operation.title',
        icon: 'ri:ship-line',
        children: [
          '/operation/pap',
          '/operation/corporation-pap',
          '/operation/fleets',
          '/operation/fleet-configs'
        ]
      },
      '/srp',
      {
        group: 'skill-planning',
        title: 'menus.skillPlanning.title',
        icon: 'ri:brain-line',
        children: [
          '/skill-planning/personal-skill-plans',
          '/skill-planning/skill-plans',
          '/skill-planning/completion-check'
        ]
      }
    ]
  },
  {
    title: 'menus.sidebar.characterTools',
    items: [
      {
        group: 'info',
        title: 'menus.info.title',
        icon: 'ri:user-star-line',
        children: [
          { path: '/info/wallet', title: 'menus.sidebar.iskWallet' },
          '/info/skill',
          '/info/npc-kills',
          '/info/ships',
          '/info/implants',
          '/info/fittings',
          '/info/assets',
          '/info/contracts'
        ]
      },
      {
        group: 'tools',
        title: 'menus.sidebar.tools',
        icon: 'ri:tools-line',
        children: [
          '/mumble',
          '/info/esi-check',
          { path: '/info/tool-bookmarks', title: 'menus.sidebar.toolBookmarks' }
        ]
      }
    ]
  },
  {
    title: 'menus.sidebar.memberServices',
    items: [
      '/admissions',
      {
        group: 'newbro',
        title: 'menus.newbro.title',
        icon: 'ri:user-heart-line',
        children: [
          '/newbro/select-mentor',
          '/newbro/captain',
          '/newbro/mentor',
          '/newbro/recruit-link',
          '/newbro/manage',
          '/newbro/mentor-manage',
          '/system/newbro-settings'
        ]
      },
      '/welfare',
      {
        group: 'shop',
        title: 'menus.sidebar.fuxiShop',
        icon: 'ri:shopping-bag-line',
        children: [
          '/shop/browse',
          { path: '/shop/wallet', title: 'menus.sidebar.fuxiWallet' },
          '/shop/manage',
          '/shop/order-manage'
        ]
      },
      '/ticket',
      '/fuxi-hall'
    ]
  },
  {
    title: 'menus.sidebar.corporationOperations',
    items: [
      {
        group: 'corporation-data',
        title: 'menus.sidebar.corporationData',
        icon: 'ri:bar-chart-line',
        children: ['/dashboard/npc-kills', '/dashboard/skill-readiness']
      },
      {
        group: 'territory-facilities',
        title: 'menus.sidebar.territoryFacilities',
        icon: 'ri:building-line',
        children: [
          '/dashboard/galaxy-registry',
          '/dashboard/corporation-structures',
          '/dashboard/fuel-officer-structures'
        ]
      },
      {
        group: 'finance',
        title: 'menus.sidebar.finance',
        icon: 'ri:coins-line',
        children: [
          '/system/salary',
          { path: '/system/wallet', title: 'menus.sidebar.fuxiWalletManagement' },
          '/system/pap',
          '/system/pap-exchange'
        ]
      }
    ]
  },
  {
    title: 'menus.sidebar.systemManagement',
    items: [
      { path: '/system/user', icon: 'ri:group-line' },
      { path: '/system/blacklist', icon: 'ri:forbid-line' },
      { path: '/system/auto-role', icon: 'ri:shield-user-line' },
      { path: '/system/task-manager', icon: 'ri:task-line' },
      { path: '/system/audit', icon: 'ri:file-search-line' },
      { path: '/system/webhook', icon: 'ri:notification-line' },
      { path: '/system/basic-config', icon: 'ri:settings-3-line' }
    ]
  }
]

function aggregateBadges(children: AppRouteRecord[]) {
  const total = children.reduce((sum, item) => sum + Number(item.meta.showTextBadge || 0), 0)
  return total > 0 ? String(total) : undefined
}

/** Project the already-authorized menu tree into left-sidebar sections without mutating it. */
export function buildSidebarSections(menuList: AppRouteRecord[]): SidebarSection[] {
  const visible = new Map<string, AppRouteRecord>()
  function collect(item: AppRouteRecord): AppRouteRecord | undefined {
    if (item.meta.isHide) return
    let entry = item
    if (item.children) {
      const children = item.children.map(collect).filter((child) => child !== undefined)
      if (!children.length) return
      entry = {
        ...item,
        children,
        meta: { ...item.meta, showTextBadge: aggregateBadges(children) }
      }
    }
    visible.set(item.path, entry)
    return entry
  }
  menuList.forEach(collect)

  function page(item: string | MenuPage): AppRouteRecord | undefined {
    if (typeof item === 'string') return visible.get(item)
    const route = visible.get(item.path)
    if (!route) return
    return {
      ...route,
      meta: {
        ...route.meta,
        title: item.title ?? route.meta.title,
        icon: item.icon ?? route.meta.icon
      }
    }
  }

  return sections
    .map(({ title, items }) => ({
      title,
      items: items
        .map((item): AppRouteRecord | undefined => {
          if (typeof item === 'string' || !('group' in item)) return page(item)
          const children = item.children.map(page).filter((child) => child !== undefined)
          if (!children.length) return
          return {
            // Pure rendering index, not a registered route or a navigation target.
            path: `sidebar:${item.group}`,
            meta: { title: item.title, icon: item.icon, showTextBadge: aggregateBadges(children) },
            children
          }
        })
        .filter((item) => item !== undefined)
    }))
    .filter((section) => section.items.length > 0)
}

/** Locate a visible page by its unchanged URL, independently of its original route parent. */
export function findSidebarSelection(sections: SidebarSection[], path: string) {
  function find(
    items: AppRouteRecord[],
    groups: string[]
  ): { path: string; groups: string[] } | undefined {
    for (const item of items) {
      if (item.children?.length) {
        const selection = find(item.children, [...groups, item.path])
        if (selection) return selection
      } else if (item.path === path) {
        return { path, groups }
      }
    }
  }
  for (const section of sections) {
    const selection = find(section.items, [])
    if (selection) return selection
  }
}
