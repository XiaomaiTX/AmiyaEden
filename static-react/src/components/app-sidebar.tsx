'use client'

import type { ComponentProps } from 'react'
import { NavMain } from '@/components/nav-main'
import { NavUser } from '@/components/nav-user'
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from '@/components/ui/sidebar'
import { useI18n } from '@/i18n'
import { buildShellMenuGroups } from '@/layout/menu-config'
import { buildEveCharacterPortraitUrl, buildEveCorporationLogoUrl } from '@/lib/eve-image'
import { SYSTEM_IDENTITY } from '@/constants/system-identity'
import { useBadgeStore, useSessionStore } from '@/stores'

export function AppSidebar({ ...props }: ComponentProps<typeof Sidebar>) {
  const { t } = useI18n()
  const characterName = useSessionStore((state) => state.characterName)
  const characterId = useSessionStore((state) => state.characterId)
  const primaryCharacterId = useSessionStore((state) => state.primaryCharacterId)
  const isLoggedIn = useSessionStore((state) => state.isLoggedIn)
  const roles = useSessionStore((state) => state.roles)
  const corpCapabilities = useSessionStore((state) => state.corpCapabilities)
  const isCurrentlyNewbro = useSessionStore((state) => state.isCurrentlyNewbro)
  const isMentorMenteeEligible = useSessionStore((state) => state.isMentorMenteeEligible)
  const badgeCounts = useBadgeStore((state) => state.counts)

  const navMainItems = buildShellMenuGroups(
    {
      isLoggedIn,
      roles,
      corpCapabilities,
      isCurrentlyNewbro,
      isMentorMenteeEligible,
    },
    badgeCounts
  ).map((group) => ({
    title: group.labelKey,
    icon: <group.icon />,
    badge: group.badge,
    items: group.items.map((item) => ({
      title: item.labelKey,
      url: item.to,
      badge: item.badge,
    })),
  }))

  return (
    <Sidebar variant="inset" {...props}>
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton size="lg" href="/">
                <img
                  className="size-8 rounded-lg object-cover"
                  src={buildEveCorporationLogoUrl(SYSTEM_IDENTITY.corporationId)}
                  alt={SYSTEM_IDENTITY.displayName}
                />
                <div className="grid flex-1 text-left text-sm leading-tight">
                  <span className="truncate font-medium">{SYSTEM_IDENTITY.displayName}</span>
                  <span className="truncate text-xs">{t('shell.subtitle')}</span>
                </div>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>
      <SidebarContent>
        <NavMain items={navMainItems} />
      </SidebarContent>
      <SidebarFooter>
        <NavUser
          user={{
            name: characterName ?? t('shell.guest'),
            email:
              roles.map((role) => t(`userAdmin.roles.${role}`)).join(', ') ||
              t('userAdmin.roles.guest'),
            avatar: buildEveCharacterPortraitUrl(primaryCharacterId ?? characterId ?? 0),
          }}
        />
      </SidebarFooter>
    </Sidebar>
  )
}
