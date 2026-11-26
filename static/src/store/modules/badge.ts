import type { BadgeCounts } from '@/api/generated/client'
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getBadgeCounts } from '@/api/generated/client'
import { unwrapContract } from '@/api/response'
import { hasNonGuestRole } from '@/router/core/menuAccess'
import { applyBadgeCountsToMenu } from './badge.helpers'
import { useMenuStore } from './menu'
import { useUserStore } from './user'

export const useBadgeStore = defineStore('badgeStore', () => {
  const badgeCounts = ref<BadgeCounts>({})
  const menuStore = useMenuStore()

  const fetchBadgeCounts = async () => {
    // badge-counts is a Login route; guests and recruiter-only accounts would only get a 403.
    const counts = hasNonGuestRole(useUserStore().info.roles ?? [])
      ? await unwrapContract(getBadgeCounts())
      : {}
    badgeCounts.value = counts
    applyBadgeCountsToMenu(menuStore.menuList, counts)
    return counts
  }

  const clearBadgeCounts = () => {
    badgeCounts.value = {}
    applyBadgeCountsToMenu(menuStore.menuList, {})
  }

  return {
    badgeCounts,
    fetchBadgeCounts,
    clearBadgeCounts
  }
})
