import { Moon, Sun } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useTheme } from '@/components/theme-context'
import { useI18n } from '@/i18n'
import { type ThemeMode } from '@/stores'

const themeOptions: Array<{ value: ThemeMode; labelKey: string }> = [
  { value: 'light', labelKey: 'common.light' },
  { value: 'dark', labelKey: 'common.dark' },
  { value: 'system', labelKey: 'common.system' },
]

export function ModeToggle() {
  const { t } = useI18n()
  const { theme, resolvedTheme, setTheme } = useTheme()

  const icon = resolvedTheme === 'dark' ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />

  return (
    <DropdownMenuTrigger>
      <Button
        type="button"
        variant="outline"
        size="icon-sm"
        aria-label={t('common.switchTheme')}
      >
        {icon}
      </Button>
      <DropdownMenu
        aria-label={t('common.switchTheme')}
        selectionMode="single"
        selectedKeys={[theme]}
        onSelectionChange={(keys) => {
          const key = Array.from(keys)[0]
          if (key === 'light' || key === 'dark' || key === 'system') {
            setTheme(key)
          }
        }}
      >
        {themeOptions.map((option) => (
          <DropdownMenuItem key={option.value} id={option.value}>
            {t(option.labelKey)}
          </DropdownMenuItem>
        ))}
      </DropdownMenu>
    </DropdownMenuTrigger>
  )
}
