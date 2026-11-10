<template>
  <div class="p-6">
    <h1>{{ t('qqbot.title') }}</h1>
    <p>{{ t('qqbot.unknown') }}</p>
    <ElButton :disabled="busy" @click="run(load)">{{ t('qqbot.refresh') }}</ElButton>
    <p role="status">{{ message }}</p>
    <ElForm label-position="top" style="max-width: 600px">
      <ElFormItem :label="t('qqbot.enabled')"><ElSwitch v-model="config.enabled" /></ElFormItem>
      <ElFormItem :label="t('qqbot.bot')"
        ><ElInputNumber v-model="config.bot_id" :min="0"
      /></ElFormItem>
      <ElFormItem :label="t('qqbot.cidrs')"><ElInput v-model="config.cidrs" /></ElFormItem>
      <ElFormItem :label="t('qqbot.interval')"
        ><ElInputNumber v-model="config.interval_ms" :min="100" :max="60000"
      /></ElFormItem>
      <ElFormItem :label="t('qqbot.token')"
        ><ElInput v-model="token" type="password" autocomplete="new-password"
      /></ElFormItem>
      <ElButton :disabled="busy" @click="run(save)">{{ t('qqbot.save') }}</ElButton>
    </ElForm>
    <p
      >{{ t('qqbot.connected') }}: {{ t(snapshot?.connected ? 'qqbot.online' : 'qqbot.offline') }}
      {{ snapshot?.last_frame }}</p
    >
    <p role="alert">{{ snapshot?.ingress_error }}</p>
    <p
      >{{ t('qqbot.rate') }}:
      {{ t(snapshot?.redis_available ? 'qqbot.online' : 'qqbot.offline') }} /
      {{ snapshot?.rate_wait_ms }}</p
    >
    <h2>{{ t('qqbot.risk') }}</h2>
    <div v-for="c in snapshot?.circuits" :key="c.class">
      {{ c.class }}: {{ c.state }} ({{ c.successes }} / {{ c.errors }})
      <ElButton :disabled="busy" @click="run(() => pauseQQBotCircuit(c.class, !c.paused))">{{
        t(c.paused ? 'qqbot.resume' : 'qqbot.pause')
      }}</ElButton>
    </div>
    <QQNotifications />
    <QQNotifications channel="fleet" />
    <QQNotifications channel="structure" />
    <QQGroupBusiness />
    <h2>{{ t('qqbot.operations') }}</h2>
    <ElInput v-model="query" :placeholder="t('qqbot.id')" />
    <ElButton
      :disabled="busy"
      @click="
        run(async () => {
          selected = query ? await fetchQQBotOperation(query) : null
        })
      "
      >{{ t('common.search') }}</ElButton
    >
    <ArtTable visual-variant="ledger" :data="selected ? [selected] : (snapshot?.operations ?? [])">
      <ElTableColumn prop="business_key" :label="t('qqbot.key')" />
      <ElTableColumn prop="kind" :label="t('qqbot.class')" />
      <ElTableColumn prop="status" :label="t('qqbot.status')" />
      <ElTableColumn prop="wait_reason" :label="t('qqbot.waiting')" />
      <ElTableColumn prop="result" :label="t('qqbot.result')" />
      <ElTableColumn :label="t('qqbot.detail')"
        ><template #default="{ row }">
          <ElButton
            v-if="['queued', 'leased', 'executing'].includes(row.status)"
            :disabled="busy || row.cancel_requested"
            @click="run(() => cancelQQBotOperation(row.id))"
            >{{ t('qqbot.cancel') }}</ElButton
          >
          <ElButton
            v-if="
              row.status === 'failed' &&
              row.failures >= 3 &&
              !row.business_key.startsWith('retry:') &&
              !/^(notification|application|card|kick|inspection):/.test(row.business_key)
            "
            :disabled="busy"
            @click="run(() => retryQQBotOperation(row.id))"
            >{{ t('qqbot.retry') }}</ElButton
          >
        </template></ElTableColumn
      >
    </ArtTable>
    <h2>{{ t('qqbot.attempts') }}</h2>
    <ArtTable visual-variant="ledger" :data="snapshot?.attempts ?? []">
      <ElTableColumn prop="operation_id" :label="t('qqbot.id')" />
      <ElTableColumn prop="classification" :label="t('qqbot.status')" />
      <ElTableColumn prop="started_at" :label="t('qqbot.detail')" />
    </ArtTable>
    <h2>{{ t('qqbot.deliveries') }}</h2>
    <ArtTable visual-variant="ledger" :data="snapshot?.deliveries ?? []">
      <ElTableColumn prop="consumer" :label="t('qqbot.key')" />
      <ElTableColumn prop="status" :label="t('qqbot.status')" />
      <ElTableColumn prop="attempts" :label="t('qqbot.failures')" />
      <ElTableColumn prop="last_error" :label="t('qqbot.detail')" />
      <ElTableColumn :label="t('qqbot.retry')"
        ><template #default="{ row }"
          ><ElButton
            v-if="row.status === 'dead' && row.attempts < 10"
            :disabled="busy"
            @click="run(() => retryQQBotDelivery(row.id))"
            >{{ t('qqbot.retry') }}</ElButton
          ></template
        ></ElTableColumn
      >
    </ArtTable>
    <h2>{{ t('qqbot.events') }}</h2>
    <details v-for="event in snapshot?.events" :key="event.id"
      ><summary>{{ event.created_at }} {{ event.id }}</summary
      ><pre>{{ event.payload }}</pre>
    </details>
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
