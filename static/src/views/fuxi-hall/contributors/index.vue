<template>
  <div class="fuxi-hall-page art-full-height" v-loading="loading">
    <ElCard class="fuxi-hall-page__card art-table-card" shadow="never">
      <div class="fuxi-hall-page__content">
        <header class="fuxi-hall-page__header">
          <p class="fuxi-hall-page__eyebrow">{{ t('fuxiHall.public.contributorsEyebrow') }}</p>
          <h1>{{ page?.title || t('fuxiHall.public.defaultContributorsTitle') }}</h1>
          <p v-if="page?.subtitle" class="fuxi-hall-page__subtitle">{{ page.subtitle }}</p>
        </header>

        <article
          v-if="page?.description_html"
          class="fuxi-hall-page__intro"
          v-html="page.description_html"
        />

        <section v-if="cards.length > 0" class="fuxi-hall-page__grid">
          <article
            v-for="card in cards"
            :key="card.id"
            class="fuxi-hall-page__member"
            :style="{ '--accent-color': card.accent_color }"
          >
            <div class="fuxi-hall-page__cover">
              <img v-if="card.cover_image" :src="card.cover_image" :alt="card.nickname" />
            </div>

            <div class="fuxi-hall-page__body">
              <img
                v-if="card.main_character_id > 0"
                class="fuxi-hall-page__avatar"
                :class="`is-${card.avatar_shape}`"
                :src="buildEveCharacterPortraitUrl(card.main_character_id, 256)"
                :alt="card.nickname"
              />
              <h3>{{ card.nickname }}</h3>
              <p class="fuxi-hall-page__main-name">{{ card.main_character_name }}</p>
              <p class="fuxi-hall-page__title">{{ card.title }}</p>
              <div class="fuxi-hall-page__description" v-html="card.description_html" />
            </div>
          </article>
        </section>

        <section v-else-if="!loading" class="fuxi-hall-page__empty">
          <h2>{{ t('fuxiHall.public.emptyTitle') }}</h2>
          <p>{{ t('fuxiHall.public.emptySubtitle') }}</p>
        </section>
      </div>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
  import { onMounted, ref } from 'vue'
  import { ElMessage } from 'element-plus'
  import { useI18n } from 'vue-i18n'

  import { fetchFuxiHallContributors } from '@/api/fuxi-hall'
  import { buildEveCharacterPortraitUrl } from '@/utils/eve-image'

  const { t } = useI18n()
  const loading = ref(false)
  const page = ref<Api.FuxiHall.PageConfig | null>(null)
  const cards = ref<Api.FuxiHall.Card[]>([])

  onMounted(() => {
    void loadPage()
  })

  async function loadPage() {
    loading.value = true
    try {
      const response = await fetchFuxiHallContributors()
      page.value = response.page
      cards.value = response.cards
    } catch (error) {
      ElMessage.error(error instanceof Error ? error.message : t('fuxiHall.public.loadFailed'))
    } finally {
      loading.value = false
    }
  }
</script>

<style scoped>
  .fuxi-hall-page {
    min-height: 100%;
    min-width: 0;
    padding: 24px;
    overflow: auto;
    background:
      radial-gradient(circle at top left, rgb(34 197 94 / 14%), transparent 34%),
      radial-gradient(circle at top right, rgb(14 165 233 / 10%), transparent 28%),
      var(--el-bg-color-page);
  }

  .fuxi-hall-page__card,
  .fuxi-hall-page__card :deep(.el-card__body) {
    min-height: 100%;
  }

  .fuxi-hall-page__content {
    display: flex;
    flex-direction: column;
    gap: 20px;
  }

  .fuxi-hall-page__eyebrow {
    margin: 0;
    font-size: 12px;
    letter-spacing: 0.08em;
    color: var(--el-text-color-secondary);
    text-transform: uppercase;
  }

  .fuxi-hall-page__header h1 {
    margin: 6px 0 0;
    color: var(--el-text-color-primary);
    font-size: clamp(24px, 4vw, 34px);
  }

  .fuxi-hall-page__subtitle {
    margin: 8px 0 0;
    color: var(--el-text-color-regular);
  }

  .fuxi-hall-page__intro {
    border-radius: 14px;
    padding: 14px;
    background: var(--el-fill-color-light);
    color: var(--el-text-color-regular);
    line-height: 1.7;
  }

  .fuxi-hall-page__grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
    gap: 18px;
  }

  .fuxi-hall-page__member {
    overflow: hidden;
    border: 1px solid color-mix(in srgb, var(--accent-color), transparent 76%);
    border-radius: 16px;
    background: var(--el-bg-color);
  }

  .fuxi-hall-page__cover {
    height: 180px;
    background: color-mix(in srgb, var(--accent-color), var(--el-fill-color-light) 70%);
  }

  .fuxi-hall-page__cover img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    display: block;
  }

  .fuxi-hall-page__body {
    padding: 14px;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .fuxi-hall-page__avatar {
    width: 88px;
    height: 88px;
    object-fit: cover;
    border: 2px solid color-mix(in srgb, var(--accent-color), transparent 64%);
    margin-bottom: 2px;
    display: block;
  }

  .fuxi-hall-page__avatar.is-circle {
    border-radius: 999px;
  }

  .fuxi-hall-page__avatar.is-rounded {
    border-radius: 14px;
  }

  .fuxi-hall-page__avatar.is-square {
    border-radius: 8px;
  }

  .fuxi-hall-page__body h3 {
    margin: 0;
    font-size: 18px;
    color: var(--el-text-color-primary);
  }

  .fuxi-hall-page__main-name,
  .fuxi-hall-page__title {
    margin: 0;
    color: var(--el-text-color-regular);
  }

  .fuxi-hall-page__description {
    margin-top: 10px;
    color: var(--el-text-color-secondary);
    line-height: 1.65;
    overflow-wrap: anywhere;
  }

  .fuxi-hall-page__empty {
    border-radius: 14px;
    padding: 24px;
    text-align: center;
    background: var(--el-fill-color-light);
    color: var(--el-text-color-secondary);
  }

  .fuxi-hall-page__empty h2 {
    margin: 0;
    color: var(--el-text-color-primary);
  }

  .fuxi-hall-page__empty p {
    margin: 8px 0 0;
  }

  @media (max-width: 768px) {
    .fuxi-hall-page {
      padding: 16px;
    }
  }
</style>
