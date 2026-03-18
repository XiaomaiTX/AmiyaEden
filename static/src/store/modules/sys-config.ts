import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { fetchBasicConfig, updateBasicConfig } from '@/api/sys-config'

export const useSysConfigStore = defineStore(
  'sysConfig',
  () => {
    const config = ref<Api.SysConfig.BasicConfig>({
      corp_id: 1,
      site_title: 'Amiya eden'
    })

    const loading = ref(false)
    const loaded = ref(false)

    const logoUrl = computed(
      () => `https://images.evetech.net/corporations/${config.value.corp_id}/logo?size=128`
    )

    const siteTitle = computed(() => config.value.site_title)

    async function loadConfig() {
      loading.value = true
      try {
        const res = await fetchBasicConfig()
        config.value = res
      } catch (error) {
        console.error('Failed to load sys config:', error)
      } finally {
        loaded.value = true
        loading.value = false
      }
    }

    async function ensureLoaded() {
      if (loaded.value || loading.value) return
      await loadConfig()
    }

    async function updateConfig(data: Api.SysConfig.UpdateBasicConfigParams) {
      await updateBasicConfig(data)
      Object.assign(config.value, data)
    }

    return {
      config,
      logoUrl,
      siteTitle,
      loading,
      loaded,
      loadConfig,
      ensureLoaded,
      updateConfig
    }
  },
  {
    persist: {
      key: 'sysConfig',
      storage: localStorage
    }
  }
)
