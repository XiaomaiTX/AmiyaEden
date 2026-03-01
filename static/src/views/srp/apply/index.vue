<!-- SRP 补损申请页面 -->
<template>
  <div class="srp-apply-page art-full-height">
    <!-- 申请补损 -->
    <ElCard class="apply-card" shadow="never">
      <template #header>
        <h2 class="section-title">{{ $t('srp.apply.formTitle') }}</h2>
      </template>

      <ElAlert type="success" :closable="false" class="mb-4" show-icon>
        <p>{{ $t('srp.apply.infoText') }}</p>
        <ElLink type="primary" class="mt-1">{{ $t('srp.apply.faqLink') }}</ElLink>
      </ElAlert>

      <ElForm ref="formRef" :model="form" :rules="rules" label-position="top">
        <ElFormItem :label="$t('srp.apply.selectCharacter')" prop="character_id">
          <ElSelect
            v-model="form.character_id"
            :placeholder="$t('srp.apply.selectCharacter')"
            style="width: 100%"
            @change="onCharacterChange"
          >
            <ElOption
              v-for="c in characters"
              :key="c.character_id"
              :label="c.character_name"
              :value="c.character_id"
            />
          </ElSelect>
        </ElFormItem>

        <ElFormItem :label="$t('srp.apply.killmail')" prop="killmail_id">
          <div class="km-select-row">
            <ElSelect
              v-model="form.killmail_id"
              :placeholder="
                form.character_id ? $t('srp.apply.selectKillmail') : $t('srp.apply.noKmHint')
              "
              :loading="kmLoading"
              :loading-text="$t('srp.apply.loadingKm')"
              class="flex-1"
              filterable
              :disabled="!form.character_id"
              @change="onKillmailSelect"
            >
              <ElOption
                v-for="km in fleetKillmails"
                :key="km.killmail_id"
                :value="km.killmail_id"
                :label="formatKmLabel(km)"
              />
            </ElSelect>
            <ElButton :disabled="!form.killmail_id" @click="openKmPreview">
              <el-icon class="mr-1"><View /></el-icon>
              {{ $t('srp.apply.previewKm') }}
            </ElButton>
          </div>
        </ElFormItem>

        <div class="fleet-info-section">
          <h3 class="fleet-info-label">{{ $t('srp.apply.fleetInfo') }}</h3>
          <ElFormItem>
            <ElSelect
              v-model="form.fleet_id"
              :placeholder="$t('srp.apply.selectFleet')"
              clearable
              filterable
              style="width: 100%"
              @change="onFleetChange"
            >
              <ElOption key="__other__" :label="$t('srp.apply.otherAction')" value="__other__" />
              <ElOption v-for="f in fleets" :key="f.id" :label="f.title" :value="f.id" />
            </ElSelect>
          </ElFormItem>
          <ElFormItem v-if="showNoteArea" :prop="noteRequired ? 'note' : ''">
            <ElInput
              v-model="form.note"
              type="textarea"
              :rows="3"
              :placeholder="$t('srp.apply.fleetNotePlaceholder')"
            />
          </ElFormItem>
        </div>

        <div class="flex justify-end mt-2">
          <ElButton type="success" :loading="submitting" @click="handleSubmit">
            {{ $t('srp.apply.submitBtnText') }}
          </ElButton>
        </div>
      </ElForm>
    </ElCard>

    <!-- 我的补损申请 -->
    <ElCard class="art-table-card mt-4" shadow="never">
      <template #header>
        <div class="table-header-bar">
          <h2 class="text-base font-medium">{{ $t('srp.apply.title') }}</h2>
        </div>
      </template>

      <ElTable v-loading="loading" :data="applications" stripe border style="width: 100%">
        <ElTableColumn prop="killmail_id" :label="$t('srp.apply.columns.id')">
          <template #default="{ row }">
            <ElLink
              :href="`https://zkillboard.com/kill/${row.killmail_id}/`"
              target="_blank"
              type="primary"
            >
              {{ row.killmail_id }}
            </ElLink>
          </template>
        </ElTableColumn>
        <ElTableColumn
          prop="character_name"
          :label="$t('srp.apply.columns.character')"
          width="130"
        />
        <ElTableColumn prop="ship_type_id" :label="$t('srp.apply.columns.ship')">
          <template #default="{ row }">
            {{ getName(row.ship_type_id, `TypeID: ${row.ship_type_id}`) }}
          </template>
        </ElTableColumn>
        <ElTableColumn
          prop="recommended_amount"
          :label="$t('srp.apply.columns.estimatedValue')"
          width="130"
          align="right"
        >
          <template #default="{ row }"> {{ formatISK(row.recommended_amount) }} ISK </template>
        </ElTableColumn>
        <ElTableColumn
          prop="review_status"
          :label="$t('srp.apply.columns.reviewStatus')"
          width="110"
          align="center"
        >
          <template #default="{ row }">
            <ElTag :type="reviewStatusType(row.review_status)" size="small">
              {{ reviewStatusLabel(row.review_status) }}
            </ElTag>
          </template>
        </ElTableColumn>
        <ElTableColumn
          prop="final_amount"
          :label="$t('srp.apply.columns.actualAmount')"
          align="right"
        >
          <template #default="{ row }">
            <template v-if="row.final_amount > 0"> {{ formatISK(row.final_amount) }} ISK </template>
            <span v-else>-</span>
          </template>
        </ElTableColumn>
        <ElTableColumn prop="payout_status" :label="$t('srp.apply.columns.paid')" align="center">
          <template #default="{ row }">
            {{ row.payout_status === 'paid' ? $t('srp.status.paid') : $t('srp.status.unpaid') }}
          </template>
        </ElTableColumn>
        <ElTableColumn
          :label="$t('srp.apply.columns.action')"
          width="100"
          align="center"
          fixed="right"
        >
          <template #default="{ row }">
            <ElButton type="primary" link size="small" @click="openTableKmPreview(row)">
              <el-icon><View /></el-icon>
            </ElButton>
          </template>
        </ElTableColumn>
      </ElTable>

      <div v-if="pagination.total > 0" class="pagination-wrapper">
        <ElPagination
          v-model:current-page="pagination.current"
          v-model:page-size="pagination.size"
          :total="pagination.total"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next, jumper"
          background
          @size-change="
            () => {
              pagination.current = 1
              loadApplications()
            }
          "
          @current-change="loadApplications"
        />
      </div>
    </ElCard>

    <!-- KM 预览弹窗 -->
    <KmPreviewDialog v-model="kmPreviewVisible" :killmail-id="previewKillmailId" />
  </div>
