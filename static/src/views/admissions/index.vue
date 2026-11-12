<template>
  <section
    class="admission-workspace"
    :class="{ 'admission-list': isList, 'art-full-height': isList }"
  >
    <ElAlert
      v-if="error || admissions.error"
      type="error"
      :closable="false"
      show-icon
      :title="t(error || 'admissions.errors.internal')"
    >
      <ElButton @click="refresh">{{ t('admissions.retry') }}</ElButton>
    </ElAlert>
    <ElCard
      v-if="(loading && !isList) || (!admissions.context && admissions.loading && !id)"
      shadow="never"
    >
      <ElSkeleton :rows="5" />
    </ElCard>
    <ElCard v-if="mode === 'new_member' || mode === 'alt'" shadow="never">
      <template #header>
        <div class="card-heading"
          ><h2>{{ t(`admissions.${mode}`) }}</h2
          ><ElButton :loading="admissions.loading" @click="refresh">{{
            t('admissions.refresh')
          }}</ElButton></div
        >
      </template>
      <AdmissionForm
        v-if="admissions.context"
        :key="mode"
        :kind="mode"
        :context="admissions.context"
        @saved="saved"
      />
      <ElEmpty v-else-if="!admissions.loading" :description="t('admissions.denied')" />
    </ElCard>
    <ElCard v-if="isList" class="art-table-card admissions-list-card" shadow="never">
      <ElTabs
        v-if="mode === 'review'"
        :model-value="reviewTab"
        @tab-change="selectStatus($event === 'all' ? '' : String($event))"
      >
        <ElTabPane
          v-for="value in ['pending', 'processed', 'all']"
          :key="value"
          :name="value"
          :label="t(`admissions.${value}`)"
        />
      </ElTabs>
      <ArtTableHeader
        :loading="loading"
        layout="refresh,size,fullscreen,settings"
        @refresh="refresh"
      >
        <template #left>
          <div v-if="mode === 'review'" class="filters">
            <ElSelect
              v-model="kind"
              :aria-label="t('admissions.kind')"
              :placeholder="`${t('admissions.kind')}: ${t('admissions.all')}`"
              @change="filterChanged"
              ><ElOption
                value=""
                :label="`${t('admissions.kind')}: ${t('admissions.all')}`" /><ElOption
                v-for="value in ['new_member', 'alt']"
                :key="value"
                :value="value"
                :label="t(`admissions.${value}`)" /></ElSelect
            ><ElSelect
              v-model="status"
              :aria-label="t('admissions.status')"
              :placeholder="`${t('admissions.status')}: ${t('admissions.all')}`"
              @change="filterChanged"
              ><ElOption
                v-for="value in ['pending', 'processed', 'invited', 'rejected', '']"
                :key="value"
                :value="value"
                :label="`${t('admissions.status')}: ${t(`admissions.${value || 'all'}`)}`" /></ElSelect
            ><ElSelect
              v-model="target"
              class="target-filter"
              :aria-label="t('admissions.target')"
              :placeholder="`${t('admissions.target')}: ${t('admissions.all')}`"
              @change="filterChanged"
              ><ElOption
                value=""
                :label="`${t('admissions.target')}: ${t('admissions.all')}`" /><ElOption
                v-for="item in reviewTargets"
                :key="item.corporation_id"
                :value="String(item.corporation_id)"
                :label="targetName(item.corporation_id)"
            /></ElSelect>
            <ElButton @click="resetFilters">{{ t('common.reset') }}</ElButton>
          </div>
          <h2 v-else>{{ t('admissions.history') }}</h2>
        </template>
      </ArtTableHeader>
      <ArtTable
        :loading="loading"
        :data="list?.list ?? []"
        :empty-text="t('admissions.empty')"
        empty-height="260px"
        :pagination="{ current: page, size: 20, total: list?.total ?? 0 }"
        :pagination-options="{ layout: 'total, prev, pager, next', align: 'right' }"
        @pagination:current-change="changePage"
        ><ElTableColumn
          prop="applicant_name"
          :label="t('admissions.applicant')"
          min-width="140" /><ElTableColumn :label="t('admissions.kind')" width="140"
          ><template #default="{ row }"
            ><span v-if="row.id">{{ t(`admissions.${row.kind}`) }}</span></template
          ></ElTableColumn
        ><ElTableColumn :label="t('admissions.target')" min-width="240"
          ><template #default="{ row }"
            ><span v-if="row.id">{{ targetName(row.target_corporation_id) }}</span></template
          ></ElTableColumn
        ><ElTableColumn :label="t('admissions.characters')" min-width="180"
          ><template #default="{ row }"
            ><span v-if="row.id">{{ characterNames(row) }}</span></template
          ></ElTableColumn
        ><ElTableColumn :label="t('admissions.status')" width="170"
          ><template #default="{ row }">
            <ElTag
              v-if="row.id"
              :type="
                row.status === 'pending'
                  ? 'warning'
                  : row.status === 'invited'
                    ? 'success'
                    : 'danger'
              "
              effect="light"
              >{{ t(`admissions.${row.status}`) }}</ElTag
            >
            <ElTag v-if="row.blacklisted" class="blacklist-tag" type="danger" effect="dark">{{
              t('admissions.blacklist.tag')
            }}</ElTag>
            <div v-if="row.closed_at" class="processing-result">
              <p>{{ t('admissions.operator') }}: {{ row.handler_name }}</p>
              <time :datetime="row.closed_at">{{ date(row.closed_at) }}</time>
              <p
                v-if="row.rejection_reason"
                class="error rejection-reason"
                :title="row.rejection_reason"
                >{{ row.rejection_reason }}</p
              >
            </div>
          </template></ElTableColumn
        ><ElTableColumn :label="t('admissions.submitted')" width="170"
          ><template #default="{ row }">{{ date(row.submitted_at) }}</template></ElTableColumn
        ><ElTableColumn :label="t('common.operation')" width="200" fixed="right"
          ><template #default="{ row }"
            ><div v-if="row.id" class="row-actions"
              ><ArtButtonTable
                v-if="canInviteFromList(row)"
                :label="t('admissions.invited')"
                el-type="success"
                :disabled="row.blacklisted"
                @click="process('invited', row)" /><ArtButtonTable
                :label="t('admissions.detail')"
                el-type="primary"
                @click="router.push(`/admissions/applications/${row.id}`)" /><ArtButtonTable
                v-if="mode === 'history' && row.status !== 'invited'"
                :label="t('admissions.editApplication')"
                el-type="warning"
                @click="
                  router.push(`/admissions/applications/${row.id}/edit`)
                " /></div></template></ElTableColumn
      ></ArtTable>
    </ElCard>

    <template v-if="!loading && id && detail">
      <ElCard v-if="isEditing" shadow="never">
        <template #header
          ><div class="card-heading"
            ><h2>{{ t('admissions.editApplication') }} #{{ id }}</h2
            ><ElButton @click="router.push(`/admissions/applications/${id}`)">{{
              t('admissions.backToDetails')
            }}</ElButton></div
          ></template
        >
        <ElAlert
          v-if="detail.application.rejection_reason"
          class="mb-4"
          type="warning"
          :closable="false"
          show-icon
          :title="detail.application.rejection_reason"
        />
        <AdmissionForm
          v-if="admissions.context && detail.is_owner && detail.application.status !== 'invited'"
          :key="`${detail.application.id}:${detail.application.version}`"
          :kind="detail.application.kind"
          :context="admissions.context"
          :application="detail.application"
          @saved="saved"
        />
        <ElEmpty v-else :description="t('admissions.denied')" />
      </ElCard>
      <template v-else>
        <ElCard shadow="never">
          <template #header>
            <div class="card-heading">
              <div class="summary-title"
                ><h2>{{ t('admissions.detail') }} #{{ detail.application.id }}</h2
                ><ElTag :type="statusType(detail.application.status)">{{
                  t(`admissions.${detail.application.status}`)
                }}</ElTag></div
              >
              <ElButton :loading="loading" @click="refresh">{{ t('admissions.refresh') }}</ElButton>
            </div>
          </template>
          <ElDescriptions :column="descriptionColumns" border>
            <ElDescriptionsItem :label="t('admissions.applicant')">{{
              detail.application.applicant_name
            }}</ElDescriptionsItem>
            <ElDescriptionsItem :label="t('admissions.kind')">{{
              t(`admissions.${detail.application.kind}`)
            }}</ElDescriptionsItem>
            <ElDescriptionsItem :label="t('admissions.target')">{{
              targetName(detail.application.target_corporation_id)
            }}</ElDescriptionsItem>
            <ElDescriptionsItem :label="t('admissions.submitted')">{{
              date(detail.application.submitted_at)
            }}</ElDescriptionsItem>
            <ElDescriptionsItem :label="t('admissions.characters')" :span="descriptionColumns">{{
              characterNames(detail.application)
            }}</ElDescriptionsItem>
            <ElDescriptionsItem
              v-if="detail.application.qq || detail.application.discord_id"
              :label="t('admissions.contacts')"
              :span="descriptionColumns"
            >
              <span v-if="detail.application.qq">QQ: {{ detail.application.qq }}</span
              ><span v-if="detail.application.discord_id" class="contact-value"
                >Discord: {{ detail.application.discord_id }}</span
              >
            </ElDescriptionsItem>
            <ElDescriptionsItem
              v-if="detail.application.closed_at"
              :label="t('admissions.operator')"
              >{{ detail.application.handler_name }}</ElDescriptionsItem
            >
            <ElDescriptionsItem
              v-if="detail.application.closed_at"
              :label="t('admissions.processedAt')"
              >{{ date(detail.application.closed_at) }}</ElDescriptionsItem
            >
            <ElDescriptionsItem
              v-if="detail.application.rejection_reason"
              :label="t('admissions.reason')"
              :span="descriptionColumns"
              ><span class="error">{{
                detail.application.rejection_reason
              }}</span></ElDescriptionsItem
            >
            <ElDescriptionsItem
              v-if="detail.application.evidence_expires_at"
              :label="t('admissions.expires')"
              :span="descriptionColumns"
              >{{ date(detail.application.evidence_expires_at) }}</ElDescriptionsItem
            >
          </ElDescriptions>
          <ElAlert
            v-if="detail.application.kind === 'new_member' && !detail.evidence.length"
            class="evidence-notice"
            type="warning"
            :closable="false"
            show-icon
            :title="t('admissions.expired')"
          />
        </ElCard>
        <div v-if="detail.evidence.length" class="evidence-grid">
          <ElCard
            v-for="evidence in detail.evidence"
            :key="`${evidence.kind}:${evidence.updated_at}`"
            shadow="never"
          >
            <template #header
              ><h2>{{ t(`admissions.${evidence.kind}`) }}</h2></template
            >
            <EvidenceImage :id="detail.application.id" :kind="evidence.kind" />
          </ElCard>
        </div>
        <ElCard v-if="detail.can_review && detail.application.status === 'pending'" shadow="never">
          <template #header
            ><h2>{{ t('admissions.reviewApplication') }}</h2></template
          >
          <div class="review-actions">
            <ElAlert
              v-if="detail.blacklist_hits.length"
              type="error"
              :closable="false"
              show-icon
              :title="t('admissions.blacklist.warning')"
            >
              <ul class="blacklist-hits">
                <li v-for="hit in detail.blacklist_hits" :key="`${hit.kind}:${hit.value}`">
                  <strong>{{ t(`blacklist.kinds.${hit.kind}`) }}</strong
                  >: {{ blacklistValue(hit) }}
                  <p v-if="hit.reason" class="blacklist-reason">{{ hit.reason }}</p>
                </li>
              </ul>
            </ElAlert>
            <ElAlert type="info" :closable="false" show-icon :title="t('admissions.manual')" />
            <ElAlert
              v-if="detail.application.kind === 'new_member'"
              type="warning"
              :closable="false"
              show-icon
              :title="t('admissions.reviewEvidence')"
            />
            <ElForm label-position="top">
              <ElFormItem :label="t('admissions.reason')">
                <ElInput
                  v-model="reason"
                  type="textarea"
                  :rows="3"
                  :maxlength="4000"
                  :placeholder="t('admissions.reasonHint')"
                  :aria-label="t('admissions.reason')"
                />
              </ElFormItem>
              <div class="actions">
                <ElButton
                  type="success"
                  :disabled="busy || detail.blacklist_hits.length > 0"
                  @click="process('invited')"
                  >{{ t('admissions.invitedAction') }}</ElButton
                >
                <ElButton
                  type="danger"
                  :disabled="busy || !reason.trim()"
                  @click="process('reject')"
                  >{{ t('admissions.rejectAction') }}</ElButton
                >
              </div>
            </ElForm>
          </div>
        </ElCard>
        <ElCard shadow="never">
          <template #header
            ><h2>{{ t('admissions.events') }}</h2></template
          >
          <ElTimeline v-if="detail.events.length">
            <ElTimelineItem
              v-for="event in detail.events"
              :key="event.id"
              :timestamp="date(event.created_at)"
              placement="top"
            >
              <p>{{ t(`admissions.${event.action}`) }} · {{ event.actor_name }}</p
              ><p v-if="event.reason" class="event-reason">{{ event.reason }}</p>
            </ElTimelineItem>
          </ElTimeline>
          <ElEmpty v-else :description="t('common.noData')" :image-size="80" />
        </ElCard>
      </template>
    </template>
  </section>
