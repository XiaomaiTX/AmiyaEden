<template>
  <div class="hall-of-fame-temple art-full-height">
    <ElCard class="hall-of-fame-temple__card art-table-card" shadow="never" v-loading="loading">
      <div v-if="temple && temple.cards.length > 0" class="hall-of-fame-temple__canvas-wrap">
        <TempleCanvas :config="temple.config" :cards="temple.cards" />
      </div>

      <div v-else-if="!loading" class="hall-of-fame-temple__empty">
        <div class="hall-of-fame-temple__empty-icon">殿</div>
        <h2>{{ t('hallOfFame.temple.emptyTitle') }}</h2>
        <p>{{ t('hallOfFame.temple.emptySubtitle') }}</p>
      </div>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
  import { onMounted, ref } from 'vue'

  import { ElMessage } from 'element-plus'
  import { useI18n } from 'vue-i18n'
  import { useRoute } from 'vue-router'

  import { fetchTemple } from '@/api/hall-of-fame'

  import {
    HALL_OF_FAME_PREVIEW_QUERY_KEY,
    readHallOfFamePreviewDraft
  } from '../manage/modules/manage-preview.helpers'
  import TempleCanvas from './modules/temple-canvas.vue'

  const { t } = useI18n()
  const route = useRoute()

  const loading = ref(false)
  const temple = ref<Api.HallOfFame.TempleResponse | null>(null)

  async function loadTemple() {
    const previewId = String(route.query[HALL_OF_FAME_PREVIEW_QUERY_KEY] ?? '')
    const previewDraft = readHallOfFamePreviewDraft(window.localStorage, previewId)

    if (previewDraft) {
      temple.value = previewDraft
      return
    }

    loading.value = true

    try {
      temple.value = await fetchTemple()
    } catch (error) {
      const message = error instanceof Error ? error.message : ''
      if (message) {
        ElMessage.error(message)
      }
    } finally {
      loading.value = false
    }
  }

  onMounted(() => {
    void loadTemple()
  })
</script>

<style scoped>
  .hall-of-fame-temple {
    display: flex;
    flex-direction: column;
    min-height: 100%;
    min-width: 0;
    padding: 24px;
    background:
      radial-gradient(circle at top left, rgba(246, 206, 112, 0.16), transparent 24%),
      radial-gradient(circle at top right, rgba(104, 164, 255, 0.14), transparent 18%),
      linear-gradient(180deg, #07111f 0%, #0f1728 58%, #131c2c 100%);
  }

  .hall-of-fame-temple__card {
    display: flex;
    flex: 1;
    min-height: 0;
    min-width: 0;
    border: 1px solid rgba(255, 255, 255, 0.06);
    border-radius: 32px;
    background: rgba(7, 15, 26, 0.6);
  }

  .hall-of-fame-temple__canvas-wrap {
    width: 100%;
    height: 100%;
    min-height: 0;
    min-width: 0;
    overflow-x: auto;
    overflow-y: auto;
  }

  .hall-of-fame-temple__card :deep(.el-card__body) {
    display: flex;
    flex: 1;
    min-height: 0;
    min-width: 0;
    overflow-x: auto;
    overflow-y: auto;
  }

  .hall-of-fame-temple__empty {
    display: flex;
    min-height: 60vh;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 12px;
    text-align: center;
    color: rgba(255, 255, 255, 0.82);
  }

  .hall-of-fame-temple__empty h2 {
    margin: 0;
    color: #fff7d6;
    font-size: 28px;
  }

  .hall-of-fame-temple__empty p {
    margin: 0;
    max-width: 420px;
    line-height: 1.6;
  }

  .hall-of-fame-temple__empty-icon {
    display: flex;
    height: 84px;
    width: 84px;
    align-items: center;
    justify-content: center;
    border-radius: 24px;
    border: 1px solid rgba(255, 215, 0, 0.22);
    background: linear-gradient(180deg, rgba(255, 215, 0, 0.18), rgba(255, 215, 0, 0.05));
    color: #ffd86a;
    font-size: 34px;
    font-weight: 700;
  }

  @media (max-width: 768px) {
    .hall-of-fame-temple {
      padding: 16px;
    }
  }
</style>
