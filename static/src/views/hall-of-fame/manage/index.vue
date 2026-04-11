<template>
  <div class="hall-of-fame-manage art-full-height" v-loading="loading">
    <CanvasToolbar
      v-if="config"
      :canvas-width="config.canvas_width"
      :canvas-height="config.canvas_height"
      :zoom-percent="canvasZoom"
      :saving="saving"
      @add-card="handleAddCard"
      @upload-background="handleBackgroundUpload"
      @update:canvas-width="handleCanvasWidthChange"
      @update:canvas-height="handleCanvasHeightChange"
      @update:zoom-percent="handleCanvasZoomChange"
      @save-layout="handleSaveLayout"
      @preview="handlePreview"
    />

    <section v-if="config" class="hall-of-fame-manage__workspace">
      <ManageCanvas
        :config="config"
        :cards="cards"
        :selected-card-id="selectedCardId"
        :zoom-ratio="canvasZoom / 100"
        @select-card="handleSelectCard"
        @update-card-position="handleCardPositionUpdate"
        @update-card-size="handleCardSizeUpdate"
      />

      <CardEditor
        ref="cardEditorRef"
        :card="selectedCard"
        @update="handleCardUpdate"
        @upload-badge-image="handleBadgeImageUpload"
        @delete="handleCardDelete"
      />
    </section>
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, ref } from 'vue'

  import { ElMessage } from 'element-plus'
  import { useRouter } from 'vue-router'
  import { useI18n } from 'vue-i18n'

  import {
    batchUpdateHofLayout,
    createHofCard,
    deleteHofCard,
    fetchHofCards,
    fetchHofConfig,
    uploadHofBadgeImage,
    updateHofCard,
    updateHofConfig,
    uploadHofBackground
  } from '@/api/hall-of-fame'

  import CanvasToolbar from './modules/canvas-toolbar.vue'
  import CardEditor from './modules/card-editor.vue'
  import ManageCanvas from './modules/manage-canvas.vue'
  import { clampHallOfFameZoom } from '../temple/modules/temple-canvas.helpers'
  import {
    buildNewCardPayload,
    patchCardById,
    toLayoutUpdates
  } from './modules/manage-canvas.helpers'
  import { saveHallOfFamePreviewDraft } from './modules/manage-preview.helpers'
  import {
    getMissingCardIdFromError,
    rebuildCardFromConfirmedState,
    queueCardUpdateRequest,
    settleCardUpdateRequest,
    type CardUpdateQueueState
  } from './modules/manage-card-sync.helpers'

  const { t } = useI18n()
  const router = useRouter()

  const loading = ref(false)
  const saving = ref(false)
  const dirty = ref(false)
  const config = ref<Api.HallOfFame.Config | null>(null)
  const cards = ref<Api.HallOfFame.Card[]>([])
  const confirmedCards = ref<Api.HallOfFame.Card[]>([])
  const selectedCardId = ref<number | null>(null)
  const cardUpdateQueues = ref<Record<number, CardUpdateQueueState>>({})
  const cardEditorRef = ref<{ flushPendingUpdates: () => void } | null>(null)
  const canvasZoom = ref(100)

  const selectedCard = computed(
    () => cards.value.find((card) => card.id === selectedCardId.value) ?? null
  )

  onMounted(() => {
    void loadManagePage()
  })

  async function loadManagePage() {
    loading.value = true

    try {
      const [nextConfig, nextCards] = await Promise.all([fetchHofConfig(), fetchHofCards()])
      config.value = nextConfig
      cards.value = nextCards
      confirmedCards.value = nextCards
      selectedCardId.value = nextCards[0]?.id ?? null
    } catch (error) {
      showError(error)
    } finally {
      loading.value = false
    }
  }

  async function handleAddCard() {
    const maxZIndex = cards.value.reduce((highest, card) => Math.max(highest, card.z_index), 0)

    try {
      const created = await createHofCard(
        buildNewCardPayload(t('hallOfFame.manage.newCardName'), maxZIndex)
      )
      cards.value = [...cards.value, created]
      confirmedCards.value = [...confirmedCards.value, created]
      selectedCardId.value = created.id
    } catch (error) {
      showError(error)
    }
  }

  function handleSelectCard(id: number) {
    selectedCardId.value = id
  }

  function handleCanvasWidthChange(value: number) {
    if (!config.value) {
      return
    }

    config.value = {
      ...config.value,
      canvas_width: value
    }
    dirty.value = true
  }

  function handleCanvasHeightChange(value: number) {
    if (!config.value) {
      return
    }

    config.value = {
      ...config.value,
      canvas_height: value
    }
    dirty.value = true
  }

  function handleCanvasZoomChange(value: number) {
    canvasZoom.value = clampHallOfFameZoom(value)
  }

  function handleCardPositionUpdate(id: number, posX: number, posY: number) {
    cards.value = patchCardById(cards.value, id, {
      pos_x: posX,
      pos_y: posY
    })
    dirty.value = true
  }

  function handleCardSizeUpdate(id: number, width: number, height: number) {
    cards.value = patchCardById(cards.value, id, {
      width,
      height
    })
    dirty.value = true
  }

  async function handleSaveLayout() {
    if (!config.value) {
      return
    }

    saving.value = true

    try {
      await Promise.all([
        batchUpdateHofLayout(toLayoutUpdates(cards.value)),
        updateHofConfig({
          canvas_width: config.value.canvas_width,
          canvas_height: config.value.canvas_height
        })
      ])
      confirmedCards.value = confirmedCards.value.map((card) => {
        const visibleCard = cards.value.find((item) => item.id === card.id)
        if (!visibleCard) {
          return card
        }

        return {
          ...card,
          pos_x: visibleCard.pos_x,
          pos_y: visibleCard.pos_y,
          width: visibleCard.width,
          height: visibleCard.height,
          z_index: visibleCard.z_index
        }
      })
      dirty.value = false
      ElMessage.success(t('hallOfFame.manage.saveSuccess'))
    } catch (error) {
      const staleCardId = getMissingCardIdFromError(error)
      if (staleCardId !== null) {
        removeCardLocally(staleCardId)
      }
      showError(error)
    } finally {
      saving.value = false
    }
  }

  async function handleBackgroundUpload(file: File) {
    if (!config.value) {
      return
    }

    try {
      const { url } = await uploadHofBackground(file)
      config.value = await updateHofConfig({ background_image: url })
      ElMessage.success(t('hallOfFame.manage.saveSuccess'))
    } catch (error) {
      showError(error)
    }
  }

  async function handleBadgeImageUpload(id: number, file: File) {
    try {
      const { url } = await uploadHofBadgeImage(file)
      handleCardUpdate(id, { badge_image: url })
    } catch (error) {
      if (error instanceof Error && error.message === 'hall_of_fame_badge_image_too_large') {
        ElMessage.error(t('hallOfFame.manage.badgeImageTooLarge'))
        return
      }

      showError(error)
    }
  }

  function handleCardUpdate(id: number, updates: Api.HallOfFame.UpdateCardParams) {
    const currentCard = cards.value.find((card) => card.id === id)
    if (!currentCard) {
      return
    }

    cards.value = patchCardById(cards.value, id, updates)

    const queuedRequest = queueCardUpdateRequest(getCardUpdateQueue(id), updates)
    setCardUpdateQueue(id, queuedRequest.state)

    if (queuedRequest.patchToSend) {
      void persistCardUpdate(id, queuedRequest.patchToSend)
    }
  }

  async function handleCardDelete(id: number) {
    try {
      await deleteHofCard(id)
      cards.value = cards.value.filter((card) => card.id !== id)
      confirmedCards.value = confirmedCards.value.filter((card) => card.id !== id)
      removeCardUpdateQueue(id)
      if (selectedCardId.value === id) {
        selectedCardId.value = cards.value[0]?.id ?? null
      }
      ElMessage.success(t('hallOfFame.manage.deleteSuccess'))
    } catch (error) {
      showError(error)
    }
  }

  function handlePreview() {
    if (!config.value) {
      return
    }

    cardEditorRef.value?.flushPendingUpdates()

    const previewUrl = saveHallOfFamePreviewDraft(
      window.localStorage,
      router.resolve({ name: 'HallOfFameTemple' }).href,
      {
        config: config.value,
        cards: cards.value.filter((card) => card.visible)
      }
    )

    window.open(previewUrl, '_blank', 'noopener')
  }

  function showError(error: unknown) {
    const message = error instanceof Error ? error.message : t('hallOfFame.manage.saveFailed')
    ElMessage.error(message)
  }

  function getCardUpdateQueue(id: number): CardUpdateQueueState {
    return cardUpdateQueues.value[id] ?? { active: null, queued: null }
  }

  function setCardUpdateQueue(id: number, queue: CardUpdateQueueState) {
    const nextQueues = { ...cardUpdateQueues.value }

    if (!queue.active && !queue.queued) {
      delete nextQueues[id]
    } else {
      nextQueues[id] = queue
    }

    cardUpdateQueues.value = nextQueues
  }

  function removeCardUpdateQueue(id: number) {
    const nextQueues = { ...cardUpdateQueues.value }
    delete nextQueues[id]
    cardUpdateQueues.value = nextQueues
  }

  function removeCardLocally(id: number) {
    cards.value = cards.value.filter((card) => card.id !== id)
    confirmedCards.value = confirmedCards.value.filter((card) => card.id !== id)
    removeCardUpdateQueue(id)

    if (selectedCardId.value === id) {
      selectedCardId.value = cards.value[0]?.id ?? null
    }
  }

  function syncCardFromConfirmed(id: number, pendingPatch: Api.HallOfFame.UpdateCardParams | null) {
    const confirmedCard = confirmedCards.value.find((card) => card.id === id)
    const visibleCard = cards.value.find((card) => card.id === id)
    if (!confirmedCard || !visibleCard) {
      return
    }

    const nextCard = rebuildCardFromConfirmedState(confirmedCard, visibleCard, pendingPatch)
    cards.value = cards.value.map((card) => (card.id === id ? nextCard : card))
  }

  async function persistCardUpdate(id: number, updates: Api.HallOfFame.UpdateCardParams) {
    try {
      await updateHofCard(id, updates)
      confirmedCards.value = patchCardById(confirmedCards.value, id, updates)
    } catch (error) {
      const staleCardId = getMissingCardIdFromError(error)
      if (staleCardId !== null) {
        const hadCard = cards.value.some((card) => card.id === staleCardId)
        removeCardLocally(staleCardId)
        if (hadCard) {
          showError(error)
        }
        return
      }
      showError(error)
    } finally {
      const nextQueue = settleCardUpdateRequest(getCardUpdateQueue(id))
      setCardUpdateQueue(id, nextQueue.state)
      syncCardFromConfirmed(id, nextQueue.state.active)

      if (nextQueue.patchToSend) {
        void persistCardUpdate(id, nextQueue.patchToSend)
      }
    }
  }
