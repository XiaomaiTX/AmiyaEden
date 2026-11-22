<template>
  <div class="hall-page" v-loading="loading">
    <header class="hall-header">
      <h1>{{ page?.title || t('fuxiHall.fame.title') }}</h1>
      <p v-if="page?.subtitle">{{ page.subtitle }}</p>
      <p>{{ t('fuxiHall.contributions.daily') }}</p>
    </header>
    <div v-if="error" role="alert" class="hall-error">
      {{ error }} <ElButton @click="loadPage">{{ t('fuxiHall.fame.retry') }}</ElButton>
    </div>
    <article v-if="page?.description_html" class="hall-intro" v-html="page.description_html" />
    <section
      v-for="(floor, index) in page?.floors ?? []"
      :key="floor.key"
      class="hall-floor"
      :class="{ 'hall-floor--top': index === 0, 'hall-floor--second': index === 1 }"
      :aria-label="floor.name || t(`fuxiHall.fame.floors.${floor.key}`)"
    >
      <div class="hall-floor-heading">
        <h2>{{ floor.name || t(`fuxiHall.fame.floors.${floor.key}`) }}</h2>
        <span>{{ t('fuxiHall.fame.memberCount', { count: floorCards(floor.key).length }) }}</span>
      </div>
      <div class="hall-grid">
        <div v-for="card in floorCards(floor.key)" :key="card.id" class="hall-card">
          <div class="hall-card-body"><FuxiHallMemberCard :card="card" /></div>
          <ElButton v-if="card.can_edit" size="small" @click="edit(card)">{{
            t('fuxiHall.fame.editMine')
          }}</ElButton>
        </div>
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
  import { onMounted, ref } from 'vue'
  import { useI18n } from 'vue-i18n'
  import { fetchFuxiHall, updateOwnFuxiHallCard } from '@/api/fuxi-hall'
  import FuxiHallMemberCard from '../components/FuxiHallMemberCard.vue'
  const { t } = useI18n()
  const page = ref<Api.FuxiHall.PageConfig | null>(null)
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
  async function loadPage() {
    loading.value = true
    error.value = ''
    try {
      const result = await fetchFuxiHall()
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
      const updated = await updateOwnFuxiHallCard(editing.value.id, {
        nickname: nickname.value,
        description_html: description.value
      })
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
  onMounted(loadPage)
</script>
<style scoped>
  .hall-page {
    min-width: 0;
    padding: 24px;
  }
  .hall-header {
    max-width: 960px;
    margin: 0 auto 32px;
    text-align: center;
  }
  .hall-header h1 {
    font-size: clamp(28px, 4vw, 36px);
    font-weight: 600;
  }
  .hall-header p {
    margin-top: 12px;
    color: var(--el-text-color-secondary);
  }
  .hall-intro {
    max-width: 960px;
    margin: 24px auto;
    line-height: 1.7;
  }
  .hall-intro :deep(img) {
    max-width: 100%;
  }
  .hall-floor {
    width: 100%;
    margin: 32px auto;
  }
  .hall-floor--top {
    max-width: 960px;
  }
  .hall-floor--second {
    max-width: 1200px;
  }
  .hall-floor-heading {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    border-bottom: 1px solid var(--el-border-color);
    padding-bottom: 12px;
    margin-bottom: 16px;
  }
  .hall-floor-heading h2 {
    font-size: 20px;
    font-weight: 600;
  }
  .hall-floor-heading span {
    font-size: 14px;
    color: var(--el-text-color-secondary);
  }
  .hall-grid {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 16px;
  }
  .hall-floor--top .hall-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .hall-floor--second .hall-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
  .hall-card {
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .hall-card-body {
    flex: 1;
    min-height: 0;
  }
  .hall-error {
    color: var(--el-color-danger);
  }
  .hall-empty {
    text-align: center;
    color: var(--el-text-color-secondary);
    padding: 48px 0;
  }
  @media (max-width: 1199px) {
    .hall-grid,
    .hall-floor--second .hall-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
  }
  @media (max-width: 639px) {
    .hall-page {
      padding: 16px;
    }
    .hall-grid,
    .hall-floor--top .hall-grid,
    .hall-floor--second .hall-grid {
      grid-template-columns: minmax(0, 1fr);
    }
  }
</style>
