<template>
  <div class="manage-canvas-shell">
    <div class="manage-canvas-shell__viewport">
      <div class="manage-canvas" :style="canvasStyle">
        <div
          v-for="card in cards"
          :key="card.id"
          class="manage-canvas__card"
          :class="{ 'is-hidden': !card.visible }"
        >
          <DraggableCard
            :card="card"
            :selected="card.id === selectedCardId"
            :canvas-width="config.canvas_width"
            :canvas-height="config.canvas_height"
            @select="emit('select-card', card.id)"
            @update:position="
              (payload) => emit('update-card-position', card.id, payload.posX, payload.posY)
            "
            @update:size="
              (payload) => emit('update-card-size', card.id, payload.width, payload.height)
            "
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue'

  import DraggableCard from './draggable-card.vue'

  const props = defineProps<{
    config: Api.HallOfFame.Config
    cards: Api.HallOfFame.Card[]
    selectedCardId: number | null
  }>()

  const emit = defineEmits<{
    'select-card': [id: number]
    'update-card-position': [id: number, posX: number, posY: number]
    'update-card-size': [id: number, width: number, height: number]
  }>()

  const canvasStyle = computed(() => ({
    width: `${props.config.canvas_width}px`,
    height: `${props.config.canvas_height}px`,
    backgroundImage: props.config.background_image
      ? `url(${props.config.background_image})`
      : undefined
  }))
</script>

<style scoped>
  .manage-canvas-shell {
    min-height: 0;
    overflow: hidden;
    border-radius: 28px;
    border: 1px solid rgba(255, 255, 255, 0.08);
    background: rgba(7, 14, 25, 0.92);
  }

  .manage-canvas-shell__viewport {
    height: 100%;
    min-height: 720px;
    overflow: auto;
    padding: 16px;
  }

  .manage-canvas {
    position: relative;
    border-radius: 24px;
    background:
      linear-gradient(180deg, rgba(10, 16, 30, 0.88), rgba(10, 16, 30, 0.96)),
      radial-gradient(circle at top, rgba(255, 215, 0, 0.14), transparent 22%);
    background-position: center;
    background-repeat: no-repeat;
    background-size: cover;
    box-shadow:
      inset 0 0 0 1px rgba(255, 255, 255, 0.08),
      0 20px 48px rgba(0, 0, 0, 0.25);
  }

  .manage-canvas::before {
    content: '';
    position: absolute;
    inset: 0;
    background-image:
      linear-gradient(rgba(255, 255, 255, 0.05) 1px, transparent 1px),
      linear-gradient(90deg, rgba(255, 255, 255, 0.05) 1px, transparent 1px);
    background-size: 48px 48px;
    pointer-events: none;
  }

  .manage-canvas__card.is-hidden {
    opacity: 0.5;
  }
</style>
