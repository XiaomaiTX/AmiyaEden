import { defineStore } from 'pinia'
import { ref } from 'vue'
import { fetchBadgeCounts as fetchBadgeCountsApi } from '@/api/badge'
import { hasNonGuestRole } from '@/router/core/menuAccess'
import { applyBadgeCountsToMenu } from './badge.helpers'
import { useMenuStore } from './menu'
import { useUserStore } from './user'

export const useBadgeStore = defineStore('badgeStore', () => {
  const badgeCounts = ref<Api.Badge.BadgeCounts>({})
  const menuStore = useMenuStore()

  const fetchBadgeCounts = async () => {
    // badge-counts is a Login route; guests and recruiter-only accounts would only get a 403.
    const counts = hasNonGuestRole(useUserStore().info.roles ?? [])
      ? await fetchBadgeCountsApi()
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
