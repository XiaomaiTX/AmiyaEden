import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getAdmissionContext } from '@/api/admissions'
import { useMenuStore } from '@/store/modules/menu'
import { admissionMenuVisibility } from './admissions.helpers'
import type { AdmissionContext } from '@/types/api/admissions'
export const useAdmissionStore = defineStore('admissions', () => {
  const context = ref<AdmissionContext | null>(null)
  const loading = ref(false)
  const error = ref(false)
  let generation = 0
  let pending: Promise<void> | null = null
  function refresh(): Promise<void> {
    if (pending) return pending
    const current = ++generation
    loading.value = true
    error.value = false
    const menus = useMenuStore()
    // A refresh is not a loss of permission. Keep the current menu and form mounted.
    pending = (async () => {
      try {
        const value = await getAdmissionContext()
        if (generation === current) context.value = value
      } catch {
        if (generation === current) {
          context.value = null
          error.value = true
        }
      } finally {
        if (generation === current) {
          loading.value = false
          pending = null
          menus.menuList = admissionMenuVisibility(menus.menuList, context.value)
        }
      }
    })()
    return pending
  }
  function clear() {
    generation++
    pending = null
    context.value = null
    loading.value = false
    error.value = false
    const menus = useMenuStore()
    menus.menuList = admissionMenuVisibility(menus.menuList, null)
  }
  return { context, loading, error, refresh, clear }
})
