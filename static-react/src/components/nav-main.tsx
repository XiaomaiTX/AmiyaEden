import { useEffect } from 'react'
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
import { useSessionStore, useSidebarNavigationStore } from '@/stores'

export function NavMain({
  items,
}: {
  items: {
    groupKey: string
    title: string
    icon: React.ReactNode
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
  const characterId = useSessionStore((state) => state.characterId)
  const ownerCharacterId = useSidebarNavigationStore((state) => state.ownerCharacterId)
  const expandedMenuGroupKeys = useSidebarNavigationStore((state) => state.expandedMenuGroupKeys)
  const expandMenuGroup = useSidebarNavigationStore((state) => state.expandMenuGroup)
  const collapseMenuGroup = useSidebarNavigationStore((state) => state.collapseMenuGroup)
  const resetForCharacter = useSidebarNavigationStore((state) => state.resetForCharacter)

  useEffect(() => {
    resetForCharacter(characterId)
  }, [characterId, resetForCharacter])

  const isActivePath = (to: string) =>
    location.pathname === to || location.pathname.startsWith(`${to}/`)
  const persistedExpandedMenuGroupKeys =
    ownerCharacterId === characterId ? expandedMenuGroupKeys : []

  return (
    <SidebarGroup>
      <SidebarGroupLabel>{t('nav.home')}</SidebarGroupLabel>
      <SidebarMenu>
        {items.map((item) => {
          const isActiveGroup = item.items?.some((subItem) => isActivePath(subItem.url)) ?? false
          const isExpanded = isActiveGroup || persistedExpandedMenuGroupKeys.includes(item.groupKey)

          return (
            <Collapsible
              key={item.groupKey}
              isExpanded={isExpanded}
              onExpandedChange={(expanded) => {
                if (expanded) {
                  expandMenuGroup(item.groupKey)
                } else if (!isActiveGroup) {
                  collapseMenuGroup(item.groupKey)
                }
              }}
              className="group/collapsible"
            >
              <SidebarMenuItem>
                <SidebarMenuButton
                  slot="trigger"
                  tooltip={t(item.title)}
                  isActive={isActiveGroup}
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
                      {item.items.map((subItem) => (
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
