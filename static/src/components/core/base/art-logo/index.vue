<!-- 系统logo -->
<template>
  <div class="flex-cc">
    <img v-if="logoUrl" :style="logoStyle" :src="logoUrl" alt="logo" class="w-full h-full" />
  </div>
</template>

<script setup lang="ts">
  import { useSysConfigStore } from '@/store/modules/sys-config'

  defineOptions({ name: 'ArtLogo' })

  interface Props {
    /** logo 大小 */
    size?: number | string
  }

  const props = withDefaults(defineProps<Props>(), {
    size: 36
  })

  const sysConfigStore = useSysConfigStore()

  const logoUrl = computed(() => sysConfigStore.logoUrl)
  const logoStyle = computed(() => ({ width: `${props.size}px` }))

  onMounted(() => {
    sysConfigStore.ensureLoaded()
  })
</script>
