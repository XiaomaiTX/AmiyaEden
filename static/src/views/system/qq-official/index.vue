<template>
  <section class="space-y-4">
    <header>
      <h1 class="text-xl font-semibold">{{ t('qqOfficial.title') }}</h1>
      <p class="text-sm text-g-600">{{ t('qqOfficial.subtitle') }}</p>
    </header>
    <ElCard v-loading="loading" shadow="never">
      <template #header>
        <div class="flex flex-wrap items-center justify-between gap-2">
          <h2 class="font-semibold">{{ t('qqOfficial.runtime') }}</h2>
          <div class="flex flex-wrap gap-2">
            <ElButton @click="refresh">{{ t('qqOfficial.localReload') }}</ElButton>
            <ElButton
              v-if="canManage && status?.has_app_secret && !status.paused"
              type="danger"
              :loading="busy"
              @click="save({ paused: true })"
            >
              {{ t('qqOfficial.pause') }}
            </ElButton>
            <ElButton v-if="canManage && status?.paused" :disabled="busy" @click="resume">
              {{ t('qqOfficial.resume') }}
            </ElButton>
          </div>
        </div>
      </template>
      <ElAlert v-if="error" :title="t('qqOfficial.loadFailed')" type="error" :closable="false" />
      <template v-if="status">
        <dl class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <div v-for="[label, value] in statusRows" :key="label">
            <dt class="text-sm text-g-600">{{ t(`qqOfficial.${label}`) }}</dt>
            <dd>{{ value }}</dd>
          </div>
        </dl>
        <div class="my-3 text-sm">
          <h3 class="text-g-600">{{ t('qqOfficial.cooldowns') }}</h3>
          <ul v-if="status.cooldowns.length">
            <li v-for="(cooldown, index) in status.cooldowns" :key="`${cooldown.scope}-${index}`">
              {{ stateLabel(cooldown.scope.startsWith('group:') ? 'group' : cooldown.scope) }} ·
              {{ formatTime(cooldown.until) }}
            </li>
          </ul>
          <span v-else>—</span>
        </div>
        <div v-if="canManage" class="flex flex-wrap items-center gap-4 border-t pt-3">
          <ElSwitch
            :model-value="status.enabled"
            :disabled="busy || (!status.enabled && !hasUsableQQCredentials(status))"
            :aria-label="t('qqOfficial.enabled')"
            :active-text="t('qqOfficial.enabled')"
            @update:model-value="save({ enabled: $event === true })"
          />
          <ElSwitch
            :model-value="status.certified"
            :disabled="busy || !status.has_app_secret"
            :aria-label="t('qqOfficial.certified')"
            :active-text="t('qqOfficial.certified')"
            @update:model-value="save({ certified: $event === true })"
          />
          <p class="text-sm text-g-600">{{ t('qqOfficial.certifyHint') }}</p>
          <p v-if="!hasUsableQQCredentials(status)" class="w-full text-sm text-g-600">
            {{
              t(
                status.has_app_secret
                  ? 'qqOfficial.credentialsUnreadable'
                  : 'qqOfficial.enableNeedsCredentials'
              )
            }}
          </p>
        </div>
      </template>
    </ElCard>
    <SettingsPanel
      v-if="status && canManage"
      :key="`settings:${scopeKey}`"
      :status="status"
      :busy="busy"
      @save="save"
    />
    <ElTabs v-if="status" :key="scopeKey" v-model="tab">
      <ElTabPane name="groups" :label="t('qqOfficial.groups')">
        <GroupsPanel
          v-if="tab === 'groups'"
          :status="status"
          :actions="actions"
          @task="taskID = $event"
          @requests="openRequests"
        />
      </ElTabPane>
      <ElTabPane name="tasks" :label="t('qqOfficial.tasks')">
        <TasksPanel v-if="tab === 'tasks'" :key="approvalScope" @task="taskID = $event" />
      </ElTabPane>
      <ElTabPane name="requests" :label="t('qqOfficial.requests')">
        <RequestsPanel
          v-if="tab === 'requests' && requestGroup"
          :key="`${requestGroup.id}:${approvalScope}`"
          :group="requestGroup"
          :status="status"
          :actions="actions"
          @task="taskID = $event"
        />
        <ElEmpty v-else :description="t('qqOfficial.requestGroupMissing')" />
      </ElTabPane>
    </ElTabs>
    <TaskDetail
      v-if="status && taskID"
      :id="taskID"
      :key="`${taskID}:${approvalScope}`"
      :actions="actions"
      @close="taskID = ''"
    />
  </section>
