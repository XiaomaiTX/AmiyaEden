import type { LucideIcon } from 'lucide-react'
import { Home, ShieldCheck } from 'lucide-react'

export interface ShellMenuItem {
  to: string
  labelKey: string
  icon: LucideIcon
}

export const shellMenuItems: ShellMenuItem[] = [
  {
    to: '/',
    labelKey: 'nav.home',
    icon: Home,
  },
  {
    to: '/admin-demo',
    labelKey: 'nav.permissionDemo',
    icon: ShieldCheck,
  },
]
