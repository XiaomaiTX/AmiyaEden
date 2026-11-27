<template>
  <ElCard shadow="never">
    <template #header>
      <h2 class="font-semibold">{{ t('qqOfficial.credentials') }}</h2>
    </template>
    <ElForm
      :model="form"
      label-position="top"
      autocomplete="off"
      @submit.prevent="reviewCredentials"
    >
      <div class="grid gap-x-4 sm:grid-cols-2">
        <ElFormItem :label="t('qqOfficial.appID')">
          <ElInput v-model="form.appID" inputmode="numeric" :disabled="busy || !!confirmation" />
        </ElFormItem>
        <ElFormItem :label="t('qqOfficial.appSecret')">
          <ElInput
            v-model="form.secret"
            type="password"
            autocomplete="new-password"
            :disabled="busy || !!confirmation"
          />
        </ElFormItem>
      </div>
      <p
        :class="draft.error ? 'text-danger' : 'text-g-600'"
        class="mb-3 text-sm"
        aria-live="polite"
      >
        {{ t(draft.error ?? 'qqOfficial.appSecretHint') }}
      </p>
      <p class="mb-3 text-sm text-g-600">{{ t('qqOfficial.credentialsHint') }}</p>
      <ElButton type="primary" native-type="submit" :disabled="busy || !draft.patch">
        {{ t('qqOfficial.saveCredentials') }}
      </ElButton>
    </ElForm>
    <div class="mt-4 space-y-1 border-t pt-3">
      <ElSwitch
        :model-value="status.approval_writes_enabled"
        :disabled="busy || (!status.approval_writes_enabled && !hasUsableQQCredentials(status))"
        :aria-label="t('qqOfficial.approvalWritesControl')"
        :aria-describedby="approvalHintID"
        :active-text="t('qqOfficial.approvalWritesControl')"
        @update:model-value="changeApprovalWrites($event === true)"
      />
      <p :id="approvalHintID" class="text-sm text-g-600">{{
        t('qqOfficial.approvalWritesHint')
      }}</p>
    </div>
    <ElDialog
      :model-value="!!confirmation"
      :title="
        t(
          confirmation === 'credentials'
            ? 'qqOfficial.saveCredentials'
            : 'qqOfficial.enableApprovalWrites'
        )
      "
      width="min(32rem, 92vw)"
      @update:model-value="(open) => !open && closeConfirmation()"
    >
      <p>
        {{
          t(
            confirmation === 'credentials'
              ? 'qqOfficial.credentialsConfirm'
              : 'qqOfficial.approvalWritesConfirm'
          )
        }}
      </p>
      <template #footer>
        <ElButton @click="closeConfirmation">{{ t('common.cancel') }}</ElButton>
        <ElButton type="danger" :disabled="busy" @click="confirm">{{
          t('common.confirm')
        }}</ElButton>
      </template>
    </ElDialog>
  </ElCard>
</template>

<script setup lang="ts">
  import { computed, reactive, ref, useId } from 'vue'
  import { useI18n } from 'vue-i18n'
  import type { QQOfficialSettings, QQOfficialStatus } from '@/api/generated/client'
  import { hasUsableQQCredentials, qqCredentialDraft } from '@/hooks/qq-official-helpers'

  const props = defineProps<{ status: QQOfficialStatus; busy: boolean }>()
  const emit = defineEmits<{ save: [patch: QQOfficialSettings] }>()
  const { t } = useI18n()
  const approvalHintID = useId()
  const form = reactive({ appID: props.status.app_id ?? '', secret: '' })
  const draft = computed(() => qqCredentialDraft(props.status.app_id, form.appID, form.secret))
  const confirmation = ref<'credentials' | 'approval'>()

  const reviewCredentials = () => {
    if (draft.value.patch && !props.busy) confirmation.value = 'credentials'
  }
  const changeApprovalWrites = (enabled: boolean) => {
    if (!enabled) emit('save', { approval_writes_enabled: false })
    else if (hasUsableQQCredentials(props.status)) confirmation.value = 'approval'
  }
  const closeConfirmation = () => {
    if (confirmation.value === 'credentials') form.secret = ''
    confirmation.value = undefined
  }
  const confirm = () => {
    const patch =
      confirmation.value === 'credentials'
        ? draft.value.patch
        : confirmation.value === 'approval' && hasUsableQQCredentials(props.status)
          ? { approval_writes_enabled: true }
          : undefined
    closeConfirmation()
    if (patch) emit('save', patch)
  }
</script>
