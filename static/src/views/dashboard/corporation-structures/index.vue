<template>
  <div class="corporation-structures-page">
    <ElCard shadow="never" class="art-card mb-4">
      <div class="flex flex-col gap-1">
        <h2 class="text-lg font-medium">{{ $t('corporationStructures.title') }}</h2>
        <p class="text-sm text-g-500">{{ $t('corporationStructures.subtitle') }}</p>
      </div>
    </ElCard>

    <ElTabs v-model="activeTab" @tab-change="handleTabChange">
      <ElTabPane :label="$t('corporationStructures.tabs.list')" name="list">
        <div class="corporation-structures-page__list-view">
          <ElCard shadow="never" class="art-card mb-4 corporation-structures-page__list-toolbar">
            <div class="flex flex-wrap items-center gap-3">
              <ElButton type="primary" plain @click="openFilterDrawer">
                {{ $t('corporationStructures.actions.openFilters') }}
              </ElButton>
              <ElButton
                type="primary"
                :loading="
                  runningTaskCorpId === filters.corporation_id && filters.corporation_id > 0
                "
                :disabled="filters.corporation_id <= 0"
                @click="handleRunTaskForSelectedCorporation"
              >
                {{ $t('corporationStructures.actions.refreshSelected') }}
              </ElButton>
            </div>
          </ElCard>

          <ElDrawer
            v-model="filterDrawerVisible"
            :title="$t('corporationStructures.filters.title')"
            direction="rtl"
            :size="filterDrawerSize"
            class="corporation-structures-page__filter-drawer"
            destroy-on-close
          >
            <div class="corporation-structures-page__filter-drawer-body">
              <div
                class="corporation-structures-page__filter-drawer-content grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-3"
              >
                <ElFormItem :label="$t('corporationStructures.filters.corporation')" class="mb-0">
                  <ElSelect
                    v-model="filters.corporation_id"
                    filterable
                    clearable
                    class="w-full"
                    @change="handleCorporationFilterChange"
                    @clear="handleCorporationFilterChange"
                  >
                    <ElOption :label="$t('corporationStructures.allCorporations')" :value="0" />
                    <ElOption
                      v-for="corp in validCorporations"
                      :key="corp.corporation_id"
                      :label="`${corp.corporation_name} (${corp.corporation_id})`"
                      :value="corp.corporation_id"
                    />
                  </ElSelect>
                </ElFormItem>

                <ElFormItem :label="$t('corporationStructures.filters.keyword')" class="mb-0">
                  <ElInput
                    v-model="filters.keyword"
                    clearable
                    :placeholder="$t('corporationStructures.placeholders.keyword')"
                  />
                </ElFormItem>

                <ElFormItem :label="$t('corporationStructures.filters.regions')" class="mb-0">
                  <ElSelect
                    v-model="filters.region_ids"
                    multiple
                    filterable
                    clearable
                    collapse-tags
                    collapse-tags-tooltip
                    class="w-full"
                  >
                    <ElOption
                      v-for="item in filterOptions.regions"
                      :key="item.region_id"
                      :label="item.region_name"
                      :value="item.region_id"
                    />
                  </ElSelect>
                </ElFormItem>

                <ElFormItem :label="$t('corporationStructures.filters.systems')" class="mb-0">
                  <ElSelect
                    v-model="filters.system_ids"
                    multiple
                    filterable
                    clearable
                    collapse-tags
                    collapse-tags-tooltip
                    class="w-full"
                  >
                    <ElOption
                      v-for="item in filterOptions.systems"
                      :key="item.system_id"
                      :label="formatSystemOption(item)"
                      :value="item.system_id"
                    />
                  </ElSelect>
                </ElFormItem>

                <ElFormItem
                  :label="$t('corporationStructures.filters.stateGroups')"
                  class="mb-0 md:col-span-2"
                >
                  <ElCheckboxGroup v-model="filters.state_groups">
                    <ElCheckboxButton value="online">{{
                      $t('corporationStructures.stateGroups.online')
                    }}</ElCheckboxButton>
                    <ElCheckboxButton value="low_power">{{
                      $t('corporationStructures.stateGroups.lowPower')
                    }}</ElCheckboxButton>
                    <ElCheckboxButton value="abandoned">{{
                      $t('corporationStructures.stateGroups.abandoned')
                    }}</ElCheckboxButton>
                    <ElCheckboxButton value="reinforced">{{
                      $t('corporationStructures.stateGroups.reinforced')
                    }}</ElCheckboxButton>
                  </ElCheckboxGroup>
                </ElFormItem>

                <ElFormItem
                  :label="$t('corporationStructures.filters.fuel')"
                  class="mb-0 md:col-span-2 xl:col-span-3"
                >
                  <div class="flex flex-wrap items-center gap-2">
                    <ElRadioGroup v-model="filters.fuel_bucket">
                      <ElRadioButton value="all">{{
                        $t('corporationStructures.fuelBuckets.all')
                      }}</ElRadioButton>
                      <ElRadioButton value="lt_24h">{{
                        $t('corporationStructures.fuelBuckets.lt24h')
                      }}</ElRadioButton>
                      <ElRadioButton value="lt_72h">{{
                        $t('corporationStructures.fuelBuckets.lt3d')
                      }}</ElRadioButton>
                      <ElRadioButton value="lt_168h">{{
                        $t('corporationStructures.fuelBuckets.lt7d')
                      }}</ElRadioButton>
                      <ElRadioButton value="custom">{{
                        $t('corporationStructures.fuelBuckets.custom')
                      }}</ElRadioButton>
                    </ElRadioGroup>
                    <template v-if="filters.fuel_bucket === 'custom'">
                      <ElInputNumber v-model="filters.fuel_min_hours" :min="0" :step="1" />
                      <span>~</span>
                      <ElInputNumber v-model="filters.fuel_max_hours" :min="0" :step="1" />
                    </template>
                  </div>
                </ElFormItem>

                <ElFormItem
                  :label="$t('corporationStructures.filters.security')"
                  class="mb-0 md:col-span-2"
                >
                  <div class="flex flex-wrap items-center gap-2">
                    <ElCheckboxGroup v-model="filters.security_bands">
                      <ElCheckboxButton value="highsec">{{
                        $t('corporationStructures.securityBands.highsec')
                      }}</ElCheckboxButton>
                      <ElCheckboxButton value="lowsec">{{
                        $t('corporationStructures.securityBands.lowsec')
                      }}</ElCheckboxButton>
                      <ElCheckboxButton value="nullsec">{{
                        $t('corporationStructures.securityBands.nullsec')
                      }}</ElCheckboxButton>
                    </ElCheckboxGroup>
                    <ElInputNumber
                      v-model="filters.security_min"
                      :min="-1"
                      :max="1"
                      :step="0.1"
                      :precision="1"
                    />
                    <span>~</span>
                    <ElInputNumber
                      v-model="filters.security_max"
                      :min="-1"
                      :max="1"
                      :step="0.1"
                      :precision="1"
                    />
                  </div>
                </ElFormItem>

                <ElFormItem :label="$t('corporationStructures.filters.types')" class="mb-0">
                  <ElSelect
                    v-model="filters.type_ids"
                    multiple
                    filterable
                    clearable
                    collapse-tags
                    collapse-tags-tooltip
                    class="w-full"
                  >
                    <ElOption
                      v-for="item in filterOptions.types"
                      :key="item.type_id"
                      :label="item.type_name"
                      :value="item.type_id"
                    />
                  </ElSelect>
                </ElFormItem>

                <ElFormItem
                  :label="$t('corporationStructures.filters.services')"
                  class="mb-0 md:col-span-2"
                >
                  <div class="flex flex-wrap items-center gap-2 w-full">
                    <ElSelect
                      v-model="filters.service_names"
                      multiple
                      filterable
                      clearable
                      class="w-full md:w-[360px]"
                    >
                      <ElOption
                        v-for="item in filterOptions.services"
                        :key="item.name"
                        :label="item.name"
                        :value="item.name"
                      />
                    </ElSelect>
                    <ElRadioGroup v-model="filters.service_match_mode">
                      <ElRadioButton value="and">{{
                        $t('corporationStructures.serviceMatch.and')
                      }}</ElRadioButton>
                      <ElRadioButton value="or">{{
                        $t('corporationStructures.serviceMatch.or')
                      }}</ElRadioButton>
                    </ElRadioGroup>
                  </div>
                </ElFormItem>

                <ElFormItem
                  :label="$t('corporationStructures.filters.timer')"
                  class="mb-0 md:col-span-2 xl:col-span-3"
                >
                  <div class="flex flex-wrap items-center gap-2">
                    <ElRadioGroup v-model="filters.timer_bucket">
                      <ElRadioButton value="all">{{
                        $t('corporationStructures.timerBuckets.all')
                      }}</ElRadioButton>
                      <ElRadioButton value="current_hour">{{
                        $t('corporationStructures.timerBuckets.currentHour')
                      }}</ElRadioButton>
                      <ElRadioButton value="next_2_hours">{{
                        $t('corporationStructures.timerBuckets.next2Hours')
                      }}</ElRadioButton>
                      <ElRadioButton value="custom">{{
                        $t('corporationStructures.timerBuckets.custom')
                      }}</ElRadioButton>
                    </ElRadioGroup>
                    <ElDatePicker
                      v-if="filters.timer_bucket === 'custom'"
                      v-model="timerRange"
                      type="datetimerange"
                      class="w-[380px]"
                      value-format="YYYY-MM-DDTHH:mm:ss"
                    />
                  </div>
                </ElFormItem>
              </div>

              <div
                class="corporation-structures-page__filter-drawer-actions flex flex-wrap items-center gap-3 mt-4"
              >
                <ElButton type="primary" :loading="loading" @click="handleSearchFromDrawer">
                  {{ $t('common.search') }}
                </ElButton>
                <ElButton @click="handleReset">{{ $t('common.reset') }}</ElButton>
              </div>
            </div>
          </ElDrawer>

          <ElCard shadow="never" class="art-table-card corporation-structures-page__list-card">
            <ArtTableHeader
              v-model:columns="columnChecks"
              :loading="loading"
              @refresh="refreshData"
            />
            <ArtTable
              :loading="loading"
              :data="data"
              :columns="columns"
              :pagination="pagination"
              :default-sort="{ prop: 'fuel_remaining_hours', order: 'ascending' }"
              :empty-text="$t('corporationStructures.empty.list')"
              @sort-change="handleSortChange"
              @pagination:size-change="handleSizeChange"
              @pagination:current-change="handleCurrentChange"
            />
          </ElCard>
          <StructureServicesDialog
            v-model:visible="servicesDialogVisible"
            :row="servicesDialogRow"
          />
        </div>
      </ElTabPane>

      <ElTabPane :label="$t('corporationStructures.tabs.settings')" name="settings">
        <ElCard shadow="never" class="art-table-card">
          <ElFormItem :label="$t('corporationStructures.settings.noticeThresholds')" class="mb-4">
            <div class="flex flex-wrap items-center gap-4">
              <div class="flex items-center gap-2">
                <span class="text-sm text-g-500">
                  {{ $t('corporationStructures.settings.fuelNoticeThreshold') }}
                </span>
                <ElInputNumber
                  v-model="noticeThresholds.fuel_notice_threshold_days"
                  :min="0"
                  :step="1"
                  step-strictly
                />
                <span class="text-sm text-g-500">{{
                  $t('corporationStructures.settings.daysUnit')
                }}</span>
              </div>
              <div class="flex items-center gap-2">
                <span class="text-sm text-g-500">
                  {{ $t('corporationStructures.settings.alertEnabled') }}
                </span>
                <ElSwitch v-model="alertEnabled" />
              </div>
              <div class="flex items-start gap-2">
                <span class="pt-2 text-sm text-g-500">
                  {{ $t('corporationStructures.settings.alertGroupIDs') }}
                </span>
                <ElInput
                  v-model="alertGroupIDsText"
                  type="textarea"
                  :rows="3"
                  class="w-72"
                  :disabled="!alertEnabled"
                  :placeholder="$t('corporationStructures.settings.alertGroupIDsPlaceholder')"
                />
                <span class="text-xs text-g-500">
                  {{ $t('corporationStructures.settings.alertGroupIDsHint') }}
                </span>
              </div>
              <div class="flex items-center gap-2">
                <span class="text-sm text-g-500">
                  {{ $t('corporationStructures.settings.timerNoticeThreshold') }}
                </span>
                <ElInputNumber
                  v-model="noticeThresholds.timer_notice_threshold_days"
                  :min="0"
                  :step="1"
                  step-strictly
                />
                <span class="text-sm text-g-500">{{
                  $t('corporationStructures.settings.daysUnit')
                }}</span>
              </div>
              <span class="text-xs text-g-500">
                {{ $t('corporationStructures.settings.noticeThresholdHint') }}
              </span>
            </div>
          </ElFormItem>

          <div class="flex flex-wrap items-center gap-3 mb-4">
            <ElButton :loading="settingsLoading" @click="loadSettings">
              {{ $t('common.refresh') }}
            </ElButton>
            <ElButton type="primary" :loading="savingAuthorizations" @click="saveAuthorizations">
              {{ $t('common.save') }}
            </ElButton>
            <ElButton
              v-if="canRunAlertScan"
              :loading="alertScanRunning"
              :disabled="!settings.alert_enabled"
              @click="runAlertScan"
            >
              {{ $t('corporationStructures.actions.runAlertScan') }}
            </ElButton>
          </div>

          <ElTable v-loading="settingsLoading" :data="settings.corporations" stripe border>
            <ElTableColumn :label="$t('corporationStructures.table.corporation')" min-width="260">
              <template #default="{ row }">
                <div class="font-medium">{{ row.corporation_name }}</div>
                <div class="text-xs text-g-500">{{ row.corporation_id }}</div>
              </template>
            </ElTableColumn>
            <ElTableColumn
              :label="$t('corporationStructures.table.directorCharacter')"
              min-width="320"
            >
              <template #default="{ row }">
                <ElSelect
                  v-model="authorizationByCorp[row.corporation_id]"
                  clearable
                  :placeholder="$t('corporationStructures.placeholders.selectDirector')"
                  class="w-full"
                  @clear="authorizationByCorp[row.corporation_id] = 0"
                >
                  <ElOption :label="$t('corporationStructures.options.disabled')" :value="0" />
                  <ElOption
                    v-for="option in row.director_characters"
                    :key="option.character_id"
                    :label="`${option.character_name} (${option.character_id})`"
                    :value="option.character_id"
                  />
                </ElSelect>
              </template>
            </ElTableColumn>
          </ElTable>

          <ElEmpty
            v-if="!settingsLoading && settings.corporations.length === 0"
            :description="$t('corporationStructures.empty.settings')"
            class="mt-4"
          />

          <ElDivider>{{ $t('corporationStructures.serviceCatalog.title') }}</ElDivider>
          <p class="mb-3 text-sm text-gray-500">{{
            $t('corporationStructures.serviceCatalog.hint')
          }}</p>
          <ElEmpty
            v-if="serviceCatalog.unmapped_activities.length === 0"
            :description="$t('corporationStructures.serviceCatalog.empty')"
            :image-size="56"
          />
          <ElTable v-else :data="serviceCatalog.unmapped_activities" border>
            <ElTableColumn
              :label="$t('corporationStructures.serviceCatalog.activity')"
              min-width="260"
            >
              <template #default="{ row }">
                <div>{{ row.activity_name }}</div>
                <div class="text-xs text-g-500"
                  >{{ row.structure_name }} ({{ row.structure_id }})</div
                >
                <div class="text-xs text-g-500">
                  {{ $t('corporationStructures.serviceCatalog.installedModules') }}:
                  {{ row.installed_module_type_ids.join(', ') || '--' }}
                </div>
              </template>
            </ElTableColumn>
            <ElTableColumn
              :label="$t('corporationStructures.serviceCatalog.module')"
              min-width="300"
            >
              <template #default="{ row }">
                <ElSelect
                  v-model="activityModuleByName[row.activity_name]"
                  class="w-full"
                  filterable
                  multiple
                >
                  <ElOption
                    v-for="module in serviceCatalog.modules"
                    :key="module.type_id"
                    :value="module.type_id"
                    :label="`${module.type_name} (${module.type_id})`"
                  />
                </ElSelect>
              </template>
            </ElTableColumn>
          </ElTable>
          <div v-if="serviceCatalog.unmapped_activities.length > 0" class="mt-3">
            <ElButton
              type="primary"
              :loading="savingServiceCatalog"
              @click="saveServiceActivityMappings"
              >{{ $t('corporationStructures.serviceCatalog.save') }}</ElButton
            >
          </div>

          <ElDivider>{{ $t('corporationStructures.serviceCatalog.modules') }}</ElDivider>
          <ElTable :data="serviceCatalog.modules" border size="small">
            <ElTableColumn
              :label="$t('corporationStructures.serviceCatalog.module')"
              prop="type_name"
              min-width="260"
            />
            <ElTableColumn
              :label="$t('corporationStructures.serviceCatalog.typeId')"
              prop="type_id"
              width="150"
            />
            <ElTableColumn
              :label="$t('corporationStructures.serviceCatalog.fuelRate')"
              prop="fuel_per_hour"
              width="170"
            />
            <ElTableColumn
              :label="$t('corporationStructures.serviceCatalog.management')"
              prop="fuel_category"
              min-width="180"
            />
          </ElTable>

          <ElDivider>{{ $t('corporationStructures.serviceCatalog.mappings') }}</ElDivider>
          <ElEmpty
            v-if="serviceCatalog.activities.length === 0"
            :description="$t('corporationStructures.serviceCatalog.noMappings')"
            :image-size="40"
          />
          <ElTable v-else :data="serviceCatalog.activities" border size="small">
            <ElTableColumn
              :label="$t('corporationStructures.serviceCatalog.activity')"
              prop="activity_name"
              min-width="260"
            />
            <ElTableColumn
              :label="$t('corporationStructures.serviceCatalog.typeId')"
              min-width="220"
            >
              <template #default="{ row }">{{ row.type_ids.join(', ') }}</template>
            </ElTableColumn>
            <ElTableColumn
              :label="$t('corporationStructures.serviceCatalog.category')"
              min-width="160"
            >
              <template #default="{ row }">{{
                row.system_managed
                  ? $t('corporationStructures.serviceCatalog.systemManaged')
                  : $t('corporationStructures.serviceCatalog.customManaged')
              }}</template>
            </ElTableColumn>
          </ElTable>
        </ElCard>
      </ElTabPane>
      <ElTabPane
        :label="$t('corporationStructures.tabs.assignmentSalary')"
        name="assignment_salary"
      >
        <ElCard shadow="never" class="art-card mb-4">
          <div class="flex flex-wrap items-center gap-3 mb-4">
            <ElButton :loading="assignmentsLoading" @click="loadAssignments">
              {{ $t('common.refresh') }}
            </ElButton>
            <ElButton type="primary" :loading="savingAssignments" @click="saveAssignments">
              {{ $t('common.save') }}
            </ElButton>
          </div>

          <ElFormItem :label="$t('corporationStructures.salary.salaryPerStructure')" class="mb-4">
            <div class="flex items-center gap-2">
              <ElInputNumber v-model="salaryPerStructureMonthly" :min="0" :step="1" step-strictly />
              <ElButton type="primary" :loading="savingSalary" @click="saveFuelSalarySettings">
                {{ $t('common.save') }}
              </ElButton>
            </div>
          </ElFormItem>

          <ElFormItem :label="$t('corporationStructures.salary.payoutMonth')" class="mb-4">
            <div class="flex items-center gap-2">
              <ElDatePicker v-model="payoutMonth" type="month" value-format="YYYY-MM" />
              <ElButton type="primary" :loading="runningPayout" @click="runSalaryPayout">
                {{ $t('corporationStructures.salary.runPayout') }}
              </ElButton>
            </div>
          </ElFormItem>
        </ElCard>
        <ElCard shadow="never" class="art-card mb-4">
          <div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-3">
            <ElFormItem :label="$t('corporationStructures.salary.targetFuelOfficer')" class="mb-0">
              <ElSelect v-model="assignmentTargetUserId" clearable filterable class="w-full">
                <ElOption
                  v-for="option in fuelOfficerOptions"
                  :key="option.user_id"
                  :label="`${option.display_name} (${option.user_id})`"
                  :value="option.user_id"
                />
              </ElSelect>
            </ElFormItem>
            <ElFormItem :label="$t('corporationStructures.salary.assignmentFilter')" class="mb-0">
              <ElRadioGroup v-model="assignmentFilterMode">
                <ElRadioButton value="all">{{
                  $t('corporationStructures.salary.assignmentFilterAll')
                }}</ElRadioButton>
                <ElRadioButton value="assigned_to_selected_officer">{{
                  $t('corporationStructures.salary.assignmentFilterAssignedToTarget')
                }}</ElRadioButton>
                <ElRadioButton value="unassigned">{{
                  $t('corporationStructures.salary.assignmentFilterUnassigned')
                }}</ElRadioButton>
              </ElRadioGroup>
            </ElFormItem>
            <ElFormItem :label="$t('corporationStructures.filters.systems')" class="mb-0">
              <ElSelect
                v-model="assignmentFilterSystemIds"
                multiple
                filterable
                clearable
                collapse-tags
                collapse-tags-tooltip
                class="w-full"
              >
                <ElOption
                  v-for="item in assignmentSystemOptions"
                  :key="item.system_id"
                  :label="item.system_name"
                  :value="item.system_id"
                />
              </ElSelect>
            </ElFormItem>
            <ElFormItem :label="$t('corporationStructures.salary.regionFilter')" class="mb-0">
              <ElSelect
                v-model="assignmentFilterRegionIds"
                multiple
                filterable
                clearable
                collapse-tags
                collapse-tags-tooltip
                class="w-full"
              >
                <ElOption
                  v-for="item in assignmentRegionOptions"
                  :key="item.region_id"
                  :label="item.region_name"
                  :value="item.region_id"
                />
              </ElSelect>
            </ElFormItem>
            <ElFormItem :label="$t('corporationStructures.filters.types')" class="mb-0">
              <ElSelect
                v-model="assignmentFilterTypeIds"
                multiple
                filterable
                clearable
                collapse-tags
                collapse-tags-tooltip
                class="w-full"
              >
                <ElOption
                  v-for="item in assignmentTypeOptions"
                  :key="item.type_id"
                  :label="item.type_name"
                  :value="item.type_id"
                />
              </ElSelect>
            </ElFormItem>
          </div>
          <div class="mt-4">
            <ElButton @click="resetAssignmentFilters">{{ $t('common.reset') }}</ElButton>
          </div>
        </ElCard>

        <ElCard shadow="never" class="art-table-card corporation-structures-page__list-card">
          <ArtTableHeader
            v-model:columns="assignmentColumnChecks"
            :loading="assignmentsLoading"
            @refresh="loadAssignments"
          />
          <ArtTable
            :loading="assignmentsLoading"
            :data="pagedAssignmentItems"
            :columns="assignmentColumnChecks"
            :pagination="assignmentPagination"
            :empty-text="$t('corporationStructures.empty.list')"
            @pagination:size-change="handleAssignmentPaginationSizeChange"
            @pagination:current-change="handleAssignmentPaginationCurrentChange"
          />
        </ElCard>
      </ElTabPane>
    </ElTabs>
  </div>
