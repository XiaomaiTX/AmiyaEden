<template>
  <div ref="cardRef" class="draggable-card" :style="cardStyle" @click="emit('select')">
    <HeroCard :card="previewCard" interactive :selected="selected" />
    <div class="draggable-card__hint">{{ t('hallOfFame.manage.dragHint') }}</div>
  </div>
</template>

<script setup lang="ts">
  import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'

  import interact from 'interactjs'
  import { useI18n } from 'vue-i18n'

  import HeroCard from '../../temple/modules/hero-card.vue'
  import { clampCardCoordinate } from './manage-canvas.helpers'

  const props = defineProps<{
    card: Api.HallOfFame.Card
    selected: boolean
    canvasWidth: number
    canvasHeight: number
  }>()

  const emit = defineEmits<{
    select: []
    'update:position': [payload: { posX: number; posY: number }]
    'update:size': [payload: { width: number; height: number }]
  }>()

  const { t } = useI18n()

  const cardRef = ref<HTMLElement | null>(null)
  const pixelLeft = ref(0)
  const pixelTop = ref(0)
  const cardWidth = ref(props.card.width || 220)
  const cardHeight = ref(props.card.height || 0)

  const previewCard = computed<Api.HallOfFame.Card>(() => ({
    ...props.card,
    width: cardWidth.value,
    height: cardHeight.value
  }))

  const cardStyle = computed(() => ({
    left: `${pixelLeft.value}px`,
    top: `${pixelTop.value}px`,
    width: `${cardWidth.value}px`,
    zIndex: String(props.card.z_index)
  }))

  watch(
    () => [
      props.card.pos_x,
      props.card.pos_y,
      props.card.width,
      props.card.height,
      props.canvasWidth,
      props.canvasHeight
    ],
    () => {
      syncFromProps()
    },
    { immediate: true }
  )

  let interaction: ReturnType<typeof interact> | null = null

  onMounted(() => {
    if (!cardRef.value) {
      return
    }

    interaction = interact(cardRef.value)
      .draggable({
        listeners: {
          move(event) {
            const target = cardRef.value
            if (!target) {
              return
            }

            const nextLeft = clampPx(
              pixelLeft.value + event.dx,
              props.canvasWidth - target.offsetWidth
            )
            const nextTop = clampPx(
              pixelTop.value + event.dy,
              props.canvasHeight - target.offsetHeight
            )

            pixelLeft.value = nextLeft
            pixelTop.value = nextTop

            emit('update:position', {
              posX: clampCardCoordinate((nextLeft / props.canvasWidth) * 100),
              posY: clampCardCoordinate((nextTop / props.canvasHeight) * 100)
            })
          }
        }
      })
      .resizable({
        edges: { right: true, bottom: true },
        listeners: {
          move(event) {
            cardWidth.value = Math.max(180, Math.round(event.rect.width))
            cardHeight.value = Math.max(0, Math.round(event.rect.height))

            emit('update:size', {
              width: cardWidth.value,
              height: cardHeight.value
            })
          }
        }
      })
  })

  onBeforeUnmount(() => {
    interaction?.unset()
  })

  function syncFromProps() {
    cardWidth.value = props.card.width || 220
    cardHeight.value = props.card.height || 0
    pixelLeft.value = (props.card.pos_x / 100) * props.canvasWidth
    pixelTop.value = (props.card.pos_y / 100) * props.canvasHeight
  }

  function clampPx(value: number, max: number) {
    if (max <= 0) {
      return 0
    }

    return Math.max(0, Math.min(max, value))
  }
</script>

<style scoped>
  .draggable-card {
    position: absolute;
    cursor: move;
    user-select: none;
  }

  .draggable-card__hint {
    position: absolute;
    right: 10px;
    top: 10px;
    border-radius: 999px;
    background: rgba(4, 10, 20, 0.72);
    color: rgba(255, 255, 255, 0.72);
    font-size: 10px;
    letter-spacing: 0.18em;
    padding: 4px 8px;
    text-transform: uppercase;
  }
</style>
