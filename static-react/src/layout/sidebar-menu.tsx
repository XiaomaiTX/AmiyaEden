import { NavLink } from 'react-router-dom'
import { useI18n } from '@/i18n'
import { shellMenuItems } from '@/layout/menu-config'
import { cn } from '@/lib/utils'
import { usePreferenceStore } from '@/stores'

export function SidebarMenu() {
  const { t } = useI18n()
  const collapsed = usePreferenceStore((state) => state.sidebarCollapsed)

  return (
    <aside
      className={cn(
        'hidden h-screen border-r bg-sidebar text-sidebar-foreground md:flex md:flex-col',
        collapsed ? 'w-20' : 'w-64'
      )}
      aria-label="primary-sidebar"
    >
      <div className="flex h-14 items-center border-b px-4">
        <span className="text-sm font-semibold tracking-wide">AmiyaEden</span>
      </div>
      <nav className="flex flex-1 flex-col gap-1 p-3">
        {shellMenuItems.map((item) => {
          const Icon = item.icon
          return (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.to === '/'}
              className={({ isActive }) =>
                cn(
                  'flex items-center gap-3 rounded-md px-3 py-2 text-sm transition-colors',
                  isActive
                    ? 'bg-sidebar-primary text-sidebar-primary-foreground'
                    : 'hover:bg-sidebar-accent hover:text-sidebar-accent-foreground'
                )
              }
            >
              <Icon className="h-4 w-4 shrink-0" />
              {!collapsed ? <span>{t(item.labelKey)}</span> : null}
            </NavLink>
          )
        })}
      </nav>
    </aside>
  )
}
