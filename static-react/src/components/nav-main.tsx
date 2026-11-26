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
import type { ShellMenuSection } from '@/layout/menu-config'
import { useSessionStore, useSidebarNavigationStore } from '@/stores'

export function NavMain({ sections }: { sections: ShellMenuSection[] }) {
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

  return sections.map(({ labelKey, entries }) => (
    <SidebarGroup key={labelKey}>
      <SidebarGroupLabel role="heading" aria-level={2}>
        {t(labelKey)}
      </SidebarGroupLabel>
      <SidebarMenu>
        {entries.map((item) => {
          if (item.kind === 'link') {
            return (
              <SidebarMenuItem key={item.to}>
                <SidebarMenuButton
                  href={item.to}
                  tooltip={t(item.labelKey)}
                  isActive={isActivePath(item.to)}
                >
                  <item.icon />
                  <span>{t(item.labelKey)}</span>
                  {item.badge ? (
                    <span className="ml-auto mr-6 rounded-full bg-primary px-1.5 text-[10px] text-primary-foreground">
                      {item.badge}
                    </span>
                  ) : null}
                </SidebarMenuButton>
              </SidebarMenuItem>
            )
          }
          const isActiveGroup = item.items.some((subItem) => isActivePath(subItem.to))
          const isExpanded = isActiveGroup || persistedExpandedMenuGroupKeys.includes(item.labelKey)

          return (
            <Collapsible
              key={item.labelKey}
              isExpanded={isExpanded}
              onExpandedChange={(expanded) => {
                if (expanded) {
                  expandMenuGroup(item.labelKey)
                } else if (!isActiveGroup) {
                  collapseMenuGroup(item.labelKey)
                }
              }}
              className="group/collapsible"
            >
              <SidebarMenuItem>
                <SidebarMenuButton
                  slot="trigger"
                  tooltip={t(item.labelKey)}
                  isActive={isActiveGroup}
                >
                  <item.icon />
                  <span>{t(item.labelKey)}</span>
                  {item.badge ? (
                    <span className="ml-auto rounded-full bg-primary px-1.5 text-[10px] text-primary-foreground">
                      {item.badge}
                    </span>
                  ) : null}
                  <ChevronRightIcon
                    className={`${item.badge ? '' : 'ml-auto '}shrink-0 transition-transform group-data-[expanded]/collapsible:rotate-90`}
                  />
                </SidebarMenuButton>
                <CollapsibleContent>
                  <SidebarMenuSub>
                    {item.items.map((subItem) => (
                      <SidebarMenuSubItem key={subItem.to}>
                        <SidebarMenuSubButton href={subItem.to} isActive={isActivePath(subItem.to)}>
                          <span>{t(subItem.labelKey)}</span>
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
              </SidebarMenuItem>
            </Collapsible>
          )
        })}
      </SidebarMenu>
    </SidebarGroup>
  ))
}
