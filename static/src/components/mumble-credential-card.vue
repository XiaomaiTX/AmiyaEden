<template>
  <div class="art-card-sm p-6 mb-4">
    <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
      <div class="max-w-2xl">
        <h2 class="text-lg font-medium">{{ t('characters.mumble.title') }}</h2>
        <p class="mt-1 text-sm text-g-500">{{ t('characters.mumble.subtitle') }}</p>
      </div>
      <ElTag v-if="status" :type="status.enabled ? 'success' : 'info'" effect="light" round>
        {{ status.enabled ? t('characters.mumble.enabled') : t('characters.mumble.disabled') }}
      </ElTag>
    </div>

    <ElAlert
      v-if="secret"
      type="warning"
      :closable="false"
      class="mt-4"
      :title="t('characters.mumble.oneTimeWarning')"
    >
      <div class="mt-3 flex flex-col gap-2 sm:flex-row sm:items-center">
        <span>{{ t('characters.mumble.password') }}</span>
        <code class="min-w-0 flex-1 break-all rounded bg-bg-100 px-3 py-2">{{ secret }}</code>
        <ElButton @click="copySecret">{{ t('common.copy') }}</ElButton>
        <ElButton text @click="secret = ''">{{ t('characters.mumble.dismissSecret') }}</ElButton>
      </div>
    </ElAlert>

    <div class="mt-5 grid gap-3 text-sm sm:grid-cols-3">
      <div>
        <span class="text-g-500">{{ t('characters.mumble.username') }}</span>
        <div class="mt-1 flex items-center gap-1">
          <p class="min-w-0 flex-1 truncate font-medium">{{ canonicalName || '—' }}</p>
          <ArtCopyButton :text="canonicalName" />
        </div>
      </div>
      <div>
        <span class="text-g-500">{{ t('characters.mumble.serverAddress') }}</span>
        <div class="mt-1 flex items-center gap-1">
          <p class="min-w-0 flex-1 truncate font-medium">{{ status?.server_address || '—' }}</p>
          <ArtCopyButton :text="status?.server_address" />
        </div>
      </div>
      <div>
        <span class="text-g-500">{{ t('characters.mumble.serverPort') }}</span>
        <div class="mt-1 flex items-center gap-1">
          <p class="min-w-0 flex-1 truncate font-medium">{{ status?.server_port || '—' }}</p>
          <ArtCopyButton :text="status?.server_port" />
        </div>
      </div>
    </div>

    <div class="mt-5 flex flex-wrap justify-end gap-2">
      <ElButton
        v-if="!status?.enabled"
        type="primary"
        :loading="busy"
        :disabled="!canonicalName"
        @click="issue(false)"
        >{{ t('characters.mumble.create') }}</ElButton
      >
      <ElButton v-if="status?.enabled" :loading="busy" @click="issue(true)">{{
        t('characters.mumble.rotate')
      }}</ElButton>
      <ElButton v-if="status?.enabled" type="danger" plain :loading="busy" @click="revoke">{{
        t('characters.mumble.revoke')
      }}</ElButton>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { ElMessage, ElMessageBox } from 'element-plus'
  import { useI18n } from 'vue-i18n'
  import {
    createMumbleCredential,
    fetchMumbleCredential,
    revokeMumbleCredential,
    rotateMumbleCredential
  } from '@/api/mumble'

  defineProps<{ canonicalName: string }>()
  const { t } = useI18n()
  const status = ref<Api.Mumble.CredentialStatus | null>(null)
  const secret = ref('')
  const busy = ref(false)

  async function load() {
    busy.value = true
    try {
      status.value = await fetchMumbleCredential()
    } catch {
      ElMessage.error(t('characters.mumble.loadFailed'))
    } finally {
      busy.value = false
    }
  }

  async function issue(rotate: boolean) {
    if (rotate) {
      try {
        await ElMessageBox.confirm(
          t('characters.mumble.rotateConfirm'),
          t('characters.mumble.title'),
          { type: 'warning' }
        )
      } catch {
        return
      }
    }
    busy.value = true
    try {
      const result = rotate ? await rotateMumbleCredential() : await createMumbleCredential()
      status.value = result.credential
      secret.value = result.password
      ElMessage.success(t('characters.mumble.issued'))
    } catch (error) {
      if (error !== 'cancel' && error !== 'close')
        ElMessage.error(t('characters.mumble.operationFailed'))
    } finally {
      busy.value = false
    }
  }

  async function revoke() {
    try {
      await ElMessageBox.confirm(
        t('characters.mumble.revokeConfirm'),
        t('characters.mumble.title'),
        { type: 'warning' }
      )
      busy.value = true
      await revokeMumbleCredential()
      if (status.value) status.value.enabled = false
      secret.value = ''
      ElMessage.success(t('characters.mumble.revoked'))
    } catch (error) {
      if (error !== 'cancel' && error !== 'close')
        ElMessage.error(t('characters.mumble.operationFailed'))
    } finally {
      busy.value = false
    }
  }

  async function copySecret() {
    try {
      await navigator.clipboard.writeText(secret.value)
      ElMessage.success(t('characters.mumble.copied'))
    } catch {
      ElMessage.error(t('common.copyFailed'))
    }
  }

  onMounted(load)
</script>
