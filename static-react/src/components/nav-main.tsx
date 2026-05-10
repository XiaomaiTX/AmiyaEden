import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible"
import {
  SidebarGroup,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
} from "@/components/ui/sidebar"
import { ChevronRightIcon } from "lucide-react"
import { NavLink, useLocation } from "react-router-dom"
import { useI18n } from "@/i18n"

export function NavMain({
  items,
}: {
  items: {
    title: string
    icon: React.ReactNode
    isActive?: boolean
    items?: {
      title: string
      url: string
    }[]
  }[]
}) {
  const { t } = useI18n()
  const location = useLocation()

  const isActivePath = (to: string) =>
    location.pathname === to || location.pathname.startsWith(`${to}/`)

  return (
    <SidebarGroup>
      <SidebarGroupLabel>{t("nav.home")}</SidebarGroupLabel>
      <SidebarMenu>
        {items.map((item) => (
          <Collapsible
            key={item.title}
            asChild
            defaultOpen={item.items?.some((subItem) => isActivePath(subItem.url))}
          >
            <SidebarMenuItem>
              <CollapsibleTrigger asChild>
                <SidebarMenuButton
                  tooltip={t(item.title)}
                  isActive={item.items?.some((subItem) => isActivePath(subItem.url))}
                >
                  {item.icon}
                  <span>{t(item.title)}</span>
                  <ChevronRightIcon className="ml-auto transition-transform group-data-[state=open]/collapsible:rotate-90" />
                </SidebarMenuButton>
              </CollapsibleTrigger>
              {item.items?.length ? (
                <>
                  <CollapsibleContent>
                    <SidebarMenuSub>
                      {item.items?.map((subItem) => (
                        <SidebarMenuSubItem key={subItem.title}>
                          <SidebarMenuSubButton
                            asChild
                            isActive={isActivePath(subItem.url)}
                          >
                            <NavLink to={subItem.url}>
                              <span>{t(subItem.title)}</span>
                            </NavLink>
                          </SidebarMenuSubButton>
                        </SidebarMenuSubItem>
                      ))}
                    </SidebarMenuSub>
                  </CollapsibleContent>
                </>
              ) : null}
            </SidebarMenuItem>
          </Collapsible>
        ))}
      </SidebarMenu>
    </SidebarGroup>
  )
}
