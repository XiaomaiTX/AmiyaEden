<template>
  <div class="fuxi-hall-manage" v-loading="loading">
    <ElCard class="fuxi-hall-manage__toolbar art-table-card" shadow="never">
      <div class="fuxi-hall-manage__toolbar-inner">
        <ElTabs v-model="currentPageKey" @tab-change="() => void loadPage()">
          <ElTabPane :label="t('fuxiHall.manage.leadershipTab')" name="leadership" />
          <ElTabPane :label="t('fuxiHall.manage.contributorsTab')" name="contributors" />
        </ElTabs>
      </div>
    </ElCard>

    <section class="fuxi-hall-manage__layout">
      <ElCard class="art-table-card" shadow="never">
        <template #header>
          <span>{{ t('fuxiHall.manage.pageConfig') }}</span>
        </template>

        <ElForm label-position="top">
          <ElFormItem :label="t('fuxiHall.manage.pageTitle')">
            <ElInput v-model="pageForm.title" />
          </ElFormItem>
          <ElFormItem :label="t('fuxiHall.manage.pageSubtitle')">
            <ElInput v-model="pageForm.subtitle" />
          </ElFormItem>
          <ElFormItem :label="t('fuxiHall.manage.pageDescription')">
            <ArtWangEditor v-model="pageForm.description_html" height="220px" />
          </ElFormItem>
          <ElFormItem>
            <ElButton type="primary" :loading="savingPage" @click="void savePage()">
              {{ t('common.save') }}
            </ElButton>
          </ElFormItem>
        </ElForm>
      </ElCard>

      <ElCard class="art-table-card" shadow="never">
        <template #header>
          <div class="fuxi-hall-manage__card-header">
            <span>{{ t('fuxiHall.manage.cardList') }}</span>
            <ElButton type="success" @click="openCreateCard">
              {{ t('fuxiHall.manage.addCard') }}
            </ElButton>
          </div>
        </template>

        <ElTable :data="cards" row-key="id">
          <ElTableColumn :label="t('fuxiHall.manage.order')" width="120">
            <template #default="{ $index }">
              <div class="fuxi-hall-manage__order-buttons">
                <ElButton size="small" text @click="void moveCard($index, -1)">↑</ElButton>
                <ElButton size="small" text @click="void moveCard($index, 1)">↓</ElButton>
              </div>
            </template>
          </ElTableColumn>
          <ElTableColumn prop="nickname" :label="t('fuxiHall.manage.nickname')" min-width="140" />
          <ElTableColumn
            prop="main_character_name"
            :label="t('fuxiHall.manage.mainCharacterName')"
            min-width="180"
          />
          <ElTableColumn
            prop="main_character_id"
            :label="t('fuxiHall.manage.mainCharacterId')"
            min-width="140"
          />
          <ElTableColumn prop="title" :label="t('fuxiHall.manage.title')" min-width="180" />
          <ElTableColumn prop="visible" :label="t('fuxiHall.manage.visible')" width="90">
            <template #default="{ row }">
              <ElTag :type="row.visible ? 'success' : 'info'">
                {{ row.visible ? t('fuxiHall.manage.visibleOn') : t('fuxiHall.manage.visibleOff') }}
              </ElTag>
            </template>
          </ElTableColumn>
          <ElTableColumn :label="t('common.operation')" width="200" fixed="right">
            <template #default="{ row }">
              <ElButton size="small" @click="openEditCard(row)">
                {{ t('common.edit') }}
              </ElButton>
              <ElButton size="small" type="danger" @click="void removeCard(row)">
                {{ t('common.delete') }}
              </ElButton>
            </template>
          </ElTableColumn>
        </ElTable>
      </ElCard>

      <ElCard class="art-table-card">
        <template #header>
          <span>{{ t('fuxiHall.manage.previewPanel') }}</span>
        </template>

        <div class="fuxi-hall-manage__preview">
          <header class="fuxi-hall-manage__preview-header">
            <h3>{{ previewPage.title || t('fuxiHall.manage.previewFallbackTitle') }}</h3>
            <p v-if="previewPage.subtitle" class="fuxi-hall-manage__preview-subtitle">
              {{ previewPage.subtitle }}
            </p>
          </header>

          <article
            v-if="previewPage.description_html"
            class="fuxi-hall-manage__preview-intro"
            v-html="previewPage.description_html"
          />

          <section v-if="previewCards.length > 0" class="fuxi-hall-manage__preview-grid">
            <article
              v-for="card in previewCards"
              :key="card.id"
              class="fuxi-hall-manage__preview-card"
              :style="{ '--accent-color': card.accent_color }"
            >
              <div class="fuxi-hall-manage__preview-body">
                <div class="fuxi-hall-manage__preview-meta">
                  <img
                    v-if="card.main_character_id > 0"
                    class="fuxi-hall-manage__preview-avatar"
                    :class="`is-${card.avatar_shape}`"
                    :src="buildEveCharacterPortraitUrl(card.main_character_id, 256)"
                    :alt="card.nickname"
                  />
                  <div>
                    <h4>{{ card.nickname }}</h4>
                    <p>{{ card.main_character_name }}</p>
                    <p>{{ card.title }}</p>
                  </div>
                </div>
                <div class="fuxi-hall-manage__preview-description" v-html="card.description_html" />
              </div>
            </article>
          </section>

          <section v-else class="fuxi-hall-manage__preview-empty">
            <span>{{ t('fuxiHall.manage.previewEmpty') }}</span>
          </section>
        </div>
      </ElCard>
    </section>

    <ElDialog v-model="cardDialogOpen" :title="cardDialogTitle" width="860px" destroy-on-close>
      <ElForm label-position="top">
        <div class="fuxi-hall-manage__dialog-grid">
          <ElFormItem :label="t('fuxiHall.manage.nickname')" required>
            <ElInput v-model="cardForm.nickname" />
          </ElFormItem>
          <ElFormItem :label="t('fuxiHall.manage.mainCharacterName')" required>
            <ElInput v-model="cardForm.main_character_name" />
          </ElFormItem>
          <ElFormItem :label="t('fuxiHall.manage.mainCharacterId')" required>
            <ElInputNumber v-model="cardForm.main_character_id" :min="1" />
          </ElFormItem>
          <ElFormItem :label="t('fuxiHall.manage.title')" required>
            <ElInput v-model="cardForm.title" />
          </ElFormItem>
          <ElFormItem :label="t('fuxiHall.manage.avatarShape')">
            <ElSelect v-model="cardForm.avatar_shape">
              <ElOption
                v-for="option in avatarShapeOptions"
                :key="option.value"
                :label="option.label"
                :value="option.value"
              />
            </ElSelect>
          </ElFormItem>
          <ElFormItem :label="t('fuxiHall.manage.accentColor')">
            <ElColorPicker v-model="cardForm.accent_color" :show-alpha="false" />
          </ElFormItem>
          <ElFormItem :label="t('fuxiHall.manage.fontScale')">
            <ElInputNumber v-model="cardForm.font_scale" :min="12" :max="20" />
          </ElFormItem>
          <ElFormItem :label="t('fuxiHall.manage.visible')">
            <ElSwitch v-model="cardForm.visible" />
          </ElFormItem>
        </div>

        <ElFormItem :label="t('fuxiHall.manage.cardDescription')">
          <ArtWangEditor v-model="cardForm.description_html" height="240px" />
        </ElFormItem>
      </ElForm>

      <template #footer>
        <ElButton @click="cardDialogOpen = false">{{ t('common.cancel') }}</ElButton>
        <ElButton type="primary" :loading="savingCard" @click="void submitCard()">
          {{ t('common.save') }}
        </ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, reactive, ref } from 'vue'
  import { ElMessage, ElMessageBox, ElTabPane, ElTabs } from 'element-plus'
  import { useI18n } from 'vue-i18n'

  import {
    createFuxiHallCard,
    deleteFuxiHallCard,
    fetchFuxiHallCards,
    fetchFuxiHallPage,
    reorderFuxiHallCards,
    updateFuxiHallCard,
    updateFuxiHallPage
  } from '@/api/fuxi-hall'
  import { buildEveCharacterPortraitUrl } from '@/utils/eve-image'

  const { t } = useI18n()

  const loading = ref(false)
  const savingPage = ref(false)
  const savingCard = ref(false)
  const currentPageKey = ref<Api.FuxiHall.PageKey>('leadership')
  const pageConfig = ref<Api.FuxiHall.PageConfig | null>(null)
  const cards = ref<Api.FuxiHall.Card[]>([])

  const cardDialogOpen = ref(false)
  const editingCardId = ref<number | null>(null)

  type PageFormState = {
    title: string
    subtitle: string
    description_html: string
  }

  type CardFormState = {
    page_key: Api.FuxiHall.PageKey
    nickname: string
    main_character_id: number
    main_character_name: string
    title: string
    description_html: string
    accent_color: string
    avatar_shape: Api.FuxiHall.AvatarShape
    font_scale: number
    visible: boolean
  }

  const pageForm = reactive<PageFormState>({
    title: '',
    subtitle: '',
    description_html: ''
  })

  const cardForm = reactive<CardFormState>({
    page_key: 'leadership',
    nickname: '',
    main_character_id: 0,
    main_character_name: '',
    title: '',
    description_html: '',
    accent_color: '#3b82f6',
    avatar_shape: 'circle',
    font_scale: 14,
    visible: true
  })

  const avatarShapeOptions = computed(() => [
    { label: t('fuxiHall.manage.avatarShapeCircle'), value: 'circle' },
    { label: t('fuxiHall.manage.avatarShapeRounded'), value: 'rounded' },
    { label: t('fuxiHall.manage.avatarShapeSquare'), value: 'square' }
  ])

  const cardDialogTitle = computed(() =>
    editingCardId.value ? t('fuxiHall.manage.editCard') : t('fuxiHall.manage.addCard')
  )

  const previewPage = computed(() => ({
    title: pageForm.title.trim(),
    subtitle: pageForm.subtitle.trim(),
    description_html: pageForm.description_html
  }))

  const previewCards = computed(() => {
    const visibleCards = cards.value.filter((card) => card.visible).map((card) => ({ ...card }))
    if (!cardDialogOpen.value) {
      return visibleCards
    }

    const previewCard = {
      id: editingCardId.value ?? -1,
      page_key: currentPageKey.value,
      nickname: cardForm.nickname.trim(),
      main_character_id: cardForm.main_character_id,
      main_character_name: cardForm.main_character_name.trim(),
      title: cardForm.title.trim(),
      description_html: cardForm.description_html,
      accent_color: cardForm.accent_color,
      avatar_shape: cardForm.avatar_shape,
      font_scale: cardForm.font_scale,
      visible: cardForm.visible,
      sort_order: editingCardId.value ? 0 : visibleCards.length + 1,
      created_at: '',
      updated_at: ''
    } as Api.FuxiHall.Card

    if (!previewCard.visible) {
      return editingCardId.value
        ? visibleCards.filter((card) => card.id !== editingCardId.value)
        : visibleCards
    }

    if (!editingCardId.value) {
      return [...visibleCards, previewCard]
    }

    const nextCards = visibleCards.filter((card) => card.id !== editingCardId.value)
    const existingIndex = cards.value.findIndex((card) => card.id === editingCardId.value)
    if (existingIndex < 0) {
      return [...nextCards, previewCard]
    }

    const insertIndex = Math.min(existingIndex, nextCards.length)
    nextCards.splice(insertIndex, 0, previewCard)
    return nextCards
  })

  onMounted(() => {
    void loadPage()
  })

  async function loadPage() {
    loading.value = true
    try {
      const [nextPage, nextCards] = await Promise.all([
        fetchFuxiHallPage(currentPageKey.value),
        fetchFuxiHallCards(currentPageKey.value)
      ])
      pageConfig.value = nextPage
      cards.value = nextCards
      pageForm.title = nextPage.title
      pageForm.subtitle = nextPage.subtitle
      pageForm.description_html = nextPage.description_html
    } catch (error) {
      ElMessage.error(error instanceof Error ? error.message : t('fuxiHall.manage.loadFailed'))
    } finally {
      loading.value = false
    }
  }

  async function savePage() {
    if (!pageForm.title?.trim()) {
      ElMessage.warning(t('fuxiHall.manage.titleRequired'))
      return
    }

    savingPage.value = true
    try {
      const saved = await updateFuxiHallPage(currentPageKey.value, {
        title: pageForm.title.trim(),
        subtitle: pageForm.subtitle?.trim() || '',
        description_html: pageForm.description_html || ''
      })
      pageConfig.value = saved
      ElMessage.success(t('fuxiHall.manage.saveSuccess'))
    } catch (error) {
      ElMessage.error(error instanceof Error ? error.message : t('fuxiHall.manage.saveFailed'))
    } finally {
      savingPage.value = false
    }
  }

  function resetCardForm() {
    cardForm.page_key = currentPageKey.value
    cardForm.nickname = ''
    cardForm.main_character_id = 0
    cardForm.main_character_name = ''
    cardForm.title = ''
    cardForm.description_html = ''
    cardForm.accent_color = '#3b82f6'
    cardForm.avatar_shape = 'circle'
    cardForm.font_scale = 14
    cardForm.visible = true
  }

  function openCreateCard() {
    editingCardId.value = null
    resetCardForm()
    cardDialogOpen.value = true
  }

  function openEditCard(card: Api.FuxiHall.Card) {
    editingCardId.value = card.id
    cardForm.page_key = card.page_key
    cardForm.nickname = card.nickname
    cardForm.main_character_id = card.main_character_id
    cardForm.main_character_name = card.main_character_name
    cardForm.title = card.title
    cardForm.description_html = card.description_html
    cardForm.accent_color = card.accent_color
    cardForm.avatar_shape = card.avatar_shape
    cardForm.font_scale = card.font_scale
    cardForm.visible = card.visible
    cardDialogOpen.value = true
  }

  async function submitCard() {
    if (
      !cardForm.nickname.trim() ||
      cardForm.main_character_id <= 0 ||
      !cardForm.main_character_name.trim() ||
      !cardForm.title.trim()
    ) {
      ElMessage.warning(t('fuxiHall.manage.requiredFields'))
      return
    }

    const payload: Api.FuxiHall.CreateCardParams = {
      page_key: currentPageKey.value,
      nickname: cardForm.nickname.trim(),
      main_character_id: cardForm.main_character_id,
      main_character_name: cardForm.main_character_name.trim(),
      title: cardForm.title.trim(),
      description_html: cardForm.description_html || '',
      accent_color: cardForm.accent_color,
      avatar_shape: cardForm.avatar_shape,
      font_scale: cardForm.font_scale,
      visible: cardForm.visible
    }

    savingCard.value = true
    try {
      if (editingCardId.value) {
        await updateFuxiHallCard(editingCardId.value, payload)
      } else {
        await createFuxiHallCard(payload)
      }
      await loadCards()
      cardDialogOpen.value = false
      ElMessage.success(t('fuxiHall.manage.saveSuccess'))
    } catch (error) {
      ElMessage.error(error instanceof Error ? error.message : t('fuxiHall.manage.saveFailed'))
    } finally {
      savingCard.value = false
    }
  }

  async function loadCards() {
    cards.value = await fetchFuxiHallCards(currentPageKey.value)
  }

  async function removeCard(card: Api.FuxiHall.Card) {
    try {
      await ElMessageBox.confirm(t('fuxiHall.manage.deleteConfirm'), t('common.tips'), {
        type: 'warning',
        confirmButtonText: t('common.confirm'),
        cancelButtonText: t('common.cancel')
      })
    } catch {
      return
    }

    try {
      await deleteFuxiHallCard(card.id)
      await loadCards()
      ElMessage.success(t('fuxiHall.manage.deleteSuccess'))
    } catch (error) {
      ElMessage.error(error instanceof Error ? error.message : t('fuxiHall.manage.saveFailed'))
    }
  }

  async function moveCard(index: number, delta: number) {
    const nextIndex = index + delta
    if (nextIndex < 0 || nextIndex >= cards.value.length) {
      return
    }

    const previous = cards.value.map((card) => ({ ...card }))
    const reordered = cards.value.map((card) => ({ ...card }))
    const current = reordered[index]
    reordered[index] = reordered[nextIndex]
    reordered[nextIndex] = current
    reordered.forEach((card, idx) => {
      card.sort_order = idx + 1
    })
    cards.value = reordered

    try {
      await reorderFuxiHallCards({
        page_key: currentPageKey.value,
        ordered_ids: reordered.map((card) => card.id)
      })
    } catch (error) {
      cards.value = previous
      ElMessage.error(error instanceof Error ? error.message : t('fuxiHall.manage.sortFailed'))
    }
  }
