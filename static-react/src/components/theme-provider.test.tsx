import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest'
import { useTheme } from '@/components/theme-context'
import { ThemeProvider } from '@/components/theme-provider'
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
})