</template>
<script setup lang="ts">
  import { computed, onBeforeUnmount, ref, watch } from 'vue'
  import { useRoute, useRouter } from 'vue-router'
  import { useI18n } from 'vue-i18n'
  import { ElMessageBox } from 'element-plus'
  import { useWindowSize } from '@vueuse/core'
  import { useBadgeStore } from '@/store/modules/badge'
  import { getAdmission, listAdmissions, processAdmission } from '@/api/admissions'
  import { useAdmissionStore } from '@/store/modules/admissions'
  import type {
    AdmissionApplication,
    AdmissionBlacklistHit,
    AdmissionDetail,
    AdmissionList
  } from '@/types/api/admissions'
  import AdmissionForm from './admission-form.vue'
  import EvidenceImage from './evidence-image.vue'
  import ArtButtonTable from '@/components/core/forms/art-button-table/index.vue'
  import { operationSalaryMessage } from '@/utils/operation-salary'
  const { t } = useI18n()
  const route = useRoute()
  const router = useRouter()
  const admissions = useAdmissionStore()
  const id = computed(() => Number(route.params.id) || 0)
  const isEditing = computed(() => !!id.value && route.path.endsWith('/edit'))
  const { width } = useWindowSize()
  const descriptionColumns = computed(() => (width.value < 768 ? 1 : 2))
  const statusType = (status: string) =>
    status === 'pending' ? 'warning' : status === 'invited' ? 'success' : 'danger'
  const mode = computed(() =>
    route.path.endsWith('/new')
      ? 'new_member'
      : route.path.endsWith('/alt')
        ? 'alt'
        : route.path.endsWith('/review')
          ? 'review'
          : 'history'
  )
  const isList = computed(() => !id.value && (mode.value === 'review' || mode.value === 'history'))
  const detail = ref<AdmissionDetail | null>(null)
  const list = ref<AdmissionList | null>(null)
  const page = ref(1)
  const status = ref('pending')
  const target = ref('')
  const kind = ref('')
  const reviewTab = computed(() =>
    status.value === 'pending' ? 'pending' : status.value === '' ? 'all' : 'processed'
  )
  const reason = ref('')
  const loading = ref(false)
  const busy = ref(false)
  const error = ref('')
  let generation = 0
  const reviewTargets = computed(
    () =>
      admissions.context?.targets.filter((item) =>
        admissions.context?.reviewable_corporation_ids.includes(item.corporation_id)
      ) ?? []
  )
  const date = (value: string) => new Date(value).toLocaleString()
  const targetName = (id: number) => t(`admissions.corporations.${id}`)
  const characterNames = (app: AdmissionApplication) =>
    app.characters.map((char) => char.character_name).join(', ')
  const blacklistValue = (hit: AdmissionBlacklistHit) =>
    hit.kind === 'character'
      ? t('blacklist.characterValue', { name: hit.display, id: hit.value })
      : hit.display
  const ownCharacterIDs = computed(
    () => new Set(admissions.context?.characters.map((char) => char.character_id) ?? [])
  )
  // 小号申请无需核对截图，可在列表直接登记已邀请；本人申请由后端拒绝处理，列表不显示入口。
  const canInviteFromList = (app: AdmissionApplication) =>
    mode.value === 'review' &&
    app.kind === 'alt' &&
    app.status === 'pending' &&
    !app.characters.some((char) => ownCharacterIDs.value.has(char.character_id))
  const errorKey = (value: unknown) =>
    value instanceof Error && value.message.startsWith('admissions.errors.')
      ? value.message
      : 'admissions.errors.internal'
  async function load() {
    const current = ++generation
    error.value = ''
    detail.value = null
    list.value = null
    if (mode.value === 'new_member' || mode.value === 'alt') {
      loading.value = false
      return
    }
    loading.value = true
    try {
      if (id.value) {
        const result = await getAdmission(id.value)
        if (current === generation) detail.value = result
      } else {
        const result = await listAdmissions(
          mode.value === 'review',
          page.value,
          mode.value === 'review' ? status.value : '',
          target.value,
          kind.value
        )
        if (current === generation) list.value = result
      }
    } catch (value) {
      if (current === generation) error.value = errorKey(value)
    } finally {
      if (current === generation) loading.value = false
    }
  }
  async function refresh() {
    await Promise.all([admissions.refresh(), load()])
  }
  function filterChanged() {
    page.value = 1
    void load()
  }
  function selectStatus(value: string) {
    status.value = value
    filterChanged()
  }
  function resetFilters() {
    kind.value = ''
    target.value = ''
    filterChanged()
  }
  function changePage(value: number) {
    page.value = value
    void load()
  }
  async function saved(id: number) {
    void useBadgeStore()
      .fetchBadgeCounts()
      .catch(() => undefined)
    await admissions.refresh()
    await router.push(`/admissions/applications/${id}`)
    await load()
  }
  async function process(action: 'invited' | 'reject', application = detail.value?.application) {
    if (!application || busy.value) return
    const submittedReason = detail.value ? reason.value : ''
    busy.value = true
    try {
      const salary =
        action === 'invited' && application.kind === 'new_member'
          ? await operationSalaryMessage('recruitment', 1, t)
          : ''
      await ElMessageBox.confirm(
        [t(action === 'invited' ? 'admissions.confirmInvited' : 'admissions.confirmReject'), salary]
          .filter(Boolean)
          .join('\n\n'),
        t('admissions.review'),
        { type: 'warning' }
      )
    } catch {
      busy.value = false
      return
    }
    const latest =
      detail.value?.application ?? list.value?.list.find((item) => item.id === application.id)
    if (latest?.id !== application.id || latest.version !== application.version) {
      busy.value = false
      return
    }
    error.value = ''
    try {
      await processAdmission(application.id, application.version, action, submittedReason)
      void useBadgeStore()
        .fetchBadgeCounts()
        .catch(() => undefined)
      await load()
    } catch (value) {
      const key = errorKey(value)
      // 黑名单可能在打开页面后刷新；重新加载以显示最新命中。
      if (key === 'admissions.errors.blacklisted') await load()
      error.value = key
    } finally {
      busy.value = false
    }
  }
  watch(
    () => route.fullPath,
    () => {
      page.value = 1
      reason.value = ''
      void refresh()
    },
    { immediate: true }
  )
  onBeforeUnmount(() => {
    generation++
  })
