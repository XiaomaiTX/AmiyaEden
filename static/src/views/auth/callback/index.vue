<!-- EVE Online SSO 回调页面 - 处理 OAuth 跳转回来后的 token -->
<template>
  <div class="callback-page">
    <div class="callback-card" :class="{ 'registration-card': registrationTicket }">
      <ContactRegistration
        v-if="registrationTicket"
        :ticket="registrationTicket"
        @completed="finishRegistration"
        @login="goLogin"
      />
      <template v-else-if="status === 'loading'">
        <div class="icon-wrap spin">
          <el-icon :size="48"><Loading /></el-icon>
        </div>
        <p class="title">{{ $t('login.callback.loading') }}</p>
        <p class="sub">{{ $t('login.callback.loadingSub') }}</p>
      </template>

      <template v-else-if="status === 'success'">
        <div class="icon-wrap success">
          <el-icon :size="48"><CircleCheckFilled /></el-icon>
        </div>
        <p class="title">{{ $t('login.callback.success') }}</p>
        <p class="sub">{{ $t('login.callback.successSub', { name: userName }) }}</p>
      </template>

      <template v-else-if="status === 'error'">
        <div class="icon-wrap error">
          <el-icon :size="48"><CircleCloseFilled /></el-icon>
        </div>
        <p class="title">{{ $t('login.callback.error') }}</p>
        <p class="sub error-msg">{{ errMsg }}</p>
        <ElButton type="primary" class="mt-6" @click="goLogin">
          {{ $t('login.callback.retry') }}
        </ElButton>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { Loading, CircleCheckFilled, CircleCloseFilled } from '@element-plus/icons-vue'
  import { useI18n } from 'vue-i18n'
  import { useUserStore } from '@/store/modules/user'
  import { fetchGetUserInfo, isUserProfileComplete } from '@/api/auth'
  import ContactRegistration from './contact-registration.vue'

  defineOptions({ name: 'AuthCallback' })

  type Status = 'loading' | 'success' | 'error'

  const status = ref<Status>('loading')
  const errMsg = ref('')
  const userName = ref('')
  const { t } = useI18n()
  const route = useRoute()
  const router = useRouter()
  const userStore = useUserStore()
  const registrationKey = 'auth:registration'
  const registrationTicket = ref('')

  const goLogin = () => {
    sessionStorage.removeItem(registrationKey)
    return router.replace({ name: 'Login' })
  }

  async function finishRegistration(token: string) {
    sessionStorage.removeItem(registrationKey)
    registrationTicket.value = ''
    await finishLogin(token)
  }

  onMounted(async () => {
    const token = route.query.token as string
    const error = route.query.error as string

    // EVE 端拒绝授权
    if (error) {
      status.value = 'error'
      errMsg.value = (route.query.error_description as string) || error
      return
    }

    if (!token) {
      const ticket = (route.query.registration as string) || sessionStorage.getItem(registrationKey)
      if (ticket) {
        registrationTicket.value = ticket
        sessionStorage.setItem(registrationKey, ticket)
        await router.replace({ path: route.path })
        return
      }
      status.value = 'error'
      errMsg.value = t('authCallback.missingToken')
      return
    }

    sessionStorage.removeItem(registrationKey)
    await finishLogin(token)
  })

  async function finishLogin(token: string) {
    try {
      // 1. 存储 token 并设置登录状态
      userStore.setToken(token)
      userStore.setLoginStatus(true)

      // 2. 拉取用户信息
      const userInfo = await fetchGetUserInfo()
      userStore.setUserInfo(userInfo)
      userStore.checkAndClearWorktabs()

      userName.value = userInfo.userName
      status.value = 'success'

      // 3. 短暂停留后跳转
      const isProfileComplete = isUserProfileComplete(userInfo)
      const redirect = isProfileComplete ? (route.query.redirect as string) || '/' : '/characters'
      setTimeout(() => {
        router.replace(redirect)
      }, 1200)
    } catch (err: any) {
      // token 无效或请求失败
      userStore.setToken('')
      userStore.setLoginStatus(false)
      status.value = 'error'
      errMsg.value = err?.message ?? t('authCallback.verifyFailed')
    }
  }
</script>

<style lang="scss" scoped>
  .callback-page {
    display: flex;
    align-items: center;
    justify-content: center;
    box-sizing: border-box;
    width: 100%;
    min-height: 100svh;
    padding: 40px 20px;
    background: var(--el-fill-color-light, #f3f6fb);
  }

  .callback-card {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
    width: 380px;
    max-width: 100%;
    padding: 48px;
    border-radius: 16px;
    background: var(--el-bg-color, #fff);
    box-shadow: 0 16px 48px -20px rgb(32 48 71 / 18%);

    &.registration-card {
      width: 460px;
      padding: 0;
      background: transparent;
      box-shadow: none;
    }

    .icon-wrap {
      @apply mb-2;

      &.spin :deep(.el-icon) {
        animation: spin 1s linear infinite;
      }

      &.success :deep(.el-icon) {
        color: var(--el-color-success);
      }

      &.error :deep(.el-icon) {
        color: var(--el-color-danger);
      }
    }

    .title {
      @apply text-xl font-semibold text-g-800;
    }

    .sub {
      @apply text-sm text-g-500 text-center;
    }

    .error-msg {
      @apply text-red-400;
    }
  }

  @keyframes spin {
    from {
      transform: rotate(0deg);
    }
    to {
      transform: rotate(360deg);
    }
  }
</style>