</script>

<style scoped>
  .hall-of-fame-manage {
    display: flex;
    min-height: 100%;
    min-width: 0;
    flex-direction: column;
    gap: 18px;
    padding: 24px;
    background:
      radial-gradient(circle at top left, rgba(255, 215, 128, 0.14), transparent 24%),
      radial-gradient(circle at right, rgba(93, 137, 255, 0.16), transparent 20%),
      linear-gradient(180deg, #08111f 0%, #10182b 56%, #131d30 100%);
  }

  .hall-of-fame-manage__hero {
    display: flex;
    align-items: flex-end;
    justify-content: space-between;
    gap: 20px;
    border-radius: 28px;
    padding: 26px 30px;
    background: linear-gradient(135deg, rgba(12, 20, 38, 0.96), rgba(9, 14, 26, 0.92));
    box-shadow: 0 20px 45px rgba(0, 0, 0, 0.2);
  }

  .hall-of-fame-manage__eyebrow {
    margin: 0 0 10px;
    color: rgba(255, 255, 255, 0.64);
    font-size: 12px;
    letter-spacing: 0.3em;
    text-transform: uppercase;
  }

  .hall-of-fame-manage__hero h1 {
    margin: 0;
    color: #fff7d6;
    font-size: clamp(28px, 4vw, 42px);
    letter-spacing: 0.05em;
  }

  .hall-of-fame-manage__summary {
    margin: 0;
    max-width: 460px;
    color: rgba(255, 255, 255, 0.76);
    line-height: 1.7;
  }

  .hall-of-fame-manage__workspace {
    display: grid;
    min-height: 0;
    min-width: 0;
    flex: 1;
    grid-template-columns: minmax(0, 1fr) 340px;
    gap: 18px;
  }

  @media (max-width: 1180px) {
    .hall-of-fame-manage__hero {
      flex-direction: column;
      align-items: flex-start;
    }

    .hall-of-fame-manage__workspace {
      grid-template-columns: 1fr;
    }
  }

  @media (max-width: 768px) {
    .hall-of-fame-manage {
      padding: 16px;
    }

    .hall-of-fame-manage__hero {
      padding: 22px 20px;
    }
  }
</style>
