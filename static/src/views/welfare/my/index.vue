<!-- 我的福利页面 -->
<template>
  <div class="welfare-my-page art-full-height">
    <ElCard class="art-table-card" shadow="never">
      <ElAlert type="success" :closable="false" class="mb-4" show-icon>
        <p class="break-all">
          {{ t('welfareMy.skillPlanningAlertPrefix') }}
          <RouterLink class="text-theme font-medium" :to="{ name: 'SkillPlanCompletionCheck' }">
            {{ '技能规划-检查完成度' }}
          </RouterLink>
        </p>
      </ElAlert>

      <ElAlert type="info" :closable="true" class="mb-4" show-icon>
        <p class="break-all">{{ t('welfareMy.multicharRewardBanner') }}</p>
      </ElAlert>

      <ElTabs v-model="activeTab" @tab-change="handleTabChange">
        <!-- 申请福利 -->
        <ElTabPane :label="t('welfareMy.applyTab')" name="apply">
          <div class="mb-3 flex items-center gap-3 flex-wrap">
            <ElSelect
              v-model="roleFilter"
              clearable
              style="width: 200px"
              :placeholder="t('welfareMy.filterRole')"
              @change="handleEligibleFilterChange"
            >
              <ElOption
                v-for="option in roleFilterOptions"
                :key="option.value"
                :label="option.label"
                :value="option.value"
              />
            </ElSelect>
            <ElSelect
              v-model="naturalPersonFilter"
              style="width: 160px"
              :placeholder="t('welfareMy.filterNaturalPerson')"
              @change="handleEligibleFilterChange"
            >
              <ElOption :label="t('welfareMy.filterAll')" value="" />
              <ElOption
                :label="t('welfareMy.filterNaturalPersonOnly')"
                :value="NATURAL_PERSON_FILTER_VALUE"
              />
            </ElSelect>
            <ElSelect
              v-model="welfareNameFilter"
              clearable
              style="width: 220px"
              :placeholder="t('welfareMy.filterWelfareName')"
              @change="handleEligibleFilterChange"
            >
              <ElOption
                v-for="option in welfareNameFilterOptions"
                :key="option.value"
                :label="option.label"
                :value="option.value"
              />
            </ElSelect>
            <ElButton @click="handleEligibleFilterReset">{{ t('common.reset') }}</ElButton>
          </div>
          <ArtTable
            :loading="eligibleLoading"
            :data="pagedEligibleRows"
            :columns="eligibleColumns"
            :empty-text="t('welfareMy.noEligibleWelfares')"
            :row-class-name="eligibleRowClassName"
            :pagination="eligiblePagination"
            @pagination:size-change="handleEligibleSizeChange"
            @pagination:current-change="handleEligibleCurrentChange"
          />
        </ElTabPane>
        <!-- 已领取福利 -->
        <ElTabPane :label="t('welfareMy.applicationsTab')" name="applications">
          <ArtTable
            :loading="applicationsLoading"
            :data="applications"
            :columns="applicationColumns"
            :empty-text="t('welfareMy.noApplications')"
            :pagination="applicationPagination"
            @pagination:size-change="handleApplicationSizeChange"
            @pagination:current-change="handleApplicationCurrentChange"
          />
        </ElTabPane>
      </ElTabs>
    </ElCard>

    <!-- 证明图片上传对话框 -->
    <ElDialog
      v-model="evidenceDialogVisible"
      :title="t('welfareMy.evidenceDialogTitle')"
      width="480px"
      destroy-on-close
    >
      <div class="flex flex-col gap-3">
        <p class="text-sm text-gray-500">{{ t('welfareMy.evidenceDialogHint') }}</p>
        <div v-if="pendingApplyRow?.exampleEvidence" class="flex flex-col gap-1">
          <span class="text-xs text-gray-400">{{ t('welfareMy.exampleEvidenceLabel') }}</span>
          <img
            :src="pendingApplyRow.exampleEvidence"
            class="rounded border"
            style="max-height: 160px; max-width: 100%; object-fit: contain"
          />
        </div>
        <ElUpload
          :show-file-list="false"
          accept="image/*"
          :before-upload="handleEvidenceFileUpload"
        >
          <ElButton size="small" :loading="evidenceUploading">
            {{ t('welfareMy.uploadEvidenceBtn') }}
          </ElButton>
        </ElUpload>
        <img
          v-if="evidenceImageUrl"
          :src="evidenceImageUrl"
          class="rounded border"
          style="max-height: 160px; max-width: 100%; object-fit: contain"
        />
      </div>
      <template #footer>
        <ElButton @click="evidenceDialogVisible = false">{{ t('common.cancel') }}</ElButton>
        <ElButton
          type="primary"
          :disabled="!evidenceImageUrl"
          :loading="applyLoading"
          @click="handleEvidenceConfirm"
        >
          {{ t('common.confirm') }}
        </ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { ElTag, ElButton, ElUpload, ElMessage, ElTooltip, ElSelect, ElOption } from 'element-plus'
  import { useTable } from '@/hooks/core/useTable'
  import { formatTime } from '@utils/common'
  import {
    getEligibleWelfares,
    applyForWelfare,
    getMyApplications,
    uploadWelfareEvidence
  } from '@/api/welfare'
  import { sortEligibleRows } from './eligibleRows'
  import {
    buildRoleFilterOptions,
    buildWelfareNameFilterOptions,
    filterEligibleRows,
    paginateEligibleRows,
    type EligibleFilters
  } from './eligibleFilters'
  import { formatWelfareIneligibleReason } from './ineligibleReason'
  import { useI18n } from 'vue-i18n'

  defineOptions({ name: 'WelfareMy' })
  const { t } = useI18n()

  // ─── Tab state ───
  const activeTab = ref('apply')
  const NATURAL_PERSON_FILTER_VALUE = 'per_user'
  const naturalPersonRoleLabel = computed(() => t('welfareMy.currentUserNaturalPerson'))

  // ─── 申请福利 Tab ───

  // 将 eligible welfares 展平为表格行
  // per_user: 一条福利一行
  // per_character: 每个可申请人物一行
  interface EligibleRow {
    welfareId: number
    welfareName: string
    description: string
    skillPlanNames: string[]
    distMode: string
    canApplyNow: boolean
    ineligibleReason?:
      | 'pap'
      | 'skill'
      | 'pap_skill'
      | 'legion_years'
      | 'pap_legion_years'
      | 'skill_legion_years'
      | 'pap_skill_legion_years'
    characterId?: number
    characterName?: string
    requireEvidence: boolean
    exampleEvidence: string
    isNaturalPersonRow: boolean
    roleFilterValue: string
  }

  const roleFilter = ref('')
  const naturalPersonFilter = ref<EligibleFilters['naturalPersonFilter']>('')
  const welfareNameFilter = ref('')
  const eligibleAllRows = ref<EligibleRow[]>([])

  const roleFilterOptions = computed(() =>
    buildRoleFilterOptions(eligibleAllRows.value, naturalPersonRoleLabel.value)
  )
  const welfareNameFilterOptions = computed(() =>
    buildWelfareNameFilterOptions(eligibleAllRows.value)
  )

  function flattenEligibleWelfares(welfares: Api.Welfare.EligibleWelfare[]): EligibleRow[] {
    const rows: EligibleRow[] = []
    for (const w of welfares) {
      const shared = {
        welfareId: w.id,
        welfareName: w.name,
        description: w.description,
        skillPlanNames: w.skill_plan_names ?? [],
        distMode: w.dist_mode,
        requireEvidence: w.require_evidence,
        exampleEvidence: w.example_evidence
      }
      if (w.dist_mode === 'per_user') {
        rows.push({
          ...shared,
          canApplyNow: w.can_apply_now,
          ineligibleReason: w.ineligible_reason,
          isNaturalPersonRow: true,
          roleFilterValue: naturalPersonRoleLabel.value
        })
      } else {
        for (const char of w.eligible_characters) {
          rows.push({
            ...shared,
            canApplyNow: char.can_apply_now,
            ineligibleReason: char.ineligible_reason,
            characterId: char.character_id,
            characterName: char.character_name,
            isNaturalPersonRow: false,
            roleFilterValue: char.character_name
          })
        }
      }
    }
    return sortEligibleRows(rows)
  }

  async function getEligibleWelfaresPage(
    params: Api.Common.CommonSearchParams
  ): Promise<Api.Common.PaginatedResponse<EligibleRow>> {
    const welfares = await getEligibleWelfares()
    const rows = flattenEligibleWelfares(welfares ?? [])
    eligibleAllRows.value = rows
    const filters: EligibleFilters = {
      roleFilter: roleFilter.value,
      naturalPersonFilter: naturalPersonFilter.value,
      welfareNameFilter: welfareNameFilter.value
    }
    const filteredRows = filterEligibleRows(rows, filters)
    const { current, size } = params
    return {
      list: paginateEligibleRows(filteredRows, current, size),
      total: filteredRows.length,
      page: current,
      pageSize: size
    }
  }

  const {
    data: pagedEligibleRows,
    loading: eligibleLoading,
    pagination: eligiblePagination,
    handleSizeChange: handleEligibleSizeChange,
    handleCurrentChange: handleEligibleCurrentChange,
    getData: loadEligibleWelfares
  } = useTable({
    core: {
      apiFn: getEligibleWelfaresPage,
      apiParams: { current: 1, size: 10 },
      immediate: false
    }
  })

  function handleEligibleFilterChange() {
    loadEligibleWelfares()
  }

  function handleEligibleFilterReset() {
    roleFilter.value = ''
    naturalPersonFilter.value = ''
    welfareNameFilter.value = ''
    loadEligibleWelfares()
  }

  const DIST_MODE_CONFIG = computed(
    () =>
      ({
        per_user: { label: t('welfareMy.distModePerUser'), type: 'primary' },
        per_character: { label: t('welfareMy.distModePerCharacter'), type: 'warning' }
      }) as Record<string, { label: string; type: string }>
  )

  const reasonMessages = computed(() => ({
    pap: t('welfareMy.ineligibleReasonPap'),
    skill: t('welfareMy.ineligibleReasonSkill'),
    legionYears: t('welfareMy.ineligibleReasonLegionYears'),
    skillPlan: (plans: string) => t('welfareMy.ineligibleReasonSkillPlan', { plans }),
    planSeparator: t('welfareMy.skillPlanJoiner'),
    reasonSeparator: t('welfareMy.ineligibleReasonJoiner')
  }))

  function getIneligibleReasonContent(row: EligibleRow) {
    return formatWelfareIneligibleReason(
      row.ineligibleReason,
      row.skillPlanNames,
      reasonMessages.value
    )
  }

  const eligibleColumns = computed(() => [
    {
      prop: 'welfareName',
      label: t('welfareMy.welfareName'),
      minWidth: 80,
      showOverflowTooltip: true
    },
    {
      prop: 'description',
      label: t('welfareMy.description'),
      minWidth: 200,
      showOverflowTooltip: true,
      formatter: (row: EligibleRow) => row.description || '-'
    },
    {
      prop: 'distMode',
      label: t('welfareMy.deliveryMode'),
      width: 120,
      formatter: (row: EligibleRow) => {
        const cfg = DIST_MODE_CONFIG.value[row.distMode] ?? {
          label: row.distMode,
          type: 'info'
        }
        return h(ElTag, { type: cfg.type as any, size: 'small', effect: 'plain' }, () => cfg.label)
      }
    },
    {
      prop: 'eligibility',
      label: t('welfareMy.eligibility'),
      width: 140,
      formatter: (row: EligibleRow) => {
        const tag = h(
          ElTag,
          {
            type: row.canApplyNow ? 'success' : 'info',
            size: 'small',
            effect: row.canApplyNow ? 'light' : 'plain'
          },
          () => (row.canApplyNow ? t('welfareMy.eligibilityNow') : t('welfareMy.eligibilityFuture'))
        )
        if (row.canApplyNow || !row.ineligibleReason) return tag
        return h(
          ElTooltip,
          { content: getIneligibleReasonContent(row), placement: 'top' },
          () => tag
        )
      }
    },
    {
      prop: 'characterName',
      label: t('welfareMy.characterName'),
      width: 160,
      formatter: (row: EligibleRow) => row.characterName || '-'
    },
    {
      prop: 'actions',
      label: '',
      width: 100,
      fixed: 'right' as const,
      formatter: (row: EligibleRow) =>
        h(
          ElButton,
          {
            type: row.canApplyNow ? 'primary' : 'info',
            size: 'small',
            plain: !row.canApplyNow,
            disabled: !row.canApplyNow,
            onClick: () => handleApply(row)
          },
          () => (row.canApplyNow ? t('welfareMy.applyBtn') : t('welfareMy.futureApplyBtn'))
        )
    }
  ])

  function eligibleRowClassName({ row }: { row: Record<string, any>; rowIndex: number }) {
    return (row as EligibleRow).canApplyNow ? '' : 'welfare-future-row'
  }

  // ─── 证明图片对话框 ───
  const evidenceDialogVisible = ref(false)
  const pendingApplyRow = ref<EligibleRow | null>(null)
  const evidenceImageUrl = ref('')
  const evidenceUploading = ref(false)
  const applyLoading = ref(false)

  function handleApply(row: EligibleRow) {
    if (row.requireEvidence) {
      pendingApplyRow.value = row
      evidenceImageUrl.value = ''
      evidenceDialogVisible.value = true
    } else {
      submitApply(row, '')
    }
  }

  async function handleEvidenceFileUpload(file: File) {
    evidenceUploading.value = true
    try {
      const res = await uploadWelfareEvidence(file)
      evidenceImageUrl.value = res.url
    } catch (e: any) {
      ElMessage.error(e?.message ?? t('welfareMy.applyFailed'))
    } finally {
      evidenceUploading.value = false
    }
    return false
  }

  async function handleEvidenceConfirm() {
    if (!pendingApplyRow.value || !evidenceImageUrl.value) return
    applyLoading.value = true
    try {
      await submitApply(pendingApplyRow.value, evidenceImageUrl.value)
      evidenceDialogVisible.value = false
    } finally {
      applyLoading.value = false
    }
  }

  async function submitApply(row: EligibleRow, evidenceImage: string) {
    try {
      await applyForWelfare({
        welfare_id: row.welfareId,
        character_id: row.characterId,
        evidence_image: evidenceImage || undefined
      })
      ElMessage.success(t('welfareMy.applySuccess'))
      loadEligibleWelfares()
      loadApplications()
    } catch (e: any) {
      ElMessage.error(e?.message ?? t('welfareMy.applyFailed'))
    }
  }

  // ─── 已领取福利 Tab ───
  const {
    data: applications,
    loading: applicationsLoading,
    pagination: applicationPagination,
    handleSizeChange: handleApplicationSizeChange,
    handleCurrentChange: handleApplicationCurrentChange,
    getData: loadApplications
  } = useTable({
    core: {
      apiFn: getMyApplications,
      apiParams: { current: 1, size: 10 },
      immediate: false
    }
  })

  const STATUS_CONFIG = computed(
    () =>
      ({
        requested: { label: t('welfareMy.statusRequested'), type: 'warning' },
        delivered: { label: t('welfareMy.statusDelivered'), type: 'success' },
        rejected: { label: t('welfareMy.statusRejected'), type: 'danger' }
      }) as Record<string, { label: string; type: string }>
  )

  const applicationColumns = computed(() => [
    {
      prop: 'welfare_name',
      label: t('welfareMy.welfareName'),
      minWidth: 70,
      showOverflowTooltip: true
    },
    {
      prop: 'character_name',
      label: t('welfareMy.characterName'),
      minWidth: 90
    },
    {
      prop: 'status',
      label: t('common.status'),
      width: 100,
      formatter: (row: Api.Welfare.MyApplication) => {
        const cfg = STATUS_CONFIG.value[row.status] ?? {
          label: row.status,
          type: 'info'
        }
        return h(ElTag, { type: cfg.type as any, size: 'small', effect: 'plain' }, () => cfg.label)
      }
    },
    {
      prop: 'reviewer_name',
      label: t('welfareMy.reviewerName'),
      width: 130,
      showOverflowTooltip: true,
      formatter: (row: Api.Welfare.MyApplication) => row.reviewer_name || '-'
    },
    {
      prop: 'created_at',
      label: t('welfareMy.appliedAt'),
      width: 170,
      formatter: (row: Api.Welfare.MyApplication) => formatTime(row.created_at)
    },
    {
      prop: 'reviewed_at',
      label: t('welfareMy.reviewedAt'),
      width: 170,
      formatter: (row: Api.Welfare.MyApplication) => formatTime(row.reviewed_at)
    }
  ])

  // ─── Tab switch & init ───
  function handleTabChange(tab: string | number) {
    if (tab === 'applications') {
      loadApplications()
    } else {
      loadEligibleWelfares()
    }
  }

  onMounted(() => {
    loadEligibleWelfares()
    loadApplications()
  })
</script>

<style scoped lang="scss">
  .welfare-my-page :deep(.welfare-future-row) {
    opacity: 0.58;
  }

  .welfare-my-page {
    :deep(.el-card__body) {
      display: flex;
      flex-direction: column;
      min-height: 0;
    }

    :deep(.el-tabs) {
      display: flex;
      flex: 1;
      flex-direction: column;
      min-height: 0;
    }

    :deep(.el-tabs__content) {
      flex: 1;
      min-height: 0;
      overflow: hidden;
    }

    :deep(.el-tab-pane) {
      display: flex;
      flex-direction: column;
      height: 100%;
      min-height: 0;
    }
  }
</style>