</script>
<style scoped>
  .admission-workspace {
    display: flex;
    flex-direction: column;
    min-width: 0;
    gap: 16px;
    margin: 0;
  }
  .admission-list {
    display: flex;
    flex-direction: column;
    max-width: none;
    margin: 0;
    min-height: 0;
    gap: 16px;
  }
  .admissions-list-card {
    flex: 1;
    min-height: 0;
  }
  .admissions-list-card :deep(.el-card__body) {
    height: 100%;
    display: flex;
    flex-direction: column;
  }
  .admissions-list-card :deep(.art-table) {
    flex: 1;
    min-height: 320px;
    height: auto !important;
    display: flex;
    flex-direction: column;
  }
  .admissions-list-card :deep(.art-table > .el-table) {
    flex: 1;
    min-height: 0;
  }
  .card-heading,
  .summary-title,
  .actions,
  .filters {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 16px;
  }
  .card-heading {
    justify-content: space-between;
  }
  h2 {
    font-size: 18px;
    font-weight: 600;
  }
  a {
    color: var(--el-color-primary);
  }
  .filters :deep(.el-select) {
    width: 180px;
  }
  .filters :deep(.target-filter) {
    width: 300px;
  }
  .filters {
    gap: 12px;
  }
  @media (max-width: 720px) {
    .filters :deep(.el-select) {
      width: 100%;
    }
    .filters {
      width: 100%;
    }
  }
  .error {
    color: var(--el-color-danger);
  }
  .processing-result {
    margin-top: 4px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }
  .rejection-reason {
    display: -webkit-box;
    overflow: hidden;
    overflow-wrap: anywhere;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 2;
  }
  .row-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
  }
  .review-actions {
    display: grid;
    gap: 16px;
  }
  .evidence-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 20px;
  }
  .evidence-grid > * {
    min-width: 0;
  }
  .contact-value {
    margin-left: 12px;
  }
  .blacklist-tag {
    margin-left: 6px;
  }
  .blacklist-hits {
    display: grid;
    gap: 6px;
    margin: 4px 0 0;
    padding-left: 16px;
    list-style: disc;
  }
  .blacklist-reason {
    white-space: pre-line;
    overflow-wrap: anywhere;
  }
  .evidence-notice {
    margin-top: 16px;
  }
  .event-reason {
    margin-top: 6px;
    color: var(--el-text-color-secondary);
  }
  :deep(.el-descriptions__content) {
    overflow-wrap: anywhere;
  }
  @media (max-width: 720px) {
    .evidence-grid {
      grid-template-columns: minmax(0, 1fr);
    }
  }
</style>
