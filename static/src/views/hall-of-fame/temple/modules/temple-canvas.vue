<template>
  <div ref="containerRef" class="temple-canvas-shell">
    <div class="temple-canvas-shell__viewport" :style="viewportStyle">
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
</template>

<script setup lang="ts">
  import { computed, ref } from 'vue'

  import { useElementSize } from '@vueuse/core'

  import HeroCard from './hero-card.vue'
  import { getTempleScale } from './temple-canvas.helpers'

  const props = defineProps<{
    config: Api.HallOfFame.Config
    cards: Api.HallOfFame.Card[]
  }>()

  const containerRef = ref<HTMLElement | null>(null)
  const { width: containerWidth } = useElementSize(containerRef)

  const scale = computed(() =>
    getTempleScale(containerWidth.value, props.config.canvas_width, props.config.canvas_height)
  )

  const viewportStyle = computed(() => ({
    height: `${scale.value.wrapperHeight}px`
  }))

  const canvasStyle = computed(() => ({
    width: `${props.config.canvas_width}px`,
    height: `${props.config.canvas_height}px`,
    transform: `scale(${scale.value.ratio})`,
    backgroundImage: props.config.background_image
      ? `url(${props.config.background_image})`
      : undefined
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
    width: 100%;
    overflow-x: hidden;
    overflow-y: auto;
  }

  .temple-canvas-shell__viewport {
    position: relative;
    width: 100%;
  }

  .temple-canvas {
    position: absolute;
    top: 0;
    left: 0;
    transform-origin: top left;
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
