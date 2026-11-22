<template>
  <div class="min-w-0">
    <ElTabs v-if="canManage" v-model="activeTab" class="px-4 pt-4 md:px-6">
      <ElTabPane name="fame" :label="t('menus.fuxiHall.fame')" />
      <ElTabPane name="manage" :label="t('menus.fuxiHall.manage')" />
    </ElTabs>
    <!-- Keep both tabs mounted so switching does not refetch and flash a loading state. -->
    <KeepAlive>
      <FuxiHallManage v-if="canManage && activeTab === 'manage'" />
      <FuxiHallPublic v-else />
    </KeepAlive>
  </div>
</template>

<script setup lang="ts">
  import { computed, defineAsyncComponent, ref, watch } from 'vue'
  import { useI18n } from 'vue-i18n'
  import { useUserStore } from '@/store/modules/user'
  import FuxiHallPublic from './hall-of-fame/index.vue'

  defineOptions({ name: 'FuxiHall' })
  const FuxiHallManage = defineAsyncComponent(() => import('./manage/index.vue'))
  const { t } = useI18n()
  const userStore = useUserStore()
  const activeTab = ref('fame')
  const canManage = computed(() =>
    (userStore.info.roles ?? []).some((role) => ['super_admin', 'admin'].includes(role))
  )
  watch(canManage, (allowed) => {
    if (!allowed) activeTab.value = 'fame'
  })
</script>
