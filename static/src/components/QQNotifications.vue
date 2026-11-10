<template>
  <section>
    <h2>{{
      t(
        channel === 'general'
          ? 'qqbot.notifications'
          : channel === 'fleet'
            ? 'qqbot.fleetNotifications'
            : 'qqbot.structureNotifications'
      )
    }}</h2>
    <ElForm label-position="top">
      <ElFormItem :label="t('qqbot.notificationEnabled')"
        ><ElSwitch v-model="enabled"
      /></ElFormItem>
      <ElFormItem :label="t('qqbot.notificationGroups')"><ElInput v-model="groups" /></ElFormItem>
      <ElButton :disabled="busy" @click="run(save)">{{ t('qqbot.save') }}</ElButton>
      <ElButton
        v-if="channel === 'general'"
        :disabled="busy"
        @click="run(() => testQQNotification(test.key, test.expires))"
        >{{ t('qqbot.notificationTest') }}</ElButton
      >
      <ElButton v-if="channel === 'general'" :disabled="busy" @click="newTest">{{
        t('qqbot.notificationNewTest')
      }}</ElButton>
      <ElButton :disabled="busy" @click="run(refresh)">{{ t('qqbot.refresh') }}</ElButton>
    </ElForm>
    <p role="status">{{ message }}</p>
    <ArtTable visual-variant="ledger" :data="targets.filter((row) => row.channel === channel)">
      <ElTableColumn prop="notification_id" :label="t('qqbot.notificationID')" />
      <ElTableColumn prop="group_id" :label="t('qqbot.notificationGroup')" />
      <ElTableColumn prop="operation_id" :label="t('qqbot.id')" />
      <ElTableColumn prop="status" :label="t('qqbot.status')" />
      <ElTableColumn prop="result" :label="t('qqbot.result')" />
    </ArtTable>
  </section>
</template>
<script setup lang="ts">
  import { onMounted, ref } from 'vue'
  import { useI18n } from 'vue-i18n'
  import {
    fetchQQNotificationChannels,
    fetchQQNotificationTargets,
    saveQQNotificationChannel,
    testQQNotification
  } from '@/api/qqbot'
  import type { QQNotificationTarget } from '@/types/api/qqbot'
  const { channel = 'general' } = defineProps<{ channel?: 'general' | 'fleet' | 'structure' }>()
  const { t } = useI18n()
  const enabled = ref(false)
  const groups = ref('')
  const targets = ref<QQNotificationTarget[]>([])
  const busy = ref(false)
  const message = ref('')
  const test = ref({
    key: crypto.randomUUID(),
    expires: new Date(Date.now() + 300000).toISOString()
  })
  function newTest() {
    test.value = { key: crypto.randomUUID(), expires: new Date(Date.now() + 300000).toISOString() }
  }
  async function refresh() {
    targets.value = await fetchQQNotificationTargets()
  }
  async function save() {
    await saveQQNotificationChannel(
      channel,
      enabled.value,
      groups.value.trim() ? groups.value.split(',').map(Number) : []
    )
  }
  async function run(fn: () => Promise<unknown>) {
    busy.value = true
    try {
      await fn()
      await refresh()
      message.value = t('qqbot.success')
    } catch {
      message.value = t('qqbot.error')
    } finally {
      busy.value = false
    }
  }
  onMounted(() =>
    run(async () => {
      const channels = await fetchQQNotificationChannels()
      const general = channels.find((c) => c.name === channel)
      enabled.value = general?.enabled ?? false
      groups.value = general
        ? ((JSON.parse(general.group_ids) as number[] | null)?.join(',') ?? '')
        : ''
    })
  )
</script>
