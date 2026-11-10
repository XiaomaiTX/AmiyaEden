<template>
  <div class="qqbot-page">
    <ElCard shadow="never">
      <template #header>
        <div class="section-heading">
          <h2 class="section-title">{{ t('qqbot.title') }}</h2>
          <ElButton :disabled="busy" @click="run(load)">{{ t('qqbot.refresh') }}</ElButton>
        </div>
      </template>

      <p class="section-description">{{ t('qqbot.unknown') }}</p>
      <p v-if="message" class="operation-message" role="status">{{ message }}</p>
      <ElForm label-position="top" class="config-form">
        <ElFormItem :label="t('qqbot.enabled')">
          <ElSwitch v-model="config.enabled" />
        </ElFormItem>
        <ElFormItem :label="t('qqbot.bot')">
          <ElInputNumber v-model="config.bot_id" :min="0" />
        </ElFormItem>
        <ElFormItem :label="t('qqbot.cidrs')">
          <ElInput v-model="config.cidrs" />
        </ElFormItem>
        <ElFormItem :label="t('qqbot.interval')">
          <ElInputNumber v-model="config.interval_ms" :min="100" :max="60000" />
        </ElFormItem>
        <ElFormItem :label="t('qqbot.token')">
          <ElInput v-model="token" type="password" autocomplete="new-password" />
        </ElFormItem>
        <ElButton type="primary" :disabled="busy" @click="run(save)">
          {{ t('qqbot.save') }}
        </ElButton>
      </ElForm>
    </ElCard>

    <ElCard shadow="never">
      <template #header>
        <h2 class="section-title">{{ t('qqbot.risk') }}</h2>
      </template>

      <div class="status-grid">
        <div class="status-item">
          <span class="status-label">{{ t('qqbot.connected') }}</span>
          <span>{{ t(snapshot?.connected ? 'qqbot.online' : 'qqbot.offline') }}</span>
          <span v-if="snapshot?.last_frame" class="status-detail">{{ snapshot.last_frame }}</span>
        </div>
        <div class="status-item">
          <span class="status-label">{{ t('qqbot.rate') }}</span>
          <span>
            {{ t(snapshot?.redis_available ? 'qqbot.online' : 'qqbot.offline') }} /
            {{ snapshot?.rate_wait_ms }}
          </span>
        </div>
      </div>
      <ElAlert
        v-if="snapshot?.ingress_error"
        class="status-alert"
        type="error"
        :closable="false"
        :title="snapshot.ingress_error"
        show-icon
      />
      <div v-if="snapshot?.circuits.length" class="circuit-list">
        <div v-for="c in snapshot.circuits" :key="c.class" class="circuit-row">
          <div class="circuit-info">
            <span class="status-label">{{ c.class }}</span>
            <span>{{ c.state }} ({{ c.successes }} / {{ c.errors }})</span>
          </div>
          <ElButton :disabled="busy" @click="run(() => pauseQQBotCircuit(c.class, !c.paused))">
            {{ t(c.paused ? 'qqbot.resume' : 'qqbot.pause') }}
          </ElButton>
        </div>
      </div>
    </ElCard>

    <QQNotifications />
    <QQNotifications channel="fleet" />
    <QQNotifications channel="structure" />
    <QQGroupBusiness />

    <ElCard shadow="never">
      <template #header>
        <h2 class="section-title">{{ t('qqbot.operations') }}</h2>
      </template>
      <div class="table-toolbar">
        <ElInput v-model="query" :placeholder="t('qqbot.id')" />
        <ElButton
          :disabled="busy"
          @click="
            run(async () => {
              selected = query ? await fetchQQBotOperation(query) : null
            })
          "
        >
          {{ t('common.search') }}
        </ElButton>
      </div>
      <ArtTable
        visual-variant="ledger"
        :data="selected ? [selected] : (snapshot?.operations ?? [])"
      >
        <ElTableColumn prop="business_key" :label="t('qqbot.key')" />
        <ElTableColumn prop="kind" :label="t('qqbot.class')" />
        <ElTableColumn prop="status" :label="t('qqbot.status')" />
        <ElTableColumn prop="wait_reason" :label="t('qqbot.waiting')" />
        <ElTableColumn prop="result" :label="t('qqbot.result')" />
        <ElTableColumn :label="t('qqbot.detail')">
          <template #default="{ row }">
            <ElButton
              v-if="['queued', 'leased', 'executing'].includes(row.status)"
              :disabled="busy || row.cancel_requested"
              @click="run(() => cancelQQBotOperation(row.id))"
            >
              {{ t('qqbot.cancel') }}
            </ElButton>
            <ElButton
              v-if="
                row.status === 'failed' &&
                row.failures >= 3 &&
                !row.business_key.startsWith('retry:') &&
                !/^(notification|application|card|kick|inspection):/.test(row.business_key)
              "
              :disabled="busy"
              @click="run(() => retryQQBotOperation(row.id))"
            >
              {{ t('qqbot.retry') }}
            </ElButton>
          </template>
        </ElTableColumn>
      </ArtTable>
    </ElCard>

    <ElCard shadow="never">
      <template #header>
        <h2 class="section-title">{{ t('qqbot.attempts') }}</h2>
      </template>
      <ArtTable visual-variant="ledger" :data="snapshot?.attempts ?? []">
        <ElTableColumn prop="operation_id" :label="t('qqbot.id')" />
        <ElTableColumn prop="classification" :label="t('qqbot.status')" />
        <ElTableColumn prop="started_at" :label="t('qqbot.detail')" />
      </ArtTable>
    </ElCard>

    <ElCard shadow="never">
      <template #header>
        <h2 class="section-title">{{ t('qqbot.deliveries') }}</h2>
      </template>
      <ArtTable visual-variant="ledger" :data="snapshot?.deliveries ?? []">
        <ElTableColumn prop="consumer" :label="t('qqbot.key')" />
        <ElTableColumn prop="status" :label="t('qqbot.status')" />
        <ElTableColumn prop="attempts" :label="t('qqbot.failures')" />
        <ElTableColumn prop="last_error" :label="t('qqbot.detail')" />
        <ElTableColumn :label="t('qqbot.retry')">
          <template #default="{ row }">
            <ElButton
              v-if="row.status === 'dead' && row.attempts < 10"
              :disabled="busy"
              @click="run(() => retryQQBotDelivery(row.id))"
            >
              {{ t('qqbot.retry') }}
            </ElButton>
          </template>
        </ElTableColumn>
      </ArtTable>
    </ElCard>

    <ElCard shadow="never">
      <template #header>
        <h2 class="section-title">{{ t('qqbot.events') }}</h2>
      </template>
      <div v-if="snapshot && snapshot.events.length" class="event-list">
        <details v-for="event in snapshot.events" :key="event.id" class="event-item">
          <summary>{{ event.created_at }} · {{ event.id }}</summary>
          <pre>{{ event.payload }}</pre>
        </details>
      </div>
    </ElCard>
  </div>