</template>

<script setup lang="ts">
  import { useRoute } from 'vue-router'
  import { useI18n } from 'vue-i18n'
  import { View } from '@element-plus/icons-vue'
  import {
    ElCard,
    ElTable,
    ElTableColumn,
    ElTag,
    ElButton,
    ElPagination,
    ElForm,
    ElFormItem,
    ElSelect,
    ElOption,
    ElInput,
    ElLink,
    ElMessage,
    ElAlert,
    type FormInstance,
    type FormRules
  } from 'element-plus'
  import KmPreviewDialog from '@/components/business/KmPreviewDialog.vue'
  import { fetchMyCharacters } from '@/api/auth'
  import { fetchFleetList } from '@/api/fleet'
  import {
    submitApplication,
    fetchMyApplications,
    fetchFleetKillmails,
    fetchMyKillmails
  } from '@/api/srp'
  import { useNameResolver } from '@/hooks'

  defineOptions({ name: 'SrpApply' })

  const OTHER_ACTION = '__other__'
  const route = useRoute()
  const { t } = useI18n()
  const { getName, resolve: resolveNames } = useNameResolver()

  /* ── 申请列表 ── */
  const applications = ref<Api.Srp.Application[]>([])
  const loading = ref(false)
  const pagination = reactive({ current: 1, size: 10, total: 0 })

  const resolveApplicationNames = async (list: Api.Srp.Application[]) => {
    const typeIds = new Set<number>()
    const solarIds = new Set<number>()
    for (const app of list) {
      if (app.ship_type_id) typeIds.add(app.ship_type_id)
      if (app.solar_system_id) solarIds.add(app.solar_system_id)
    }
    await resolveNames({
      ids: {
        ...(typeIds.size ? { type: [...typeIds] } : {}),
        ...(solarIds.size ? { solar_system: [...solarIds] } : {})
      }
    })
  }

  const loadApplications = async () => {
    loading.value = true
    try {
      const res = await fetchMyApplications({
        current: pagination.current,
        size: pagination.size
      })
      applications.value = res?.records ?? []
      pagination.total = res?.total ?? 0
      if (applications.value.length) await resolveApplicationNames(applications.value)
    } catch {
      applications.value = []
    } finally {
      loading.value = false
    }
  }

  /* ── 角色 & 舰队 ── */
  const characters = ref<Api.Auth.EveCharacter[]>([])
  const loadCharacters = async () => {
    try {
      characters.value = (await fetchMyCharacters()) ?? []
    } catch {
      characters.value = []
    }
  }

  const fleets = ref<Api.Fleet.FleetItem[]>([])
  const loadFleets = async () => {
    try {
      const res = await fetchFleetList({ size: 200 } as any)
      fleets.value = res?.records ?? []
    } catch {
      fleets.value = []
    }
  }

  /* ── 表单 ── */
  const formRef = ref<FormInstance>()
  const submitting = ref(false)
  const kmLoading = ref(false)
  const fleetKillmails = ref<Api.Srp.FleetKillmailItem[]>([])

  const form = reactive({
    character_id: 0,
    fleet_id: '',
    killmail_id: 0,
    note: '',
    final_amount: 0,
    recommended_amount: 0
  })

  const showNoteArea = computed(() => form.fleet_id === OTHER_ACTION)
  const noteRequired = computed(() => form.fleet_id === OTHER_ACTION || !form.fleet_id)

  const rules: FormRules = {
    character_id: [{ required: true, message: t('srp.apply.selectCharacter'), trigger: 'change' }],
    killmail_id: [
      {
        required: true,
        validator: (_r, v, cb) => (v > 0 ? cb() : cb(new Error(t('srp.apply.selectKillmail')))),
        trigger: 'change'
      }
    ],
    note: [
      {
        validator: (_r: any, v: string, cb: (e?: Error) => void) => {
          if (noteRequired.value && !v) return cb(new Error(t('srp.apply.noteRequired')))
          cb()
        },
        trigger: 'blur'
      }
    ]
  }

  const formatKmLabel = (km: Api.Srp.FleetKillmailItem) =>
    `${km.killmail_id}: ${getName(km.ship_type_id, `TypeID: ${km.ship_type_id}`)}` +
    `(${km.victim_name}) - ${formatTime(km.killmail_time)}` +
    ` @${getName(km.solar_system_id, String(km.solar_system_id))}`

  const loadKillmails = async () => {
    if (!form.character_id) {
      fleetKillmails.value = []
      return
    }
    kmLoading.value = true
    fleetKillmails.value = []
    form.killmail_id = 0
    try {
      if (form.fleet_id && form.fleet_id !== OTHER_ACTION) {
        const list = await fetchFleetKillmails(form.fleet_id)
        fleetKillmails.value = list ?? []
        if (!list?.length) ElMessage.info(t('srp.apply.noKmFound'))
      } else {
        const list = await fetchMyKillmails(form.character_id)
        fleetKillmails.value = list ?? []
      }

      if (fleetKillmails.value.length) {
        const typeIds = [
          ...new Set(fleetKillmails.value.map((km) => km.ship_type_id).filter(Boolean))
        ]
        const solarIds = [
          ...new Set(fleetKillmails.value.map((km) => km.solar_system_id).filter(Boolean))
        ]
        const idsToResolve: Record<string, number[]> = {}
        if (typeIds.length) idsToResolve.type = typeIds
        if (solarIds.length) idsToResolve.solar_system = solarIds
        if (Object.keys(idsToResolve).length > 0) {
          await resolveNames({ ids: idsToResolve })
        }
      }
    } catch {
      fleetKillmails.value = []
    } finally {
      kmLoading.value = false
    }
  }

  const onCharacterChange = () => {
    form.killmail_id = 0
    form.recommended_amount = 0
    loadKillmails()
  }

  const onFleetChange = () => {
    form.killmail_id = 0
    if (form.fleet_id !== OTHER_ACTION) {
      form.note = ''
    }
    loadKillmails()
  }

  const onKillmailSelect = (_: number) => {
    form.recommended_amount = 0
  }

  const handleSubmit = async () => {
    await formRef.value?.validate()
    submitting.value = true
    try {
      const fleetId = form.fleet_id === OTHER_ACTION ? null : form.fleet_id || null
      await submitApplication({
        character_id: form.character_id,
        killmail_id: form.killmail_id,
        fleet_id: fleetId,
        note: form.note,
        final_amount: form.final_amount
      })
      ElMessage.success(t('srp.apply.submitSuccess'))
      formRef.value?.resetFields()
      form.fleet_id = ''
      form.recommended_amount = 0
      fleetKillmails.value = []
      loadApplications()
    } catch {
      /* handled */
    } finally {
      submitting.value = false
    }
  }

  /* ── KM 预览 ── */
  const kmPreviewVisible = ref(false)
  const previewKillmailId = ref(0)

  const openKmPreview = () => {
    if (!form.killmail_id) return
    previewKillmailId.value = form.killmail_id
    kmPreviewVisible.value = true
  }

  const openTableKmPreview = (row: Api.Srp.Application) => {
    previewKillmailId.value = row.killmail_id
    kmPreviewVisible.value = true
  }

  const openZkillboard = (killmailId: number) => {
    window.open(`https://zkillboard.com/kill/${killmailId}/`, '_blank')
  }

  /* ── 工具函数 ── */
  const formatTime = (v: string) => (v ? new Date(v).toLocaleString() : '-')
  const formatISK = (v: number) =>
    new Intl.NumberFormat('en-US', {
      minimumFractionDigits: 2,
      maximumFractionDigits: 2
    }).format(v ?? 0)

  type TagType = 'primary' | 'success' | 'warning' | 'info' | 'danger'
  const reviewStatusType = (s: string): TagType =>
    (({ pending: 'info', approved: 'success', rejected: 'danger' }) as Record<string, TagType>)[
      s
    ] ?? 'info'
  const reviewStatusLabel = (s: string) =>
    ({
      pending: t('srp.status.pending'),
      approved: t('srp.status.approved'),
      rejected: t('srp.status.rejected')
    })[s as 'pending' | 'approved' | 'rejected'] ?? s

  /* ── 初始化 ── */
  onMounted(() => {
    const fid = route.query.fleet_id as string
    if (fid) {
      form.fleet_id = fid
    }
    loadCharacters()
    loadFleets()
    loadApplications()
  })
</script>

<style scoped>
  .apply-card :deep(.el-card__header) {
    padding: 12px 16px;
  }

  .km-select-row {
    display: flex;
    gap: 8px;
    width: 100%;
  }

  .fleet-info-section {
    border-top: 1px solid #ebeef5;
    padding-top: 16px;
    margin-top: 8px;
  }

  .fleet-info-label {
    font-size: 14px;
    font-weight: 500;
    margin-bottom: 12px;
    color: #606266;
  }

  .table-header-bar {
    margin: -20px;
    padding: 12px 20px;
    border-radius: 4px 4px 0 0;
  }

  .pagination-wrapper {
    display: flex;
    justify-content: flex-end;
    margin-top: 16px;
  }
</style>