</script>

<style scoped>
  .fuxi-hall-manage {
    display: flex;
    flex-direction: column;
    gap: 16px;
    min-height: 0;
    padding: 20px;
  }

  .fuxi-hall-manage__toolbar-inner {
    display: flex;
    align-items: flex-end;
    gap: 12px;
    flex-wrap: wrap;
  }

  .fuxi-hall-manage__toolbar-inner :deep(.el-tabs__header) {
    margin: 0;
  }

  .fuxi-hall-manage__card-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }

  .fuxi-hall-manage__layout {
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    gap: 16px;
    min-height: 0;
  }

  .fuxi-hall-manage__order-buttons {
    display: flex;
    gap: 4px;
  }

  .fuxi-hall-manage__dialog-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
    gap: 0 12px;
  }

  .fuxi-hall-manage__preview {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .fuxi-hall-manage__preview-header h3 {
    margin: 0;
    font-size: 22px;
    color: var(--el-text-color-primary);
  }

  .fuxi-hall-manage__preview-subtitle {
    margin: 8px 0 0;
    color: var(--el-text-color-regular);
  }

  .fuxi-hall-manage__preview-intro {
    border-radius: 12px;
    background: var(--el-fill-color-light);
    color: var(--el-text-color-regular);
    line-height: 1.65;
    padding: 12px;
  }

  .fuxi-hall-manage__preview-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
    gap: 16px;
  }

  .fuxi-hall-manage__preview-card {
    border: 1px solid color-mix(in srgb, var(--accent-color), transparent 75%);
    border-radius: 16px;
    overflow: hidden;
    background: var(--el-bg-color);
  }

  .fuxi-hall-manage__preview-body {
    padding: 14px;
    color: var(--el-text-color-regular);
  }

  .fuxi-hall-manage__preview-meta {
    display: flex;
    align-items: flex-start;
    gap: 12px;
  }

  .fuxi-hall-manage__preview-avatar {
    width: 72px;
    height: 72px;
    flex-shrink: 0;
    object-fit: cover;
    border: 2px solid color-mix(in srgb, var(--accent-color), transparent 62%);
    display: block;
  }

  .fuxi-hall-manage__preview-avatar.is-circle {
    border-radius: 999px;
  }

  .fuxi-hall-manage__preview-avatar.is-rounded {
    border-radius: 12px;
  }

  .fuxi-hall-manage__preview-avatar.is-square {
    border-radius: 8px;
  }

  .fuxi-hall-manage__preview-body h4 {
    margin: 0;
    color: var(--el-text-color-primary);
    font-size: 17px;
  }

  .fuxi-hall-manage__preview-body p {
    margin: 4px 0 0;
  }

  .fuxi-hall-manage__preview-description {
    margin-top: 8px;
    color: var(--el-text-color-secondary);
    line-height: 1.6;
    overflow-wrap: anywhere;
  }

  .fuxi-hall-manage__preview-empty {
    border-radius: 12px;
    background: var(--el-fill-color-light);
    color: var(--el-text-color-secondary);
    padding: 16px;
    text-align: center;
  }

  @media (max-width: 768px) {
    .fuxi-hall-manage {
      padding: 14px;
    }

    .fuxi-hall-manage__preview-grid {
      grid-template-columns: minmax(0, 1fr);
    }
  }
</style>
