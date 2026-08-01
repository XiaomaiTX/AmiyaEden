import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, test } from 'vitest'
import { AppSidebar } from '@/components/app-sidebar'
import { ThemeProvider } from '@/components/theme-provider'
import { SidebarProvider } from '@/components/ui/sidebar'
import { I18nProvider } from '@/i18n'
import { usePreferenceStore, useSessionStore, useSidebarNavigationStore } from '@/stores'
import { SIDEBAR_NAVIGATION_STORE_KEY } from '@/stores/persistence-keys'

function renderSidebar(initialPath: string) {
  return render(
    <ThemeProvider>
      <I18nProvider>
        <SidebarProvider>
          <MemoryRouter initialEntries={[initialPath]}>
            <AppSidebar />
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

    expect(screen.getByRole('button', { name: 'EVE 人物信息' })).toHaveAttribute(
      'aria-expanded',
      'true'
    )
    expect(screen.getByRole('link', { name: '人物资产' })).toHaveAttribute('data-active', 'true')
    expect(screen.getByText('钱包流水')).toBeInTheDocument()
  })

  test('persists manual group expansion and restores it after remounting', async () => {
    const user = userEvent.setup()
    const firstRender = renderSidebar('/dashboard/console')

    const infoGroup = screen.getByRole('button', { name: 'EVE 人物信息' })
    await user.click(infoGroup)

    expect(useSidebarNavigationStore.getState().expandedMenuGroupKeys).toEqual(['nav.group.info'])
    expect(infoGroup).toHaveAttribute('aria-expanded', 'true')

    firstRender.unmount()
    renderSidebar('/dashboard/console')

    expect(screen.getByRole('button', { name: 'EVE 人物信息' })).toHaveAttribute(
      'aria-expanded',
      'true'
    )
  })

  test('keeps the current route group open when its trigger is clicked', async () => {
    const user = userEvent.setup()
    renderSidebar('/info/assets')

    const infoGroup = screen.getByRole('button', { name: 'EVE 人物信息' })
    await user.click(infoGroup)

    expect(infoGroup).toHaveAttribute('aria-expanded', 'true')
  })

  test('filters routes that the current session cannot access', () => {
    renderSidebar('/dashboard/console')

    expect(screen.getByRole('button', { name: '仪表盘' })).toBeInTheDocument()
    expect(screen.queryByText('军团刷怪报表')).not.toBeInTheDocument()
  })
})
