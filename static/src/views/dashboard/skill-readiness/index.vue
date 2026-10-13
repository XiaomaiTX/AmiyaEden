<template>
  <div class="skill-readiness-page">
    <ElCard shadow="never" class="mb-4">
      <div class="mb-4">
        <h2 class="text-lg font-semibold">{{ $t('dashboardSkillReadiness.title') }}</h2>
        <p class="mt-1 text-sm text-g-500">{{ $t('dashboardSkillReadiness.subtitle') }}</p>
      </div>
      <div
        class="grid grid-cols-1 items-end gap-3 md:grid-cols-[minmax(220px,1fr)_minmax(220px,1fr)_auto_auto]"
      >
        <ElFormItem :label="$t('dashboardSkillReadiness.filters.corporations')" class="mb-0">
          <ElSelect
            v-model="corporationIds"
            multiple
            filterable
            clearable
            collapse-tags
            collapse-tags-tooltip
            class="w-full"
            :placeholder="$t('dashboardSkillReadiness.allCorporations')"
            :loading="optionsLoading"
          >
            <ElOption
              v-for="item in options.corporations"
              :key="item.corporation_id"
              :label="`${item.corporation_name || item.corporation_id} (${item.corporation_id})`"
              :value="item.corporation_id"
            />
          </ElSelect>
        </ElFormItem>
        <ElFormItem :label="$t('dashboardSkillReadiness.filters.plan')" class="mb-0">
          <ElSelect
            v-model="planId"
            filterable
            clearable
            class="w-full"
            :placeholder="$t('dashboardSkillReadiness.selectPlan')"
            :loading="optionsLoading"
          >
            <ElOption
              v-for="plan in options.plans"
              :key="plan.id"
              :label="plan.title"
              :value="plan.id"
            />
          </ElSelect>
        </ElFormItem>
        <ElButton type="primary" :disabled="!planId" :loading="loading" @click="runSearch">
          {{ $t('dashboardSkillReadiness.search') }}
        </ElButton>
        <ElButton :disabled="!planId" :loading="loading" @click="loadResult">
          {{ $t('dashboardSkillReadiness.refresh') }}
        </ElButton>
      </div>
      <ElAlert
        v-if="optionsLoaded && !options.plans.length"
        class="mt-3"
        type="info"
        :closable="false"
        :title="$t('dashboardSkillReadiness.noPlans')"
      />
    </ElCard>

    <div v-if="result" class="mb-4 grid grid-cols-2 gap-3 xl:grid-cols-5">
      <ElCard v-for="item in statItems" :key="item.key" shadow="never" class="text-center">
        <div class="text-sm text-g-500">{{ $t(`dashboardSkillReadiness.stats.${item.key}`) }}</div>
        <div class="mt-1 text-2xl font-semibold tabular-nums">{{ item.value }}</div>
      </ElCard>
    </div>

    <ElCard shadow="never" class="art-table-card">
      <ElAlert v-if="error" class="mb-3" type="error" :closable="false" :title="error" />
      <ElTable
        v-loading="loading || optionsLoading"
        :data="result?.list ?? []"
        row-key="user_id"
        stripe
        @expand-change="onExpandChange"
      >
        <ElTableColumn type="expand">
          <template #default="{ row }">
            <div class="px-4 py-2">
              <div v-if="userLoading[row.user_id]" class="py-4 text-center text-g-500">
                {{ $t('dashboardSkillReadiness.loading') }}
              </div>
              <ElAlert
                v-else-if="userErrors[row.user_id]"
                type="error"
                :closable="false"
                :title="userErrors[row.user_id]"
              />
              <ElEmpty
                v-else-if="!(userCharacters[row.user_id] ?? []).length"
                :description="$t('dashboardSkillReadiness.noCharacters')"
              />
              <ElTable v-else :data="userCharacters[row.user_id] ?? []" size="small" border>
                <ElTableColumn
                  prop="character_name"
                  :label="$t('dashboardSkillReadiness.character')"
                  min-width="180"
                >
                  <template #default="{ row: character }">
                    <div class="font-medium">{{ character.character_name }}</div>
                    <div class="text-xs text-g-500">{{ character.character_id }}</div>
                  </template>
                </ElTableColumn>
                <ElTableColumn :label="$t('dashboardSkillReadiness.corporation')" min-width="180">
                  <template #default="{ row: character }">
                    {{
                      character.corporation_name ||
                      (character.corporation_id > 0
                        ? character.corporation_id
                        : $t('dashboardSkillReadiness.unknownCorporation'))
                    }}
                  </template>
                </ElTableColumn>
                <ElTableColumn :label="$t('common.status')" width="140">
                  <template #default="{ row: character }">
                    <ElTag :type="statusTagType(character.status)" effect="light">
                      {{ $t(`dashboardSkillReadiness.status.${character.status}`) }}
                    </ElTag>
                  </template>
                </ElTableColumn>
                <ElTableColumn
                  :label="$t('dashboardSkillReadiness.skills')"
                  width="120"
                  align="center"
                >
                  <template #default="{ row: character }">
                    {{
                      character.status === 'unknown'
                        ? '—'
                        : `${character.matched_skills}/${character.total_skills}`
                    }}
                  </template>
                </ElTableColumn>
                <ElTableColumn :label="$t('dashboardSkillReadiness.updatedAt')" min-width="190">
                  <template #default="{ row: character }">
                    {{ formatTimestamp(character.skill_updated_at) }}
                  </template>
                </ElTableColumn>
                <ElTableColumn
                  :label="$t('dashboardSkillReadiness.columns.actions')"
                  width="120"
                  fixed="right"
                >
                  <template #default="{ row: character }">
                    <ElButton link type="primary" @click="openCharacter(character)">
                      {{ $t('dashboardSkillReadiness.details') }}
                    </ElButton>
                  </template>
                </ElTableColumn>
              </ElTable>
            </div>
          </template>
        </ElTableColumn>
        <ElTableColumn :label="$t('dashboardSkillReadiness.columns.user')" min-width="180">
          <template #default="{ row }">
            {{ row.nickname || `${$t('dashboardSkillReadiness.columns.userId')} ${row.user_id}` }}
          </template>
        </ElTableColumn>
        <ElTableColumn
          prop="user_id"
          :label="$t('dashboardSkillReadiness.columns.userId')"
          width="110"
        />
        <ElTableColumn
          prop="character_count"
          :label="$t('dashboardSkillReadiness.columns.characters')"
          width="140"
          align="center"
        />
        <ElTableColumn
          prop="satisfied_character_count"
          :label="$t('dashboardSkillReadiness.columns.satisfiedCharacters')"
          width="140"
          align="center"
        />
        <ElTableColumn
          prop="unknown_character_count"
          :label="$t('dashboardSkillReadiness.columns.unknownCharacters')"
          width="140"
          align="center"
        />
      </ElTable>
      <div v-if="result" class="mt-4 flex flex-wrap items-center justify-between gap-3">
        <ElSelect v-model="pageSize" class="w-32" @change="changePageSize">
          <ElOption v-for="size in [20, 50, 100]" :key="size" :label="size" :value="size" />
        </ElSelect>
        <ElPagination
          v-model:current-page="page"
          :page-size="pageSize"
          :total="result.total"
          layout="prev, pager, next, total"
          @current-change="changePage"
        />
      </div>
    </ElCard>

    <ElDialog
      v-model="detailOpen"
      :title="detail?.character.character_name || $t('dashboardSkillReadiness.details')"
      width="min(1000px, 94vw)"
      destroy-on-close
    >
      <template v-if="detail">
        <div class="mb-3 flex flex-wrap items-center gap-2 text-sm">
          <ElTag :type="statusTagType(detail.character.status)" effect="light">
            {{ $t(`dashboardSkillReadiness.status.${detail.character.status}`) }}
          </ElTag>
          <span
            >{{ $t('dashboardSkillReadiness.corporation') }}:
            {{
              detail.character.corporation_name ||
              (detail.character.corporation_id > 0
                ? detail.character.corporation_id
                : $t('dashboardSkillReadiness.unknownCorporation'))
            }}</span
          >
          <span
            >{{ $t('dashboardSkillReadiness.updatedAt') }}:
            {{ formatTimestamp(detail.character.skill_updated_at) }}</span
          >
        </div>
        <ElAlert
          v-if="detail.character.token_invalid"
          class="mb-3"
          type="warning"
          :closable="false"
          :title="$t('dashboardSkillReadiness.tokenInvalid')"
        />
        <ElAlert
          v-if="detail.character.status === 'unknown'"
          class="mb-3"
          type="info"
          :closable="false"
          :title="$t('dashboardSkillReadiness.unknownHint')"
        />
      </template>
      <div v-if="detailLoading" class="py-8 text-center text-g-500">{{
        $t('dashboardSkillReadiness.loading')
      }}</div>
      <ElAlert v-else-if="detailError" type="error" :closable="false" :title="detailError" />
      <ElTabs v-else-if="detail" v-model="detailTab">
        <ElTabPane :label="$t('dashboardSkillReadiness.comparison')" name="comparison">
          <div class="mb-3 flex flex-wrap items-center gap-3">
            <ElInput
              v-model="skillSearch"
              clearable
              class="max-w-sm"
              :placeholder="$t('dashboardSkillReadiness.searchSkills')"
            />
            <ElCheckbox v-model="matchedOnly">{{
              $t('dashboardSkillReadiness.matchedOnly')
            }}</ElCheckbox>
          </div>
          <ElTable :data="filteredRequirements" stripe max-height="440">
            <ElTableColumn
              prop="skill_name"
              :label="$t('dashboardSkillReadiness.skillName')"
              min-width="220"
            />
            <ElTableColumn
              prop="group_name"
              :label="$t('dashboardSkillReadiness.group')"
              min-width="180"
            />
            <ElTableColumn
              prop="required_level"
              :label="$t('dashboardSkillReadiness.requiredLevel')"
              width="120"
              align="center"
            />
            <ElTableColumn
              :label="$t('dashboardSkillReadiness.currentLevel')"
              width="120"
              align="center"
            >
              <template #default="{ row }">{{ row.current_level ?? '—' }}</template>
            </ElTableColumn>
            <ElTableColumn :label="$t('common.status')" width="120" align="center">
              <template #default="{ row }">
                <ElTag
                  v-if="row.satisfied !== undefined"
                  :type="row.satisfied ? 'success' : 'danger'"
                  effect="light"
                >
                  {{
                    $t(
                      row.satisfied
                        ? 'dashboardSkillReadiness.status.satisfied'
                        : 'dashboardSkillReadiness.status.unsatisfied'
                    )
                  }}
                </ElTag>
                <span v-else>—</span>
              </template>
            </ElTableColumn>
          </ElTable>
          <ElEmpty
            v-if="!filteredRequirements.length"
            :description="
              $t(
                matchedOnly
                  ? 'dashboardSkillReadiness.noMissingSkills'
                  : 'dashboardSkillReadiness.noSkills'
              )
            "
          />
        </ElTabPane>
        <ElTabPane :label="$t('dashboardSkillReadiness.allSkills')" name="skills">
          <ElInput
            v-model="skillSearch"
            clearable
            class="mb-3 max-w-sm"
            :placeholder="$t('dashboardSkillReadiness.searchSkills')"
          />
          <ElTable :data="filteredSkills" stripe max-height="440">
            <ElTableColumn
              prop="skill_name"
              :label="$t('dashboardSkillReadiness.skillName')"
              min-width="220"
            />
            <ElTableColumn
              prop="group_name"
              :label="$t('dashboardSkillReadiness.group')"
              min-width="180"
            />
            <ElTableColumn
              prop="trained_level"
              :label="$t('dashboardSkillReadiness.currentLevel')"
              width="120"
              align="center"
            />
            <ElTableColumn
              prop="active_level"
              :label="$t('dashboardSkillReadiness.activeLevel')"
              width="130"
              align="center"
            />
            <ElTableColumn
              :label="$t('dashboardSkillReadiness.skillPoints')"
              width="150"
              align="right"
            >
              <template #default="{ row }">{{
                row.skillpoints_in_skill?.toLocaleString(locale) ?? '—'
              }}</template>
            </ElTableColumn>
          </ElTable>
          <ElEmpty
            v-if="!filteredSkills.length"
            :description="$t('dashboardSkillReadiness.noSkills')"
          />
        </ElTabPane>
      </ElTabs>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, nextTick, onMounted, ref, watch } from 'vue'
  import { useI18n } from 'vue-i18n'
  import { useRoute, useRouter } from 'vue-router'
  import {
    fetchSkillReadinessCharacterSkills,
    fetchSkillReadinessOptions,
    fetchSkillReadinessUserCharacters,
    querySkillReadiness
  } from '@/api/skill-readiness'

  type Filters = { corporationIds: number[]; planId: number; page: number; pageSize: number }

  const route = useRoute()
  const router = useRouter()
  const { locale, t } = useI18n()
  const options = ref<Api.SkillReadiness.Options>({ corporations: [], plans: [] })
  const optionsLoading = ref(true)
  const optionsLoaded = ref(false)
  const corporationIds = ref<number[]>([])
  const planId = ref(0)
  const page = ref(1)
  const pageSize = ref(20)
  const result = ref<Api.SkillReadiness.Result | null>(null)
  const loading = ref(false)
  const error = ref('')
  const userCharacters = ref<Record<number, Api.SkillReadiness.CharacterRow[]>>({})
  const userLoading = ref<Record<number, boolean>>({})
  const userErrors = ref<Record<number, string>>({})
  const detailOpen = ref(false)
  const detail = ref<Api.SkillReadiness.CharacterDetail | null>(null)
  const detailLoading = ref(false)
  const detailError = ref('')
  const detailTab = ref<'comparison' | 'skills'>('comparison')
  const matchedOnly = ref(false)
  const skillSearch = ref('')
  const querySequence = ref(0)
  const detailSequence = ref(0)
  const userScopeSequence = ref(0)
  let writingQuery = false
  let restoringRoute = false

  function queryFilters(value: Filters): Api.SkillReadiness.Request {
    return {
      corporation_ids: value.corporationIds,
      plan_id: value.planId,
      page: value.page,
      page_size: value.pageSize,
      language: locale.value.startsWith('zh') ? 'zh' : 'en'
    }
  }

  function readRouteFilters(): Filters {
    const corpValue = route.query.corporations
    const rawCorporations = Array.isArray(corpValue) ? corpValue[0] : corpValue
    const corporations = (rawCorporations || '')
      .split(',')
      .map(Number)
      .filter((id) => Number.isSafeInteger(id) && id > 0)
    const parsePositive = (value: unknown, fallback: number) => {
      const parsed = Number(Array.isArray(value) ? value[0] : value)
      return Number.isSafeInteger(parsed) && parsed > 0 ? parsed : fallback
    }
    const selectedSize = parsePositive(route.query.page_size, 20)
    return {
      corporationIds: [...new Set(corporations)],
      planId: parsePositive(route.query.plan, 0),
      page: parsePositive(route.query.page, 1),
      pageSize: [20, 50, 100].includes(selectedSize) ? selectedSize : 20
    }
  }

  async function writeRouteFilters(value: Filters) {
    const nextQuery = {
      ...route.query,
      corporations: value.corporationIds.length ? value.corporationIds.join(',') : undefined,
      plan: value.planId || undefined,
      page: value.page === 1 ? undefined : value.page,
      page_size: value.pageSize === 20 ? undefined : value.pageSize
    }
    if (JSON.stringify(nextQuery) === JSON.stringify(route.query)) return
    writingQuery = true
    await router.replace({ query: nextQuery })
  }

  async function loadResult() {
    if (!planId.value) return
    const sequence = ++querySequence.value
    loading.value = true
    error.value = ''
    try {
      const data = await querySkillReadiness(
        queryFilters({
          corporationIds: corporationIds.value,
          planId: planId.value,
          page: page.value,
          pageSize: pageSize.value
        })
      )
      if (sequence === querySequence.value) result.value = data
    } catch (caughtError) {
      if (sequence === querySequence.value) {
        result.value = null
        error.value =
          caughtError instanceof Error
            ? caughtError.message
            : t('dashboardSkillReadiness.queryFailed')
      }
    } finally {
      if (sequence === querySequence.value) loading.value = false
    }
  }

  async function runSearch() {
    page.value = 1
    await writeRouteFilters({
      corporationIds: corporationIds.value,
      planId: planId.value,
      page: 1,
      pageSize: pageSize.value
    })
    await loadResult()
  }

  async function onExpandChange(
    row: Api.SkillReadiness.UserRow,
    expandedRows: Api.SkillReadiness.UserRow[]
  ) {
    if (
      !expandedRows.some((item) => item.user_id === row.user_id) ||
      userCharacters.value[row.user_id]
    )
      return
    const scopeSequence = userScopeSequence.value
    userLoading.value = { ...userLoading.value, [row.user_id]: true }
    userErrors.value = { ...userErrors.value, [row.user_id]: '' }
    try {
      const rows = await fetchSkillReadinessUserCharacters(
        row.user_id,
        queryFilters({
          corporationIds: corporationIds.value,
          planId: planId.value,
          page: page.value,
          pageSize: pageSize.value
        })
      )
      if (scopeSequence === userScopeSequence.value) {
        userCharacters.value = { ...userCharacters.value, [row.user_id]: rows }
      }
    } catch (caughtError) {
      if (scopeSequence === userScopeSequence.value) {
        userErrors.value = {
          ...userErrors.value,
          [row.user_id]:
            caughtError instanceof Error
              ? caughtError.message
              : t('dashboardSkillReadiness.loadFailed')
        }
      }
    } finally {
      if (scopeSequence === userScopeSequence.value) {
        userLoading.value = { ...userLoading.value, [row.user_id]: false }
      }
    }
  }

  async function openCharacter(character: Api.SkillReadiness.CharacterRow) {
    const sequence = ++detailSequence.value
    detailOpen.value = true
    detail.value = null
    detailError.value = ''
    detailLoading.value = true
    detailTab.value = 'comparison'
    matchedOnly.value = false
    skillSearch.value = ''
    try {
      const data = await fetchSkillReadinessCharacterSkills(
        character.character_id,
        queryFilters({
          corporationIds: corporationIds.value,
          planId: planId.value,
          page: page.value,
          pageSize: pageSize.value
        })
      )
      if (sequence === detailSequence.value) detail.value = data
    } catch (caughtError) {
      if (sequence === detailSequence.value)
        detailError.value =
          caughtError instanceof Error
            ? caughtError.message
            : t('dashboardSkillReadiness.loadFailed')
    } finally {
      if (sequence === detailSequence.value) detailLoading.value = false
    }
  }

  function statusTagType(status: Api.SkillReadiness.Status): 'success' | 'danger' | 'info' {
    if (status === 'satisfied') return 'success'
    if (status === 'unsatisfied') return 'danger'
    return 'info'
  }

  function formatTimestamp(value: number | null) {
    return value ? new Date(value * 1000).toLocaleString(locale.value) : '—'
  }

  const statItems = computed(() => {
    if (!result.value) return []
    return [
      { key: 'characters', value: result.value.summary.character_count },
      { key: 'users', value: result.value.summary.user_count },
      { key: 'satisfiedUsers', value: result.value.summary.satisfied_user_count },
      { key: 'satisfiedCharacters', value: result.value.summary.satisfied_character_count },
      { key: 'unknownCharacters', value: result.value.summary.unknown_character_count }
    ]
  })

  const filteredRequirements = computed(() => {
    const rows = detail.value?.requirements ?? []
    const search = skillSearch.value.trim().toLocaleLowerCase()
    return rows.filter((row) => {
      const matchesSearch =
        !search || `${row.skill_name} ${row.group_name}`.toLocaleLowerCase().includes(search)
      return matchesSearch && (!matchedOnly.value || row.satisfied === false)
    })
  })

  const filteredSkills = computed(() => {
    const rows = detail.value?.skills ?? []
    const search = skillSearch.value.trim().toLocaleLowerCase()
    return rows.filter(
      (row) => !search || `${row.skill_name} ${row.group_name}`.toLocaleLowerCase().includes(search)
    )
  })

  watch(
    () => route.query,
    async () => {
      if (writingQuery) {
        writingQuery = false
        return
      }
      const restored = readRouteFilters()
      restoringRoute = true
      corporationIds.value = restored.corporationIds
      planId.value = restored.planId
      page.value = restored.page
      pageSize.value = restored.pageSize
      userCharacters.value = {}
      userLoading.value = {}
      userErrors.value = {}
      userScopeSequence.value++
      detailOpen.value = false
      detailSequence.value++
      querySequence.value++
      loading.value = false
      error.value = ''
      await nextTick()
      restoringRoute = false
      if (restored.planId) await loadResult()
      else result.value = null
    }
  )

  watch([corporationIds, planId], () => {
    if (restoringRoute) return
    page.value = 1
    result.value = null
    userCharacters.value = {}
    userLoading.value = {}
    userErrors.value = {}
    userScopeSequence.value++
    detailOpen.value = false
    querySequence.value++
    loading.value = false
    error.value = ''
    detailSequence.value++
  })

  onMounted(async () => {
    const restored = readRouteFilters()
    restoringRoute = true
    corporationIds.value = restored.corporationIds
    planId.value = restored.planId
    page.value = restored.page
    pageSize.value = restored.pageSize
    await nextTick()
    restoringRoute = false
    try {
      options.value = await fetchSkillReadinessOptions()
    } catch (caughtError) {
      error.value =
        caughtError instanceof Error ? caughtError.message : t('dashboardSkillReadiness.loadFailed')
    } finally {
      optionsLoading.value = false
      optionsLoaded.value = true
    }
    if (restored.planId) await loadResult()
  })

  async function changePage(nextPage: number) {
    page.value = nextPage
    await writeRouteFilters({
      corporationIds: corporationIds.value,
      planId: planId.value,
      page: nextPage,
      pageSize: pageSize.value
    })
    await loadResult()
  }

  async function changePageSize(nextSize: number) {
    pageSize.value = nextSize
    page.value = 1
    await writeRouteFilters({
      corporationIds: corporationIds.value,
      planId: planId.value,
      page: 1,
      pageSize: nextSize
    })
    await loadResult()
  }
</script>
