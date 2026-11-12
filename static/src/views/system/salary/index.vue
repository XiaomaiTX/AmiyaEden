<template>
  <ElCard shadow="never">
    <template #header>
      <div class="salary-header">
        <h2>{{ t('salaryManagement.title') }}</h2>
        <ElButton :disabled="loading || saving" @click="load">{{
          t('salaryManagement.refresh')
        }}</ElButton>
      </div>
    </template>
    <ElAlert
      :title="t('salaryManagement.description')"
      type="info"
      :closable="false"
      class="mb-4"
    />
    <p v-if="loading" role="status">{{ t('salaryManagement.loading') }}</p>
    <ElAlert v-if="error" :title="error" type="error" :closable="false" class="mb-4" />
    <ElForm v-if="form" label-position="top" :disabled="loading || saving" @submit.prevent="save">
      <h3 class="mb-4">{{ t('salaryManagement.operations') }}</h3>
      <div class="salary-grid">
        <ElFormItem v-for="key in operationKeys" :key="key" :label="t(`salaryManagement.${key}`)">
          <div class="salary-field">
            <ElInputNumber
              v-model="form[key]"
              :min="0"
              :max="1000000000000"
              :precision="2"
              :controls="false"
            />
            <span>{{
              t(
                key === 'srp_payout' ? 'salaryManagement.perOperationUnit' : 'salaryManagement.unit'
              )
            }}</span>
          </div>
        </ElFormItem>
      </div>
      <ElDivider />
      <h3 class="mb-4">{{ t('salaryManagement.fcSection') }}</h3>
      <p class="mb-4 text-sm text-secondary">{{ t('salaryManagement.fcHint') }}</p>
      <div class="salary-grid">
        <ElFormItem :label="t('salaryManagement.fc_salary')">
          <ElInputNumber
            v-model="form.fc_salary"
            :min="0"
            :max="1000000000000"
            :precision="2"
            :controls="false"
          />
        </ElFormItem>
        <ElFormItem :label="t('salaryManagement.fc_salary_monthly_limit')">
          <ElInputNumber
            v-model="form.fc_salary_monthly_limit"
            :min="0"
            :precision="0"
            :controls="false"
          />
        </ElFormItem>
      </div>
      <ElButton type="primary" native-type="submit" :loading="saving">{{
        t('salaryManagement.save')
      }}</ElButton>
    </ElForm>
  </ElCard>
</template>
<script setup lang="ts">
  import { ref, onMounted } from 'vue'
  import { useI18n } from 'vue-i18n'
  import { ElMessage } from 'element-plus'
  import { fetchSalarySettings, updateSalarySettings } from '@/api/salary'
  import type { SalarySettings } from '@/types/api/salary'
  const { t } = useI18n()
  const operationKeys = ['recruitment', 'welfare_delivery', 'shop_delivery', 'srp_payout'] as const
  const form = ref<SalarySettings | null>(null)
  const loading = ref(false)
  const saving = ref(false)
  const error = ref('')
  async function load() {
    loading.value = true
    error.value = ''
    form.value = null
    try {
      form.value = await fetchSalarySettings()
    } catch {
      error.value = t('salaryManagement.loadFailed')
    } finally {
      loading.value = false
    }
  }
  async function save() {
    if (!form.value || loading.value || saving.value) return
    saving.value = true
    error.value = ''
    try {
      form.value = await updateSalarySettings(form.value)
      ElMessage.success(t('salaryManagement.saved'))
    } catch {
      error.value = t('salaryManagement.saveFailed')
    } finally {
      saving.value = false
    }
  }
  onMounted(load)
</script>
<style scoped>
  .salary-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
  }
  .salary-header h2 {
    margin: 0;
    font-size: 18px;
  }
  .salary-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(min(100%, 280px), 1fr));
    gap: 0 32px;
    max-width: 900px;
  }
  .salary-field {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    align-items: center;
  }
  .salary-field span {
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }
</style>
