import { HeaderBar } from '@/layout/header-bar'
import { GlobalHost } from '@/layout/global-host'
import { PageContent } from '@/layout/page-content'
import { SidebarMenu } from '@/layout/sidebar-menu'

export function AppShell() {
  return (
    <div className="flex min-h-screen bg-background">
      <SidebarMenu />
      <div className="flex min-w-0 flex-1 flex-col">
        <HeaderBar />
        <PageContent />
      </div>
      <GlobalHost />
    </div>
  )
}