</template>

<script setup lang="ts">
  import { ElCheckbox, ElLink, ElMessage, ElMessageBox, ElTag } from 'element-plus'
  import { useI18n } from 'vue-i18n'
  import { useRoute, useRouter } from 'vue-router'
  import { useCorpCapability } from '@/hooks/core/useCorpCapability'
  import { useTable } from '@/hooks/core/useTable'
  import type { ColumnOption } from '@/types/component'
  import { formatTime, fuelExpiryMonthOffset } from '@/utils/common'
  import StructureServicesDialog from './modules/structure-services-dialog.vue'
  import {
    fetchCorporationStructureAssignments,
    fetchCorporationStructureFilterOptions,
    fetchCorporationStructureList,
    fetchCorporationStructureSettings,
    fetchStructureServiceCatalog,
    fetchFuelSalarySettings,
    runFuelSalaryPayout,
    runCorporationStructuresTask,
    updateCorporationStructureAssignments,
    updateCorporationStructureAuthorizations,
    updateFuelSalarySettings,
    updateStructureServiceCatalog
  } from '@/api/corporation-structures'
  import { runTask } from '@/api/task-manager'

  defineOptions({ name: 'DashboardCorporationStructures' })

  type StructureTab = 'list' | 'settings' | 'assignment_salary'
  type StructureRow = Api.Dashboard.CorporationStructureRow
  type AssignmentRow = Api.Dashboard.CorporationStructureAssignmentItem
  type AssignmentFilterMode = 'all' | 'assigned_to_selected_officer' | 'unassigned'
  type TableSort = { prop?: string; order?: 'ascending' | 'descending' | null }
  type TagType = '' | 'success' | 'warning' | 'info' | 'primary' | 'danger'

  const { t } = useI18n()
  const route = useRoute()
  const router = useRouter()
  const { hasCapability } = useCorpCapability()

  const settings = ref<Api.Dashboard.CorporationStructuresSettings>({
    corporations: [],
    fuel_notice_threshold_days: 7,
    timer_notice_threshold_days: 7,
    alert_enabled: false,
    alert_group_ids: []
  })
  const alertEnabled = ref(false)
  const alertGroupIDsText = ref('')
  const noticeThresholds = reactive({
    fuel_notice_threshold_days: 7,
    timer_notice_threshold_days: 7
  })
  const settingsLoading = ref(false)
  const serviceCatalog = ref<Api.Dashboard.StructureServiceCatalog>({
    modules: [],
    activities: [],
    unmapped_activities: []
  })
  const activityModuleByName = reactive<Record<string, number[]>>({})
  const savingServiceCatalog = ref(false)
  const savingAuthorizations = ref(false)
  const alertScanRunning = ref(false)
  const runningTaskCorpId = ref<number>(0)
  const canRunAlertScan = computed(() => hasCapability('system.task.run'))
  const authorizationByCorp = reactive<Record<number, number>>({})
  const timerRange = ref<[string, string] | null>(null)
  const filterDrawerVisible = ref(false)
  const { width: windowWidth } = useWindowSize()
  const filterDrawerSize = computed(() => (windowWidth.value < 768 ? '100%' : '560px'))

  const servicesDialogVisible = ref(false)
  const servicesDialogRow = ref<StructureRow | null>(null)
  const openServicesDialog = (row: StructureRow) => {
    servicesDialogRow.value = row
    servicesDialogVisible.value = true
  }

  const filterOptions = ref<Api.Dashboard.CorporationStructureFilterOptionsResponse>({
    systems: [],
    regions: [],
    types: [],
    services: []
  })

  const buildDefaultFilters = () => ({
    corporation_id: 0,
    keyword: '',
    state_groups: [] as string[],
    fuel_bucket: 'all' as Api.Dashboard.CorporationStructureListRequest['fuel_bucket'],
    fuel_min_hours: undefined as number | undefined,
    fuel_max_hours: undefined as number | undefined,
    system_ids: [] as number[],
    region_ids: [] as number[],
    security_bands: [] as ('highsec' | 'lowsec' | 'nullsec')[],
    security_min: undefined as number | undefined,
    security_max: undefined as number | undefined,
    type_ids: [] as number[],
    service_names: [] as string[],
    service_match_mode: 'and' as const,
    timer_bucket: 'all' as Api.Dashboard.CorporationStructureListRequest['timer_bucket']
  })

  const filters = reactive(buildDefaultFilters())

  const normalizeTab = (value: unknown): StructureTab => {
    const queryValue = Array.isArray(value) ? value[0] : value
    if (queryValue === 'settings') return 'settings'
    if (queryValue === 'assignment_salary') return 'assignment_salary'
    return 'list'
  }

  const activeTab = ref<StructureTab>(normalizeTab(route.query.tab))
  const assignmentsLoading = ref(false)
  const savingAssignments = ref(false)
  const savingSalary = ref(false)
  const runningPayout = ref(false)
  const payoutMonth = ref<string>('')
  const assignmentItems = ref<AssignmentRow[]>([])
  const fuelOfficerOptions = ref<Api.Dashboard.FuelOfficerUserOption[]>([])
  const assignmentByStructure = reactive<Record<number, number>>({})
  const assignmentTargetUserId = ref<number>(0)
  const assignmentFilterMode = ref<AssignmentFilterMode>('all')
  const assignmentFilterSystemIds = ref<number[]>([])
  const assignmentFilterRegionIds = ref<number[]>([])
  const assignmentFilterTypeIds = ref<number[]>([])
  const assignmentPagination = reactive({ current: 1, size: 20, total: 0 })
  const salaryPerStructureMonthly = ref(0)
  const validCorporations = computed(() =>
    settings.value.corporations.filter((corp) => (corp.authorized_character_id || 0) > 0)
  )
  const assignmentSystemOptions = computed(() => {
    const optionByID = new Map<number, { system_id: number; system_name: string }>()
    assignmentItems.value.forEach((item) => {
      if (item.system_id > 0 && !optionByID.has(item.system_id)) {
        optionByID.set(item.system_id, {
          system_id: item.system_id,
          system_name: item.system_name || `System-${item.system_id}`
        })
      }
    })
    return [...optionByID.values()].sort((a, b) => a.system_name.localeCompare(b.system_name))
  })
  const assignmentRegionOptions = computed(() => {
    const optionByID = new Map<number, { region_id: number; region_name: string }>()
    assignmentItems.value.forEach((item) => {
      if (item.region_id > 0 && !optionByID.has(item.region_id)) {
        optionByID.set(item.region_id, {
          region_id: item.region_id,
          region_name: item.region_name || `Region-${item.region_id}`
        })
      }
    })
    return [...optionByID.values()].sort((a, b) => a.region_name.localeCompare(b.region_name))
  })
  const assignmentTypeOptions = computed(() => {
    const optionByID = new Map<number, { type_id: number; type_name: string }>()
    assignmentItems.value.forEach((item) => {
      if (item.type_id > 0 && !optionByID.has(item.type_id)) {
        optionByID.set(item.type_id, {
          type_id: item.type_id,
          type_name: item.type_name || `Type-${item.type_id}`
        })
      }
    })
    return [...optionByID.values()].sort((a, b) => a.type_name.localeCompare(b.type_name))
  })
  const filteredAssignmentItems = computed(() => {
    return assignmentItems.value.filter((item) => {
      if (
        assignmentFilterMode.value === 'assigned_to_selected_officer' &&
        assignmentTargetUserId.value > 0 &&
        assignmentByStructure[item.structure_id] !== assignmentTargetUserId.value
      ) {
        return false
      }
      if (
        assignmentFilterMode.value === 'assigned_to_selected_officer' &&
        assignmentTargetUserId.value <= 0
      ) {
        return false
      }
      if (
        assignmentFilterMode.value === 'unassigned' &&
        (assignmentByStructure[item.structure_id] || 0) > 0
      ) {
        return false
      }
      if (
        assignmentFilterSystemIds.value.length > 0 &&
        !assignmentFilterSystemIds.value.includes(item.system_id)
      ) {
        return false
      }
      if (
        assignmentFilterRegionIds.value.length > 0 &&
        !assignmentFilterRegionIds.value.includes(item.region_id)
      ) {
        return false
      }
      if (
        assignmentFilterTypeIds.value.length > 0 &&
        !assignmentFilterTypeIds.value.includes(item.type_id)
      ) {
        return false
      }
      return true
    })
  })
  const pagedAssignmentItems = computed(() => {
    assignmentPagination.total = filteredAssignmentItems.value.length
    const start = (assignmentPagination.current - 1) * assignmentPagination.size
    return filteredAssignmentItems.value.slice(start, start + assignmentPagination.size)
  })
  const assignmentColumns = computed<ColumnOption<AssignmentRow>[]>(() => [
    {
      prop: 'assigned_to_target',
      label: t('corporationStructures.salary.assignedToTarget'),
      width: 170,
      formatter: (row: AssignmentRow) =>
        h(ElCheckbox, {
          modelValue:
            assignmentTargetUserId.value > 0 &&
            assignmentByStructure[row.structure_id] === assignmentTargetUserId.value,
          disabled: assignmentTargetUserId.value <= 0,
          onChange: (checked: string | number | boolean) =>
            toggleAssignmentToTarget(row, Boolean(checked))
        })
    },
    {
      prop: 'corporation_name',
      label: t('corporationStructures.table.corporation'),
      minWidth: 180,
      showOverflowTooltip: true
    },
    {
      prop: 'structure_name',
      label: t('corporationStructures.table.name'),
      minWidth: 220,
      showOverflowTooltip: true
    },
    {
      prop: 'system_name',
      label: t('corporationStructures.table.system'),
      minWidth: 220,
      formatter: (row: AssignmentRow) =>
        h('div', { class: 'leading-5' }, [
          h('div', {}, row.system_name || '--'),
          h(
            'div',
            { class: 'text-xs text-g-500' },
            `${row.region_name || '--'} / ${formatSecurity(row.security)}`
          )
        ])
    },
    {
      prop: 'type_name',
      label: t('corporationStructures.table.type'),
      minWidth: 180,
      showOverflowTooltip: true
    },
    {
      prop: 'assigned_character_name',
      label: t('corporationStructures.salary.assignedFuelOfficer'),
      minWidth: 220,
      formatter: (row: AssignmentRow) =>
        row.assigned_character_name || t('corporationStructures.salary.unassignedLabel')
    }
  ])
  const assignmentColumnChecks = ref<ColumnOption[]>([])

  const normalizeFuelHours = (value: number | undefined) => {
    if (value == null || Number.isNaN(value)) {
      return undefined
    }
    return Math.max(0, Math.floor(value))
  }

  const normalizeThresholdDays = (value: number) => {
    if (Number.isNaN(value)) {
      return 0
    }
    return Math.max(0, Math.floor(value))
  }

  const fetchStructurePage = async (
    params: Api.Dashboard.CorporationStructureListRequest & { current: number; size: number }
  ): Promise<Api.Common.PaginatedResponse<StructureRow>> => {
    const corpId = params.corporation_id ?? 0
    const response = await fetchCorporationStructureList({
      corporation_id: corpId > 0 ? corpId : undefined,
      page: params.current,
      page_size: params.size,
      keyword: params.keyword || undefined,
      state_groups: params.state_groups?.length ? params.state_groups : undefined,
      fuel_bucket: params.fuel_bucket,
      fuel_min_hours: normalizeFuelHours(params.fuel_min_hours),
      fuel_max_hours: normalizeFuelHours(params.fuel_max_hours),
      system_ids: params.system_ids?.length ? params.system_ids : undefined,
      region_ids: params.region_ids?.length ? params.region_ids : undefined,
      security_bands: params.security_bands?.length ? params.security_bands : undefined,
      security_min: params.security_min,
      security_max: params.security_max,
      type_ids: params.type_ids?.length ? params.type_ids : undefined,
      service_names: params.service_names?.length ? params.service_names : undefined,
      service_match_mode: params.service_match_mode,
      timer_bucket: params.timer_bucket,
      timer_start: params.timer_start,
      timer_end: params.timer_end,
      sort_by: params.sort_by,
      sort_order: params.sort_order
    })
    return {
      list: response?.items ?? [],
      total: response?.total ?? 0,
      page: response?.page ?? params.current,
      pageSize: response?.page_size ?? params.size
    }
  }

  const stateTagTypeMap: Record<string, TagType> = {
    shield_vulnerable: 'success',
    low_power: 'warning',
    abandoned: 'info',
    shield_reinforce: 'danger',
    armor_reinforce: 'danger',
    armor_vulnerable: 'danger',
    hull_reinforce: 'danger',
    hull_vulnerable: 'danger'
  }

  const {
    columns,
    columnChecks,
    data,
    loading,
    pagination,
    searchParams,
    handleSizeChange,
    handleCurrentChange,
    refreshData,
    getData
  } = useTable({
    core: {
      apiFn: fetchStructurePage,
      apiParams: {
        corporation_id: 0,
        keyword: '',
        state_groups: [],
        fuel_bucket: 'all',
        fuel_min_hours: undefined,
        fuel_max_hours: undefined,
        system_ids: [],
        region_ids: [],
        security_bands: [],
        security_min: undefined,
        security_max: undefined,
        type_ids: [],
        service_names: [],
        service_match_mode: 'and',
        timer_bucket: 'all',
        timer_start: undefined,
        timer_end: undefined,
        sort_by: 'fuel_remaining_hours',
        sort_order: 'asc',
        current: 1,
        size: 20
      },
      immediate: false,
      columnsFactory: () => [
        {
          prop: 'corporation_name',
          label: t('corporationStructures.table.corporation'),
          minWidth: 180,
          showOverflowTooltip: true
        },
        {
          prop: 'assigned_character_name',
          label: t('corporationStructures.salary.assignedFuelOfficer'),
          minWidth: 220,
          showOverflowTooltip: true,
          formatter: (row: StructureRow) =>
            row.assigned_user_id > 0
              ? row.assigned_character_name || '--'
              : t('corporationStructures.salary.unassignedLabel')
        },
        {
          prop: 'state',
          label: t('corporationStructures.table.state'),
          width: 180,
          formatter: (row: StructureRow) =>
            h(
              ElTag,
              {
                type: stateTagTypeMap[row.state] || 'info',
                size: 'small',
                effect: 'plain'
              },
              () => mapStateLabel(row.state)
            )
        },
        {
          prop: 'system_name',
          label: t('corporationStructures.table.system'),
          minWidth: 220,
          sortable: 'custom' as const,
          formatter: (row: StructureRow) =>
            h('div', { class: 'leading-5' }, [
              h('div', {}, row.system_name || '--'),
              h(
                'div',
                { class: 'text-xs text-g-500' },
                `${row.region_name || '--'} / ${formatSecurity(row.security)}`
              )
            ])
        },
        {
          prop: 'name',
          label: t('corporationStructures.table.name'),
          minWidth: 200,
          sortable: 'custom' as const,
          showOverflowTooltip: true
        },
        {
          prop: 'type_name',
          label: t('corporationStructures.table.type'),
          minWidth: 180,
          sortable: 'custom' as const,
          showOverflowTooltip: true
        },
        {
          prop: 'services',
          label: t('corporationStructures.table.services'),
          width: 110,
          align: 'center',
          formatter: (row: StructureRow) =>
            row.services && row.services.length > 0
              ? h(
                  ElLink,
                  { type: 'primary', underline: 'never', onClick: () => openServicesDialog(row) },
                  () => String(row.services.length)
                )
              : t('corporationStructures.noServices')
        },
        {
          prop: 'fuel_remaining_hours',
          label: t('corporationStructures.table.fuelRemaining'),
          width: 170,
          sortable: 'custom' as const,
          formatter: (row: StructureRow) => row.fuel_remaining || '--'
        },
        {
          prop: 'fuel_per_hour',
          label: t('corporationStructures.table.fuelPerHour'),
          width: 180,
          sortable: 'custom' as const,
          formatter: (row: StructureRow) =>
            row.fuel_estimate_incomplete
              ? formatFuelEstimateStatus(row.fuel_estimate_status)
              : row.fuel_per_hour != null
                ? row.fuel_per_hour
                : '--'
        },
        {
          prop: 'fuel_to_month_end',
          label: t('corporationStructures.table.fuelToMonthEnd'),
          width: 200,
          sortable: 'custom' as const,
          formatter: (row: StructureRow) => formatFuelToMonthEnd(row)
        },
        {
          prop: 'reinforce_hour',
          label: t('corporationStructures.table.reinforceHour'),
          width: 150,
          sortable: 'custom' as const,
          formatter: (row: StructureRow) => formatReinforceHour(row.reinforce_hour)
        },
        {
          prop: 'state_timer_end',
          label: t('corporationStructures.table.timerEnd'),
          width: 190,
          sortable: 'custom' as const,
          formatter: (row: StructureRow) => formatTimeText(row.state_timer_end)
        },
        {
          prop: 'updated_at',
          label: t('corporationStructures.table.updatedAt'),
          width: 190,
          sortable: 'custom' as const,
          formatter: (row: StructureRow) => formatUpdatedAt(row.updated_at)
        }
      ]
    }
  })

  const syncAuthorizationsFromSettings = () => {
    Object.keys(authorizationByCorp).forEach((key) => {
      delete authorizationByCorp[Number(key)]
    })
    settings.value.corporations.forEach((item) => {
      authorizationByCorp[item.corporation_id] = item.authorized_character_id || 0
    })
    noticeThresholds.fuel_notice_threshold_days = normalizeThresholdDays(
      settings.value.fuel_notice_threshold_days
    )
    noticeThresholds.timer_notice_threshold_days = normalizeThresholdDays(
      settings.value.timer_notice_threshold_days
    )
    alertEnabled.value = settings.value.alert_enabled
    alertGroupIDsText.value = settings.value.alert_group_ids.join('\n')
  }

  const loadSettings = async () => {
    settingsLoading.value = true
    try {
      settings.value = await fetchCorporationStructureSettings()
      serviceCatalog.value = await fetchStructureServiceCatalog()
      serviceCatalog.value.unmapped_activities.forEach((item) => {
        if (!activityModuleByName[item.activity_name]) {
          activityModuleByName[item.activity_name] = []
        }
      })
      syncAuthorizationsFromSettings()

      const validCorpSet = new Set(validCorporations.value.map((item) => item.corporation_id))
      if (filters.corporation_id > 0 && !validCorpSet.has(filters.corporation_id)) {
        filters.corporation_id = 0
      }
    } finally {
      settingsLoading.value = false
    }
  }

  const loadFilterOptions = async () => {
    filterOptions.value = (await fetchCorporationStructureFilterOptions({
      corporation_id: filters.corporation_id > 0 ? filters.corporation_id : undefined
    })) || {
      systems: [],
      regions: [],
      types: [],
      services: []
    }
  }

  const copyFiltersToSearchParams = () => {
    searchParams.corporation_id = filters.corporation_id
    searchParams.keyword = filters.keyword
    searchParams.state_groups = [...filters.state_groups]
    searchParams.fuel_bucket = filters.fuel_bucket
    searchParams.fuel_min_hours =
      filters.fuel_bucket === 'custom' ? filters.fuel_min_hours : undefined
    searchParams.fuel_max_hours =
      filters.fuel_bucket === 'custom' ? filters.fuel_max_hours : undefined
    searchParams.system_ids = [...filters.system_ids]
    searchParams.region_ids = [...filters.region_ids]
    searchParams.security_bands = [...filters.security_bands]
    searchParams.security_min = filters.security_min
    searchParams.security_max = filters.security_max
    searchParams.type_ids = [...filters.type_ids]
    searchParams.service_names = [...filters.service_names]
    searchParams.service_match_mode = filters.service_match_mode
    searchParams.timer_bucket = filters.timer_bucket
    searchParams.timer_start =
      filters.timer_bucket === 'custom' && timerRange.value ? timerRange.value[0] : undefined
    searchParams.timer_end =
      filters.timer_bucket === 'custom' && timerRange.value ? timerRange.value[1] : undefined
  }

  const handleSearch = () => {
    copyFiltersToSearchParams()
    searchParams.current = 1
    getData()
  }

  const handleSearchFromDrawer = () => {
    handleSearch()
    filterDrawerVisible.value = false
  }

  const handleReset = () => {
    Object.assign(filters, buildDefaultFilters())
    timerRange.value = null
    copyFiltersToSearchParams()
    searchParams.sort_by = 'fuel_remaining_hours'
    searchParams.sort_order = 'asc'
    searchParams.current = 1
    getData()
    void loadFilterOptions()
  }

  const saveAuthorizations = async () => {
    const authorizations: Api.Dashboard.CorporationStructureAuthorizationBinding[] =
      settings.value.corporations.map((corp) => ({
        corporation_id: corp.corporation_id,
        character_id: authorizationByCorp[corp.corporation_id] || 0
      }))

    savingAuthorizations.value = true
    try {
      await updateCorporationStructureAuthorizations({
        authorizations,
        fuel_notice_threshold_days: normalizeThresholdDays(
          noticeThresholds.fuel_notice_threshold_days
        ),
        timer_notice_threshold_days: normalizeThresholdDays(
          noticeThresholds.timer_notice_threshold_days
        ),
        alert_enabled: alertEnabled.value,
        alert_group_ids: parseAlertGroupIDs(alertGroupIDsText.value)
      })
      await loadSettings()
      ElMessage.success(t('corporationStructures.messages.authorizationSaved'))
    } finally {
      savingAuthorizations.value = false
    }
  }

  const saveServiceActivityMappings = async () => {
    const activityNames = [
      ...new Set(serviceCatalog.value.unmapped_activities.map((item) => item.activity_name))
    ]
    const activities = activityNames
      .filter((name) => (activityModuleByName[name] || []).length > 0)
      .map((name) => ({ activity_name: name, type_ids: activityModuleByName[name] }))
    if (activities.length === 0) return
    savingServiceCatalog.value = true
    try {
      await updateStructureServiceCatalog({ modules: [], activities })
      await loadSettings()
      ElMessage.success(t('corporationStructures.messages.serviceCatalogSaved'))
    } finally {
      savingServiceCatalog.value = false
    }
  }

  const parseAlertGroupIDs = (value: string): number[] =>
    value
      .split(/\r?\n/)
      .map((item) => item.trim())
      .filter((item) => item.length > 0)
      .map((item) => Number(item))

  const runAlertScan = async () => {
    try {
      await ElMessageBox.confirm(
        t('corporationStructures.confirm.runAlertScan'),
        t('common.tips'),
        {
          confirmButtonText: t('common.confirm'),
          cancelButtonText: t('common.cancel'),
          type: 'warning'
        }
      )
    } catch {
      return
    }

    alertScanRunning.value = true
    try {
      await runTask('corporation_structure_alert_scan')
      ElMessage.success(t('corporationStructures.messages.alertScanTriggered'))
    } catch {
      ElMessage.error(t('corporationStructures.messages.alertScanTriggerFailed'))
    } finally {
      alertScanRunning.value = false
    }
  }

  const handleRunTaskForCorporation = async (corporationId: number) => {
    runningTaskCorpId.value = corporationId
    try {
      const result = await runCorporationStructuresTask({ corporation_id: corporationId })
      if (result.running) {
        ElMessage.warning(
          result.message || t('corporationStructures.messages.refreshAlreadyRunning')
        )
        return
      }
      ElMessage.success(result.message || t('corporationStructures.messages.refreshQueued'))
    } finally {
      runningTaskCorpId.value = 0
    }
  }

  const handleRunTaskForSelectedCorporation = async () => {
    if (filters.corporation_id <= 0) {
      ElMessage.warning(t('corporationStructures.messages.selectCorporationFirst'))
      return
    }
    await handleRunTaskForCorporation(filters.corporation_id)
  }

  const handleCorporationFilterChange = async () => {
    await loadFilterOptions()
  }

  const openFilterDrawer = () => {
    filterDrawerVisible.value = true
  }

  const handleSortChange = (sort: TableSort) => {
    if (!sort?.prop || !sort.order) {
      searchParams.sort_by = 'fuel_remaining_hours'
      searchParams.sort_order = 'asc'
    } else {
      searchParams.sort_by = sort.prop as Api.Dashboard.CorporationStructureListRequest['sort_by']
      searchParams.sort_order = sort.order === 'descending' ? 'desc' : 'asc'
    }
    searchParams.current = 1
    getData()
  }

  const formatFuelEstimateStatus = (
    status: Api.Dashboard.CorporationStructureRow['fuel_estimate_status']
  ) => {
    const keyByStatus: Record<string, string> = {
      authorization_required: 'fuelEstimateAuthorizationRequired',
      activity_mapping_required: 'fuelEstimateActivityMappingRequired',
      module_mismatch: 'fuelEstimateModuleMismatch',
      rate_unavailable: 'fuelEstimateRateUnavailable',
      ambiguous_module: 'fuelEstimateAmbiguousModule'
    }
    return t(`corporationStructures.table.${keyByStatus[status] || 'fuelEstimateIncomplete'}`)
  }

  // 数值后的 +N 徽标表示目标月底（fuel_expires 所在月）距当前月还有几个整月
  const formatFuelToMonthEnd = (row: StructureRow) => {
    if (row.fuel_estimate_incomplete) {
      return formatFuelEstimateStatus(row.fuel_estimate_status)
    }
    if (row.fuel_to_month_end == null) {
      return '--'
    }
    const monthOffset = fuelExpiryMonthOffset(row.fuel_expires)
    if (monthOffset == null || monthOffset < 1) {
      return row.fuel_to_month_end
    }
    return h('span', { class: 'inline-flex items-center gap-1' }, [
      String(row.fuel_to_month_end),
      h(ElTag, { size: 'small', type: 'info', effect: 'plain' }, () => `+${monthOffset}`)
    ])
  }

  const formatSecurity = (security: number) => {
    if (typeof security !== 'number' || Number.isNaN(security)) {
      return '--'
    }
    return security.toFixed(1)
  }

  const formatUpdatedAt = (updatedAt: number) => {
    if (!updatedAt) {
      return '--'
    }
    return formatTime(new Date(updatedAt * 1000).toISOString())
  }

  const formatTimeText = (value: string) => {
    if (!value) {
      return '--'
    }
    const parsed = new Date(value)
    if (Number.isNaN(parsed.getTime())) {
      return value
    }
    return formatTime(parsed.toISOString())
  }

  const formatReinforceHour = (hour: number) => {
    if (!Number.isInteger(hour) || hour < 0 || hour > 23) {
      return '--'
    }
    return String(hour).padStart(2, '0')
  }

  const mapStateLabel = (state: string) => {
    const key = `corporationStructures.states.${state}`
    const translated = t(key)
    if (translated === key) {
      return state || '--'
    }
    return translated
  }

  const formatSystemOption = (item: Api.Dashboard.CorporationStructureSystemOption) => {
    const regionText = item.region_name ? ` / ${item.region_name}` : ''
    return `${item.system_name}${regionText} (${formatSecurity(item.security)})`
  }

  const handleTabChange = (tab: string | number) => {
    activeTab.value = normalizeTab(tab)
  }

  const syncAssignmentColumnChecks = () => {
    assignmentColumnChecks.value = assignmentColumns.value.map((col) => ({
      ...col,
      checked: col.checked ?? true,
      visible: col.visible ?? true
    }))
  }

  const resetAssignmentFilters = () => {
    assignmentFilterMode.value = 'all'
    assignmentFilterSystemIds.value = []
    assignmentFilterRegionIds.value = []
    assignmentFilterTypeIds.value = []
    assignmentPagination.current = 1
  }

  const toggleAssignmentToTarget = (row: AssignmentRow, checked: boolean) => {
    const target = assignmentTargetUserId.value
    if (target <= 0) {
      return
    }
    if (checked) {
      assignmentByStructure[row.structure_id] = target
      row.assigned_user_id = target
      row.assigned_character_name =
        fuelOfficerOptions.value.find((item) => item.user_id === target)?.display_name || ''
      return
    }

    if (assignmentByStructure[row.structure_id] === target) {
      assignmentByStructure[row.structure_id] = 0
      row.assigned_user_id = 0
      row.assigned_character_name = ''
    }
  }

  const handleAssignmentPaginationSizeChange = (size: number) => {
    assignmentPagination.size = size
    assignmentPagination.current = 1
  }

  const handleAssignmentPaginationCurrentChange = (current: number) => {
    assignmentPagination.current = current
  }

  const loadAssignments = async () => {
    assignmentsLoading.value = true
    try {
      const response = await fetchCorporationStructureAssignments({
        corporation_id: filters.corporation_id > 0 ? filters.corporation_id : undefined
      })
      assignmentItems.value = response.items
      fuelOfficerOptions.value = response.fuel_officers
      Object.keys(assignmentByStructure).forEach((key) => {
        delete assignmentByStructure[Number(key)]
      })
      for (const item of response.items) {
        assignmentByStructure[item.structure_id] = item.assigned_user_id || 0
      }
      if (
        assignmentTargetUserId.value > 0 &&
        !response.fuel_officers.some((item) => item.user_id === assignmentTargetUserId.value)
      ) {
        assignmentTargetUserId.value = 0
      }
      if (assignmentTargetUserId.value <= 0 && response.fuel_officers.length > 0) {
        assignmentTargetUserId.value = response.fuel_officers[0].user_id
      }
      resetAssignmentFilters()
      syncAssignmentColumnChecks()
    } finally {
      assignmentsLoading.value = false
    }
  }

  const saveAssignments = async () => {
    savingAssignments.value = true
    try {
      await updateCorporationStructureAssignments({
        assignments: assignmentItems.value.map((item) => ({
          corporation_id: item.corporation_id,
          structure_id: item.structure_id,
          user_id: assignmentByStructure[item.structure_id] || 0
        }))
      })
      ElMessage.success(t('common.saveSuccess'))
      await loadAssignments()
    } finally {
      savingAssignments.value = false
    }
  }

  const loadFuelSalarySettings = async () => {
    const settingsRes = await fetchFuelSalarySettings()
    salaryPerStructureMonthly.value = Math.max(0, settingsRes.salary_per_structure_monthly || 0)
  }

  const saveFuelSalarySettings = async () => {
    savingSalary.value = true
    try {
      await updateFuelSalarySettings({
        salary_per_structure_monthly: Math.max(0, Math.floor(salaryPerStructureMonthly.value || 0))
      })
      ElMessage.success(t('common.saveSuccess'))
      await loadFuelSalarySettings()
    } finally {
      savingSalary.value = false
    }
  }

  const runSalaryPayout = async () => {
    if (!payoutMonth.value) {
      ElMessage.warning(t('corporationStructures.salary.selectPayoutMonth'))
      return
    }
    runningPayout.value = true
    try {
      const result = await runFuelSalaryPayout({ settlement_month: payoutMonth.value })
      ElMessage.success(
        t('corporationStructures.salary.payoutSuccess', { count: result.items.length })
      )
    } finally {
      runningPayout.value = false
    }
  }

  watch(
    () => route.query.tab,
    (value) => {
      const nextTab = normalizeTab(value)
      if (nextTab !== activeTab.value) {
        activeTab.value = nextTab
      }
    }
  )

  watch(activeTab, (tab) => {
    const queryTab = normalizeTab(route.query.tab)
    if (queryTab === tab && route.query.tab) {
      return
    }
    void router.replace({
      query: {
        ...route.query,
        tab
      }
    })
  })

  watch(
    () => [
      assignmentFilterMode.value,
      assignmentTargetUserId.value,
      assignmentFilterSystemIds.value.join(','),
      assignmentFilterRegionIds.value.join(','),
      assignmentFilterTypeIds.value.join(',')
    ],
    () => {
      assignmentPagination.current = 1
    }
  )

  watch(
    () => filteredAssignmentItems.value.length,
    (total) => {
      const maxPage = Math.max(1, Math.ceil(total / assignmentPagination.size))
      if (assignmentPagination.current > maxPage) {
        assignmentPagination.current = maxPage
      }
    }
  )

  onMounted(async () => {
    if (!route.query.tab || normalizeTab(route.query.tab) !== route.query.tab) {
      await router.replace({
        query: {
          ...route.query,
          tab: activeTab.value
        }
      })
    }

    await loadSettings()
    await loadFilterOptions()
    await loadAssignments()
    await loadFuelSalarySettings()
    copyFiltersToSearchParams()
    await getData()
  })
</script>

<style scoped lang="scss">
  .corporation-structures-page {
    &__filter-drawer {
      :deep(.el-drawer__body) {
        display: flex;
        flex-direction: column;
        min-height: 0;
        overflow: hidden;
      }
    }

    &__filter-drawer-body {
      display: flex;
      flex: 1;
      flex-direction: column;
      min-height: 0;
    }

    &__filter-drawer-content {
      flex: 1;
      min-height: 0;
      overflow: auto;
      padding-right: 4px;
    }

    &__filter-drawer-actions {
      position: sticky;
      bottom: 0;
      margin-top: 16px;
      padding-top: 16px;
      background: var(--el-bg-color);
      border-top: 1px solid var(--el-border-color-lighter);
    }
  }
</style>
