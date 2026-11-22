<template>
  <ElConfigProvider size="default" :locale="locales[language]" :z-index="3000">
    <RouterView></RouterView>
  </ElConfigProvider>
</template>

<script setup lang="ts">
  import { useUserStore } from './store/modules/user'
  import { useAdmissionStore } from './store/modules/admissions'
  import zh from 'element-plus/es/locale/lang/zh-cn'
  import en from 'element-plus/es/locale/lang/en'
  import { systemUpgrade } from './utils/sys'
  import { toggleTransition } from './utils/ui/animation'
  import { checkStorageCompatibility } from './utils/storage'
  import { initializeTheme } from './hooks/core/useTheme'

  const userStore = useUserStore()
  const admissions = useAdmissionStore()
  const admissionIdentityKey = computed(() => {
    if (!userStore.isLogin) return ''
    const info = userStore.info
    const charKey = (info.characters ?? [])
      .map((char) => `${char.character_id}:${char.corporation_id ?? 0}`)
      .join(',')
    return [
      info.userId ?? '',
      info.profileComplete ? '1' : '0',
      info.qq ?? '',
      info.discordId ?? '',
      (info.roles ?? []).join(','),
      charKey
    ].join('|')
  })
  watch(
    () => [userStore.isLogin, userStore.info.userId, admissionIdentityKey.value] as const,
    ([isLogin, userId], [prevIsLogin, prevUserId]) => {
      if (!isLogin) {
        admissions.clear()
        return
      }
      if (!prevIsLogin || userId !== prevUserId) admissions.clear()
      void admissions.refresh()
    }
  )
  const { language } = storeToRefs(userStore)

  const locales = {
    zh: zh,
    en: en
  }

  onBeforeMount(() => {
    toggleTransition(true)
    initializeTheme()
  })

  onMounted(() => {
    checkStorageCompatibility()
    toggleTransition(false)
    systemUpgrade()
  })
</script>
