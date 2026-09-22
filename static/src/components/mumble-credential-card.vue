<template>
  <div class="art-card-sm p-6 mb-4">
    <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
      <div class="max-w-2xl">
        <h2 class="text-lg font-medium">{{ t('mumble.credential.title') }}</h2>
        <p class="mt-1 text-sm text-g-500">{{ t('mumble.credential.subtitle') }}</p>
      </div>
      <ElTag v-if="status" :type="status.enabled ? 'success' : 'info'" effect="light" round>
        {{ status.enabled ? t('mumble.credential.enabled') : t('mumble.credential.disabled') }}
      </ElTag>
    </div>

    <ElAlert
      v-if="secret"
      type="warning"
      :closable="false"
      class="mt-4"
      :title="t('mumble.credential.oneTimeWarning')"
    >
      <div class="mt-3 flex flex-col gap-2 sm:flex-row sm:items-center">
        <span>{{ t('mumble.credential.password') }}</span>
        <code class="min-w-0 flex-1 break-all rounded bg-bg-100 px-3 py-2">{{ secret }}</code>
        <ElButton @click="copySecret">{{ t('common.copy') }}</ElButton>
        <ElButton text @click="secret = ''">{{ t('mumble.credential.dismissSecret') }}</ElButton>
      </div>
    </ElAlert>

    <div class="mt-5 flex flex-col gap-3 text-sm">
      <div>
        <span class="text-g-500">{{ t('mumble.credential.username') }}</span>
        <div class="mt-1 flex items-center gap-1">
          <p class="min-w-0 flex-1 truncate font-medium">{{ canonicalName || '—' }}</p>
          <ArtCopyButton :text="canonicalName" />
        </div>
      </div>
      <div>
        <span class="text-g-500">{{ t('mumble.credential.nodes') }}</span>
        <p v-if="!status?.servers?.length" class="mt-1 text-g-500">
          {{ t('mumble.credential.noNodes') }}
        </p>
        <div v-else class="mt-1 flex flex-col gap-2">
          <div
            v-for="node in status?.servers ?? []"
            :key="`${node.address}:${node.port}`"
            class="flex items-start gap-2"
          >
            <div class="min-w-0 flex-1">
              <div class="flex flex-wrap items-baseline gap-x-2">
                <span class="font-medium">{{ node.name || node.address }}</span>
                <code class="break-all rounded bg-bg-100 px-2 py-0.5">{{
                  node.port > 0 ? `${node.address}:${node.port}` : node.address
                }}</code>
              </div>
              <p v-if="node.description" class="mt-0.5 text-xs text-g-500">{{
                node.description
              }}</p>
            </div>
            <ArtCopyButton :text="node.port > 0 ? `${node.address}:${node.port}` : node.address" />
          </div>
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
        >{{ t('mumble.credential.create') }}</ElButton
      >
      <ElButton v-if="status?.enabled" :loading="busy" @click="issue(true)">{{
        t('mumble.credential.rotate')
      }}</ElButton>
      <ElButton v-if="status?.enabled" type="danger" plain :loading="busy" @click="revoke">{{
        t('mumble.credential.revoke')
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
      ElMessage.error(t('mumble.credential.loadFailed'))
    } finally {
      busy.value = false
    }
  }

  async function issue(rotate: boolean) {
    if (rotate) {
      try {
        await ElMessageBox.confirm(
          t('mumble.credential.rotateConfirm'),
          t('mumble.credential.title'),
          {
            type: 'warning'
          }
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
      ElMessage.success(t('mumble.credential.issued'))
    } catch (error) {
      if (error !== 'cancel' && error !== 'close')
        ElMessage.error(t('mumble.credential.operationFailed'))
    } finally {
      busy.value = false
    }
  }

  async function revoke() {
    try {
      await ElMessageBox.confirm(
        t('mumble.credential.revokeConfirm'),
        t('mumble.credential.title'),
        {
          type: 'warning'
        }
      )
      busy.value = true
      await revokeMumbleCredential()
      if (status.value) status.value.enabled = false
      secret.value = ''
      ElMessage.success(t('mumble.credential.revoked'))
    } catch (error) {
      if (error !== 'cancel' && error !== 'close')
        ElMessage.error(t('mumble.credential.operationFailed'))
    } finally {
      busy.value = false
    }
  }

  async function copySecret() {
    try {
      await navigator.clipboard.writeText(secret.value)
      ElMessage.success(t('mumble.credential.copied'))
    } catch {
      ElMessage.error(t('common.copyFailed'))
    }
  }

  onMounted(load)
</script>
