"use client"

import type { ComponentProps } from "react"
import { NavMain } from "@/components/nav-main"
import { NavUser } from "@/components/nav-user"
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar"
import { TerminalIcon } from "lucide-react"
import { NavLink } from "react-router-dom"
import { useI18n } from "@/i18n"
import { buildShellMenuGroups } from "@/layout/menu-config"
import { useBadgeStore, usePreferenceStore, useSessionStore } from "@/stores"

export function AppSidebar({ ...props }: ComponentProps<typeof Sidebar>) {
  const { t } = useI18n()
  const locale = usePreferenceStore((state) => state.locale)
  const characterName = useSessionStore((state) => state.characterName)
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
            <SidebarMenuButton size="lg" asChild>
              <NavLink to="/">
                <div className="flex aspect-square size-8 items-center justify-center rounded-lg bg-sidebar-primary text-sidebar-primary-foreground">
                  <TerminalIcon className="size-4" />
                </div>
                <div className="grid flex-1 text-left text-sm leading-tight">
                  <span className="truncate font-medium">AmiyaEden</span>
                  <span className="truncate text-xs">{t("shell.runtime")} · {locale}</span>
                </div>
              </NavLink>
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
            name: characterName ?? t("shell.guest"),
            email:
              roles.map((role) => t(`userAdmin.roles.${role}`)).join(", ") ||
              t("userAdmin.roles.guest"),
            avatar: "",
          }}
        />
      </SidebarFooter>
    </Sidebar>
  )
}
