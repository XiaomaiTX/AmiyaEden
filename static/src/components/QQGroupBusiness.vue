<template>
  <div class="group-business-page">
    <ElCard shadow="never">
      <template #header>
        <h2 class="section-title">{{ t('qqbot.groupBusiness') }}</h2>
      </template>
      <p class="section-description">{{ t('qqbot.groupPolicy') }}</p>
      <ElForm label-position="top" class="rule-form">
        <ElFormItem :label="t('qqbot.notificationGroup')">
          <ElInputNumber v-model="rule.group_id" :min="1" />
        </ElFormItem>
        <ElFormItem :label="t('qqbot.corporations')">
          <ElInput v-model="corps" />
        </ElFormItem>
        <ElFormItem :label="t('qqbot.roles')">
          <ElInput v-model="roles" />
        </ElFormItem>
        <ElFormItem v-for="key in switches" :key="key" :label="t(`qqbot.${key}`)">
          <ElSwitch v-model="rule[key]" />
        </ElFormItem>
        <ElFormItem :label="t('qqbot.cardTemplate')">
          <ElInput v-model="rule.card_template" />
          <div class="form-hint">{{ t('qqbot.cardVariables') }}</div>
        </ElFormItem>
        <ElFormItem :label="t('qqbot.confirmations')">
          <ElInputNumber v-model="rule.mismatch_confirmations" :min="1" :max="100" />
        </ElFormItem>
        <ElFormItem :label="t('qqbot.observation')">
          <ElInputNumber v-model="rule.observation_hours" :min="0" :max="8760" />
        </ElFormItem>
        <div class="form-actions">
          <ElButton type="primary" :disabled="busy" @click="run(true)">
            {{ t('qqbot.save') }}
          </ElButton>
          <ElButton :disabled="busy" @click="run(false)">{{ t('qqbot.refresh') }}</ElButton>
        </div>
      </ElForm>
      <p v-if="message" class="operation-message" role="status">{{ message }}</p>
    </ElCard>

    <ElCard shadow="never">
      <template #header>
        <h3 class="section-title">{{ t('qqbot.rules') }}</h3>
      </template>
      <ArtTable visual-variant="ledger" :data="audit?.rules ?? []">
        <ElTableColumn prop="group_id" :label="t('qqbot.notificationGroup')" />
        <ElTableColumn prop="version" :label="t('qqbot.version')" />
        <ElTableColumn :label="t('qqbot.detail')">
          <template #default="{ row }">
            <ElButton @click="edit(row)">{{ t('qqbot.editRule') }}</ElButton>
          </template>
        </ElTableColumn>
      </ArtTable>
    </ElCard>

    <ElCard v-for="table in tables" :key="table.key" shadow="never">
      <template #header>
        <h3 class="section-title">{{ t(`qqbot.${table.key}`) }}</h3>
      </template>
      <ArtTable visual-variant="ledger" :data="audit?.[table.key] ?? []">
        <ElTableColumn
          v-for="field in table.fields"
          :key="field"
          :prop="field"
          :label="t(`qqbot.field_${field}`)"
        />
      </ArtTable>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
  import { ref, onMounted } from 'vue'
  import { useI18n } from 'vue-i18n'
  import { fetchQQGroupBusiness, saveQQGroupRule } from '@/api/qqbot'
  import type { QQGroupRule, QQGroupBusinessAudit } from '@/types/api/qqbot'

  const { t } = useI18n()
  const rule = ref<QQGroupRule>({
    group_id: 0,
    version: 0,
    admission_enabled: false,
    auto_reject: false,
    corporation_ids: [],
    roles: [],
    card_enabled: false,
    card_template: '{character_name}',
    inspection_enabled: false,
    auto_kick: false,
    mismatch_confirmations: 3,
    observation_hours: 24,
    last_inspection_at: '',
    updated_at: ''
  })
  const audit = ref<QQGroupBusinessAudit | null>(null)
  const corps = ref('')
  const roles = ref('')
  const busy = ref(false)
  const message = ref('')
  const switches = [
    'admission_enabled',
    'auto_reject',
    'card_enabled',
    'inspection_enabled',
    'auto_kick'
  ] as const
  const tables = [
    { key: 'applications', fields: ['group_id', 'qq', 'status', 'evidence', 'operation_id'] },
    {
      key: 'members',
      fields: [
        'group_id',
        'qq',
        'present',
        'confirmations',
        'evidence',
        'card_target',
        'card_status',
        'card_operation_id',
        'kick_status',
        'kick_operation_id'
      ]
    },
    { key: 'inspections', fields: ['id', 'group_id', 'rule_version', 'status', 'operation_id'] },
    { key: 'judgements', fields: ['run_id', 'qq', 'done', 'decision', 'evidence', 'operation_id'] }
  ] as const

  function edit(value: QQGroupRule) {
    rule.value = { ...value }
    corps.value = (value.corporation_ids ?? []).join(',')
    roles.value = (value.roles ?? []).join(',')
  }

  async function run(save: boolean) {
    busy.value = true
    try {
      if (save)
        await saveQQGroupRule({
          ...rule.value,
          corporation_ids: corps.value.trim() ? corps.value.split(',').map(Number) : [],
          roles: roles.value
            .split(',')
            .map((value) => value.trim())
            .filter(Boolean)
        })
      audit.value = await fetchQQGroupBusiness()
      message.value = t('qqbot.success')
    } catch {
      message.value = t('qqbot.error')
    } finally {
      busy.value = false
    }
  }

  onMounted(() => run(false))
</script>

<style scoped>
  .group-business-page {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .section-title {
    margin: 0;
    font-size: 16px;
    font-weight: 600;
  }

  .section-description,
  .operation-message,
  .form-hint {
    color: var(--el-text-color-secondary);
  }

  .section-description {
    margin: 0 0 16px;
  }

  .rule-form {
    max-width: 680px;
  }

  .form-hint {
    margin-top: 6px;
    font-size: 12px;
    line-height: 1.6;
  }

  .form-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }

  .operation-message {
    margin: 12px 0 0;
  }
</style>
