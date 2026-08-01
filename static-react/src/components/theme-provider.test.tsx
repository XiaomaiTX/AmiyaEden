import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import { useTheme } from '@/components/theme-context'
import { ModeToggle } from '@/components/mode-toggle'
import { ThemeProvider } from '@/components/theme-provider'
import { I18nProvider } from '@/i18n'
import { usePreferenceStore } from '@/stores'

function ThemeConsumer() {
  const { resolvedTheme, setTheme } = useTheme()

  return (
    <div>
      <span data-testid="theme">{resolvedTheme}</span>
      <button type="button" onClick={() => setTheme('dark')}>
        dark
      </button>
    </div>
  )
}

function ModeToggleHarness() {
  return (
    <ThemeProvider>
      <I18nProvider>
        <ModeToggle />
      </I18nProvider>
    </ThemeProvider>
  )
}

describe('ThemeProvider', () => {
  beforeEach(() => {
    usePreferenceStore.setState({
      locale: 'zh-CN',
      sidebarCollapsed: false,
      theme: 'system',
    })

    vi.stubGlobal('matchMedia', (query: string) => ({
      matches: query.includes('dark') ? false : false,
      media: query,
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }))
    document.documentElement.classList.remove('light', 'dark')
    document.documentElement.style.colorScheme = ''
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  test('applies the selected theme to the root element', async () => {
    const user = userEvent.setup()

    render(
      <ThemeProvider>
        <ThemeConsumer />
      </ThemeProvider>
    )

    await waitFor(() => {
      expect(document.documentElement.classList.contains('light')).toBe(true)
    })

    await user.click(screen.getByRole('button', { name: 'dark' }))

    await waitFor(() => {
      expect(document.documentElement.classList.contains('dark')).toBe(true)
      expect(document.documentElement.style.colorScheme).toBe('dark')
    })
  })

  test('applies light mode selected from the theme menu', async () => {
    const user = userEvent.setup()

    render(<ModeToggleHarness />)
    await user.click(screen.getByRole('button', { name: '切换主题' }))
    await user.click(screen.getByRole('menuitemradio', { name: '浅色' }))

    await waitFor(() => {
      expect(document.documentElement.classList.contains('light')).toBe(true)
      expect(document.documentElement.classList.contains('dark')).toBe(false)
      expect(usePreferenceStore.getState().theme).toBe('light')
    })
  })

  test('falls back to system mode for an invalid persisted theme', async () => {
    usePreferenceStore.setState({ theme: 'invalid' as never })

    render(
      <ThemeProvider>
        <ThemeConsumer />
      </ThemeProvider>
    )

    await waitFor(() => {
      expect(screen.getByTestId('theme')).toHaveTextContent('light')
      expect(document.documentElement.classList.contains('light')).toBe(true)
    })
  })
})
