/**
 * useCorpCapability - 军团 capability 客户端门禁
 *
 * 用于按钮级和组件级权限判断：判断当前用户的军团 capability 集合（`/me.corp_capabilities`）
 * 是否满足给定动作。后端仍是最终授权，这里只负责避免入口可见但点击后 403 的体验问题。
 *
 * 语义：
 * - `hasCapability(key)`：用户是否拥有单个 capability（或为 super_admin / full_access）。
 * - `hasAllCapabilities(keys)`：全部命中。
 * - `hasAnyCapability(keys)`：至少命中一项。
 *
 * Stage 0A 之前用户端只检查路由 meta，按钮永远可见，导致 `shop.order.create` /
 * `ticket.user.reply` / `system.task.run` / `system.wallet.adjust` /
 * `system.audit.export` 等动作点击后才会收到 403。
 */

import { useUserStore } from '@/store/modules/user'

export const useCorpCapability = () => {
  const userStore = useUserStore()

  const roles = (): string[] => userStore.info?.roles ?? []
  const capabilities = (): string[] => userStore.info?.corpCapabilities ?? []

  const isSuperAdmin = (): boolean => roles().includes('super_admin')

  const hasCapability = (key: string): boolean => {
    if (isSuperAdmin()) return true
    return capabilities().includes(key)
  }

  const hasAllCapabilities = (keys: string[]): boolean => {
    if (keys.length === 0) return true
    if (isSuperAdmin()) return true
    return keys.every((key) => capabilities().includes(key))
  }

  const hasAnyCapability = (keys: string[]): boolean => {
    if (keys.length === 0) return true
    if (isSuperAdmin()) return true
    return keys.some((key) => capabilities().includes(key))
  }

  return {
    hasCapability,
    hasAllCapabilities,
    hasAnyCapability
  }
}
