<template>
  <div>
    <p
      >{{ t('refund.bounty') }}: {{ value.bounty.toFixed(2) }} {{ t('refund.isk') }} ·
      {{ t('refund.taxRate') }}: {{ value.settings.tax_rate }}% · {{ t('refund.reward') }}:
      {{ value.reward.toFixed(2) }} {{ t('refund.coin') }}</p
    >
    <p
      >{{ t('refund.system') }}: {{ value.request.system_id }} · {{ t('refund.period') }}:
      {{ formatTime(value.request.start) }} — {{ formatTime(value.request.end) }}</p
    >
    <p>{{ t('refund.ignored') }}: {{ value.ignored_names.join(', ') || '—' }}</p>
    <ElAlert
      v-if="!value.has_newbro"
      :title="t('refund.noNewbro')"
      type="warning"
      :closable="false"
    />
    <ElTable :data="value.members">
      <ElTableColumn prop="character_name" :label="t('refund.character')" />
      <ElTableColumn :label="t('refund.newbro')"
        ><template #default="{ row }"
          >{{ t('refund.' + (row.eligibility.is_currently_newbro ? 'yes' : 'no'))
          }}<span v-if="row.eligibility.disqualified_reason">
            · {{ t('refund.reason.' + row.eligibility.disqualified_reason) }}</span
          ></template
        ></ElTableColumn
      >
      <ElTableColumn prop="eligibility.total_sp" :label="t('refund.sp')">
        <template #default="{ row }">{{
          Number(row.eligibility.total_sp).toLocaleString()
        }}</template>
      </ElTableColumn>
      <ElTableColumn prop="eligibility.oldest_birthday" :label="t('refund.birthday')" />
      <ElTableColumn prop="bounty" :label="t('refund.bounty')">
        <template #default="{ row }">
          {{ Number(row.bounty).toFixed(2) }} {{ t('refund.isk') }}
        </template>
      </ElTableColumn>
      <ElTableColumn :label="t('refund.sync')"
        ><template #default="{ row }">{{
          row.wallet_updated_at ? formatTime(row.wallet_updated_at) : t('refund.unknownSync')
        }}</template></ElTableColumn
      >
    </ElTable>
    <ElAlert
      v-for="c in value.conflicts"
      :key="c.character_id + ':' + c.journal_id"
      type="error"
      :closable="false"
      :title="`${t('refund.conflict')}: ${c.character_id} · ${formatTime(c.journal_at)} · #${c.refund_id}`"
    />
    <details
      ><summary>{{ t('refund.journals') }} ({{ value.journals.length }})</summary
      ><ElTable :data="value.journals"
        ><ElTableColumn prop="character_id" :label="t('refund.character')" /><ElTableColumn
          prop="journal_id"
          :label="t('refund.journal')"
        /><ElTableColumn prop="date" :label="t('refund.time')">
          <template #default="{ row }">{{ formatTime(row.date) }}</template>
        </ElTableColumn>
        <ElTableColumn prop="amount" :label="t('refund.bounty')">
          <template #default="{ row }">
            {{ Number(row.amount).toFixed(2) }} {{ t('refund.isk') }}
          </template>
        </ElTableColumn>
        <ElTableColumn prop="match_journal_id" :label="t('refund.match')" />
        <ElTableColumn prop="match_date" :label="t('refund.time')">
          <template #default="{ row }">{{ formatTime(row.match_date) }}</template>
        </ElTableColumn>
      </ElTable></details
    >
  </div>
</template>
<script setup lang="ts">
  import { useI18n } from 'vue-i18n'
  import type { RefundCalculation } from '@/types/api/captain-refund'
  import { formatTime } from '@utils/common'
  defineProps<{ value: RefundCalculation }>()
  const { t } = useI18n()
</script>
