<template>
  <div class="newbro-settings-page">
    <ElCard shadow="never" class="mb-4">
      <div class="page-header">
        <div class="page-title">{{ t('system.newbroSettings.title') }}</div>
        <div class="page-subtitle">{{ t('system.newbroSettings.subtitle') }}</div>
      </div>
    </ElCard>

    <ElForm
      ref="formRef"
      :model="form"
      :rules="rules"
      label-width="200px"
      v-loading="loading"
      class="settings-form"
    >
      <ElCard shadow="never" class="mt-4">
        <template #header>
          <div class="section-title">{{ t('system.newbroSettings.rewardTitle') }}</div>
          <div class="section-hint">{{ t('system.newbroSettings.rewardHint') }}</div>
        </template>

        <ElFormItem :label="t('system.newbroSettings.bonusRate')" prop="bonus_rate">
          <ElInputNumber
            v-model="form.bonus_rate"
            :min="0"
            :step="1"
            :precision="2"
            :controls="false"
            class="number-input"
            :placeholder="t('system.newbroSettings.bonusRatePlaceholder')"
          />
          <span class="input-suffix">%</span>
        </ElFormItem>
      </ElCard>

      <ElCard shadow="never">
        <template #header>
          <div class="section-title">{{ t('system.newbroSettings.eligibilityTitle') }}</div>
          <div class="section-hint">{{ t('system.newbroSettings.eligibilityHint') }}</div>
        </template>

        <ElFormItem :label="t('system.newbroSettings.maxCharacterSP')" prop="max_character_sp">
          <ElInputNumber
            v-model="form.max_character_sp"
            :min="1"
            :step="1000000"
            :controls="false"
            class="number-input"
            :placeholder="t('system.newbroSettings.maxCharacterSPPlaceholder')"
          />
        </ElFormItem>

        <ElFormItem :label="t('system.newbroSettings.multiCharacterSP')" prop="multi_character_sp">
          <ElInputNumber
            v-model="form.multi_character_sp"
            :min="1"
            :step="1000000"
            :controls="false"
            class="number-input"
            :placeholder="t('system.newbroSettings.multiCharacterSPPlaceholder')"
          />
        </ElFormItem>

        <ElFormItem
          :label="t('system.newbroSettings.multiCharacterThreshold')"
          prop="multi_character_threshold"
        >
          <ElInputNumber
            v-model="form.multi_character_threshold"
            :min="1"
            :controls="false"
            class="number-input"
            :placeholder="t('system.newbroSettings.multiCharacterThresholdPlaceholder')"
          />
        </ElFormItem>
      </ElCard>

      <ElCard shadow="never" class="mt-4">
        <template #header>
          <div class="section-title">{{ t('system.newbroSettings.refreshTitle') }}</div>
          <div class="section-hint">{{ t('system.newbroSettings.refreshHint') }}</div>
        </template>

        <ElFormItem
          :label="t('system.newbroSettings.refreshIntervalDays')"
          prop="refresh_interval_days"
        >
          <ElInputNumber
            v-model="form.refresh_interval_days"
            :min="1"
            :controls="false"
            class="number-input"
            :placeholder="t('system.newbroSettings.refreshIntervalDaysPlaceholder')"
          />
        </ElFormItem>
      </ElCard>

      <ElCard shadow="never" class="mt-4">
        <ElButton type="primary" :disabled="saving" @click="handleSave">
          {{ t('system.newbroSettings.save') }}
        </ElButton>
      </ElCard>
    </ElForm>
  </div>
</template>

<script setup lang="ts">
  import { fetchAdminNewbroSettings, updateAdminNewbroSettings } from '@/api/newbro'
  import {
    ElButton,
    ElCard,
    ElForm,
    ElFormItem,
    ElInputNumber,
    ElMessage,
    type FormInstance,
    type FormRules
  } from 'element-plus'
  import { useI18n } from 'vue-i18n'

  defineOptions({ name: 'NewbroSettings' })

  const { t } = useI18n()

  const formRef = ref<FormInstance>()
  const loading = ref(false)
  const saving = ref(false)

  const form = reactive<Api.Newbro.Settings>({
    max_character_sp: 20000000,
    multi_character_sp: 10000000,
    multi_character_threshold: 3,
    refresh_interval_days: 7,
    bonus_rate: 20
  })

  const rules: FormRules = {
    max_character_sp: [
      {
        required: true,
        type: 'number',
        min: 1,
        message: t('system.newbroSettings.maxCharacterSPPlaceholder'),
        trigger: 'blur'
      }
    ],
    multi_character_sp: [
      {
        required: true,
        type: 'number',
        min: 1,
        message: t('system.newbroSettings.multiCharacterSPPlaceholder'),
        trigger: 'blur'
      }
    ],
    multi_character_threshold: [
      {
        required: true,
        type: 'number',
        min: 1,
        message: t('system.newbroSettings.multiCharacterThresholdPlaceholder'),
        trigger: 'blur'
      }
    ],
    refresh_interval_days: [
      {
        required: true,
        type: 'number',
        min: 1,
        message: t('system.newbroSettings.refreshIntervalDaysPlaceholder'),
        trigger: 'blur'
      }
    ],
    bonus_rate: [
      {
        required: true,
        type: 'number',
        min: 0,
        message: t('system.newbroSettings.bonusRatePlaceholder'),
        trigger: 'blur'
      }
    ]
  }

  async function loadSettings() {
    loading.value = true
    try {
      const data = await fetchAdminNewbroSettings()
      form.max_character_sp = data.max_character_sp
      form.multi_character_sp = data.multi_character_sp
      form.multi_character_threshold = data.multi_character_threshold
      form.refresh_interval_days = data.refresh_interval_days
      form.bonus_rate = data.bonus_rate
    } finally {
      loading.value = false
    }
  }

  async function handleSave() {
    await formRef.value?.validate()

    saving.value = true
    try {
      const data = await updateAdminNewbroSettings({
        max_character_sp: form.max_character_sp,
        multi_character_sp: form.multi_character_sp,
        multi_character_threshold: form.multi_character_threshold,
        refresh_interval_days: form.refresh_interval_days,
        bonus_rate: form.bonus_rate
      })
      form.max_character_sp = data.max_character_sp
      form.multi_character_sp = data.multi_character_sp
      form.multi_character_threshold = data.multi_character_threshold
      form.refresh_interval_days = data.refresh_interval_days
      form.bonus_rate = data.bonus_rate
      ElMessage.success(t('system.newbroSettings.saveSuccess'))
    } finally {
      saving.value = false
    }
  }

  onMounted(() => {
    loadSettings()
  })
</script>

<style scoped>
  .page-header {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .page-title {
    font-size: 18px;
    font-weight: 600;
  }

  .page-subtitle {
    font-size: 13px;
    color: var(--el-text-color-secondary);
  }

  .section-title {
    font-size: 15px;
    font-weight: 600;
  }

  .section-hint {
    margin-top: 4px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  .number-input {
    width: 280px;
  }

  .input-suffix {
    margin-left: 8px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }
</style>
