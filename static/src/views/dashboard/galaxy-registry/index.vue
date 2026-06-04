<template>
  <div class="galaxy-registry-page">
    <ElCard shadow="never" class="art-card mb-4">
      <div class="flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
        <div>
          <h2 class="text-lg font-medium">{{ $t('galaxyRegistry.title') }}</h2>
          <p class="text-sm text-g-500">{{ $t('galaxyRegistry.subtitle') }}</p>
        </div>
        <div class="flex flex-wrap gap-2">
          <ElButton :loading="loading.systems" @click="loadSystems">
            {{ $t('common.refresh') }}
          </ElButton>
        </div>
      </div>
    </ElCard>

    <div class="galaxy-registry-page__summary mb-4">
      <ElCard shadow="never" class="art-card">
        <div class="text-sm text-g-500">{{ $t('galaxyRegistry.summary.idle') }}</div>
        <div class="text-2xl font-semibold">{{ systemsSummary.idle_count }}</div>
      </ElCard>
      <ElCard shadow="never" class="art-card">
        <div class="text-sm text-g-500">{{ $t('galaxyRegistry.summary.busy') }}</div>
        <div class="text-2xl font-semibold">{{ systemsSummary.busy_count }}</div>
      </ElCard>
      <ElCard shadow="never" class="art-card">
        <div class="text-sm text-g-500">{{ $t('galaxyRegistry.summary.overdue') }}</div>
        <div class="text-2xl font-semibold">{{ systemsSummary.overdue_count }}</div>
      </ElCard>
    </div>

    <ElCard shadow="never" class="art-card mb-4">
      <template #header>
        <div class="flex items-center justify-between">
          <span>{{ $t('galaxyRegistry.systems.title') }}</span>
        </div>
      </template>

      <ElTable :data="systems" v-loading="loading.systems" stripe border>
        <ElTableColumn
          prop="solar_system_name"
          :label="$t('galaxyRegistry.columns.system')"
          min-width="180"
        >
          <template #default="{ row }">
            <div class="font-medium">{{ row.solar_system_name }}</div>
            <div class="text-xs text-g-500">
              {{ row.region_name }} / {{ row.constellation_name }}
            </div>
          </template>
        </ElTableColumn>
        <ElTableColumn prop="security" :label="$t('galaxyRegistry.columns.security')" width="110">
          <template #default="{ row }">{{ formatSecurity(row.security) }}</template>
        </ElTableColumn>
        <ElTableColumn prop="note" :label="$t('galaxyRegistry.columns.note')" min-width="180" />
        <ElTableColumn :label="$t('galaxyRegistry.columns.minBounty')" width="160">
          <template #default="{ row }">{{ formatIsk(row.min_bounty_amount) }}</template>
        </ElTableColumn>
        <ElTableColumn :label="$t('galaxyRegistry.columns.status')" width="140">
          <template #default="{ row }">
            <ElTag :type="statusTagType(row.status)" effect="plain">
              {{ statusLabel(row.status) }}
            </ElTag>
          </template>
        </ElTableColumn>
        <ElTableColumn :label="$t('galaxyRegistry.columns.captain')" min-width="180">
          <template #default="{ row }">
            <span v-if="row.active_entry">
              {{ row.active_entry.captain_nickname }}
              <span class="text-g-500">({{ row.active_entry.captain_character_name }})</span>
            </span>
            <span v-else>--</span>
          </template>
        </ElTableColumn>
        <ElTableColumn :label="$t('galaxyRegistry.columns.expectedEndAt')" width="180">
          <template #default="{ row }">
            {{ formatDateTime(row.active_entry?.expected_end_at) }}
          </template>
        </ElTableColumn>
        <ElTableColumn :label="$t('common.operation')" width="220" fixed="right">
          <template #default="{ row }">
            <div class="flex flex-wrap gap-2">
              <ElButton
                v-if="canCreateEntry(row)"
                type="primary"
                size="small"
                @click="openCreateDialog(row)"
              >
                {{ $t('galaxyRegistry.actions.createEntry') }}
              </ElButton>
              <ElButton
                v-if="canEndEntry(row)"
                type="warning"
                size="small"
                :loading="endingEntryId === row.active_entry?.entry_id"
                @click="handleEndEntry(row)"
              >
                {{ $t('galaxyRegistry.actions.endEntry') }}
              </ElButton>
            </div>
          </template>
        </ElTableColumn>
      </ElTable>
    </ElCard>

    <ElCard v-if="isCaptain" shadow="never" class="art-card mb-4">
      <template #header>
        <div class="flex items-center justify-between">
          <span>{{ $t('galaxyRegistry.myEntries.title') }}</span>
          <div class="flex flex-wrap gap-2">
            <ElSelect
              v-model="myFilters.status"
              clearable
              class="w-[140px]"
              @change="loadMyEntries"
            >
              <ElOption value="active" :label="$t('galaxyRegistry.entryStatus.active')" />
              <ElOption value="completed" :label="$t('galaxyRegistry.entryStatus.completed')" />
            </ElSelect>
            <ElSelect
              v-model="myFilters.validation_status"
              clearable
              class="w-[140px]"
              @change="loadMyEntries"
            >
              <ElOption value="pending" :label="$t('galaxyRegistry.validationStatus.pending')" />
              <ElOption value="valid" :label="$t('galaxyRegistry.validationStatus.valid')" />
              <ElOption
                value="violation"
                :label="$t('galaxyRegistry.validationStatus.violation')"
              />
            </ElSelect>
          </div>
        </div>
      </template>

      <ElTable :data="myEntries" v-loading="loading.myEntries" stripe border>
        <ElTableColumn
          prop="solar_system_name"
          :label="$t('galaxyRegistry.columns.system')"
          min-width="160"
        />
        <ElTableColumn :label="$t('galaxyRegistry.columns.actualStartAt')" width="180">
          <template #default="{ row }">{{ formatDateTime(row.actual_start_at) }}</template>
        </ElTableColumn>
        <ElTableColumn :label="$t('galaxyRegistry.columns.actualEndAt')" width="180">
          <template #default="{ row }">{{ formatDateTime(row.actual_end_at) }}</template>
        </ElTableColumn>
        <ElTableColumn :label="$t('galaxyRegistry.columns.expectedEndAt')" width="180">
          <template #default="{ row }">{{ formatDateTime(row.expected_end_at) }}</template>
        </ElTableColumn>
        <ElTableColumn :label="$t('galaxyRegistry.columns.status')" width="120">
          <template #default="{ row }">
            <ElTag size="small" :type="statusTagType(row.status)">{{
              statusLabel(row.status)
            }}</ElTag>
          </template>
        </ElTableColumn>
        <ElTableColumn :label="$t('galaxyRegistry.columns.validationStatus')" width="120">
          <template #default="{ row }">
            <ElTag size="small" :type="validationTagType(row.validation_status)">
              {{ validationLabel(row.validation_status) }}
            </ElTag>
          </template>
        </ElTableColumn>
        <ElTableColumn :label="$t('galaxyRegistry.columns.validatedBounty')" width="160">
          <template #default="{ row }">{{ formatIsk(row.validated_bounty_amount) }}</template>
        </ElTableColumn>
        <ElTableColumn
          prop="violation_reason"
          :label="$t('galaxyRegistry.columns.violationReason')"
          min-width="160"
        >
          <template #default="{ row }">{{ violationReasonLabel(row.violation_reason) }}</template>
        </ElTableColumn>
      </ElTable>
    </ElCard>

    <ElCard v-if="isAdmin" shadow="never" class="art-card">
      <ElTabs v-model="adminTab">
        <ElTabPane :label="$t('galaxyRegistry.admin.systemsTab')" name="systems">
          <div class="flex flex-wrap gap-2 mb-4">
            <ElInput
              v-model="adminSystemSearch.keyword"
              class="w-[260px]"
              :placeholder="$t('galaxyRegistry.admin.searchPlaceholder')"
              @keyup.enter="searchSdeSystems"
            />
            <ElButton type="primary" :loading="loading.sdeSearch" @click="searchSdeSystems">
              {{ $t('common.search') }}
            </ElButton>
          </div>

          <ElTable :data="sdeSystems" v-loading="loading.sdeSearch" stripe border class="mb-4">
            <ElTableColumn
              prop="solar_system_name"
              :label="$t('galaxyRegistry.columns.system')"
              min-width="160"
            />
            <ElTableColumn :label="$t('galaxyRegistry.columns.security')" width="100">
              <template #default="{ row }">{{ formatSecurity(row.security) }}</template>
            </ElTableColumn>
            <ElTableColumn
              :label="$t('galaxyRegistry.columns.regionConstellation')"
              min-width="220"
            >
              <template #default="{ row }"
                >{{ row.region_name }} / {{ row.constellation_name }}</template
              >
            </ElTableColumn>
            <ElTableColumn :label="$t('common.operation')" width="120">
              <template #default="{ row }">
                <ElButton size="small" type="primary" @click="handleCreateSystem(row)">
                  {{ $t('galaxyRegistry.admin.addSystem') }}
                </ElButton>
              </template>
            </ElTableColumn>
          </ElTable>

          <ElTable :data="adminSystems" v-loading="loading.adminSystems" stripe border>
            <ElTableColumn
              prop="solar_system_name"
              :label="$t('galaxyRegistry.columns.system')"
              min-width="160"
            />
            <ElTableColumn
              :label="$t('galaxyRegistry.columns.regionConstellation')"
              min-width="220"
            >
              <template #default="{ row }"
                >{{ row.region_name }} / {{ row.constellation_name }}</template
              >
            </ElTableColumn>
            <ElTableColumn :label="$t('galaxyRegistry.columns.note')" min-width="180">
              <template #default="{ row }">
                <ElInput v-model="row.note" />
              </template>
            </ElTableColumn>
            <ElTableColumn :label="$t('galaxyRegistry.columns.minBounty')" width="180">
              <template #default="{ row }">
                <ElInputNumber
                  v-model="row.min_bounty_amount"
                  :min="0"
                  :step="1000000"
                  class="w-full"
                />
              </template>
            </ElTableColumn>
            <ElTableColumn :label="$t('galaxyRegistry.columns.enabled')" width="120">
              <template #default="{ row }">
                <ElSwitch v-model="row.is_enabled" />
              </template>
            </ElTableColumn>
            <ElTableColumn :label="$t('common.operation')" width="180" fixed="right">
              <template #default="{ row }">
                <div class="flex gap-2">
                  <ElButton
                    size="small"
                    type="primary"
                    :loading="savingSystemId === row.id"
                    @click="handleSaveSystem(row)"
                  >
                    {{ $t('common.save') }}
                  </ElButton>
                  <ElButton
                    size="small"
                    type="danger"
                    :loading="deletingSystemId === row.id"
                    @click="handleDeleteSystem(row)"
                  >
                    {{ $t('common.delete') }}
                  </ElButton>
                </div>
              </template>
            </ElTableColumn>
          </ElTable>
        </ElTabPane>

        <ElTabPane :label="$t('galaxyRegistry.admin.entriesTab')" name="entries">
          <div class="flex flex-wrap gap-2 mb-4">
            <ElInput
              v-model="adminEntryFilters.keyword"
              class="w-[220px]"
              :placeholder="$t('galaxyRegistry.admin.entryKeywordPlaceholder')"
              @keyup.enter="loadAdminEntries"
            />
            <ElSelect
              v-model="adminEntryFilters.status"
              clearable
              class="w-[140px]"
              @change="loadAdminEntries"
            >
              <ElOption value="active" :label="$t('galaxyRegistry.entryStatus.active')" />
              <ElOption value="completed" :label="$t('galaxyRegistry.entryStatus.completed')" />
            </ElSelect>
            <ElSelect
              v-model="adminEntryFilters.validation_status"
              clearable
              class="w-[140px]"
              @change="loadAdminEntries"
            >
              <ElOption value="pending" :label="$t('galaxyRegistry.validationStatus.pending')" />
              <ElOption value="valid" :label="$t('galaxyRegistry.validationStatus.valid')" />
              <ElOption
                value="violation"
                :label="$t('galaxyRegistry.validationStatus.violation')"
              />
            </ElSelect>
            <ElButton type="primary" @click="loadAdminEntries">{{ $t('common.search') }}</ElButton>
          </div>

          <ElTable :data="adminEntries" v-loading="loading.adminEntries" stripe border>
            <ElTableColumn
              prop="solar_system_name"
              :label="$t('galaxyRegistry.columns.system')"
              min-width="160"
            />
            <ElTableColumn :label="$t('galaxyRegistry.columns.captain')" min-width="180">
              <template #default="{ row }">
                {{ row.captain_nickname }} ({{ row.captain_character_name }})
              </template>
            </ElTableColumn>
            <ElTableColumn :label="$t('galaxyRegistry.columns.status')" width="120">
              <template #default="{ row }">
                <ElTag size="small" :type="statusTagType(row.status)">{{
                  statusLabel(row.status)
                }}</ElTag>
              </template>
            </ElTableColumn>
            <ElTableColumn :label="$t('galaxyRegistry.columns.validationStatus')" width="120">
              <template #default="{ row }">
                <ElTag size="small" :type="validationTagType(row.validation_status)">
                  {{ validationLabel(row.validation_status) }}
                </ElTag>
              </template>
            </ElTableColumn>
            <ElTableColumn :label="$t('galaxyRegistry.columns.actualStartAt')" width="180">
              <template #default="{ row }">{{ formatDateTime(row.actual_start_at) }}</template>
            </ElTableColumn>
            <ElTableColumn :label="$t('galaxyRegistry.columns.actualEndAt')" width="180">
              <template #default="{ row }">{{ formatDateTime(row.actual_end_at) }}</template>
            </ElTableColumn>
            <ElTableColumn :label="$t('galaxyRegistry.columns.validatedBounty')" width="160">
              <template #default="{ row }">{{ formatIsk(row.validated_bounty_amount) }}</template>
            </ElTableColumn>
            <ElTableColumn :label="$t('common.operation')" width="140" fixed="right">
              <template #default="{ row }">
                <ElButton
                  v-if="row.status === 'active'"
                  size="small"
                  type="warning"
                  :loading="forceEndingEntryId === row.id"
                  @click="handleForceEndEntry(row)"
                >
                  {{ $t('galaxyRegistry.admin.forceEnd') }}
                </ElButton>
              </template>
            </ElTableColumn>
          </ElTable>
        </ElTabPane>

        <ElTabPane :label="$t('galaxyRegistry.admin.analyticsTab')" name="analytics">
          <div class="flex flex-wrap gap-2 mb-4">
            <ElDatePicker
              v-model="analyticsRange"
              type="daterange"
              value-format="YYYY-MM-DD"
              :range-separator="$t('galaxyRegistry.admin.rangeSeparator')"
              :start-placeholder="$t('galaxyRegistry.admin.rangeStart')"
              :end-placeholder="$t('galaxyRegistry.admin.rangeEnd')"
            />
            <ElButton type="primary" :loading="loading.analytics" @click="loadAnalytics">
              {{ $t('common.search') }}
            </ElButton>
          </div>

          <div class="galaxy-registry-page__summary mb-4" v-if="analytics">
            <ElCard shadow="never" class="art-card">
              <div class="text-sm text-g-500">{{ $t('galaxyRegistry.summary.idle') }}</div>
              <div class="text-2xl font-semibold">{{ analytics.current_snapshot.idle_count }}</div>
            </ElCard>
            <ElCard shadow="never" class="art-card">
              <div class="text-sm text-g-500">{{ $t('galaxyRegistry.summary.busy') }}</div>
              <div class="text-2xl font-semibold">{{ analytics.current_snapshot.busy_count }}</div>
            </ElCard>
            <ElCard shadow="never" class="art-card">
              <div class="text-sm text-g-500">{{ $t('galaxyRegistry.summary.overdue') }}</div>
              <div class="text-2xl font-semibold">{{
                analytics.current_snapshot.overdue_count
              }}</div>
            </ElCard>
          </div>

          <ElTable v-if="analytics" :data="analytics.top_systems" stripe border class="mb-4">
            <ElTableColumn
              prop="solar_system_name"
              :label="$t('galaxyRegistry.admin.topSystems')"
              min-width="220"
            />
            <ElTableColumn
              prop="register_count"
              :label="$t('galaxyRegistry.admin.registerCount')"
              width="140"
            />
          </ElTable>

          <ElTable v-if="analytics" :data="analytics.recent_violations" stripe border>
            <ElTableColumn
              prop="solar_system_name"
              :label="$t('galaxyRegistry.columns.system')"
              min-width="160"
            />
            <ElTableColumn :label="$t('galaxyRegistry.columns.captain')" min-width="180">
              <template #default="{ row }">
                {{ row.captain_nickname }} ({{ row.captain_character_name }})
              </template>
            </ElTableColumn>
            <ElTableColumn
              prop="violation_reason"
              :label="$t('galaxyRegistry.columns.violationReason')"
              min-width="180"
            >
              <template #default="{ row }">{{
                violationReasonLabel(row.violation_reason)
              }}</template>
            </ElTableColumn>
            <ElTableColumn :label="$t('galaxyRegistry.columns.actualEndAt')" width="180">
              <template #default="{ row }">{{ formatDateTime(row.actual_end_at) }}</template>
            </ElTableColumn>
          </ElTable>
        </ElTabPane>
      </ElTabs>
    </ElCard>

    <ElDialog
      v-model="createDialogVisible"
      :title="$t('galaxyRegistry.actions.createEntry')"
      width="420px"
    >
      <ElForm label-position="top">
        <ElFormItem :label="$t('galaxyRegistry.columns.system')">
          <ElInput :model-value="selectedSystem?.solar_system_name || ''" disabled />
        </ElFormItem>
        <ElFormItem :label="$t('galaxyRegistry.columns.expectedEndAt')">
          <ElDatePicker
            v-model="createForm.expected_end_at"
            type="datetime"
            value-format="YYYY-MM-DD HH:mm:ss"
            class="w-full"
          />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="createDialogVisible = false">{{ $t('common.cancel') }}</ElButton>
        <ElButton type="primary" :loading="creatingEntry" @click="handleSubmitCreateEntry">
          {{ $t('common.confirm') }}
        </ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, reactive, ref } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import { useI18n } from 'vue-i18n'
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
    updateAdminGalaxyRegistrySystem
  } from '@/api/galaxy-registry'

  type TagType = 'primary' | 'success' | 'warning' | 'info' | 'danger'

  const { t } = useI18n()
  const userStore = useUserStore()

  const roles = computed(() => userStore.info.roles || [])
  const isCaptain = computed(
    () =>
      roles.value.includes('captain') ||
      roles.value.includes('admin') ||
      roles.value.includes('super_admin')
  )
  const isAdmin = computed(
    () => roles.value.includes('admin') || roles.value.includes('super_admin')
  )

  const loading = reactive({
    systems: false,
    myEntries: false,
    adminSystems: false,
    adminEntries: false,
    analytics: false,
    sdeSearch: false
  })

  const systems = ref<Api.Dashboard.GalaxyRegistrySystemItem[]>([])
  const systemsSummary = reactive<Api.Dashboard.GalaxyRegistrySystemSummary>({
    idle_count: 0,
    busy_count: 0,
    overdue_count: 0
  })

  const myEntries = ref<Api.Dashboard.GalaxyRegistryEntryItem[]>([])
  const adminSystems = ref<Api.Dashboard.GalaxyRegistryAdminSystem[]>([])
  const adminEntries = ref<Api.Dashboard.GalaxyRegistryEntryItem[]>([])
  const analytics = ref<Api.Dashboard.GalaxyRegistryAdminAnalytics | null>(null)
  const sdeSystems = ref<Api.Dashboard.GalaxyRegistrySdeSystem[]>([])
  const adminTab = ref('systems')
  const analyticsRange = ref<[string, string] | null>(null)

  const myFilters = reactive<Api.Dashboard.GalaxyRegistryEntryListParams>({
    current: 1,
    size: 20,
    status: '',
    validation_status: ''
  })
  const adminEntryFilters = reactive<Api.Dashboard.GalaxyRegistryEntryListParams>({
    current: 1,
    size: 20,
    keyword: '',
    status: '',
    validation_status: ''
  })
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

  const creatingEntry = ref(false)
  const endingEntryId = ref<number>(0)
  const savingSystemId = ref<number>(0)
  const deletingSystemId = ref<number>(0)
  const forceEndingEntryId = ref<number>(0)

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

  const loadMyEntries = async () => {
    if (!isCaptain.value) return
    loading.myEntries = true
    try {
      const response = await fetchMyGalaxyRegistryEntries(myFilters)
      myEntries.value = response.list || []
    } finally {
      loading.myEntries = false
    }
  }

  const loadAdminSystems = async () => {
    if (!isAdmin.value) return
    loading.adminSystems = true
    try {
      adminSystems.value = (await fetchAdminGalaxyRegistrySystems()) || []
    } finally {
      loading.adminSystems = false
    }
  }

  const loadAdminEntries = async () => {
    if (!isAdmin.value) return
    loading.adminEntries = true
    try {
      const response = await fetchAdminGalaxyRegistryEntries(adminEntryFilters)
      adminEntries.value = response.list || []
    } finally {
      loading.adminEntries = false
    }
  }

  const loadAnalytics = async () => {
    if (!isAdmin.value) return
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
      await Promise.all([loadSystems(), loadMyEntries()])
    } finally {
      creatingEntry.value = false
    }
  }

  const handleEndEntry = async (row: Api.Dashboard.GalaxyRegistrySystemItem) => {
    const entryId = row.active_entry?.entry_id
    if (!entryId) return
    endingEntryId.value = entryId
    try {
      await endGalaxyRegistryEntry(entryId)
      ElMessage.success(t('galaxyRegistry.messages.entryEnded'))
      await Promise.all([loadSystems(), loadMyEntries()])
    } finally {
      endingEntryId.value = 0
    }
  }

  const handleCreateSystem = async (row: Api.Dashboard.GalaxyRegistrySdeSystem) => {
    await createAdminGalaxyRegistrySystem({
      solar_system_id: row.solar_system_id,
      note: '',
      min_bounty_amount: 10000000,
      is_enabled: true
    })
    ElMessage.success(t('galaxyRegistry.messages.systemCreated'))
    await loadAdminSystems()
  }

  const handleSaveSystem = async (row: Api.Dashboard.GalaxyRegistryAdminSystem) => {
    savingSystemId.value = row.id
    try {
      await updateAdminGalaxyRegistrySystem(row.id, {
        note: row.note,
        min_bounty_amount: row.min_bounty_amount,
        is_enabled: row.is_enabled
      })
      ElMessage.success(t('galaxyRegistry.messages.systemSaved'))
      await Promise.all([loadAdminSystems(), loadSystems()])
    } finally {
      savingSystemId.value = 0
    }
  }

  const handleDeleteSystem = async (row: Api.Dashboard.GalaxyRegistryAdminSystem) => {
    await ElMessageBox.confirm(t('galaxyRegistry.messages.deleteConfirm'), t('common.tips'), {
      type: 'warning'
    })
    deletingSystemId.value = row.id
    try {
      await deleteAdminGalaxyRegistrySystem(row.id)
      ElMessage.success(t('galaxyRegistry.messages.systemDeleted'))
      await Promise.all([loadAdminSystems(), loadSystems()])
    } finally {
      deletingSystemId.value = 0
    }
  }

  const handleForceEndEntry = async (row: Api.Dashboard.GalaxyRegistryEntryItem) => {
    forceEndingEntryId.value = row.id
    try {
      await forceEndAdminGalaxyRegistryEntry(row.id)
      ElMessage.success(t('galaxyRegistry.messages.entryForceEnded'))
      await Promise.all([loadSystems(), loadAdminEntries(), loadMyEntries(), loadAnalytics()])
    } finally {
      forceEndingEntryId.value = 0
    }
  }

  const canCreateEntry = (row: Api.Dashboard.GalaxyRegistrySystemItem) =>
    isCaptain.value && row.is_enabled && row.status === 'idle'

  const canEndEntry = (row: Api.Dashboard.GalaxyRegistrySystemItem) =>
    isCaptain.value && row.active_entry?.is_mine

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

  onMounted(async () => {
    await loadSystems()
    await Promise.all([loadMyEntries(), loadAdminSystems(), loadAdminEntries(), loadAnalytics()])
  })
</script>

<style scoped lang="scss">
  .galaxy-registry-page {
    &__summary {
      display: grid;
      grid-template-columns: repeat(1, minmax(0, 1fr));
      gap: 12px;
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
