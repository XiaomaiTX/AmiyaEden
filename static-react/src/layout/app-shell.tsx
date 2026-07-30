import { AppSidebar } from '@/components/app-sidebar'
import { SidebarInset, SidebarProvider } from '@/components/ui/sidebar'
import { HeaderBar } from '@/layout/header-bar'
import { GlobalHost } from '@/layout/global-host'
import { PageContent } from '@/layout/page-content'
import { usePreferenceStore } from '@/stores'
import { WorktabBar } from '@/layout/worktab-bar'

export function AppShell() {
  const collapsed = usePreferenceStore((state) => state.sidebarCollapsed)
  const setSidebarCollapsed = usePreferenceStore((state) => state.setSidebarCollapsed)

  return (
    <SidebarProvider open={!collapsed} onOpenChange={(open) => setSidebarCollapsed(!open)}>
      <AppSidebar />
      <SidebarInset>
        <HeaderBar />
        <WorktabBar />
        <PageContent />
      </SidebarInset>
      <GlobalHost />
    </SidebarProvider>
  )
}