</template>
<script setup lang="ts">
  import { ref, onMounted } from 'vue'
  import QQGroupBusiness from '@/components/QQGroupBusiness.vue'
  import QQNotifications from '@/components/QQNotifications.vue'
  import { useI18n } from 'vue-i18n'
  import {
    fetchQQBotConfig,
    fetchQQBotSnapshot,
    fetchQQBotOperation,
    saveQQBotConfig,
    cancelQQBotOperation,
    retryQQBotOperation,
    retryQQBotDelivery,
    pauseQQBotCircuit
  } from '@/api/qqbot'
  import type { QQBotConfig, QQBotSnapshot, QQBotOperation } from '@/types/api/qqbot'

  const { t } = useI18n()
  const config = ref<QQBotConfig>({
    enabled: false,
    bot_id: 0,
    cidrs: '127.0.0.1/32',
    interval_ms: 1000
  })
  const snapshot = ref<QQBotSnapshot | null>(null)
  const query = ref('')
  const selected = ref<QQBotOperation | null>(null)
  const token = ref('')
  const message = ref('')
  const busy = ref(false)

  async function load() {
    const [c, s] = await Promise.all([fetchQQBotConfig(), fetchQQBotSnapshot()])
    config.value = c
    snapshot.value = s
  }

  async function save() {
    await saveQQBotConfig({ ...config.value, ...(token.value ? { token: token.value } : {}) })
    token.value = ''
  }

  async function run(fn: () => Promise<unknown>) {
    busy.value = true
    try {
      await fn()
      await load()
      if (selected.value) selected.value = await fetchQQBotOperation(selected.value.id)
      message.value = t('qqbot.success')
    } catch {
      message.value = t('qqbot.error')
    } finally {
      busy.value = false
    }
  }

  onMounted(() => run(load))
</script>

<style scoped>
  .qqbot-page {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .section-heading,
  .circuit-row,
  .table-toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }

  .section-title {
    margin: 0;
    font-size: 16px;
    font-weight: 600;
  }

  .section-description,
  .operation-message {
    margin: 0 0 16px;
    color: var(--el-text-color-secondary);
  }

  .config-form {
    max-width: 600px;
  }

  .status-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(min(100%, 280px), 1fr));
    gap: 12px;
  }

  .status-item,
  .circuit-info {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .status-label {
    font-size: 13px;
    font-weight: 600;
  }

  .status-detail {
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }

  .status-alert,
  .circuit-list {
    margin-top: 16px;
  }

  .circuit-list {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .circuit-row {
    padding-top: 10px;
    border-top: 1px solid var(--el-border-color-lighter);
  }

  .table-toolbar {
    justify-content: flex-start;
    margin-bottom: 12px;
  }

  .table-toolbar .el-input {
    max-width: 320px;
  }

  .event-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .event-item {
    padding: 10px 12px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: var(--el-border-radius-base);
  }

  .event-item summary {
    cursor: pointer;
  }

  .event-item pre {
    overflow: auto;
    margin: 12px 0 0;
    padding: 12px;
    border-radius: var(--el-border-radius-base);
    background: var(--el-fill-color-light);
    white-space: pre-wrap;
    overflow-wrap: anywhere;
  }

  @media (max-width: 640px) {
    .section-heading,
    .table-toolbar {
      align-items: stretch;
      flex-direction: column;
    }
  }
</style>
