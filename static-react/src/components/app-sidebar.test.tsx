import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, useNavigate } from 'react-router-dom'
import { RouterProvider as AriaRouterProvider } from 'react-aria-components'
import { beforeEach, describe, expect, test } from 'vitest'
import { AppSidebar } from '@/components/app-sidebar'
import { ThemeProvider } from '@/components/theme-provider'
import { SidebarProvider } from '@/components/ui/sidebar'
import { I18nProvider } from '@/i18n'
import { usePreferenceStore, useSessionStore, useSidebarNavigationStore } from '@/stores'
import { SIDEBAR_NAVIGATION_STORE_KEY } from '@/stores/persistence-keys'

function RoutedSidebar() {
  const navigate = useNavigate()
  return (
    <AriaRouterProvider navigate={(target) => navigate(target)}>
      <AppSidebar />
    </AriaRouterProvider>
  )
}

function renderSidebar(initialPath: string) {
  return render(
    <ThemeProvider>
      <I18nProvider>
        <SidebarProvider>
          <MemoryRouter initialEntries={[initialPath]}>
            <RoutedSidebar />
          </MemoryRouter>
        </SidebarProvider>
      </I18nProvider>
    </ThemeProvider>
  )
}

describe('AppSidebar', () => {
  beforeEach(() => {
    localStorage.removeItem(SIDEBAR_NAVIGATION_STORE_KEY)
    useSidebarNavigationStore.getState().clear()
    usePreferenceStore.setState({
      locale: 'zh-CN',
      sidebarCollapsed: false,
      theme: 'system',
    })

    useSessionStore.setState({
      isLoggedIn: true,
      accessToken: 'token-123',
      characterId: 1001,
      characterName: 'Amiya',
      roles: ['member'],
      corpCapabilities: ['menu.dashboard', 'menu.info', 'info.assets.read', 'info.wallet.read'],
      isCurrentlyNewbro: false,
      isMentorMenteeEligible: false,
      hydratedAt: null,
    })
  })

  test('auto-expands the active route group and marks the active child', () => {
    renderSidebar('/info/assets')

    expect(screen.getByRole('button', { name: '人物信息' })).toHaveAttribute(
      'aria-expanded',
      'true'
    )
    expect(screen.getByRole('link', { name: '人物资产' })).toHaveAttribute('data-active', 'true')
    expect(screen.getByText('ISK 钱包流水')).toBeInTheDocument()
  })

  test('persists manual group expansion and restores it after remounting', async () => {
    const user = userEvent.setup()
    const firstRender = renderSidebar('/dashboard/console')

    const infoGroup = screen.getByRole('button', { name: '人物信息' })
    await user.click(infoGroup)

    expect(useSidebarNavigationStore.getState().expandedMenuGroupKeys).toEqual(['nav.group.info'])
    expect(infoGroup).toHaveAttribute('aria-expanded', 'true')

    firstRender.unmount()
    renderSidebar('/dashboard/console')

    expect(screen.getByRole('button', { name: '人物信息' })).toHaveAttribute(
      'aria-expanded',
      'true'
    )
  })

  test('keeps the current route group open when its trigger is clicked', async () => {
    const user = userEvent.setup()
    renderSidebar('/info/assets')

    const infoGroup = screen.getByRole('button', { name: '人物信息' })
    await user.click(infoGroup)

    expect(infoGroup).toHaveAttribute('aria-expanded', 'true')
  })

  test('filters routes that the current session cannot access', () => {
    renderSidebar('/dashboard/console')

    expect(screen.getByRole('heading', { name: '首页' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: '控制台' })).toHaveAttribute('data-active', 'true')
    expect(screen.getByRole('link', { name: '军团公告' })).toHaveAttribute('href', '/announcements')
    expect(screen.queryByRole('heading', { name: '军团运营' })).not.toBeInTheDocument()
    expect(screen.queryByRole('heading', { name: '系统管理' })).not.toBeInTheDocument()
  })

  test('Mumble is under Utilities while Fuxi Hall stays a direct link', async () => {
    const user = userEvent.setup()
    useSessionStore.setState({ corpCapabilities: ['menu.fuxi_hall'] })
    renderSidebar('/fuxi-hall')

    const hall = screen.getByRole('link', { name: '伏羲名人堂' })
    const utilities = screen.getByRole('button', { name: '实用工具' })
    expect(utilities).toHaveAttribute('aria-expanded', 'false')
    expect(screen.queryByRole('link', { name: 'Mumble 凭据' })).not.toBeInTheDocument()
    await user.click(utilities)
    const mumble = screen.getByRole('link', { name: 'Mumble 凭据' })
    expect(hall).toHaveAttribute('href', '/fuxi-hall')
    expect(hall).toHaveAttribute('data-active', 'true')
    expect(screen.queryByRole('button', { name: '伏羲名人堂' })).not.toBeInTheDocument()
    expect(screen.queryByText('伏羲大厅管理')).not.toBeInTheDocument()

    await user.click(mumble)
    expect(mumble).toHaveAttribute('data-active', 'true')
    await user.click(hall)
    expect(hall).toHaveAttribute('data-active', 'true')
  })
})
