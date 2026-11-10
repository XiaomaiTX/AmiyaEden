<template>
  <div class="newbro-admin-settings-panel">
    <ElCard shadow="never" class="mb-4">
      <div class="page-header">
        <div class="page-title">{{ t(titleKey) }}</div>
        <div class="page-subtitle">{{ t(subtitleKey) }}</div>
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
          <div class="section-title">{{ t('system.newbroSettings.recruitTitle') }}</div>
          <div class="section-hint">{{ t('system.newbroSettings.recruitHint') }}</div>
        </template>

        <ElFormItem :label="t('system.newbroSettings.recruitQQURL')" prop="recruit_qq_url">
          <ElInput
            v-model="form.recruit_qq_url"
            class="url-input"
            :placeholder="t('system.newbroSettings.recruitQQURLPlaceholder')"
          />
        </ElFormItem>

        <ElFormItem
          :label="t('system.newbroSettings.recruitRewardAmount')"
          prop="recruit_reward_amount"
        >
          <ElInputNumber
            v-model="form.recruit_reward_amount"
            :min="0"
            :step="1"
            :precision="2"
            :controls="false"
            class="number-input"
            :placeholder="t('system.newbroSettings.recruitRewardAmountPlaceholder')"
          />
          <span class="input-suffix">{{ t('system.newbroSettings.fuxiCoinUnit') }}</span>
        </ElFormItem>

        <ElFormItem
          :label="t('system.newbroSettings.recruitCooldownDays')"
          prop="recruit_cooldown_days"
        >
          <ElInputNumber
            v-model="form.recruit_cooldown_days"
            :min="1"
            :controls="false"
            class="number-input"
            :placeholder="t('system.newbroSettings.recruitCooldownDaysPlaceholder')"
          />
          <span class="input-suffix">{{ t('system.newbroSettings.daysUnit') }}</span>
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
  import { fetchAdminNewbroRecruitSettings, updateAdminNewbroRecruitSettings } from '@/api/newbro'
  import {
    ElButton,
    ElCard,
    ElForm,
    ElFormItem,
    ElInput,
    ElInputNumber,
    ElMessage,
    type FormInstance,
    type FormRules
  } from 'element-plus'
  import { useI18n } from 'vue-i18n'

  defineOptions({ name: 'NewbroAdminSettingsPanel' })

  const { t } = useI18n()

  const formRef = ref<FormInstance>()
  const loading = ref(false)
  const saving = ref(false)
  const titleKey = 'newbro.recruitLink.settingsTab'
  const subtitleKey = 'newbro.recruitLink.settingsSubtitle'
  const form = reactive<Api.Newbro.RecruitSettings>({
    recruit_qq_url: '',
    recruit_reward_amount: 50,
    recruit_cooldown_days: 90
  })

  const rules: FormRules = {
    recruit_qq_url: [
      {
        required: false,
        type: 'string',
        trigger: 'blur'
      }
    ],
    recruit_reward_amount: [
      {
        required: true,
        type: 'number',
        min: 0,
        message: t('system.newbroSettings.validation.mustBeZeroOrGreater'),
        trigger: 'blur'
      }
    ],
    recruit_cooldown_days: [
      {
        required: true,
        type: 'number',
        min: 1,
        message: t('system.newbroSettings.validation.mustBeGreaterThanZero'),
        trigger: 'blur'
      }
    ]
  }

  const applyRecruitSettings = (data: Api.Newbro.RecruitSettings) => {
    form.recruit_qq_url = data.recruit_qq_url
    form.recruit_reward_amount = data.recruit_reward_amount
    form.recruit_cooldown_days = data.recruit_cooldown_days
  }

  async function loadSettings() {
    loading.value = true
    try {
      const data = await fetchAdminNewbroRecruitSettings()
      applyRecruitSettings(data)
    } finally {
      loading.value = false
    }
  }

  async function handleSave() {
    await formRef.value?.validate()

    saving.value = true
    try {
      const data = await updateAdminNewbroRecruitSettings({
        recruit_qq_url: form.recruit_qq_url,
        recruit_reward_amount: form.recruit_reward_amount,
        recruit_cooldown_days: form.recruit_cooldown_days
      })
      applyRecruitSettings(data)
      ElMessage.success(t('system.newbroSettings.saveSuccess'))
    } finally {
      saving.value = false
    }
  }

  defineExpose({
    reloadSettings: loadSettings
  })

  onMounted(() => {
    void loadSettings()
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

  .url-input {
    width: 480px;
    max-width: 100%;
  }

  .input-suffix {
    margin-left: 8px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }
</style>
