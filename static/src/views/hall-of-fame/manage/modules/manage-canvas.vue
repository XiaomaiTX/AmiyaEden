<template>
  <div class="manage-canvas-shell">
    <div
      ref="topScrollbarRef"
      class="manage-canvas-shell__top-scrollbar"
      @scroll="handleTopScrollbarScroll"
    >
      <div class="manage-canvas-shell__top-scrollbar-track" :style="scrollTrackStyle" />
    </div>

    <div ref="viewportRef" class="manage-canvas-shell__viewport" @scroll="handleViewportScroll">
      <div class="manage-canvas-shell__stage" :style="stageStyle">
        <div class="manage-canvas" :style="canvasStyle">
          <div v-for="card in cards" :key="card.id" class="manage-canvas__card">
            <DraggableCard
              :card="card"
              :selected="card.id === selectedCardId"
              :canvas-width="config.canvas_width"
              :canvas-height="config.canvas_height"
              :zoom-ratio="zoomRatio"
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
  </div>
</template>

<script setup lang="ts">
  import { computed, ref } from 'vue'

  import DraggableCard from './draggable-card.vue'

  const props = defineProps<{
    config: Api.HallOfFame.Config
    cards: Api.HallOfFame.Card[]
    selectedCardId: number | null
    zoomRatio: number
  }>()

  const emit = defineEmits<{
    'select-card': [id: number]
    'update-card-position': [id: number, posX: number, posY: number]
    'update-card-size': [id: number, width: number, height: number]
  }>()

  const topScrollbarRef = ref<HTMLElement | null>(null)
  const viewportRef = ref<HTMLElement | null>(null)
  let syncingScroll = false

  const canvasStyle = computed(() => ({
    width: `${props.config.canvas_width}px`,
    height: `${props.config.canvas_height}px`,
    transform: `scale(${props.zoomRatio})`,
    transformOrigin: 'top left',
    backgroundImage: props.config.background_image
      ? `url(${props.config.background_image})`
      : undefined
  }))

  const scrollTrackStyle = computed(() => ({
    width: `${props.config.canvas_width * props.zoomRatio + 32}px`
  }))

  const stageStyle = computed(() => ({
    width: `${props.config.canvas_width * props.zoomRatio}px`,
    height: `${props.config.canvas_height * props.zoomRatio}px`
  }))

  function syncScrollPosition(source: HTMLElement | null, target: HTMLElement | null) {
    if (!source || !target || syncingScroll) {
      return
    }

    syncingScroll = true
    target.scrollLeft = source.scrollLeft

    queueMicrotask(() => {
      syncingScroll = false
    })
  }

  function handleTopScrollbarScroll(event: Event) {
    syncScrollPosition(event.target as HTMLElement, viewportRef.value)
  }

  function handleViewportScroll(event: Event) {
    syncScrollPosition(event.target as HTMLElement, topScrollbarRef.value)
  }
</script>

<style scoped>
  .manage-canvas-shell {
    min-height: 0;
    min-width: 0;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    gap: 8px;
    border-radius: 28px;
    border: 1px solid rgba(255, 255, 255, 0.08);
    background: rgba(7, 14, 25, 0.92);
  }

  .manage-canvas-shell__top-scrollbar {
    overflow-x: auto;
    overflow-y: hidden;
    padding: 12px 16px 0;
  }

  .manage-canvas-shell__top-scrollbar-track {
    height: 1px;
  }

  .manage-canvas-shell__viewport {
    height: 100%;
    min-height: 720px;
    min-width: 0;
    overflow-x: auto;
    overflow-y: auto;
    padding: 16px;
  }

  .manage-canvas-shell__stage {
    position: relative;
  }

  .manage-canvas {
    position: absolute;
    top: 0;
    left: 0;
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
</style>
