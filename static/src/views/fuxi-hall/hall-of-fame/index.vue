<template>
  <div class="fx-gallery" v-loading="loading">
    <FuxiHallBanner :title="page?.title || t('fuxiHall.fame.title')" :subtitle="page?.subtitle">
      <nav
        v-if="page && page.floors.length > 1"
        class="fx-index"
        :aria-label="t('fuxiHall.fame.floorIndex')"
      >
        <span>{{ t('fuxiHall.fame.floorIndex') }}</span>
        <button
          v-for="floor in page.floors"
          :key="floor.key"
          type="button"
          :aria-controls="`fx-floor-${floor.key}`"
          @click="showFloor(floor.key)"
        >
          {{ floor.name || t(`fuxiHall.fame.floors.${floor.key}`) }}
        </button>
      </nav>
    </FuxiHallBanner>
    <div v-if="error" role="alert" class="hall-error">
      {{ error }} <ElButton @click="loadPage()">{{ t('fuxiHall.fame.retry') }}</ElButton>
    </div>
    <article v-if="page?.description_html" class="fx-intro" v-html="page.description_html" />
    <section
      v-for="floor in page?.floors ?? []"
      :key="floor.key"
      :id="`fx-floor-${floor.key}`"
      tabindex="-1"
      class="fx-floor"
      :aria-label="floor.name || t(`fuxiHall.fame.floors.${floor.key}`)"
    >
      <div class="fx-floor__heading">
        <h2>{{ floor.name || t(`fuxiHall.fame.floors.${floor.key}`) }}</h2>
        <span>{{ t('fuxiHall.fame.memberCount', { count: floorCards(floor.key).length }) }}</span>
      </div>
      <div
        class="fx-grid"
        :style="{ maxWidth: `${Math.max(1, floorCards(floor.key).length) * 396 - 16}px` }"
      >
        <FuxiHallMemberCard v-for="card in floorCards(floor.key)" :key="card.id" :card="card">
          <template v-if="card.can_edit" #action>
            <ElButton size="small" @click="edit(card)">{{ t('fuxiHall.fame.editMine') }}</ElButton>
          </template>
        </FuxiHallMemberCard>
      </div>
    </section>
    <p v-if="!loading && !error && cards.length === 0" class="hall-empty">{{
      t('fuxiHall.public.emptyTitle')
    }}</p>
    <ElDialog
      :model-value="!!editing"
      :title="t('fuxiHall.fame.editMine')"
      width="min(720px, 94vw)"
      :close-on-click-modal="!saving"
      :close-on-press-escape="!saving"
      :show-close="!saving"
      @close="editing = null"
    >
      <ElForm label-position="top" @submit.prevent="save">
        <ElFormItem :label="t('fuxiHall.manage.nickname')" required
          ><ElInput
            v-model="nickname"
            :maxlength="256"
            :disabled="saving"
            :aria-label="t('fuxiHall.manage.nickname')"
        /></ElFormItem>
        <ElFormItem :label="t('fuxiHall.manage.cardDescription')"
          ><ArtWangEditor v-model="description" height="240px"
        /></ElFormItem>
        <p v-if="editError" role="alert" class="hall-error">{{ editError }}</p>
        <ElButton :disabled="saving" @click="editing = null">{{ t('common.cancel') }}</ElButton>
        <ElButton
          type="primary"
          native-type="submit"
          :loading="saving"
          :disabled="saving || !nickname.trim()"
          >{{ t('common.save') }}</ElButton
        >
      </ElForm>
    </ElDialog>
  </div>
</template>
<script setup lang="ts">
  import type { HallPage } from '@/api/generated/client'
  import { onActivated, onMounted, ref } from 'vue'
  import { useI18n } from 'vue-i18n'
  import { getHall, updateHallOwnCard } from '@/api/generated/client'
  import { unwrapContract } from '@/api/response'
  import FuxiHallMemberCard from '../components/FuxiHallMemberCard.vue'
  import FuxiHallBanner from '../components/FuxiHallBanner.vue'
  const { t } = useI18n()
  function showFloor(key: string) {
    const section = document.getElementById(`fx-floor-${key}`)
    // The Vue header overlays both the desktop scroll pane and mobile document.
    const headerHeight = document.getElementById('app-header')?.getBoundingClientRect().height ?? 0
    section?.style.setProperty('--fx-sticky-header-height', `${headerHeight}px`)
    section?.focus({ preventScroll: true })
    section?.scrollIntoView({ block: 'start' })
  }
  const page = ref<HallPage | null>(null)
  const cards = ref<Api.FuxiHall.Card[]>([])
  const loading = ref(false)
  const error = ref('')
  const editing = ref<Api.FuxiHall.Card | null>(null)
  const nickname = ref('')
  const description = ref('')
  const saving = ref(false)
  const editError = ref('')
  function message(caught: unknown) {
    const value = caught instanceof Error ? caught.message : 'fuxiHall.errors.load'
    return value.startsWith('fuxiHall.') ? t(value) : value
  }
  function floorCards(key: string) {
    return cards.value.filter((card) => card.floor_key === key)
  }
  // quiet refreshes keep the current cards on screen instead of showing the loading mask.
  async function loadPage(quiet = false) {
    loading.value = !quiet
    error.value = ''
    try {
      const result = await unwrapContract(getHall({ showErrorMessage: false }))
      page.value = result.page
      cards.value = result.cards
    } catch (caught) {
      error.value = message(caught)
    } finally {
      loading.value = false
    }
  }
  function edit(card: Api.FuxiHall.Card) {
    editing.value = card
    nickname.value = card.nickname
    description.value = card.description_html
    editError.value = ''
  }
  async function save() {
    if (!editing.value || saving.value) return
    saving.value = true
    editError.value = ''
    try {
      const updated = await unwrapContract(
        updateHallOwnCard(
          editing.value.id,
          {
            nickname: nickname.value,
            description_html: description.value
          },
          { showErrorMessage: false }
        )
      )
      cards.value = cards.value.map((card) =>
        card.id === updated.id
          ? { ...card, nickname: updated.nickname, description_html: updated.description_html }
          : card
      )
      editing.value = null
    } catch (caught) {
      editError.value = message(caught)
    } finally {
      saving.value = false
    }
  }
  onMounted(() => void loadPage())
  // Returning from the cached manage tab picks up card edits made there.
  onActivated(() => {
    if (page.value) void loadPage(true)
  })
</script>
<style scoped>
  .hall-error {
    color: var(--el-color-danger);
  }
  .hall-empty {
    text-align: center;
    color: var(--el-text-color-secondary);
    padding: 48px 0;
  }
</style>
