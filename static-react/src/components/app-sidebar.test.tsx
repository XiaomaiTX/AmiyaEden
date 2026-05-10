import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, test } from 'vitest'
import { AppSidebar } from '@/components/app-sidebar'
import { ThemeProvider } from '@/components/theme-provider'
import { SidebarProvider } from '@/components/ui/sidebar'
import { I18nProvider } from '@/i18n'
import { usePreferenceStore, useSessionStore } from '@/stores'

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
      authList: [],
      isCurrentlyNewbro: false,
      isMentorMenteeEligible: false,
      hydratedAt: null,
    })
  })

  test('auto-expands the active route group and marks the active child', () => {
    renderSidebar('/info/assets')

    expect(screen.getByRole('button', { name: 'EVE 人物信息' })).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByRole('link', { name: '人物资产' })).toHaveAttribute('data-active', 'true')
    expect(screen.getByText('钱包流水')).toBeInTheDocument()
  })

  test('filters routes that the current session cannot access', () => {
    renderSidebar('/dashboard/console')

    expect(screen.getByRole('button', { name: '仪表盘' })).toBeInTheDocument()
    expect(screen.queryByText('军团刷怪报表')).not.toBeInTheDocument()
  })
})
