<template>
  <ElCard v-loading="loading || busy">
    <h2>{{ t('refund.settingsTitle') }}</h2>
    <p>{{ t('refund.settingsHint') }}</p>
    <ElAlert v-if="message" :title="message" :type="messageType" :closable="false" class="mb-4" />
    <ElForm v-if="cfg" label-position="top">
      <ElFormItem v-for="key in keys" :key="key" :label="t('refund.' + key)">
        <ElInputNumber
          v-model="cfg[key]"
          :min="0"
          :max="key === 'tax_rate' ? 100 : key === 'max_age_days' ? 36500 : undefined"
          :step="key === 'tax_rate' ? 0.01 : 1"
          :precision="key === 'tax_rate' ? 2 : 0"
        />
      </ElFormItem>
      <ElFormItem :label="t('refund.allowedSystems')">
        <ElInput v-model="systems" type="textarea" :rows="4" />
      </ElFormItem>
      <ElButton type="primary" :disabled="loading || busy" @click="save">
        {{ t('common.save') }}
      </ElButton>
    </ElForm>
  </ElCard>
</template>

<script setup lang="ts">
  import { onMounted, ref } from 'vue'
  import { useI18n } from 'vue-i18n'
  import { getRefundSettings, saveRefundSettings } from '@/api/captain-refund'
  import type { RefundSettings } from '@/types/api/captain-refund'

  const { t } = useI18n()
  const cfg = ref<RefundSettings | null>(null)
  const systems = ref('')
  const loading = ref(true)
  const busy = ref(false)
  const message = ref('')
  const messageType = ref<'success' | 'error'>('error')
  const keys = ['max_total_sp', 'max_age_days', 'tax_rate'] as const

  onMounted(async () => {
    try {
      cfg.value = await getRefundSettings()
      systems.value = cfg.value.allowed_system_ids.join('\n')
    } catch {
      messageType.value = 'error'
      message.value = t('refund.failed')
    } finally {
      loading.value = false
    }
  })

  async function save() {
    if (!cfg.value || loading.value || busy.value) return
    message.value = ''
    const settings = cfg.value
    const entries = systems.value.split(/[\s,;]+/).filter(Boolean)
    const ids = entries.map(Number)
    if (
      !Number.isInteger(settings.max_total_sp) ||
      settings.max_total_sp < 0 ||
      !Number.isInteger(settings.max_age_days) ||
      settings.max_age_days < 0 ||
      settings.max_age_days > 36500 ||
      !Number.isFinite(settings.tax_rate) ||
      settings.tax_rate < 0 ||
      settings.tax_rate > 100 ||
      entries.some((entry) => !/^\d+$/.test(entry)) ||
      ids.some((id) => !Number.isSafeInteger(id) || id <= 0)
    ) {
      messageType.value = 'error'
      message.value = t('refund.settingsInvalid')
      return
    }

    busy.value = true
    try {
      const allowedSystemIDs = [...new Set(ids)]
      cfg.value = await saveRefundSettings({ ...settings, allowed_system_ids: allowedSystemIDs })
      systems.value = cfg.value.allowed_system_ids.join('\n')
      messageType.value = 'success'
      message.value = t('refund.saved')
    } catch {
      messageType.value = 'error'
      message.value = t('refund.failed')
    } finally {
      busy.value = false
    }
  }
</script>
