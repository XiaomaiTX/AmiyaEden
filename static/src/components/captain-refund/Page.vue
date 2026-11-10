<template>
  <div class="space-y-4">
    <h1>{{ t(admin ? 'refund.manage' : 'refund.title') }}</h1>
    <ElAlert v-if="error" :title="error" type="error" :closable="false" />
    <ElAlert v-if="success" :title="success" type="success" :closable="false" />
    <ElCard v-if="!admin" v-loading="busy">
      <p>{{ t('refund.hint') }}</p>
      <ElForm label-position="top">
        <ElFormItem :label="t('refund.captain')">
          <ElSelect v-model="form.captain_character_id">
            <ElOption
              v-for="character in options?.characters"
              :key="character.character_id"
              :value="character.character_id"
              :label="character.character_name"
            />
          </ElSelect>
        </ElFormItem>
        <ElFormItem :label="t('refund.system')">
          <ElSelect v-model="form.system_id">
            <ElOption
              v-for="systemID in options?.settings.allowed_system_ids"
              :key="systemID"
              :value="systemID"
              :label="String(systemID)"
            />
          </ElSelect>
        </ElFormItem>
        <ElAlert
          v-if="options?.settings.allowed_system_ids.length === 0"
          :title="t('refund.noSystems')"
          type="warning"
          :closable="false"
        />
        <ElFormItem :label="t('refund.members')">
          <ElInput v-model="names" type="textarea" :rows="5" />
        </ElFormItem>
        <ElFormItem :label="t('refund.start')">
          <ElInput v-model="start" type="datetime-local" step="1" />
        </ElFormItem>
        <ElFormItem :label="t('refund.end')">
          <ElInput v-model="end" type="datetime-local" step="1" />
        </ElFormItem>
      </ElForm>
      <ElButton
        :disabled="busy || !form.captain_character_id || !form.system_id || !start || !end"
        @click="run(calculatePreview)"
      >
        {{ t('refund.preview') }}
      </ElButton>
      <template v-if="preview">
        <Calculation :value="preview" />
        <ElButton type="primary" :disabled="busy || !preview.can_submit" @click="run(submit)">
          {{ t('refund.submit') }}
        </ElButton>
      </template>
    </ElCard>

    <ElCard class="art-table-card" shadow="never">
      <template #header>
        <div class="flex items-center justify-between gap-3">
          <span>{{ t('refund.history') }}</span>
          <ArtTableHeader
            v-model:columns="columnChecks"
            :loading="loading"
            @refresh="refreshData"
          />
        </div>
      </template>
      <ArtTable
        :loading="loading"
        :data="rows"
        :columns="columns"
        :pagination="pagination"
        visual-variant="ledger"
        @pagination:size-change="handleSizeChange"
        @pagination:current-change="handleCurrentChange"
      />
      <ElEmpty v-if="!loading && rows.length === 0" :description="t('common.empty')" />
    </ElCard>

    <ElCard v-if="selected" v-loading="busy">
      <h2>#{{ selected.id }} · {{ t('refund.' + selected.status) }}</h2>
      <Calculation :value="selected.calculation" />
      <p>
        {{ t('refund.reviewed') }}: {{ selected.reviewed_by || '—' }} ·
        {{ formatTime(selected.reviewed_at) }}
      </p>
      <p v-if="selected.reject_reason">
        {{ t('refund.rejectReason') }}: {{ selected.reject_reason }}
      </p>
      <template v-if="admin && selected.status === 'pending'">
        <ElFormItem :label="t('refund.rejectReason')">
          <ElInput v-model="reason" type="textarea" />
        </ElFormItem>
        <ElButton type="primary" :disabled="busy" @click="run(() => review(true))">
          {{ t('refund.approve') }}
        </ElButton>
        <ElButton
          type="danger"
          :disabled="busy || !reason.trim()"
          @click="run(() => review(false))"
        >
          {{ t('refund.reject') }}
        </ElButton>
      </template>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
  import { h, onMounted, reactive, ref, watch } from 'vue'
  import { ElButton, ElTag } from 'element-plus'
  import { useI18n } from 'vue-i18n'
  import {
    refundOptions,
    previewRefund,
    createRefund,
    listRefunds,
    getRefund,
    reviewRefund
  } from '@/api/captain-refund'
  import type {
    RefundOptions,
    RefundApplication,
    RefundCalculation,
    RefundRequest
  } from '@/types/api/captain-refund'
  import { useTable } from '@/hooks/core/useTable'
  import { formatTime } from '@utils/common'
  import Calculation from './Calculation.vue'

  const props = withDefaults(defineProps<{ admin?: boolean }>(), { admin: false })
  const { t } = useI18n()
  const options = ref<RefundOptions | null>(null)
  const selected = ref<RefundApplication | null>(null)
  const preview = ref<RefundCalculation | null>(null)
  const busy = ref(false)
  const error = ref('')
  const success = ref('')
  const names = ref('')
  const start = ref('')
  const end = ref('')
  const reason = ref('')
  const form = reactive({ captain_character_id: 0, system_id: 0 })

  const {
    columns,
    columnChecks,
    data: rows,
    loading,
    pagination,
    getData: load,
    refreshData,
    handleSizeChange,
    handleCurrentChange
  } = useTable({
    core: {
      apiFn: (params: Api.Common.CommonSearchParams) => listRefunds(props.admin, params),
      apiParams: { current: 1, size: 200 },
      columnsFactory: () => [
        { prop: 'id', label: t('refund.application'), width: 120 },
        {
          prop: 'captain',
          label: t('refund.captain'),
          minWidth: 180,
          formatter: (row: RefundApplication) =>
            row.calculation.members.find((member) => member.is_captain)?.character_name || '—'
        },
        {
          prop: 'status',
          label: t('common.status'),
          width: 130,
          formatter: (row: RefundApplication) =>
            h(ElTag, { type: refundStatusTag(row.status) }, () => t('refund.' + row.status))
        },
        {
          prop: 'reward',
          label: t('refund.reward'),
          width: 160,
          formatter: (row: RefundApplication) => `${row.reward.toFixed(2)} ${t('refund.coin')}`
        },
        {
          prop: 'created_at',
          label: t('refund.submittedAt'),
          minWidth: 180,
          formatter: (row: RefundApplication) => formatTime(row.created_at)
        },
        {
          prop: 'operation',
          label: t('common.operation'),
          width: 120,
          fixed: 'right',
          formatter: (row: RefundApplication) =>
            h(ElButton, { link: true, type: 'primary', onClick: () => openDetail(row) }, () =>
              t('refund.details')
            )
        }
      ]
    }
  })

  watch([form, names, start, end], () => {
    preview.value = null
  })

  function refundStatusTag(status: RefundApplication['status']) {
    if (status === 'approved') return 'success'
    if (status === 'pending') return 'warning'
    return 'info'
  }

  async function run(fn: () => Promise<void>) {
    if (busy.value) return
    busy.value = true
    error.value = ''
    success.value = ''
    try {
      await fn()
    } catch {
      error.value = t('refund.failed')
    } finally {
      busy.value = false
    }
  }

  function request(): RefundRequest {
    return {
      ...form,
      member_names: names.value
        .split(/[,;\n]+/)
        .map((name) => name.trim())
        .filter(Boolean),
      start: new Date(start.value + 'Z').toISOString(),
      end: new Date(end.value + 'Z').toISOString()
    }
  }

  async function calculatePreview() {
    preview.value = await previewRefund(request())
  }

  function openDetail(row: RefundApplication) {
    void run(async () => {
      selected.value = await getRefund(props.admin, row.id)
      reason.value = ''
    })
  }

  async function submit() {
    const payload = request()
    try {
      selected.value = await createRefund(payload)
      preview.value = null
      success.value = t('refund.submitted')
      await load()
    } catch {
      try {
        preview.value = await previewRefund(payload)
        error.value = t(preview.value.conflicts.length > 0 ? 'refund.conflict' : 'refund.failed')
      } catch {
        error.value = t('refund.failed')
      }
    }
  }

  async function review(approve: boolean) {
    if (!selected.value) return
    await reviewRefund(selected.value.id, approve, reason.value)
    selected.value = await getRefund(true, selected.value.id)
    await load()
  }

  onMounted(() => {
    void run(async () => {
      if (!props.admin) {
        options.value = await refundOptions()
        form.captain_character_id = options.value.characters[0]?.character_id ?? 0
        form.system_id = options.value.settings.allowed_system_ids[0] ?? 0
      }
    })
  })
</script>
