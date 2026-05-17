<!-- 系统管理 - 基础配置 -->
<template>
  <div class="basic-config-page">
    <ElCard shadow="never">
      <template #header>
        <h2 class="section-title">{{ $t('system.basicConfig.allowCorporations') }}</h2>
      </template>

      <ElForm
        :model="allowCorpsForm"
        label-width="120px"
        style="max-width: 680px"
        v-loading="loadingAllowCorpsConfig"
      >
        <ElFormItem
          :label="$t('system.basicConfig.allowCorporationsLabel')"
          prop="allow_corporations"
        >
          <ElInput
            v-model="allowCorpsInput"
            type="textarea"
            :rows="6"
            clearable
            :placeholder="$t('system.basicConfig.allowCorporationsPlaceholder')"
          />
          <div class="form-hint">
            {{ $t('system.basicConfig.allowCorporationsHint', SYSTEM_IDENTITY_I18N) }}
          </div>
        </ElFormItem>

        <ElFormItem>
          <ElButton type="primary" :loading="savingAllowCorps" @click="handleSaveAllowCorps">
            {{ $t('system.basicConfig.save') }}
          </ElButton>
        </ElFormItem>
      </ElForm>
    </ElCard>

    <ElCard shadow="never" style="margin-top: 16px">
      <template #header>
        <h2 class="section-title">{{ $t('system.basicConfig.sdeConfig') }}</h2>
      </template>

      <ElForm
        :model="sdeForm"
        label-width="120px"
        style="max-width: 680px"
        v-loading="loadingSDEConfig"
      >
        <ElAlert
          v-if="sdeStatus.has_update"
          type="warning"
          :closable="false"
          :title="$t('system.basicConfig.sdeUpdateAvailable')"
          class="sde-status-alert"
        />

        <div class="sde-status-grid">
          <div class="sde-status-item">
            <span class="sde-status-label">{{ $t('system.basicConfig.sdeCurrentVersion') }}</span>
            <span class="sde-status-value">{{ sdeStatus.current_version || '-' }}</span>
          </div>
          <div class="sde-status-item">
            <span class="sde-status-label">{{ $t('system.basicConfig.sdeLatestVersion') }}</span>
            <span class="sde-status-value">{{ sdeStatus.latest_version || '-' }}</span>
          </div>
          <div class="sde-status-item">
            <span class="sde-status-label">{{ $t('system.basicConfig.sdeLastCheckAt') }}</span>
            <span class="sde-status-value">{{ formatTimestamp(sdeStatus.last_check_at) }}</span>
          </div>
          <div class="sde-status-item">
            <span class="sde-status-label">{{ $t('system.basicConfig.sdeLastUpdateAt') }}</span>
            <span class="sde-status-value">{{ formatTimestamp(sdeStatus.last_update_at) }}</span>
          </div>
          <div class="sde-status-item sde-status-item--full" v-if="sdeLastError">
            <span class="sde-status-label">{{ $t('system.basicConfig.sdeLastError') }}</span>
            <span class="sde-status-value sde-status-value--error">{{ sdeLastError }}</span>
          </div>
        </div>

        <ElFormItem :label="$t('system.basicConfig.sdeApiKey')" prop="api_key">
          <ElInput
            v-model="sdeForm.api_key"
            clearable
            show-password
            :placeholder="$t('system.basicConfig.sdeApiKeyPlaceholder')"
            style="width: 400px"
          />
        </ElFormItem>

        <ElFormItem :label="$t('system.basicConfig.sdeProxy')" prop="proxy">
          <ElInput
            v-model="sdeForm.proxy"
            clearable
            :placeholder="$t('system.basicConfig.sdeProxyPlaceholder')"
            style="width: 400px"
          />
        </ElFormItem>

        <ElFormItem :label="$t('system.basicConfig.sdeDownloadUrl')" prop="download_url">
          <ElInput
            v-model="sdeForm.download_url"
            clearable
            :placeholder="$t('system.basicConfig.sdeDownloadUrlPlaceholder')"
            style="width: 500px"
          />
        </ElFormItem>

        <ElFormItem>
          <ElButton :loading="checkingSDE" @click="handleCheckSDE">
            {{ $t('system.basicConfig.sdeCheckVersion') }}
          </ElButton>
          <ElButton
            type="warning"
            :loading="updatingSDE"
            :disabled="!sdeStatus.has_update"
            @click="handleUpdateSDE"
          >
            {{ $t('system.basicConfig.sdeRunUpdate') }}
          </ElButton>
          <ElButton type="primary" :loading="savingSDE" @click="handleSaveSDE">
            {{ $t('system.basicConfig.save') }}
          </ElButton>
        </ElFormItem>
      </ElForm>
    </ElCard>

    <ElCard shadow="never" style="margin-top: 16px">
      <template #header>
        <h2 class="section-title">{{ $t('system.basicConfig.corporationAccessPolicies') }}</h2>
      </template>

      <div v-loading="loadingCorpPolicies">
        <div class="form-hint">{{ $t('system.basicConfig.corporationAccessPoliciesHint') }}</div>

        <ElFormItem
          :label="$t('system.basicConfig.selectedCorporation')"
          class="corp-policy-selector"
        >
          <ElSelect
            v-model="selectedCorporationId"
            :placeholder="$t('system.basicConfig.selectCorporationToConfigure')"
            style="width: 280px"
          >
            <ElOption
              v-for="corporationID in allowCorpsForm.allow_corporations"
              :key="corporationID"
              :label="String(corporationID)"
              :value="corporationID"
            />
          </ElSelect>
        </ElFormItem>

        <div v-if="selectedPolicy" class="corp-policy-row">
          <div class="corp-policy-header">
            <span>
              {{ $t('system.basicConfig.corporationId') }}: {{ selectedPolicy.corporation_id }}
            </span>
            <ElSwitch
              v-model="selectedPolicy.full_access"
              :active-text="$t('system.basicConfig.fullAccess')"
            />
          </div>

          <div class="corp-capability-groups">
            <div
              v-for="group in corpCapabilityGroups"
              :key="group.labelKey"
              class="corp-capability-group"
            >
              <div class="corp-capability-group-title">{{ $t(group.labelKey) }}</div>
              <ElCheckboxGroup v-model="selectedPolicy.capabilities">
                <ElCheckbox
                  v-for="capability in group.capabilities"
                  :key="capability"
                  :label="capability"
                >
                  {{ $t(corpCapabilityLabelKeys[capability]) }}
                </ElCheckbox>
              </ElCheckboxGroup>
            </div>
          </div>

          <ElFormItem
            :label="$t('system.basicConfig.srpRecommendationMultiplier')"
            class="corp-policy-multiplier"
          >
            <ElInputNumber
              v-model="selectedPolicy.multiplier"
              :precision="2"
              :step="0.1"
              :min="0"
              :max="1"
              controls-position="right"
            />
          </ElFormItem>
        </div>

        <ElButton
          type="primary"
          :loading="savingCorpPolicies"
          :disabled="!selectedPolicy"
          @click="handleSaveCorpPolicies"
        >
          {{ $t('system.basicConfig.saveCurrentCorporationPolicy') }}
        </ElButton>
      </div>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
  import { useI18n } from 'vue-i18n'
  import {
    ElCard,
    ElForm,
    ElFormItem,
    ElInput,
    ElButton,
    ElMessage,
    ElAlert,
    ElSwitch,
    ElCheckboxGroup,
    ElCheckbox,
    ElInputNumber,
    ElSelect,
    ElOption
  } from 'element-plus'
  import {
    fetchSDEConfig,
    updateSDEConfig,
    fetchSDEStatus,
    checkSDEVersion,
    triggerSDEUpdate,
    fetchAllowCorporations,
    updateAllowCorporations,
    fetchCorporationAccessPolicies,
    updateCorporationAccessPolicies
  } from '@/api/sys-config'
  import { SYSTEM_IDENTITY, SYSTEM_IDENTITY_I18N } from '@/constants/system-identity'
  import { formatTime } from '@/utils/common'

  defineOptions({ name: 'BasicConfig' })

  const { t } = useI18n()
  const loadingSDEConfig = ref(false)
  const savingSDE = ref(false)
  const checkingSDE = ref(false)
  const updatingSDE = ref(false)
  const loadingAllowCorpsConfig = ref(false)
  const savingAllowCorps = ref(false)
  const loadingCorpPolicies = ref(false)
  const savingCorpPolicies = ref(false)
  const REQUIRED_ALLOW_CORPORATION_ID = SYSTEM_IDENTITY.corporationId
  const corpCapabilityGroups: Array<{
    labelKey: string
    capabilities: Api.SysConfig.CorporationCapability[]
  }> = [
    {
      labelKey: 'system.basicConfig.corpCapabilityGroups.menus',
      capabilities: [
        'menu.dashboard',
        'menu.operation',
        'menu.role',
        'menu.newbro',
        'menu.fuxi_hall',
        'menu.ticket',
        'menu.shop',
        'menu.system',
        'menu.info',
        'menu.skill_planning',
        'menu.srp',
        'menu.welfare'
      ]
    },
    {
      labelKey: 'system.basicConfig.corpCapabilityGroups.srp',
      capabilities: ['srp.user', 'srp.manage']
    },
    {
      labelKey: 'system.basicConfig.corpCapabilityGroups.welfare',
      capabilities: ['welfare.user', 'welfare.approval', 'welfare.settings']
    },
    {
      labelKey: 'system.basicConfig.corpCapabilityGroups.management',
      capabilities: ['ticket.manage', 'shop.manage', 'system.manage']
    }
  ]
  const corpCapabilityOptions = corpCapabilityGroups.flatMap((group) => group.capabilities)
  const corpCapabilityLabelKeys: Record<Api.SysConfig.CorporationCapability, string> = {
    'menu.dashboard': 'system.basicConfig.corpCapabilities.menuDashboard',
    'menu.operation': 'system.basicConfig.corpCapabilities.menuOperation',
    'menu.role': 'system.basicConfig.corpCapabilities.menuRole',
    'menu.newbro': 'system.basicConfig.corpCapabilities.menuNewbro',
    'menu.fuxi_hall': 'system.basicConfig.corpCapabilities.menuFuxiHall',
    'menu.ticket': 'system.basicConfig.corpCapabilities.menuTicket',
    'menu.shop': 'system.basicConfig.corpCapabilities.menuShop',
    'menu.system': 'system.basicConfig.corpCapabilities.menuSystem',
    'menu.info': 'system.basicConfig.corpCapabilities.menuInfo',
    'menu.skill_planning': 'system.basicConfig.corpCapabilities.menuSkillPlanning',
    'menu.srp': 'system.basicConfig.corpCapabilities.menuSrp',
    'menu.welfare': 'system.basicConfig.corpCapabilities.menuWelfare',
    'srp.user': 'system.basicConfig.corpCapabilities.srpUser',
    'srp.manage': 'system.basicConfig.corpCapabilities.srpManage',
    'welfare.user': 'system.basicConfig.corpCapabilities.welfareUser',
    'welfare.approval': 'system.basicConfig.corpCapabilities.welfareApproval',
    'welfare.settings': 'system.basicConfig.corpCapabilities.welfareSettings',
    'ticket.manage': 'system.basicConfig.corpCapabilities.ticketManage',
    'shop.manage': 'system.basicConfig.corpCapabilities.shopManage',
    'system.manage': 'system.basicConfig.corpCapabilities.systemManage'
  }

  const sdeForm = reactive<Api.SysConfig.SDEConfig>({
    api_key: '',
    proxy: '',
    download_url: ''
  })
  const sdeStatus = reactive<Api.SysConfig.SDEStatus>({
    current_version: '',
    latest_version: '',
    has_update: false,
    last_check_at: 0,
    last_check_success: false,
    last_check_error: '',
    last_update_at: 0,
    last_update_success: false,
    last_update_error: ''
  })

  const allowCorpsForm = reactive<Api.SysConfig.AllowCorporationsConfig>({
    allow_corporations: []
  })

  const allowCorpsInput = ref('')
  const corpPoliciesVersion = ref(1)
  const selectedCorporationId = ref<number>()
  const corpPolicyRows = ref<
    Array<{
      corporation_id: number
      full_access: boolean
      capabilities: Api.SysConfig.CorporationCapability[]
      multiplier: number
    }>
  >([])
  const selectedPolicy = computed(() =>
    corpPolicyRows.value.find((row) => row.corporation_id === selectedCorporationId.value)
  )
  const sdeLastError = computed(() => sdeStatus.last_update_error || sdeStatus.last_check_error)

  const normalizeAllowCorporations = (corporations: number[]) => {
    const seen = new Set<number>([REQUIRED_ALLOW_CORPORATION_ID])
    return [
      REQUIRED_ALLOW_CORPORATION_ID,
      ...corporations.filter((corporationID) => {
        if (seen.has(corporationID)) {
          return false
        }
        seen.add(corporationID)
        return true
      })
    ]
  }

  const parseCorporationId = (value: string) => {
    if (!/^\d+$/.test(value)) {
      throw new Error(t('system.basicConfig.invalidCorpId'))
    }

    const corporationId = Number.parseInt(value, 10)
    if (!Number.isSafeInteger(corporationId) || corporationId <= 0) {
      throw new Error(t('system.basicConfig.invalidCorpId'))
    }

    return corporationId
  }

  const loadSDEConfig = async () => {
    loadingSDEConfig.value = true
    try {
      const res = await fetchSDEConfig()
      sdeForm.api_key = res.api_key
      sdeForm.proxy = res.proxy
      sdeForm.download_url = res.download_url
    } catch {
      /* empty */
    } finally {
      loadingSDEConfig.value = false
    }
  }

  const syncSDEStatus = (status: Api.SysConfig.SDEStatus) => {
    sdeStatus.current_version = status.current_version
    sdeStatus.latest_version = status.latest_version
    sdeStatus.has_update = status.has_update
    sdeStatus.last_check_at = status.last_check_at
    sdeStatus.last_check_success = status.last_check_success
    sdeStatus.last_check_error = status.last_check_error
    sdeStatus.last_update_at = status.last_update_at
    sdeStatus.last_update_success = status.last_update_success
    sdeStatus.last_update_error = status.last_update_error
  }

  const loadSDEStatus = async () => {
    try {
      const status = await fetchSDEStatus()
      syncSDEStatus(status)
    } catch {
      /* empty */
    }
  }

  const formatTimestamp = (unixSeconds: number) => {
    if (!unixSeconds) {
      return '-'
    }
    return formatTime(new Date(unixSeconds * 1000).toISOString())
  }

  const handleSaveSDE = async () => {
    savingSDE.value = true
    try {
      await updateSDEConfig({
        api_key: sdeForm.api_key,
        proxy: sdeForm.proxy,
        download_url: sdeForm.download_url
      })
      ElMessage.success(t('system.basicConfig.saveSuccess'))
    } catch {
      /* empty */
    } finally {
      savingSDE.value = false
    }
  }

  const handleCheckSDE = async () => {
    checkingSDE.value = true
    try {
      const status = await checkSDEVersion()
      syncSDEStatus(status)
      ElMessage.success(t('system.basicConfig.sdeCheckSuccess'))
    } catch {
      /* empty */
    } finally {
      checkingSDE.value = false
    }
  }

  const handleUpdateSDE = async () => {
    updatingSDE.value = true
    try {
      const status = await triggerSDEUpdate()
      syncSDEStatus(status)
      ElMessage.success(t('system.basicConfig.sdeUpdateSuccess'))
    } catch {
      /* empty */
    } finally {
      updatingSDE.value = false
    }
  }

  const loadAllowCorpsConfig = async () => {
    loadingAllowCorpsConfig.value = true
    try {
      const res = await fetchAllowCorporations()
      const corporations = normalizeAllowCorporations(res.allow_corporations)
      allowCorpsForm.allow_corporations = corporations
      allowCorpsInput.value = corporations.join('\n')
    } catch {
      /* empty */
    } finally {
      loadingAllowCorpsConfig.value = false
    }
  }

  const handleSaveAllowCorps = async () => {
    try {
      const lines = allowCorpsInput.value
        .split('\n')
        .map((line) => line.trim())
        .filter((line) => line !== '')
      const corps = normalizeAllowCorporations(lines.map(parseCorporationId))

      savingAllowCorps.value = true
      await updateAllowCorporations({ allow_corporations: corps })
      allowCorpsForm.allow_corporations = corps
      allowCorpsInput.value = corps.join('\n')
      ElMessage.success(t('system.basicConfig.saveSuccess'))
    } catch (error) {
      ElMessage.error(
        error instanceof Error && error.message
          ? error.message
          : t('system.basicConfig.invalidCorpId')
      )
    } finally {
      savingAllowCorps.value = false
    }
  }

  const clampMultiplier = (value: number) => {
    if (!Number.isFinite(value)) return 1
    if (value < 0 || value > 1) return NaN
    return value
  }

  const loadCorpPolicies = async () => {
    loadingCorpPolicies.value = true
    try {
      const res = await fetchCorporationAccessPolicies()
      corpPoliciesVersion.value = res.version || 1
      const policyMap = new Map(
        (res.policies || []).map((policy) => [
          policy.corporation_id,
          {
            full_access: !!policy.full_access,
            capabilities: Array.isArray(policy.capabilities) ? policy.capabilities : [],
            rules: policy.rules ?? {}
          }
        ])
      )
      corpPolicyRows.value = allowCorpsForm.allow_corporations.map((corporationID) => {
        const policy = policyMap.get(corporationID)
        const rawMultiplier = policy?.rules?.['srp.recommendation_multiplier']
        const multiplier =
          typeof rawMultiplier === 'number' && Number.isFinite(rawMultiplier) ? rawMultiplier : 1
        return {
          corporation_id: corporationID,
          full_access: policy?.full_access ?? false,
          capabilities: (policy?.capabilities ?? []).filter((capability) =>
            corpCapabilityOptions.includes(capability as Api.SysConfig.CorporationCapability)
          ) as Api.SysConfig.CorporationCapability[],
          multiplier: clampMultiplier(multiplier) || 1
        }
      })
      if (
        !selectedCorporationId.value ||
        !allowCorpsForm.allow_corporations.includes(selectedCorporationId.value)
      ) {
        selectedCorporationId.value = allowCorpsForm.allow_corporations[0]
      }
    } catch {
      /* empty */
    } finally {
      loadingCorpPolicies.value = false
    }
  }

  const handleSaveCorpPolicies = async () => {
    if (!selectedPolicy.value) {
      return
    }

    if (Number.isNaN(clampMultiplier(selectedPolicy.value.multiplier))) {
      ElMessage.error(t('system.basicConfig.invalidMultiplier'))
      return
    }

    savingCorpPolicies.value = true
    try {
      await updateCorporationAccessPolicies({
        version: corpPoliciesVersion.value,
        default_mode: 'deny',
        policies: corpPolicyRows.value.map((row) => ({
          corporation_id: row.corporation_id,
          full_access: row.full_access,
          capabilities: row.capabilities,
          rules: {
            'srp.recommendation_multiplier': row.multiplier
          }
        }))
      })
      ElMessage.success(t('system.basicConfig.saveSuccess'))
      await loadCorpPolicies()
    } catch {
      /* empty */
    } finally {
      savingCorpPolicies.value = false
    }
  }

  onMounted(() => {
    loadAllowCorpsConfig().then(loadCorpPolicies)
    loadSDEConfig()
    loadSDEStatus()
  })
