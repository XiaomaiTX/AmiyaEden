import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getAdmissionContext } from '@/api/generated/client'
import { unwrapContract } from '@/api/response'
import { useMenuStore } from '@/store/modules/menu'
import { admissionMenuVisibility, createAdmissionLoader } from './admissions.helpers'
import type { AdmissionContext } from '@/api/generated/client'
export const useAdmissionStore = defineStore('admissions', () => {
  const context = ref<AdmissionContext | null>(null)
  const loading = ref(false)
  const loaded = ref(false)
  const error = ref(false)
  const loader = createAdmissionLoader(
    () =>
      unwrapContract(
        getAdmissionContext({ showErrorMessage: false }),
        'admissions.errors.internal'
      ),
    {
      setContext: (value) => {
        context.value = value
      },
      setLoading: (value) => {
        loading.value = value
      },
      setLoaded: (value) => {
        loaded.value = value
      },
      setError: (value) => {
        error.value = value
      },
      onSettled: (nextContext) => {
        const menus = useMenuStore()
        menus.menuList = admissionMenuVisibility(menus.menuList, nextContext)
      }
    }
  )
  return {
    context,
    loading,
    loaded,
    error,
    refresh: loader.refresh,
    ensureLoaded: loader.ensureLoaded,
    clear: loader.clear
  }
})
