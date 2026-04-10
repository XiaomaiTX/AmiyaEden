<template>
  <div class="temple-canvas-shell">
    <div class="temple-canvas-shell__viewport">
      <div class="temple-canvas-shell__stage" :style="stageStyle">
        <div class="temple-canvas" :style="canvasStyle">
          <div
            v-for="card in cards"
            :key="card.id"
            class="temple-canvas__card"
            :style="getCardPositionStyle(card)"
          >
            <HeroCard :card="card" interactive />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue'

  import HeroCard from './hero-card.vue'

  const props = defineProps<{
    config: Api.HallOfFame.Config
    cards: Api.HallOfFame.Card[]
  }>()

  const canvasStyle = computed(() => ({
    width: `${props.config.canvas_width}px`,
    height: `${props.config.canvas_height}px`,
    backgroundImage: props.config.background_image
      ? `url(${props.config.background_image})`
      : undefined
  }))

  const stageStyle = computed(() => ({
    width: `${props.config.canvas_width}px`,
    height: `${props.config.canvas_height}px`
  }))

  function getCardPositionStyle(card: Api.HallOfFame.Card) {
    return {
      left: `${card.pos_x}%`,
      top: `${card.pos_y}%`,
      zIndex: String(card.z_index)
    }
  }
</script>

<style scoped>
  .temple-canvas-shell {
    width: max-content;
    min-width: 100%;
  }

  .temple-canvas-shell__viewport {
    overflow: visible;
  }

  .temple-canvas-shell__stage {
    position: relative;
  }

  .temple-canvas {
    position: absolute;
    inset: 0 auto auto 0;
    border-radius: 32px;
    overflow: hidden;
    background:
      radial-gradient(circle at top, rgba(250, 216, 122, 0.16), transparent 24%),
      linear-gradient(180deg, #0a1020 0%, #091320 42%, #111827 100%);
    background-repeat: no-repeat;
    background-position: center;
    background-size: cover;
    box-shadow:
      inset 0 0 0 1px rgba(255, 255, 255, 0.08),
      0 24px 60px rgba(0, 0, 0, 0.34);
  }

  .temple-canvas::after {
    content: '';
    position: absolute;
    inset: 0;
    background:
      linear-gradient(180deg, rgba(5, 7, 15, 0.24), rgba(5, 7, 15, 0.5)),
      radial-gradient(circle at bottom, rgba(255, 255, 255, 0.06), transparent 30%);
    pointer-events: none;
  }

  .temple-canvas__card {
    position: absolute;
  }
</style>
