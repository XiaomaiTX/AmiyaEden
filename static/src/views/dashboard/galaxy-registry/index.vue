<template>
  <div class="galaxy-registry-page">
    <ElCard shadow="never" class="art-card mb-4">
      <div class="flex flex-col gap-1">
        <h2 class="text-lg font-medium">{{ t('galaxyRegistry.title') }}</h2>
        <p class="text-sm text-g-500">{{ t('galaxyRegistry.subtitle') }}</p>
      </div>
    </ElCard>

    <ElTabs v-model="activeTab">
      <ElTabPane :label="t('galaxyRegistry.tabs.current')" name="current">
        <div class="galaxy-registry-page__summary mb-4">
          <ElCard shadow="never" class="art-card">
            <div class="text-sm text-g-500">{{ t('galaxyRegistry.summary.idle') }}</div>
            <div class="text-2xl font-semibold">{{ systemsSummary.idle_count }}</div>
          </ElCard>
          <ElCard shadow="never" class="art-card">
            <div class="text-sm text-g-500">{{ t('galaxyRegistry.summary.busy') }}</div>
            <div class="text-2xl font-semibold">{{ systemsSummary.busy_count }}</div>
          </ElCard>
          <ElCard shadow="never" class="art-card">
            <div class="text-sm text-g-500">{{ t('galaxyRegistry.summary.overdue') }}</div>
            <div class="text-2xl font-semibold">{{ systemsSummary.overdue_count }}</div>
          </ElCard>
        </div>

        <ElCard shadow="never" class="art-card">
          <template #header>
            <div class="flex items-center justify-between gap-3 flex-wrap">
              <span>{{ t('galaxyRegistry.current.title') }}</span>
              <ElButton :loading="loading.systems" @click="loadSystems">
                {{ t('common.refresh') }}
              </ElButton>
            </div>
          </template>

          <ElTable :data="systems" v-loading="loading.systems" stripe border>
            <ElTableColumn
              prop="solar_system_name"
              :label="t('galaxyRegistry.columns.system')"
              min-width="180"
            >
              <template #default="{ row }">
                <div class="font-medium">{{ row.solar_system_name }}</div>
                <div class="text-xs text-g-500">
                  {{ row.region_name }} / {{ row.constellation_name }}
                </div>
              </template>
            </ElTableColumn>
            <ElTableColumn
              prop="security"
              :label="t('galaxyRegistry.columns.security')"
              width="110"
            >
              <template #default="{ row }">{{ formatSecurity(row.security) }}</template>
            </ElTableColumn>
            <ElTableColumn prop="note" :label="t('galaxyRegistry.columns.note')" min-width="180" />
            <ElTableColumn :label="t('galaxyRegistry.columns.minBounty')" width="170">
              <template #default="{ row }">{{ formatIsk(row.min_bounty_amount) }}</template>
            </ElTableColumn>
            <ElTableColumn :label="t('galaxyRegistry.columns.status')" width="140">
              <template #default="{ row }">
                <ElTag :type="statusTagType(row.status)" effect="plain">
                  {{ statusLabel(row.status) }}
                </ElTag>
              </template>
            </ElTableColumn>
            <ElTableColumn :label="t('galaxyRegistry.columns.captain')" min-width="220">
              <template #default="{ row }">
                <span v-if="row.active_entry">
                  {{ row.active_entry.captain_nickname }}
                  <span class="text-g-500">({{ row.active_entry.captain_character_name }})</span>
                </span>
                <span v-else>--</span>
              </template>
            </ElTableColumn>
            <ElTableColumn :label="t('galaxyRegistry.columns.expectedEndAt')" width="190">
              <template #default="{ row }">
                {{ formatDateTime(row.active_entry?.expected_end_at) }}
              </template>
            </ElTableColumn>
            <ElTableColumn :label="t('common.operation')" width="260" fixed="right">
              <template #default="{ row }">
                <div class="flex flex-wrap gap-2">
                  <ElButton
                    v-if="canCreateEntry(row)"
                    type="primary"
                    size="small"
                    @click="openCreateDialog(row)"
                  >
                    {{ t('galaxyRegistry.actions.createEntry') }}
                  </ElButton>
                  <ElButton
                    v-if="canEditExpectedEnd(row)"
                    size="small"
                    @click="openExpectedEndDialogFromSystem(row)"
                  >
                    {{ t('galaxyRegistry.actions.editExpectedEnd') }}
                  </ElButton>
                  <ElButton
                    v-if="canEndEntry(row)"
                    type="warning"
                    size="small"
                    :loading="endingEntryId === row.active_entry?.entry_id"
                    @click="handleEndEntryById(row.active_entry?.entry_id)"
                  >
                    {{ t('galaxyRegistry.actions.endEntry') }}
                  </ElButton>
                </div>
              </template>
            </ElTableColumn>
          </ElTable>
        </ElCard>
      </ElTabPane>

      <ElTabPane v-if="canCaptainTab" :label="t('galaxyRegistry.tabs.captain')" name="captain">
        <ElCard shadow="never" class="art-card mb-4">
          <template #header>
            <span>{{ t('galaxyRegistry.captain.activeTitle') }}</span>
          </template>

          <div v-if="myActiveSystems.length" class="galaxy-registry-page__active-list">
            <div
              v-for="row in myActiveSystems"
              :key="row.system_config_id"
              class="galaxy-registry-page__active-card"
            >
              <div>
                <div class="font-medium">{{ row.solar_system_name }}</div>
                <div class="text-xs text-g-500">
                  {{ row.region_name }} / {{ row.constellation_name }}
                </div>
                <div class="text-sm mt-2">
                  {{ t('galaxyRegistry.columns.expectedEndAt') }}:
                  {{ formatDateTime(row.active_entry?.expected_end_at) }}
                </div>
              </div>
              <div class="flex flex-wrap gap-2">
                <ElButton size="small" @click="openExpectedEndDialogFromSystem(row)">
                  {{ t('galaxyRegistry.actions.editExpectedEnd') }}
                </ElButton>
                <ElButton
                  size="small"
                  type="warning"
                  :loading="endingEntryId === row.active_entry?.entry_id"
                  @click="handleEndEntryById(row.active_entry?.entry_id)"
                >
                  {{ t('galaxyRegistry.actions.endEntry') }}
                </ElButton>
              </div>
            </div>
          </div>
          <ElEmpty v-else :description="t('galaxyRegistry.captain.noActiveEntry')" />
        </ElCard>

        <ElCard shadow="never" class="art-table-card">
          <div class="flex flex-wrap gap-3 mb-4">
            <ElSelect
              v-model="mySearchParams.status"
              clearable
              class="w-[160px]"
              @change="loadMyEntries"
            >
              <ElOption value="active" :label="t('galaxyRegistry.entryStatus.active')" />
              <ElOption value="completed" :label="t('galaxyRegistry.entryStatus.completed')" />
            </ElSelect>
            <ElSelect
              v-model="mySearchParams.validation_status"
              clearable
              class="w-[160px]"
              @change="loadMyEntries"
            >
              <ElOption value="pending" :label="t('galaxyRegistry.validationStatus.pending')" />
              <ElOption value="valid" :label="t('galaxyRegistry.validationStatus.valid')" />
              <ElOption value="violation" :label="t('galaxyRegistry.validationStatus.violation')" />
            </ElSelect>
            <ElButton type="primary" @click="() => loadMyEntries()">
              {{ t('common.search') }}
            </ElButton>
          </div>

          <ArtTableHeader
            v-model:columns="myColumnChecks"
            :loading="myEntriesLoading"
            @refresh="refreshMyEntries"
          />
          <ArtTable
            :loading="myEntriesLoading"
            :data="myEntries"
            :columns="myColumns"
            :pagination="myPagination"
            visual-variant="ledger"
            @pagination:size-change="handleMySizeChange"
            @pagination:current-change="handleMyCurrentChange"
          />
        </ElCard>
      </ElTabPane>

      <ElTabPane v-if="canAdminTab" :label="t('galaxyRegistry.tabs.admin')" name="admin">
        <ElCard shadow="never" class="art-card mb-4">
          <div class="flex items-center justify-between gap-3 flex-wrap">
            <div class="flex flex-wrap gap-2">
              <ElButton :loading="loading.adminSystems" @click="loadAdminSystems">
                {{ t('common.refresh') }}
              </ElButton>
              <ElButton
                type="primary"
                :disabled="!hasDirtySystems"
                :loading="savingAllSystems"
                @click="handleSaveAllSystems"
              >
                {{ t('galaxyRegistry.admin.saveAllSystems') }}
              </ElButton>
            </div>
            <div class="text-sm text-g-500">
              {{ t('galaxyRegistry.admin.pendingChanges', { count: dirtySystemCount }) }}
            </div>
          </div>
        </ElCard>

        <ElCard shadow="never" class="art-card mb-4">
          <template #header>
            <span>{{ t('galaxyRegistry.admin.addSystemsTitle') }}</span>
          </template>

          <div class="flex flex-wrap gap-3 mb-4">
            <ElInput
              v-model="adminSystemSearch.keyword"
              class="w-[280px]"
              :placeholder="t('galaxyRegistry.admin.searchPlaceholder')"
              @keyup.enter="searchSdeSystems"
            />
            <ElButton type="primary" :loading="loading.sdeSearch" @click="searchSdeSystems">
              {{ t('common.search') }}
            </ElButton>
          </div>

          <ElTable :data="sdeSystems" v-loading="loading.sdeSearch" stripe border>
            <ElTableColumn
              prop="solar_system_name"
              :label="t('galaxyRegistry.columns.system')"
              min-width="160"
            />
            <ElTableColumn :label="t('galaxyRegistry.columns.security')" width="100">
              <template #default="{ row }">{{ formatSecurity(row.security) }}</template>
            </ElTableColumn>
            <ElTableColumn :label="t('galaxyRegistry.columns.regionConstellation')" min-width="240">
              <template #default="{ row }">
                {{ row.region_name }} / {{ row.constellation_name }}
              </template>
            </ElTableColumn>
            <ElTableColumn :label="t('common.operation')" width="140">
              <template #default="{ row }">
                <ElButton
                  size="small"
                  type="primary"
                  :disabled="hasSystemDraft(row.solar_system_id)"
                  @click="handleAddSystemDraft(row)"
                >
                  {{ t('galaxyRegistry.admin.addSystem') }}
                </ElButton>
              </template>
            </ElTableColumn>
          </ElTable>
        </ElCard>

        <ElCard shadow="never" class="art-card mb-4">
          <template #header>
            <span>{{ t('galaxyRegistry.admin.systemConfigTitle') }}</span>
          </template>

          <ElTable
            :data="adminSystems"
            v-loading="loading.adminSystems"
            stripe
            border
            row-key="local_id"
          >
            <ElTableColumn
              prop="solar_system_name"
              :label="t('galaxyRegistry.columns.system')"
              min-width="160"
            />
            <ElTableColumn :label="t('galaxyRegistry.columns.regionConstellation')" min-width="240">
              <template #default="{ row }">
                {{ row.region_name }} / {{ row.constellation_name }}
              </template>
            </ElTableColumn>
            <ElTableColumn :label="t('galaxyRegistry.columns.note')" min-width="180">
              <template #default="{ row }">
                <ElInput v-model="row.note" @input="markSystemDirty(row)" />
              </template>
            </ElTableColumn>
            <ElTableColumn :label="t('galaxyRegistry.columns.minBounty')" width="190">
              <template #default="{ row }">
                <ElInputNumber
                  v-model="row.min_bounty_amount"
                  :min="0"
                  :step="1000000"
                  class="w-full"
                  @change="markSystemDirty(row)"
                />
              </template>
            </ElTableColumn>
            <ElTableColumn :label="t('galaxyRegistry.columns.enabled')" width="120">
              <template #default="{ row }">
                <ElSwitch v-model="row.is_enabled" @change="markSystemDirty(row)" />
              </template>
            </ElTableColumn>
            <ElTableColumn :label="t('galaxyRegistry.columns.pending')" width="120">
              <template #default="{ row }">
                <ElTag v-if="row.is_new" type="primary" size="small">
                  {{ t('galaxyRegistry.admin.newSystem') }}
                </ElTag>
                <ElTag v-else-if="row.is_dirty" type="warning" size="small">
                  {{ t('galaxyRegistry.admin.modifiedSystem') }}
                </ElTag>
                <span v-else>--</span>
              </template>
            </ElTableColumn>
            <ElTableColumn :label="t('common.operation')" width="120" fixed="right">
              <template #default="{ row }">
                <ElButton size="small" type="danger" @click="handleDeleteSystem(row)">
                  {{ t('common.delete') }}
                </ElButton>
              </template>
            </ElTableColumn>
          </ElTable>
        </ElCard>

        <ElCard shadow="never" class="art-card mb-4">
          <template #header>
            <div class="flex items-center justify-between gap-3 flex-wrap">
              <span>{{ t('galaxyRegistry.admin.analyticsTitle') }}</span>
              <div class="flex flex-wrap gap-2">
                <ElDatePicker
                  v-model="analyticsRange"
                  type="daterange"
                  value-format="YYYY-MM-DD"
                  :range-separator="t('galaxyRegistry.admin.rangeSeparator')"
                  :start-placeholder="t('galaxyRegistry.admin.rangeStart')"
                  :end-placeholder="t('galaxyRegistry.admin.rangeEnd')"
                />
                <ElButton type="primary" :loading="loading.analytics" @click="loadAnalytics">
                  {{ t('common.search') }}
                </ElButton>
              </div>
            </div>
          </template>

          <template v-if="analytics">
            <div class="galaxy-registry-page__summary mb-4">
              <ElCard shadow="never" class="art-card">
                <div class="text-sm text-g-500">{{ t('galaxyRegistry.summary.idle') }}</div>
                <div class="text-2xl font-semibold">
                  {{ analytics.current_snapshot.idle_count }}
                </div>
              </ElCard>
              <ElCard shadow="never" class="art-card">
                <div class="text-sm text-g-500">{{ t('galaxyRegistry.summary.busy') }}</div>
                <div class="text-2xl font-semibold">
                  {{ analytics.current_snapshot.busy_count }}
                </div>
              </ElCard>
              <ElCard shadow="never" class="art-card">
                <div class="text-sm text-g-500">{{ t('galaxyRegistry.summary.overdue') }}</div>
                <div class="text-2xl font-semibold">
                  {{ analytics.current_snapshot.overdue_count }}
                </div>
              </ElCard>
            </div>

            <div class="galaxy-registry-page__summary mb-4">
              <ElCard shadow="never" class="art-card">
                <div class="text-sm text-g-500">{{ t('galaxyRegistry.admin.recent7d') }}</div>
                <div class="text-lg font-semibold">
                  {{ analytics.recent_7d.valid_count }} / {{ analytics.recent_7d.entry_count }}
                </div>
              </ElCard>
              <ElCard shadow="never" class="art-card">
                <div class="text-sm text-g-500">{{ t('galaxyRegistry.admin.recent30d') }}</div>
                <div class="text-lg font-semibold">
                  {{ analytics.recent_30d.valid_count }} / {{ analytics.recent_30d.entry_count }}
                </div>
              </ElCard>
              <ElCard shadow="never" class="art-card">
                <div class="text-sm text-g-500">{{ t('galaxyRegistry.admin.violations') }}</div>
                <div class="text-lg font-semibold">{{ analytics.recent_30d.violation_count }}</div>
              </ElCard>
            </div>

            <ElTable :data="analytics.top_systems" stripe border class="mb-4">
              <ElTableColumn
                prop="solar_system_name"
                :label="t('galaxyRegistry.admin.topSystems')"
                min-width="220"
              />
              <ElTableColumn
                prop="register_count"
                :label="t('galaxyRegistry.admin.registerCount')"
                width="140"
              />
            </ElTable>

            <ElTable :data="analytics.recent_violations" stripe border>
              <ElTableColumn
                prop="solar_system_name"
                :label="t('galaxyRegistry.columns.system')"
                min-width="160"
              />
              <ElTableColumn :label="t('galaxyRegistry.columns.captain')" min-width="200">
                <template #default="{ row }">
                  {{ row.captain_nickname }} ({{ row.captain_character_name }})
                </template>
              </ElTableColumn>
              <ElTableColumn
                prop="violation_reason"
                :label="t('galaxyRegistry.columns.violationReason')"
                min-width="180"
              >
                <template #default="{ row }">
                  {{ violationReasonLabel(row.violation_reason) }}
                </template>
              </ElTableColumn>
              <ElTableColumn :label="t('galaxyRegistry.columns.actualEndAt')" width="180">
                <template #default="{ row }">{{ formatDateTime(row.actual_end_at) }}</template>
              </ElTableColumn>
            </ElTable>
          </template>
        </ElCard>

        <ElCard shadow="never" class="art-table-card">
          <div class="flex flex-wrap gap-3 mb-4">
            <ElInput
              v-model="adminSearchParams.keyword"
              class="w-[220px]"
              :placeholder="t('galaxyRegistry.admin.entryKeywordPlaceholder')"
              @keyup.enter="loadAdminEntries"
            />
            <ElSelect
              v-model="adminSearchParams.status"
              clearable
              class="w-[160px]"
              @change="loadAdminEntries"
            >
              <ElOption value="active" :label="t('galaxyRegistry.entryStatus.active')" />
              <ElOption value="completed" :label="t('galaxyRegistry.entryStatus.completed')" />
            </ElSelect>
            <ElSelect
              v-model="adminSearchParams.validation_status"
              clearable
              class="w-[160px]"
              @change="loadAdminEntries"
            >
              <ElOption value="pending" :label="t('galaxyRegistry.validationStatus.pending')" />
              <ElOption value="valid" :label="t('galaxyRegistry.validationStatus.valid')" />
              <ElOption value="violation" :label="t('galaxyRegistry.validationStatus.violation')" />
            </ElSelect>
            <ElButton type="primary" @click="() => loadAdminEntries()">
              {{ t('common.search') }}
            </ElButton>
          </div>

          <ArtTableHeader
            v-model:columns="adminColumnChecks"
            :loading="adminEntriesLoading"
            @refresh="refreshAdminEntries"
          />
          <ArtTable
            :loading="adminEntriesLoading"
            :data="adminEntries"
            :columns="adminColumns"
            :pagination="adminPagination"
            visual-variant="ledger"
            @pagination:size-change="handleAdminSizeChange"
            @pagination:current-change="handleAdminCurrentChange"
          />
        </ElCard>
      </ElTabPane>
    </ElTabs>

    <ElDialog
      v-model="createDialogVisible"
      :title="t('galaxyRegistry.actions.createEntry')"
      width="420px"
    >
      <ElForm label-position="top">
        <ElFormItem :label="t('galaxyRegistry.columns.system')">
          <ElInput :model-value="selectedSystem?.solar_system_name || ''" disabled />
        </ElFormItem>
        <ElFormItem :label="t('galaxyRegistry.columns.expectedEndAt')">
          <ElDatePicker
            v-model="createForm.expected_end_at"
            type="datetime"
            value-format="YYYY-MM-DD HH:mm:ss"
            class="w-full"
          />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="createDialogVisible = false">{{ t('common.cancel') }}</ElButton>
        <ElButton type="primary" :loading="creatingEntry" @click="handleSubmitCreateEntry">
          {{ t('common.confirm') }}
        </ElButton>
      </template>
    </ElDialog>

    <ElDialog
      v-model="expectedEndDialogVisible"
      :title="t('galaxyRegistry.actions.editExpectedEnd')"
      width="420px"
    >
      <ElForm label-position="top">
        <ElFormItem :label="t('galaxyRegistry.columns.system')">
          <ElInput :model-value="expectedEndDialog.system_name" disabled />
        </ElFormItem>
        <ElFormItem :label="t('galaxyRegistry.columns.expectedEndAt')">
          <ElDatePicker
            v-model="expectedEndDialog.expected_end_at"
            type="datetime"
            value-format="YYYY-MM-DD HH:mm:ss"
            class="w-full"
          />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="expectedEndDialogVisible = false">{{ t('common.cancel') }}</ElButton>
        <ElButton
          type="primary"
          :loading="updatingExpectedEnd"
          @click="handleSubmitExpectedEndUpdate"
        >
          {{ t('common.save') }}
        </ElButton>
      </template>
    </ElDialog>

    <ElDialog
      v-model="validationDialogVisible"
      :title="t('galaxyRegistry.admin.validationDialogTitle')"
      width="420px"
    >
      <ElForm label-position="top">
        <ElFormItem :label="t('galaxyRegistry.columns.system')">
          <ElInput :model-value="validationDialog.system_name" disabled />
        </ElFormItem>
        <ElFormItem :label="t('galaxyRegistry.columns.validationStatus')">
          <ElSelect v-model="validationDialog.validation_status" class="w-full">
            <ElOption value="valid" :label="t('galaxyRegistry.validationStatus.valid')" />
            <ElOption value="violation" :label="t('galaxyRegistry.validationStatus.violation')" />
          </ElSelect>
        </ElFormItem>
        <ElFormItem
          v-if="validationDialog.validation_status === 'violation'"
          :label="t('galaxyRegistry.columns.violationReason')"
        >
          <ElSelect v-model="validationDialog.violation_reason" clearable class="w-full">
            <ElOption
              value="no_bounty_in_window"
              :label="t('galaxyRegistry.violationReasons.no_bounty_in_window')"
            />
            <ElOption
              value="bounty_below_threshold"
              :label="t('galaxyRegistry.violationReasons.bounty_below_threshold')"
            />
          </ElSelect>
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="validationDialogVisible = false">{{ t('common.cancel') }}</ElButton>
        <ElButton
          type="primary"
          :loading="updatingValidation"
          @click="handleSubmitValidationUpdate"
        >
          {{ t('common.save') }}
        </ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import type { ColumnOption } from '@/types/component'
  import { h, computed, onMounted, reactive, ref, watch } from 'vue'
  import { ElButton, ElMessage, ElMessageBox, ElTag } from 'element-plus'
  import { useI18n } from 'vue-i18n'
  import { useTable } from '@/hooks/core/useTable'
  import { useUserStore } from '@/store/modules/user'
  import { formatTime } from '@/utils/common/time'
  import {
    createAdminGalaxyRegistrySystem,
    createGalaxyRegistryEntry,
    deleteAdminGalaxyRegistrySystem,
    endGalaxyRegistryEntry,
    fetchAdminGalaxyRegistryAnalytics,
    fetchAdminGalaxyRegistryEntries,
    fetchAdminGalaxyRegistrySystems,
    fetchGalaxyRegistrySystems,
    fetchMyGalaxyRegistryEntries,
    forceEndAdminGalaxyRegistryEntry,
    searchGalaxyRegistrySdeSystems,
    updateAdminGalaxyRegistryEntryValidation,
    updateAdminGalaxyRegistrySystem,
    updateGalaxyRegistryEntryExpectedEndAt
  } from '@/api/galaxy-registry'

  defineOptions({ name: 'DashboardGalaxyRegistry' })

  type TagType = 'primary' | 'success' | 'warning' | 'info' | 'danger'

  type AdminSystemDraft = Api.Dashboard.GalaxyRegistryAdminSystem & {
    local_id: string
    is_new: boolean
    is_dirty: boolean
  }

  const { t } = useI18n()
  const userStore = useUserStore()

  const roles = computed(() => userStore.info.roles || [])
  const canCaptainTab = computed(
    () =>
      roles.value.includes('captain') ||
      roles.value.includes('admin') ||
      roles.value.includes('super_admin')
  )
  const canAdminTab = computed(
    () => roles.value.includes('admin') || roles.value.includes('super_admin')
  )

  const activeTab = ref<'current' | 'captain' | 'admin'>('current')

  const loading = reactive({
    systems: false,
    adminSystems: false,
    sdeSearch: false,
    analytics: false
  })

  const systems = ref<Api.Dashboard.GalaxyRegistrySystemItem[]>([])
  const systemsSummary = reactive<Api.Dashboard.GalaxyRegistrySystemSummary>({
    idle_count: 0,
    busy_count: 0,
    overdue_count: 0
  })
  const analytics = ref<Api.Dashboard.GalaxyRegistryAdminAnalytics | null>(null)
  const analyticsRange = ref<[string, string] | null>(null)
  const sdeSystems = ref<Api.Dashboard.GalaxyRegistrySdeSystem[]>([])
  const adminSystems = ref<AdminSystemDraft[]>([])
  const adminSystemSearch = reactive<Api.Dashboard.GalaxyRegistrySdeSystemSearchParams>({
    keyword: '',
    limit: 20
  })

  const createDialogVisible = ref(false)
  const selectedSystem = ref<Api.Dashboard.GalaxyRegistrySystemItem | null>(null)
  const createForm = reactive<Api.Dashboard.GalaxyRegistryCreateEntryRequest>({
    system_config_id: 0,
    expected_end_at: ''
  })

  const expectedEndDialogVisible = ref(false)
  const expectedEndDialog = reactive({
    entry_id: 0,
    system_name: '',
    expected_end_at: ''
  })

  const validationDialogVisible = ref(false)
  const validationDialog = reactive<
    Api.Dashboard.GalaxyRegistryAdminUpdateValidationRequest & {
      entry_id: number
      system_name: string
    }
  >({
    entry_id: 0,
    system_name: '',
    validation_status: 'valid',
    violation_reason: ''
  })

  const creatingEntry = ref(false)
  const endingEntryId = ref(0)
  const updatingExpectedEnd = ref(false)
  const updatingValidation = ref(false)
  const savingAllSystems = ref(false)
  const forceEndingEntryId = ref(0)

  const adaptPage = <T,>(response: Api.Common.PaginatedResponse<T>) => ({
    list: response.list || [],
    total: response.total || 0,
    current: response.page || 1,
    size: response.pageSize || 200
  })

  const myTable = useTable({
    core: {
      apiFn: fetchMyGalaxyRegistryEntries,
      apiParams: {
        current: 1,
        size: 200,
        status: '',
        validation_status: ''
      },
      immediate: false,
      columnsFactory: () =>
        [
          {
            prop: 'solar_system_name',
            label: t('galaxyRegistry.columns.system'),
            minWidth: 160
          },
          {
            prop: 'actual_start_at',
            label: t('galaxyRegistry.columns.actualStartAt'),
            width: 180,
            formatter: (row: Api.Dashboard.GalaxyRegistryEntryItem) =>
              formatDateTime(row.actual_start_at)
          },
          {
            prop: 'actual_end_at',
            label: t('galaxyRegistry.columns.actualEndAt'),
            width: 180,
            formatter: (row: Api.Dashboard.GalaxyRegistryEntryItem) =>
              formatDateTime(row.actual_end_at)
          },
          {
            prop: 'expected_end_at',
            label: t('galaxyRegistry.columns.expectedEndAt'),
            width: 180,
            formatter: (row: Api.Dashboard.GalaxyRegistryEntryItem) =>
              formatDateTime(row.expected_end_at)
          },
          {
            prop: 'status',
            label: t('galaxyRegistry.columns.status'),
            width: 120,
            formatter: (row: Api.Dashboard.GalaxyRegistryEntryItem) =>
              h(ElTag, { size: 'small', type: statusTagType(row.status) }, () =>
                statusLabel(row.status)
              )
          },
          {
            prop: 'validation_status',
            label: t('galaxyRegistry.columns.validationStatus'),
            width: 120,
            formatter: (row: Api.Dashboard.GalaxyRegistryEntryItem) =>
              h(ElTag, { size: 'small', type: validationTagType(row.validation_status) }, () =>
                validationLabel(row.validation_status)
              )
          },
          {
            prop: 'validated_bounty_amount',
            label: t('galaxyRegistry.columns.validatedBounty'),
            width: 170,
            formatter: (row: Api.Dashboard.GalaxyRegistryEntryItem) =>
              formatIsk(row.validated_bounty_amount)
          },
          {
            prop: 'violation_reason',
            label: t('galaxyRegistry.columns.violationReason'),
            minWidth: 180,
            formatter: (row: Api.Dashboard.GalaxyRegistryEntryItem) =>
              violationReasonLabel(row.violation_reason)
          },
          {
            label: t('common.operation'),
            width: 200,
            fixed: 'right',
            formatter: (row: Api.Dashboard.GalaxyRegistryEntryItem) => {
              if (row.status !== 'active') {
                return '--'
              }
              return h('div', { class: 'flex flex-wrap gap-2' }, [
                h(
                  ElButton,
                  {
                    size: 'small',
                    onClick: () => openExpectedEndDialogFromEntry(row)
                  },
                  () => t('galaxyRegistry.actions.editExpectedEnd')
                ),
                h(
                  ElButton,
                  {
                    size: 'small',
                    type: 'warning',
                    loading: endingEntryId.value === row.id,
                    onClick: () => handleEndEntryById(row.id)
                  },
                  () => t('galaxyRegistry.actions.endEntry')
                )
              ])
            }
          }
        ] as ColumnOption<Api.Dashboard.GalaxyRegistryEntryItem>[]
    },
    transform: {
      responseAdapter: adaptPage
    }
  })

  const adminTable = useTable({
    core: {
      apiFn: fetchAdminGalaxyRegistryEntries,
      apiParams: {
        current: 1,
        size: 200,
        keyword: '',
        status: '',
        validation_status: ''
      },
      immediate: false,
      columnsFactory: () =>
        [
          {
            prop: 'solar_system_name',
            label: t('galaxyRegistry.columns.system'),
            minWidth: 160
          },
          {
            prop: 'captain_character_name',
            label: t('galaxyRegistry.columns.captain'),
            minWidth: 220,
            formatter: (row: Api.Dashboard.GalaxyRegistryEntryItem) =>
              `${row.captain_nickname} (${row.captain_character_name})`
          },
          {
            prop: 'status',
            label: t('galaxyRegistry.columns.status'),
            width: 120,
            formatter: (row: Api.Dashboard.GalaxyRegistryEntryItem) =>
              h(ElTag, { size: 'small', type: statusTagType(row.status) }, () =>
                statusLabel(row.status)
              )
          },
          {
            prop: 'validation_status',
            label: t('galaxyRegistry.columns.validationStatus'),
            width: 120,
            formatter: (row: Api.Dashboard.GalaxyRegistryEntryItem) =>
              h(ElTag, { size: 'small', type: validationTagType(row.validation_status) }, () =>
                validationLabel(row.validation_status)
              )
          },
          {
            prop: 'actual_start_at',
            label: t('galaxyRegistry.columns.actualStartAt'),
            width: 180,
            formatter: (row: Api.Dashboard.GalaxyRegistryEntryItem) =>
              formatDateTime(row.actual_start_at)
          },
          {
            prop: 'actual_end_at',
            label: t('galaxyRegistry.columns.actualEndAt'),
            width: 180,
            formatter: (row: Api.Dashboard.GalaxyRegistryEntryItem) =>
              formatDateTime(row.actual_end_at)
          },
          {
            prop: 'validated_bounty_amount',
            label: t('galaxyRegistry.columns.validatedBounty'),
            width: 170,
            formatter: (row: Api.Dashboard.GalaxyRegistryEntryItem) =>
              formatIsk(row.validated_bounty_amount)
          },
          {
            label: t('common.operation'),
            width: 220,
            fixed: 'right',
            formatter: (row: Api.Dashboard.GalaxyRegistryEntryItem) => {
              const actions = []
              if (row.status === 'active') {
                actions.push(
                  h(
                    ElButton,
                    {
                      size: 'small',
                      type: 'warning',
                      loading: forceEndingEntryId.value === row.id,
                      onClick: () => handleForceEndEntry(row)
                    },
                    () => t('galaxyRegistry.admin.forceEnd')
                  )
                )
              }
              if (row.status === 'completed') {
                actions.push(
                  h(
                    ElButton,
                    {
                      size: 'small',
                      onClick: () => openValidationDialog(row)
                    },
                    () => t('galaxyRegistry.admin.overrideValidation')
                  )
                )
              }
              return actions.length ? h('div', { class: 'flex flex-wrap gap-2' }, actions) : '--'
            }
          }
        ] as ColumnOption<Api.Dashboard.GalaxyRegistryEntryItem>[]
    },
    transform: {
      responseAdapter: adaptPage
    }
  })

  const {
    columns: myColumns,
    columnChecks: myColumnChecks,
    data: myEntries,
    loading: myEntriesLoading,
    pagination: myPagination,
    searchParams: mySearchParams,
    handleSizeChange: handleMySizeChange,
    handleCurrentChange: handleMyCurrentChange,
    getData: loadMyEntries,
    refreshData: refreshMyEntries
  } = myTable

  const {
    columns: adminColumns,
    columnChecks: adminColumnChecks,
    data: adminEntries,
    loading: adminEntriesLoading,
    pagination: adminPagination,
    searchParams: adminSearchParams,
    handleSizeChange: handleAdminSizeChange,
    handleCurrentChange: handleAdminCurrentChange,
    getData: loadAdminEntries,
    refreshData: refreshAdminEntries
  } = adminTable

  const myActiveSystems = computed(() => systems.value.filter((row) => row.active_entry?.is_mine))
  const dirtySystemCount = computed(
    () => adminSystems.value.filter((row) => row.is_dirty || row.is_new).length
  )
  const hasDirtySystems = computed(() => dirtySystemCount.value > 0)

  const loadSystems = async () => {
    loading.systems = true
    try {
      const response = await fetchGalaxyRegistrySystems()
      systems.value = response.items || []
      Object.assign(systemsSummary, response.summary || systemsSummary)
    } finally {
      loading.systems = false
    }
  }

  const loadAdminSystems = async () => {
    if (!canAdminTab.value) return
    loading.adminSystems = true
    try {
      const rows = (await fetchAdminGalaxyRegistrySystems()) || []
      adminSystems.value = rows.map((row) => ({
        ...row,
        local_id: `persisted-${row.id}`,
        is_new: false,
        is_dirty: false
      }))
    } finally {
      loading.adminSystems = false
    }
  }

  const loadAnalytics = async () => {
    if (!canAdminTab.value) return
    loading.analytics = true
    try {
      analytics.value = await fetchAdminGalaxyRegistryAnalytics({
        start_date: analyticsRange.value?.[0],
        end_date: analyticsRange.value?.[1]
      })
    } finally {
      loading.analytics = false
    }
  }

  const searchSdeSystems = async () => {
    if (!adminSystemSearch.keyword.trim()) {
      sdeSystems.value = []
      return
    }
    loading.sdeSearch = true
    try {
      sdeSystems.value = (await searchGalaxyRegistrySdeSystems(adminSystemSearch)) || []
    } finally {
      loading.sdeSearch = false
    }
  }

  const refreshCaptainViews = async () => {
    await Promise.all([loadSystems(), refreshMyEntries()])
  }

  const refreshAdminViews = async () => {
    const tasks: Array<Promise<unknown>> = [
      loadSystems(),
      refreshAdminEntries(),
      loadAnalytics(),
      loadAdminSystems()
    ]
    if (canCaptainTab.value) {
      tasks.push(refreshMyEntries())
    }
    await Promise.all(tasks)
  }

  const openCreateDialog = (row: Api.Dashboard.GalaxyRegistrySystemItem) => {
    selectedSystem.value = row
    createForm.system_config_id = row.system_config_id
    createForm.expected_end_at = ''
    createDialogVisible.value = true
  }

  const handleSubmitCreateEntry = async () => {
    if (!createForm.system_config_id || !createForm.expected_end_at) {
      ElMessage.warning(t('galaxyRegistry.messages.expectedEndAtRequired'))
      return
    }
    creatingEntry.value = true
    try {
      await createGalaxyRegistryEntry(createForm)
      ElMessage.success(t('galaxyRegistry.messages.entryCreated'))
      createDialogVisible.value = false
      await refreshCaptainViews()
    } finally {
      creatingEntry.value = false
    }
  }

  const handleEndEntryById = async (entryId?: number) => {
    if (!entryId) return
    endingEntryId.value = entryId
    try {
      await endGalaxyRegistryEntry(entryId)
      ElMessage.success(t('galaxyRegistry.messages.entryEnded'))
      await refreshCaptainViews()
      if (canAdminTab.value) {
        await Promise.all([refreshAdminEntries(), loadAnalytics()])
      }
    } finally {
      endingEntryId.value = 0
    }
  }

  const openExpectedEndDialogFromSystem = (row: Api.Dashboard.GalaxyRegistrySystemItem) => {
    if (!row.active_entry?.entry_id) return
    expectedEndDialog.entry_id = row.active_entry.entry_id
    expectedEndDialog.system_name = row.solar_system_name
    expectedEndDialog.expected_end_at = row.active_entry.expected_end_at
    expectedEndDialogVisible.value = true
  }

  const openExpectedEndDialogFromEntry = (row: Api.Dashboard.GalaxyRegistryEntryItem) => {
    expectedEndDialog.entry_id = row.id
    expectedEndDialog.system_name = row.solar_system_name
    expectedEndDialog.expected_end_at = row.expected_end_at
    expectedEndDialogVisible.value = true
  }

  const handleSubmitExpectedEndUpdate = async () => {
    if (!expectedEndDialog.entry_id || !expectedEndDialog.expected_end_at) {
      ElMessage.warning(t('galaxyRegistry.messages.expectedEndAtRequired'))
      return
    }
    updatingExpectedEnd.value = true
    try {
      await updateGalaxyRegistryEntryExpectedEndAt(expectedEndDialog.entry_id, {
        expected_end_at: expectedEndDialog.expected_end_at
      })
      ElMessage.success(t('galaxyRegistry.messages.expectedEndUpdated'))
      expectedEndDialogVisible.value = false
      await refreshCaptainViews()
    } finally {
      updatingExpectedEnd.value = false
    }
  }

  const hasSystemDraft = (solarSystemId: number) =>
    adminSystems.value.some((row) => row.solar_system_id === solarSystemId)

  const handleAddSystemDraft = (row: Api.Dashboard.GalaxyRegistrySdeSystem) => {
    if (hasSystemDraft(row.solar_system_id)) {
      return
    }
    adminSystems.value.unshift({
      id: 0,
      solar_system_id: row.solar_system_id,
      solar_system_name: row.solar_system_name,
      region_id: row.region_id,
      region_name: row.region_name,
      constellation_id: row.constellation_id,
      constellation_name: row.constellation_name,
      security: row.security,
      note: '',
      min_bounty_amount: 10000000,
      is_enabled: true,
      created_at: '',
      updated_at: '',
      local_id: `new-${row.solar_system_id}`,
      is_new: true,
      is_dirty: true
    })
  }

  const markSystemDirty = (row: AdminSystemDraft) => {
    row.is_dirty = true
  }

  const handleSaveAllSystems = async () => {
    if (!hasDirtySystems.value) {
      return
    }
    savingAllSystems.value = true
    try {
      for (const row of adminSystems.value) {
        if (row.is_new) {
          await createAdminGalaxyRegistrySystem({
            solar_system_id: row.solar_system_id,
            note: row.note,
            min_bounty_amount: row.min_bounty_amount,
            is_enabled: row.is_enabled
          })
          continue
        }
        if (row.is_dirty) {
          await updateAdminGalaxyRegistrySystem(row.id, {
            note: row.note,
            min_bounty_amount: row.min_bounty_amount,
            is_enabled: row.is_enabled
          })
        }
      }
      ElMessage.success(t('galaxyRegistry.messages.systemSaved'))
      await Promise.all([loadAdminSystems(), loadSystems()])
    } finally {
      savingAllSystems.value = false
    }
  }

  const handleDeleteSystem = async (row: AdminSystemDraft) => {
    if (row.is_new) {
      adminSystems.value = adminSystems.value.filter((item) => item.local_id !== row.local_id)
      return
    }
    await ElMessageBox.confirm(t('galaxyRegistry.messages.deleteConfirm'), t('common.tips'), {
      type: 'warning'
    })
    await deleteAdminGalaxyRegistrySystem(row.id)
    ElMessage.success(t('galaxyRegistry.messages.systemDeleted'))
    await Promise.all([loadAdminSystems(), loadSystems()])
  }

  const handleForceEndEntry = async (row: Api.Dashboard.GalaxyRegistryEntryItem) => {
    forceEndingEntryId.value = row.id
    try {
      await forceEndAdminGalaxyRegistryEntry(row.id)
      ElMessage.success(t('galaxyRegistry.messages.entryForceEnded'))
      await refreshAdminViews()
    } finally {
      forceEndingEntryId.value = 0
    }
  }

  const openValidationDialog = (row: Api.Dashboard.GalaxyRegistryEntryItem) => {
    validationDialog.entry_id = row.id
    validationDialog.system_name = row.solar_system_name
    validationDialog.validation_status =
      row.validation_status === 'violation' ? 'violation' : 'valid'
    validationDialog.violation_reason = row.violation_reason || ''
    validationDialogVisible.value = true
  }

  const handleSubmitValidationUpdate = async () => {
    if (!validationDialog.entry_id) return
    updatingValidation.value = true
    try {
      await updateAdminGalaxyRegistryEntryValidation(validationDialog.entry_id, {
        validation_status: validationDialog.validation_status,
        violation_reason:
          validationDialog.validation_status === 'violation'
            ? validationDialog.violation_reason || undefined
            : undefined
      })
      ElMessage.success(t('galaxyRegistry.messages.validationUpdated'))
      validationDialogVisible.value = false
      await Promise.all([refreshAdminEntries(), loadAnalytics()])
    } finally {
      updatingValidation.value = false
    }
  }

  const canCreateEntry = (row: Api.Dashboard.GalaxyRegistrySystemItem) =>
    canCaptainTab.value && row.is_enabled && row.status === 'idle'
  const canEndEntry = (row: Api.Dashboard.GalaxyRegistrySystemItem) =>
    canCaptainTab.value && !!row.active_entry?.is_mine
  const canEditExpectedEnd = (row: Api.Dashboard.GalaxyRegistrySystemItem) =>
    canCaptainTab.value && !!row.active_entry?.is_mine

  const statusLabel = (status: string) => t(`galaxyRegistry.systemStatus.${status}`)
  const validationLabel = (status: string) => t(`galaxyRegistry.validationStatus.${status}`)
  const violationReasonLabel = (reason: string) =>
    reason ? t(`galaxyRegistry.violationReasons.${reason}`) : '--'

  const statusTagMap: Record<string, TagType> = {
    idle: 'success',
    busy: 'warning',
    overdue: 'danger',
    active: 'warning',
    completed: 'info'
  }
  const validationTagMap: Record<string, TagType> = {
    pending: 'warning',
    valid: 'success',
    violation: 'danger'
  }

  const statusTagType = (status: string): TagType => statusTagMap[status] || 'info'
  const validationTagType = (status: string): TagType => validationTagMap[status] || 'info'

  const formatDateTime = (value?: string | null) => formatTime(value)
  const formatSecurity = (value: number) =>
    typeof value === 'number' && !Number.isNaN(value) ? value.toFixed(1) : '--'
  const formatIsk = (value: number) =>
    new Intl.NumberFormat('en-US', { maximumFractionDigits: 0 }).format(value || 0)

  watch([canCaptainTab, canAdminTab], ([captain, admin]) => {
    if (activeTab.value === 'admin' && !admin) {
      activeTab.value = captain ? 'captain' : 'current'
      return
    }
    if (activeTab.value === 'captain' && !captain) {
      activeTab.value = 'current'
    }
  })

  onMounted(async () => {
    await loadSystems()
    if (canCaptainTab.value) {
      await loadMyEntries()
    }
    if (canAdminTab.value) {
      await Promise.all([loadAdminSystems(), loadAdminEntries(), loadAnalytics()])
    }
  })
</script>

<style scoped lang="scss">
  .galaxy-registry-page {
    &__summary {
      display: grid;
      grid-template-columns: repeat(1, minmax(0, 1fr));
      gap: 12px;
    }

    &__active-list {
      display: grid;
      gap: 12px;
    }

    &__active-card {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 16px;
      padding: 16px;
      border: 1px solid var(--el-border-color-light);
      border-radius: 12px;
      background: var(--el-bg-color-page);
    }
  }

  @media (min-width: 768px) {
    .galaxy-registry-page {
      &__summary {
        grid-template-columns: repeat(3, minmax(0, 1fr));
      }
    }
  }
</style>
