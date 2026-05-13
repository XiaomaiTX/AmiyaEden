<template>
  <div class="fuxi-hall-page art-full-height" v-loading="loading">
    <ElCard class="fuxi-hall-page__card art-table-card" shadow="never">
      <div class="fuxi-hall-page__content">
        <header class="fuxi-hall-page__header">
          <p class="fuxi-hall-page__eyebrow">{{ t('fuxiHall.public.leadershipEyebrow') }}</p>
          <h1>{{ page?.title || t('fuxiHall.public.defaultLeadershipTitle') }}</h1>
          <p v-if="page?.subtitle" class="fuxi-hall-page__subtitle">{{ page.subtitle }}</p>
        </header>

        <article
          v-if="page?.description_html"
          class="fuxi-hall-page__intro"
          v-html="page.description_html"
        />

        <section v-if="cards.length > 0" class="fuxi-hall-page__grid">
          <FuxiHallMemberCard v-for="card in cards" :key="card.id" :card="card" />
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

  import { fetchFuxiHallLeadership } from '@/api/fuxi-hall'
  import FuxiHallMemberCard from '../components/FuxiHallMemberCard.vue'

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
      const response = await fetchFuxiHallLeadership()
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
      radial-gradient(circle at top left, rgb(59 130 246 / 16%), transparent 32%),
      radial-gradient(circle at top right, rgb(2 132 199 / 12%), transparent 30%),
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
