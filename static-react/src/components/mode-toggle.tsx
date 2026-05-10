import * as React from 'react'
import { Moon, Sun } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useTheme } from '@/components/theme-context'
import { useI18n } from '@/i18n'
import { cn } from '@/lib/utils'
import { type ThemeMode } from '@/stores'

const themeOptions: Array<{ value: ThemeMode; labelKey: string }> = [
  { value: 'light', labelKey: 'common.light' },
  { value: 'dark', labelKey: 'common.dark' },
  { value: 'system', labelKey: 'common.system' },
]

export function ModeToggle() {
  const { t } = useI18n()
  const { theme, resolvedTheme, setTheme } = useTheme()
  const [open, setOpen] = React.useState(false)
  const ref = React.useRef<HTMLDivElement | null>(null)

  React.useEffect(() => {
    if (!open) {
      return
    }

    const handlePointerDown = (event: PointerEvent) => {
      if (ref.current && !ref.current.contains(event.target as Node)) {
        setOpen(false)
      }
    }

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setOpen(false)
      }
    }

    document.addEventListener('pointerdown', handlePointerDown)
    document.addEventListener('keydown', handleKeyDown)

    return () => {
      document.removeEventListener('pointerdown', handlePointerDown)
      document.removeEventListener('keydown', handleKeyDown)
    }
  }, [open])

  const icon =
    resolvedTheme === 'dark' ? (
      <Sun className="h-4 w-4" />
    ) : (
      <Moon className="h-4 w-4" />
    )

  return (
    <div ref={ref} className="relative">
      <Button
        type="button"
        variant="outline"
        size="icon-sm"
        title={t('common.switchTheme')}
        aria-label={t('common.switchTheme')}
        aria-haspopup="menu"
        aria-expanded={open}
        onClick={() => setOpen((previous) => !previous)}
      >
        {icon}
      </Button>

      {open ? (
        <div
          role="menu"
          aria-label={t('common.switchTheme')}
          className="absolute right-0 top-full z-50 mt-2 min-w-28 rounded-md border bg-popover p-1 text-popover-foreground shadow-md"
        >
          {themeOptions.map((option) => {
            const isActive = theme === option.value
            return (
              <button
                key={option.value}
                type="button"
                role="menuitemradio"
                aria-checked={isActive}
                className={cn(
                  'flex w-full items-center rounded-sm px-3 py-2 text-left text-sm transition-colors hover:bg-muted',
                  isActive && 'bg-muted font-medium'
                )}
                onClick={() => {
                  setTheme(option.value)
                  setOpen(false)
                }}
              >
                {t(option.labelKey)}
              </button>
            )
          })}
        </div>
      ) : null}
    </div>
  )
}
