<template>
  <div class="hall-of-fame-temple art-full-height">
    <section class="hall-of-fame-temple__hero">
      <div class="hall-of-fame-temple__copy">
        <p class="hall-of-fame-temple__eyebrow">{{ t('hallOfFame.temple.eyebrow') }}</p>
        <h1>{{ t('menus.hallOfFame.temple') }}</h1>
      </div>
    </section>

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

  import { fetchTemple } from '@/api/hall-of-fame'

  import TempleCanvas from './modules/temple-canvas.vue'

  const { t } = useI18n()

  const loading = ref(false)
  const temple = ref<Api.HallOfFame.TempleResponse | null>(null)

  async function loadTemple() {
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
    gap: 20px;
    min-height: 100%;
    padding: 24px;
    background:
      radial-gradient(circle at top left, rgba(246, 206, 112, 0.16), transparent 24%),
      radial-gradient(circle at top right, rgba(104, 164, 255, 0.14), transparent 18%),
      linear-gradient(180deg, #07111f 0%, #0f1728 58%, #131c2c 100%);
  }

  .hall-of-fame-temple__hero {
    position: relative;
    overflow: hidden;
    border-radius: 28px;
    padding: 28px 32px;
    background:
      linear-gradient(135deg, rgba(19, 32, 58, 0.95), rgba(12, 19, 35, 0.92)),
      linear-gradient(180deg, rgba(255, 215, 0, 0.08), transparent);
    box-shadow: 0 18px 45px rgba(0, 0, 0, 0.22);
  }

  .hall-of-fame-temple__hero::after {
    content: '';
    position: absolute;
    inset: 0;
    background:
      linear-gradient(120deg, transparent 10%, rgba(255, 255, 255, 0.08) 38%, transparent 65%),
      radial-gradient(circle at right, rgba(255, 215, 0, 0.16), transparent 25%);
    pointer-events: none;
  }

  .hall-of-fame-temple__copy {
    position: relative;
    z-index: 1;
  }

  .hall-of-fame-temple__copy h1 {
    margin: 0;
    color: #fff7d6;
    font-size: clamp(30px, 4vw, 46px);
    font-weight: 700;
    letter-spacing: 0.06em;
  }

  .hall-of-fame-temple__eyebrow {
    margin: 0 0 10px;
    color: rgba(255, 255, 255, 0.72);
    font-size: 12px;
    letter-spacing: 0.32em;
    text-transform: uppercase;
  }

  .hall-of-fame-temple__card {
    flex: 1;
    border: 1px solid rgba(255, 255, 255, 0.06);
    border-radius: 32px;
    background: rgba(7, 15, 26, 0.6);
  }

  .hall-of-fame-temple__canvas-wrap {
    min-height: 60vh;
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

    .hall-of-fame-temple__hero {
      padding: 22px 20px;
    }
  }
</style>
