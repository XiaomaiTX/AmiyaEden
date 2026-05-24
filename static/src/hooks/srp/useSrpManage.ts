import { ref, reactive, computed, watch, onMounted, h, type Component } from 'vue'
import { useI18n } from 'vue-i18n'
import { useTable } from '@/hooks/core/useTable'
import { useNameResolver } from '@/hooks'
import { useEnterSearch } from '@/hooks/core/useEnterSearch'
import { useUserStore } from '@/store/modules/user'
import { fetchApplicationList, fetchSrpFleetOptions } from '@/api/srp'
import { formatIskSmart, formatTime } from '@utils/common'
import { ElTag, ElTooltip, ElLink } from 'element-plus'

type SrpApp = Api.Srp.Application
type TagType = 'primary' | 'success' | 'warning' | 'info' | 'danger'

export function useSrpManage(callbacks: {
  openReviewDialog: (row: SrpApp, action: 'approve' | 'reject') => void
  handlePayoutAction: (row: SrpApp) => void
  openKmPreview: (row: SrpApp) => void
  components: {
    ArtButtonTable: Component
    ArtCopyButton: Component
  }
}) {
  const { t } = useI18n()
  const { ArtButtonTable, ArtCopyButton } = callbacks.components
  const { getName, resolve: resolveNames } = useNameResolver()
  const { createEnterSearchHandler } = useEnterSearch()
  const userStore = useUserStore()

  const canPayout = computed(() => {
    const roles = userStore.getUserInfo?.roles ?? []
    return roles.some((r) => ['super_admin', 'admin', 'srp'].includes(r))
  })

  // ─── Fleets ───
  const fleets = ref<Api.Srp.FleetOption[]>([])
  const fleetMap = computed(() => new Map(fleets.value.map((f) => [f.fleet_id, f])))
  const loadFleets = async () => {
    try {
      fleets.value = (await fetchSrpFleetOptions()) ?? []
    } catch {
      fleets.value = []
    }
  }

  // ─── Filters & Tabs ───
  const activeTab = ref('pending')
  const payoutMode = ref<Api.Srp.PayoutMode>('fuxi_coin')
  const filter = reactive({ review_status: '', fleet_id: '', keyword: '' })
  const advancedFilter = reactive({
    corporation_id: undefined as number | undefined,
    ship_type_id: undefined as number | undefined,
    solar_system_id: undefined as number | undefined,
    has_recommended_match: undefined as boolean | undefined
  })

  // ─── Formatters ───
  const reviewStatusType = (s: string): TagType =>
    (({ pending: 'info', approved: 'success', rejected: 'danger' }) as Record<string, TagType>)[
      s
    ] ?? 'info'

  const reviewStatusLabel = (s: string) =>
    ({
      submitted: t('srp.status.submitted'),
      approved: t('srp.status.approved'),
      rejected: t('srp.status.rejected')
    })[s as 'submitted' | 'approved' | 'rejected'] ?? s

  const payoutStatusType = (s: string): TagType => (s === 'paid' ? 'success' : 'warning')

  const formatFleetLabel = (f: Api.Srp.FleetOption) =>
    f.fleet_fc_name
      ? `${f.fleet_fc_name}: ${f.fleet_title || f.fleet_id}`
      : f.fleet_title || f.fleet_id

  // ─── Table ───
  const {
    columns,
    columnChecks,
    data,
    loading,
    pagination,
    handleSizeChange,
    handleCurrentChange,
    refreshData,
    getData,
    searchParams
  } = useTable({
    core: {
      apiFn: fetchApplicationList,
      apiParams: {
        current: 1,
        size: 200,
        tab: 'pending',
        sort_by: 'created_at',
        sort_order: 'desc'
      },
      columnsFactory: () => [
        { type: 'index', width: 40, label: '#' },
        {
          prop: 'review_status',
          label: t('srp.manage.columns.review'),
          width: 80,
          formatter: (row: SrpApp) => {
            const tag = h(ElTag, { type: reviewStatusType(row.review_status), size: 'small' }, () =>
              reviewStatusLabel(row.review_status)
            )
            if (row.review_note) {
              return h(ElTooltip, { content: row.review_note, placement: 'top' }, () => tag)
            }
            return tag
          }
        },
        {
          prop: 'payout_status',
          label: t('srp.manage.columns.payout'),
          width: 80,
          formatter: (row: SrpApp) =>
            h(ElTag, { type: payoutStatusType(row.payout_status), size: 'small' }, () =>
              row.payout_status === 'paid' ? t('srp.status.paid') : t('srp.status.notpaid')
            )
        },
        {
          prop: 'nickname',
          label: t('srp.manage.columns.nickname'),
          width: 120,
          showOverflowTooltip: true,
          formatter: (row: SrpApp) =>
            h('span', { class: row.nickname ? '' : 'text-gray-400' }, row.nickname || '-')
        },
        {
          prop: 'character_name',
          label: t('srp.manage.columns.character'),
          width: 140,
          sortable: 'custom',
          formatter: (row: SrpApp) =>
            h('div', { class: 'flex items-center gap-1 min-w-0' }, [
              h('span', { class: 'truncate' }, row.character_name || '-'),
              h(ArtCopyButton, { text: row.character_name })
            ])
        },
        {
          prop: 'ship_type_id',
          label: t('srp.manage.columns.ship'),
          width: 150,
          sortable: 'custom',
          showOverflowTooltip: true,
          formatter: (row: SrpApp) =>
            h('span', {}, getName(row.ship_type_id, `TypeID: ${row.ship_type_id}`, 'type'))
        },
        {
          prop: 'recommended_amount',
          label: t('srp.manage.columns.recommendedAmount'),
          width: 90,
          sortable: 'custom',
          formatter: (row: SrpApp) => h('span', {}, formatIskSmart(row.recommended_amount))
        },
        {
          prop: 'final_amount',
          label: t('srp.manage.columns.finalAmount'),
          width: 90,
          sortable: 'custom',
          formatter: (row: SrpApp) =>
            h('span', { class: 'font-semibold text-blue-600' }, formatIskSmart(row.final_amount))
        },
        {
          prop: 'fleet_title',
          label: t('srp.manage.columns.fleet'),
          width: 150,
          formatter: (row: SrpApp) => {
            if (!row.fleet_id) return h('span', { class: 'text-gray-400' }, '-')
            const fleet = fleetMap.value.get(row.fleet_id)
            const tooltipContent = fleet
              ? formatFleetLabel(fleet)
              : row.fleet_fc_name
                ? `${row.fleet_fc_name}: ${row.fleet_title || row.fleet_id}`
                : row.fleet_title || row.fleet_id
            return h(ElTooltip, { content: tooltipContent, placement: 'top' }, () =>
              h('span', { class: 'cursor-default' }, row.fleet_title || row.fleet_id || '')
            )
          }
        },
        {
          prop: 'solar_system_id',
          label: t('srp.manage.columns.system'),
          width: 128,
          sortable: 'custom',
          showOverflowTooltip: true,
          formatter: (row: SrpApp) =>
            h('span', {}, getName(row.solar_system_id, String(row.solar_system_id), 'solar_system'))
        },
        {
          prop: 'killmail_id',
          label: t('srp.manage.columns.killId'),
          width: 96,
          formatter: (row: SrpApp) =>
            h(
              ElLink,
              {
                href: `https://zkillboard.com/kill/${row.killmail_id}/`,
                target: '_blank',
                type: 'primary'
              },
              () => String(row.killmail_id)
            )
        },
        {
          prop: 'killmail_time',
          label: t('srp.manage.columns.kmTime'),
          width: 160,
          sortable: 'custom',
          formatter: (row: SrpApp) => h('span', {}, formatTime(row.killmail_time))
        },
        {
          prop: 'corporation_id',
          label: t('srp.manage.columns.corporation'),
          width: 150,
          sortable: 'custom',
          showOverflowTooltip: true,
          formatter: (row: SrpApp) =>
            h(
              'span',
              {},
              getName(
                row.corporation_id,
                row.corporation_id ? `ID: ${row.corporation_id}` : '-',
                'esi'
              )
            )
        },
        {
          prop: 'alliance_id',
          label: t('srp.manage.columns.alliance'),
          width: 150,
          showOverflowTooltip: true,
          formatter: (row: SrpApp) =>
            h(
              'span',
              {},
              getName(row.alliance_id, row.alliance_id ? `ID: ${row.alliance_id}` : '-', 'esi')
            )
        },
        {
          prop: 'note',
          label: t('srp.manage.columns.note'),
          minWidth: 150,
          showOverflowTooltip: true,
          formatter: (row: SrpApp) =>
            h('span', { class: row.note ? '' : 'text-gray-400' }, row.note || '-')
        },
        {
          prop: 'review_note',
          label: t('srp.manage.columns.reviewNote'),
          minWidth: 170,
          showOverflowTooltip: true,
          formatter: (row: SrpApp) =>
            h('span', { class: row.review_note ? '' : 'text-gray-400' }, row.review_note || '-')
        },
        {
          prop: 'last_actor_nickname',
          label: t('srp.manage.columns.lastActor'),
          width: 130,
          showOverflowTooltip: true,
          formatter: (row: SrpApp) =>
            h(
              'span',
              { class: row.last_actor_nickname ? '' : 'text-gray-400' },
              row.last_actor_nickname || '-'
            )
        },
        {
          prop: 'actions',
          label: t('srp.manage.columns.action'),
          width: 280,
          fixed: 'right',
          formatter: (row: SrpApp) => {
            const btns: ReturnType<typeof h>[] = [
              h(ArtButtonTable, { type: 'view', onClick: () => callbacks.openKmPreview(row) })
            ]
            if (row.review_status === 'submitted') {
              btns.push(
                h(ArtButtonTable, {
                  label: t('srp.manage.approveBtn'),
                  elType: 'success',
                  onClick: () => callbacks.openReviewDialog(row, 'approve')
                }),
                h(ArtButtonTable, {
                  label: t('srp.manage.rejectBtn'),
                  elType: 'danger',
                  onClick: () => callbacks.openReviewDialog(row, 'reject')
                })
              )
            } else if (row.review_status === 'approved' && row.payout_status === 'notpaid') {
              if (canPayout.value) {
                btns.push(
                  h(ArtButtonTable, {
                    label: t('srp.manage.payoutBtn'),
                    elType: 'primary',
                    onClick: () => callbacks.handlePayoutAction(row)
                  })
                )
              }
              btns.push(
                h(ArtButtonTable, {
                  label: t('srp.manage.editBtn'),
                  elType: 'warning',
                  onClick: () => callbacks.openReviewDialog(row, 'approve')
                }),
                h(ArtButtonTable, {
                  label: t('srp.manage.reRejectBtn'),
                  elType: 'danger',
                  onClick: () => callbacks.openReviewDialog(row, 'reject')
                })
              )
            } else if (row.review_status === 'rejected') {
              btns.push(
                h(ArtButtonTable, {
                  label: t('srp.manage.reApproveBtn'),
                  elType: 'success',
                  onClick: () => callbacks.openReviewDialog(row, 'approve')
                })
              )
            }
            return h(
              'div',
              { class: 'flex items-center gap-1 flex-nowrap whitespace-nowrap' },
              btns
            )
          }
        }
      ]
    }
  })

  // ─── Name Resolution ───
  watch(data, async (list) => {
    if (list.length) await resolveManageNames(list)
  })

  const resolveManageNames = async (list: SrpApp[]) => {
    const typeIds = new Set<number>()
    const solarIds = new Set<number>()
    const esiIds = new Set<number>()
    for (const app of list) {
      if (app.ship_type_id) typeIds.add(app.ship_type_id)
      if (app.solar_system_id) solarIds.add(app.solar_system_id)
      if (app.corporation_id) esiIds.add(app.corporation_id)
      if (app.alliance_id) esiIds.add(app.alliance_id)
    }
    await resolveNames({
      ids: {
        ...(typeIds.size ? { type: [...typeIds] } : {}),
        ...(solarIds.size ? { solar_system: [...solarIds] } : {})
      },
      esi: esiIds.size ? [...esiIds] : undefined
    })
  }

  // ─── Search & Filter ───
  const handleSearch = () => {
    Object.assign(searchParams, {
      current: 1,
      tab: activeTab.value,
      review_status: filter.review_status || undefined,
      fleet_id: filter.fleet_id || undefined,
      keyword: filter.keyword.trim() || undefined,
      corporation_id: advancedFilter.corporation_id,
      ship_type_id: advancedFilter.ship_type_id,
      solar_system_id: advancedFilter.solar_system_id,
      has_recommended_match: advancedFilter.has_recommended_match
    })
    getData()
  }

  const handleKeywordSearchKeyup = createEnterSearchHandler(handleSearch)

  const resetFilter = () => {
    filter.review_status = ''
    filter.fleet_id = ''
    filter.keyword = ''
    advancedFilter.corporation_id = undefined
    advancedFilter.ship_type_id = undefined
    advancedFilter.solar_system_id = undefined
    advancedFilter.has_recommended_match = undefined
    Object.assign(searchParams, {
      current: 1,
      tab: activeTab.value,
      review_status: undefined,
      fleet_id: undefined,
      keyword: undefined,
      corporation_id: undefined,
      ship_type_id: undefined,
      solar_system_id: undefined,
      has_recommended_match: undefined,
      sort_by: 'created_at',
      sort_order: 'desc'
    })
    getData()
  }

  const handleTabChange = () => {
    filter.review_status = ''
    filter.fleet_id = ''
    filter.keyword = ''
    advancedFilter.corporation_id = undefined
    advancedFilter.ship_type_id = undefined
    advancedFilter.solar_system_id = undefined
    advancedFilter.has_recommended_match = undefined
    Object.assign(searchParams, {
      current: 1,
      tab: activeTab.value,
      review_status: undefined,
      fleet_id: undefined,
      keyword: undefined,
      corporation_id: undefined,
      ship_type_id: undefined,
      solar_system_id: undefined,
      has_recommended_match: undefined,
      sort_by: 'created_at',
      sort_order: 'desc'
    })
    getData()
  }

  const handleSortChange = (sort: { prop?: string; order?: 'ascending' | 'descending' | null }) => {
    if (!sort?.prop || !sort.order) {
      searchParams.sort_by = 'created_at'
      searchParams.sort_order = 'desc'
    } else {
      searchParams.sort_by = sort.prop as NonNullable<Api.Srp.ApplicationSearchParams['sort_by']>
      searchParams.sort_order = sort.order === 'descending' ? 'desc' : 'asc'
    }
    searchParams.current = 1
    getData()
  }

  const corporationOptions = computed(() => {
    const ids = Array.from(
      new Set(
        data.value.map((app) => app.corporation_id).filter((id) => Number.isFinite(id) && id > 0)
      )
    ).sort((a, b) => a - b)
    return ids.map((id) => ({
      value: id,
      label: getName(id, `ID: ${id}`, 'esi')
    }))
  })

  const shipTypeOptions = computed(() => {
    const ids = Array.from(
      new Set(
        data.value.map((app) => app.ship_type_id).filter((id) => Number.isFinite(id) && id > 0)
      )
    ).sort((a, b) => a - b)
    return ids.map((id) => ({
      value: id,
      label: getName(id, `TypeID: ${id}`, 'type')
    }))
  })

  const solarSystemOptions = computed(() => {
    const ids = Array.from(
      new Set(
        data.value.map((app) => app.solar_system_id).filter((id) => Number.isFinite(id) && id > 0)
      )
    ).sort((a, b) => a - b)
    return ids.map((id) => ({
      value: id,
      label: getName(id, String(id), 'solar_system')
    }))
  })

  // ─── Export ───
  const manageExportHeaders = computed(() => ({
    character_name: t('srp.manage.exportColumns.character'),
    ship_name: t('srp.manage.exportColumns.ship'),
    solar_system: t('srp.manage.exportColumns.system'),
    killmail_id: t('srp.manage.exportColumns.killId'),
    killmail_time: t('srp.manage.exportColumns.kmTime'),
    corporation: t('srp.manage.exportColumns.corporation'),
    alliance: t('srp.manage.exportColumns.alliance'),
    fleet_title: t('srp.manage.exportColumns.fleet'),
    fleet_fc_name: t('srp.manage.exportColumns.fc'),
    note: t('srp.manage.exportColumns.note'),
    recommended_amount: t('srp.manage.exportColumns.recommendedAmount'),
    final_amount: t('srp.manage.exportColumns.finalAmount'),
    review_status: t('srp.manage.exportColumns.reviewStatus'),
    review_note: t('srp.manage.exportColumns.reviewNote'),
    last_actor_nickname: t('srp.manage.exportColumns.lastActor'),
    payout_status: t('srp.manage.exportColumns.payoutStatus')
  }))

  const exportManageData = computed(() =>
    data.value.map((app) => ({
      character_name: app.character_name,
      ship_name: getName(app.ship_type_id, `TypeID: ${app.ship_type_id}`, 'type'),
      solar_system: getName(app.solar_system_id, String(app.solar_system_id), 'solar_system'),
      killmail_id: app.killmail_id,
      killmail_time: formatTime(app.killmail_time),
      corporation: getName(
        app.corporation_id,
        app.corporation_id ? `ID: ${app.corporation_id}` : '-',
        'esi'
      ),
      alliance: getName(app.alliance_id, app.alliance_id ? `ID: ${app.alliance_id}` : '-', 'esi'),
      fleet_title: app.fleet_title || '-',
      fleet_fc_name: app.fleet_fc_name || '-',
      note: app.note || '-',
      recommended_amount: app.recommended_amount,
      final_amount: app.final_amount,
      review_status: reviewStatusLabel(app.review_status),
      review_note: app.review_note || '-',
      last_actor_nickname: app.last_actor_nickname || '-',
      payout_status: app.payout_status === 'paid' ? t('srp.status.paid') : t('srp.status.notpaid')
    }))
  )

  // ─── KM Preview ───
  const kmPreviewVisible = ref(false)
  const previewKillmailId = ref(0)
  const openKmPreview = (row: SrpApp) => {
    previewKillmailId.value = row.killmail_id
    kmPreviewVisible.value = true
  }

  onMounted(() => {
    loadFleets()
  })

  return {
    // permissions
    canPayout,
    // fleets
    fleets,
    fleetMap,
    formatFleetLabel,
    // filters
    activeTab,
    payoutMode,
    filter,
    advancedFilter,
    handleSearch,
    handleKeywordSearchKeyup,
    resetFilter,
    handleTabChange,
    handleSortChange,
    corporationOptions,
    shipTypeOptions,
    solarSystemOptions,
    // table
    columns,
    columnChecks,
    data,
    loading,
    pagination,
    handleSizeChange,
    handleCurrentChange,
    refreshData,
    // formatters
    formatISK: formatIskSmart,
    // export
    manageExportHeaders,
    exportManageData,
    // km preview
    kmPreviewVisible,
    previewKillmailId,
    openKmPreview
  }
}
