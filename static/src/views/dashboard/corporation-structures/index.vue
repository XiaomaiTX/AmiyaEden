<template>
  <div class="corporation-structures-page art-full-height">
    <ElCard shadow="never" class="art-card mb-4">
      <div class="flex flex-col gap-1">
        <h2 class="text-lg font-medium">{{ $t('corporationStructures.title') }}</h2>
        <p class="text-sm text-g-500">{{ $t('corporationStructures.subtitle') }}</p>
      </div>
    </ElCard>

    <ElTabs v-model="activeTab" @tab-change="handleTabChange">
      <ElTabPane :label="$t('corporationStructures.tabs.list')" name="list">
        <ElCard shadow="never" class="art-table-card">
          <div class="flex flex-wrap items-center gap-3 mb-4">
            <ElSelect
              v-model="selectedCorpId"
              class="w-64"
              clearable
              :placeholder="$t('corporationStructures.filters.corporation')"
              @clear="selectedCorpId = 0"
            >
              <ElOption :label="$t('corporationStructures.allCorporations')" :value="0" />
              <ElOption
                v-for="corp in settings.corporations"
                :key="corp.corporation_id"
                :label="`${corp.corporation_name} (${corp.corporation_id})`"
                :value="corp.corporation_id"
              />
            </ElSelect>

            <ElButton :loading="listLoading" @click="loadStructures">
              {{ $t('common.search') }}
            </ElButton>
            <ElButton
              type="primary"
              :loading="refreshingCorpId === selectedCorpId && selectedCorpId > 0"
              :disabled="selectedCorpId === 0"
              @click="handleRefreshSelectedCorporation"
            >
              {{ $t('corporationStructures.actions.refreshSelected') }}
            </ElButton>
          </div>

          <ElTable v-loading="listLoading" :data="rows" stripe border>
            <ElTableColumn
              prop="corporation_name"
              :label="$t('corporationStructures.table.corporation')"
              min-width="180"
              show-overflow-tooltip
            />
            <ElTableColumn
              prop="state"
              :label="$t('corporationStructures.table.state')"
              width="140"
            />
            <ElTableColumn :label="$t('corporationStructures.table.system')" min-width="180">
              <template #default="{ row }">
                <span>{{ row.system_name }}</span>
                <span class="text-xs text-g-500 ml-1">({{ formatSecurity(row.security) }})</span>
              </template>
            </ElTableColumn>
            <ElTableColumn
              prop="name"
              :label="$t('corporationStructures.table.name')"
              min-width="200"
              show-overflow-tooltip
            />
            <ElTableColumn
              prop="type_name"
              :label="$t('corporationStructures.table.type')"
              min-width="180"
              show-overflow-tooltip
            />
            <ElTableColumn :label="$t('corporationStructures.table.services')" min-width="260">
              <template #default="{ row }">
                {{ formatServices(row.services) }}
              </template>
            </ElTableColumn>
            <ElTableColumn
              prop="fuel_remaining"
              :label="$t('corporationStructures.table.fuelRemaining')"
              width="160"
            />
            <ElTableColumn
              prop="reinforce_hour"
              :label="$t('corporationStructures.table.reinforceHour')"
              width="140"
            />
            <ElTableColumn :label="$t('corporationStructures.table.updatedAt')" width="190">
              <template #default="{ row }">
                {{ formatUpdatedAt(row.updated_at) }}
              </template>
            </ElTableColumn>
          </ElTable>

          <ElEmpty
            v-if="!listLoading && rows.length === 0"
            :description="$t('corporationStructures.empty.list')"
            class="mt-4"
          />
        </ElCard>
      </ElTabPane>

      <ElTabPane :label="$t('corporationStructures.tabs.settings')" name="settings">
        <ElCard shadow="never" class="art-table-card">
          <div class="flex flex-wrap items-center gap-3 mb-4">
            <ElButton :loading="settingsLoading" @click="loadSettings">
              {{ $t('common.refresh') }}
            </ElButton>
            <ElButton type="primary" :loading="savingAuthorizations" @click="saveAuthorizations">
              {{ $t('common.save') }}
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
                  <ElOption
                    v-for="option in row.director_characters"
                    :key="option.character_id"
                    :label="`${option.character_name} (${option.character_id})`"
                    :value="option.character_id"
                  />
                </ElSelect>
              </template>
            </ElTableColumn>
            <ElTableColumn :label="$t('corporationStructures.table.actions')" width="180">
              <template #default="{ row }">
                <ElButton
                  size="small"
                  type="primary"
                  :loading="refreshingCorpId === row.corporation_id"
                  @click="handleRefreshCorporation(row.corporation_id)"
                >
                  {{ $t('corporationStructures.actions.refreshThisCorporation') }}
                </ElButton>
              </template>
            </ElTableColumn>
          </ElTable>

          <ElEmpty
            v-if="!settingsLoading && settings.corporations.length === 0"
            :description="$t('corporationStructures.empty.settings')"
            class="mt-4"
          />
        </ElCard>
      </ElTabPane>
    </ElTabs>
  </div>