</template>

<script setup lang="ts">
  import { computed, ref, shallowRef, watch } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import { useI18n } from 'vue-i18n'
  import { getQQOfficialStatus, updateQQOfficialSettings } from '@/api/generated/client'
  import type { QQOfficialGroup, QQOfficialSettings } from '@/api/generated/client'
  import { unwrapContract } from '@/api/response'
  import {
    qqReadOptions,
    useQQOfficialActions,
    useQQOfficialRead
  } from '@/hooks/useQQOfficialConsole'
  import { hasUsableQQCredentials } from '@/hooks/qq-official-helpers'
  import { formatTime } from '@/utils/common/time'
  import SettingsPanel from './modules/SettingsPanel.vue'
  import GroupsPanel from './modules/GroupsPanel.vue'
  import TasksPanel from './modules/TasksPanel.vue'
  import RequestsPanel from './modules/RequestsPanel.vue'
  import TaskDetail from './modules/TaskDetail.vue'

  defineOptions({ name: 'QQOfficialConsole' })
  const { t } = useI18n()
  const actions = useQQOfficialActions()
  const { busy, canManage } = actions
  const {
    data: status,
    error,
    loading,
    refresh,
    replace
  } = useQQOfficialRead(
    (signal) => unwrapContract(getQQOfficialStatus({ signal, ...qqReadOptions })),
    15_000
  )
  const tab = ref('groups')
  const taskID = ref('')
  const requestGroup = shallowRef<QQOfficialGroup>()
  const credentialRevision = ref(0)
  const scopeKey = computed(() => `${status.value?.app_id ?? ''}:${credentialRevision.value}`)
  const approvalScope = computed(() => String(status.value?.approval_writes_enabled))
  watch(scopeKey, () => {
    tab.value = 'groups'
    requestGroup.value = undefined
    taskID.value = ''
  })
  const stateLabel = (state: string) => t(`qqOfficial.state.${state}`, t('qqOfficial.unknownState'))
  const yesNo = (value: boolean) => t(value ? 'qqOfficial.yes' : 'qqOfficial.no')
  const statusRows = computed(() => {
    const value = status.value
    if (!value) return []
    return [
      ['configured', yesNo(value.configured)],
      ['appID', value.app_id ?? '—'],
      ['appSecretStored', yesNo(value.has_app_secret)],
      ['enabled', yesNo(value.enabled)],
      ['paused', yesNo(value.paused)],
      ['certified', yesNo(value.certified)],
      ['approvalWrites', yesNo(value.approval_writes_enabled)],
      ['token', stateLabel(value.token_state)],
      [
        'gateway',
        t(`qqOfficial.gatewayState.${value.gateway_state}`, t('qqOfficial.unknownState'))
      ],
      ['lastEvent', formatTime(value.last_event_at)],
      ['queueDepth', value.queue_depth],
      ['attemptsMinute', value.attempts_last_minute],
      ['messagesMinute', value.messages_last_minute],
      ['messageBudget', value.message_budget],
      ['nextRequest', formatTime(value.next_request_at)],
      ['oldestQueued', formatTime(value.oldest_queued_at)]
    ]
  })
  const save = async (patch: QQOfficialSettings) => {
    const credentialsChanged = patch.app_secret !== undefined
    const saved = await actions.run(() =>
      unwrapContract(updateQQOfficialSettings(patch, { ...qqReadOptions, sensitive: true }))
    )
    if (saved) {
      if (credentialsChanged) credentialRevision.value++
      replace(saved)
      ElMessage.success(t('qqOfficial.saved'))
    } else await refresh()
  }
  const resume = async () => {
    try {
      await ElMessageBox.confirm(t('qqOfficial.resumeConfirm'), t('qqOfficial.resume'), {
        type: 'warning',
        confirmButtonText: t('common.confirm'),
        cancelButtonText: t('common.cancel')
      })
    } catch {
      return
    }
    await save({ paused: false })
  }
  const openRequests = (group: QQOfficialGroup) => {
    requestGroup.value = group
    tab.value = 'requests'
  }
</script>