</script>

<style scoped>
  .section-title {
    font-size: 15px;
    font-weight: 600;
    margin: 0;
  }

  .form-hint {
    font-size: 12px;
    color: var(--el-text-color-secondary);
    margin-top: 4px;
  }

  .sde-status-alert {
    margin-bottom: 12px;
  }

  .sde-status-grid {
    display: grid;
    gap: 8px 12px;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    margin-bottom: 16px;
  }

  .sde-status-item {
    display: flex;
    gap: 8px;
    font-size: 13px;
  }

  .sde-status-item--full {
    grid-column: 1 / -1;
  }

  .sde-status-label {
    color: var(--el-text-color-secondary);
  }

  .sde-status-value {
    color: var(--el-text-color-primary);
    word-break: break-all;
  }

  .sde-status-value--error {
    color: var(--el-color-danger);
  }

  .corp-policy-row {
    border: 1px solid var(--el-border-color);
    border-radius: 8px;
    padding: 12px;
    margin: 12px 0;
  }

  .corp-policy-selector {
    margin-top: 12px;
    margin-bottom: 12px;
  }

  .corp-policy-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 10px;
    font-size: 13px;
  }

  .corp-capability-groups {
    display: grid;
    gap: 10px;
  }

  .corp-capability-group {
    border-top: 1px dashed var(--el-border-color);
    padding-top: 8px;
  }

  .corp-capability-group-title {
    color: var(--el-text-color-secondary);
    font-size: 12px;
    margin-bottom: 4px;
  }

  .corp-policy-multiplier {
    margin-top: 12px;
    margin-bottom: 0;
  }
</style>