</template>

<script setup lang="ts">
  import { ElMessage } from 'element-plus'
  import { useI18n } from 'vue-i18n'
  import { useRoute, useRouter } from 'vue-router'
  import {
    fetchCorporationStructureList,
    fetchCorporationStructureSettings,
    refreshCorporationStructures,
    updateCorporationStructureAuthorizations
  } from '@/api/corporation-structures'

  defineOptions({ name: 'DashboardCorporationStructures' })

  type StructureTab = 'list' | 'settings'

  const { t } = useI18n()
  const route = useRoute()
  const router = useRouter()

  const settings = ref<Api.Dashboard.CorporationStructuresSettings>({
    corporations: []
  })
  const rows = ref<Api.Dashboard.CorporationStructureRow[]>([])
  const settingsLoading = ref(false)
  const listLoading = ref(false)
  const savingAuthorizations = ref(false)
  const refreshingCorpId = ref<number>(0)
  const selectedCorpId = ref<number>(0)
  const authorizationByCorp = reactive<Record<number, number>>({})

  const normalizeTab = (value: unknown): StructureTab => {
    const queryValue = Array.isArray(value) ? value[0] : value
    return queryValue === 'settings' ? 'settings' : 'list'
  }

  const activeTab = ref<StructureTab>(normalizeTab(route.query.tab))

  const syncAuthorizationsFromSettings = () => {
    Object.keys(authorizationByCorp).forEach((key) => {
      delete authorizationByCorp[Number(key)]
    })
    settings.value.corporations.forEach((item) => {
      authorizationByCorp[item.corporation_id] = item.authorized_character_id || 0
    })
  }

  const loadSettings = async () => {
    settingsLoading.value = true
    try {
      settings.value = await fetchCorporationStructureSettings()
      syncAuthorizationsFromSettings()

      const managedCorpSet = new Set(settings.value.corporations.map((item) => item.corporation_id))
      if (selectedCorpId.value > 0 && !managedCorpSet.has(selectedCorpId.value)) {
        selectedCorpId.value = 0
      }
    } finally {
      settingsLoading.value = false
    }
  }

  const loadStructures = async () => {
    listLoading.value = true
    try {
      const payload: Api.Dashboard.CorporationStructureListRequest = {}
      if (selectedCorpId.value > 0) {
        payload.corporation_id = selectedCorpId.value
      }
      const result = await fetchCorporationStructureList(payload)
      rows.value = result?.items || []
    } finally {
      listLoading.value = false
    }
  }

  const saveAuthorizations = async () => {
    const authorizations: Api.Dashboard.CorporationStructureAuthorizationBinding[] =
      settings.value.corporations.map((corp) => ({
        corporation_id: corp.corporation_id,
        character_id: authorizationByCorp[corp.corporation_id] || 0
      }))

    savingAuthorizations.value = true
    try {
      await updateCorporationStructureAuthorizations({ authorizations })
      await loadSettings()
      ElMessage.success(t('corporationStructures.messages.authorizationSaved'))
    } finally {
      savingAuthorizations.value = false
    }
  }

  const handleRefreshCorporation = async (corporationId: number) => {
    refreshingCorpId.value = corporationId
    try {
      const result = await refreshCorporationStructures(corporationId)
      if (result.running) {
        ElMessage.warning(
          result.message || t('corporationStructures.messages.refreshAlreadyRunning')
        )
        return
      }
      ElMessage.success(result.message || t('corporationStructures.messages.refreshQueued'))
    } finally {
      refreshingCorpId.value = 0
    }
  }

  const handleRefreshSelectedCorporation = async () => {
    if (selectedCorpId.value <= 0) {
      ElMessage.warning(t('corporationStructures.messages.selectCorporationFirst'))
      return
    }
    await handleRefreshCorporation(selectedCorpId.value)
  }

  const formatServices = (services: Api.Dashboard.CorporationStructureServiceInfo[]) => {
    if (!services || services.length === 0) {
      return t('corporationStructures.noServices')
    }
    return services.map((service) => `${service.name} (${service.state})`).join(' / ')
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
    return new Date(updatedAt * 1000).toLocaleString()
  }

  const handleTabChange = (tab: string | number) => {
    activeTab.value = normalizeTab(tab)
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
    await loadStructures()
  })
</script>
