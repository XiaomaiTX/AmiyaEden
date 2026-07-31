import { Collapsible, CollapsibleContent } from '@/components/ui/collapsible'
import {
  SidebarGroup,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuItem,
  SidebarMenuButton,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
} from '@/components/ui/sidebar'
import { ChevronRightIcon } from 'lucide-react'
import { useLocation } from 'react-router-dom'
import { useI18n } from '@/i18n'

export function NavMain({
  items,
}: {
  items: {
    title: string
    icon: React.ReactNode
    isActive?: boolean
    badge?: number
    items?: {
      title: string
      url: string
      badge?: number
    }[]
  }[]
}) {
  const { t } = useI18n()
  const location = useLocation()

  const isActivePath = (to: string) =>
    location.pathname === to || location.pathname.startsWith(`${to}/`)

  return (
    <SidebarGroup>
      <SidebarGroupLabel>{t('nav.home')}</SidebarGroupLabel>
      <SidebarMenu>
        {items.map((item) => {
          const expandedDefault = item.items?.some((subItem) => isActivePath(subItem.url))
          return (
            <Collapsible
              key={item.title}
              defaultExpanded={expandedDefault}
              className="group/collapsible"
            >
              <SidebarMenuItem>
                <SidebarMenuButton
                  slot="trigger"
                  tooltip={t(item.title)}
                  isActive={item.items?.some((subItem) => isActivePath(subItem.url))}
                >
                  {item.icon}
                  <span>{t(item.title)}</span>
                  {item.badge ? (
                    <span className="ml-auto rounded-full bg-primary px-1.5 text-[10px] text-primary-foreground">
                      {item.badge}
                    </span>
                  ) : null}
                  <ChevronRightIcon className="ml-auto transition-transform group-data-[expanded]/collapsible:rotate-90" />
                </SidebarMenuButton>
                {item.items?.length ? (
                  <CollapsibleContent>
                    <SidebarMenuSub>
                      {item.items?.map((subItem) => (
                        <SidebarMenuSubItem key={subItem.title}>
                          <SidebarMenuSubButton href={subItem.url} isActive={isActivePath(subItem.url)}>
                            <span>{t(subItem.title)}</span>
                            {subItem.badge ? (
                              <span className="ml-auto rounded-full bg-primary px-1.5 text-[10px] text-primary-foreground">
                                {subItem.badge}
                              </span>
                            ) : null}
                          </SidebarMenuSubButton>
                        </SidebarMenuSubItem>
                      ))}
                    </SidebarMenuSub>
                  </CollapsibleContent>
                ) : null}
              </SidebarMenuItem>
            </Collapsible>
          )
        })}
      </SidebarMenu>
    </SidebarGroup>
  )
}
