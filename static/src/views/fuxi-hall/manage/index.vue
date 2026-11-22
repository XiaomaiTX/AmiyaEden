<template>
  <div class="fuxi-hall-manage" v-loading="loading">
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
          <fieldset class="hall-floors">
            <legend>{{ t('fuxiHall.fame.floorSettings') }}</legend
            ><p>{{ t('fuxiHall.fame.floorHint') }}</p>
            <div v-for="(floor, index) in pageForm.floors" :key="floor.key" class="hall-floor-row">
              <ElInput
                v-model="floor.name"
                :aria-label="t('fuxiHall.fame.floorName')"
                :maxlength="64"
                :placeholder="
                  ['top', 'second', 'first'].includes(floor.key)
                    ? t(`fuxiHall.fame.floors.${floor.key}`)
                    : t('fuxiHall.fame.floorName')
                "
              />
              <ElInputNumber
                v-model="floor.capacity"
                :aria-label="t('fuxiHall.fame.capacity')"
                :min="1"
                :max="10000"
              />
              <ElButton
                :aria-label="t('fuxiHall.fame.moveUp')"
                :disabled="index === 0"
                @click="moveFloor(index, -1)"
                >↑</ElButton
              >
              <ElButton
                :aria-label="t('fuxiHall.fame.moveDown')"
                :disabled="index === pageForm.floors.length - 1"
                @click="moveFloor(index, 1)"
                >↓</ElButton
              >
              <ElButton @click="pageForm.floors.splice(index, 1)">{{
                t('common.delete')
              }}</ElButton>
            </div>
            <ElButton
              :disabled="pageForm.floors.length >= 12"
              @click="
                pageForm.floors.push({
                  key: `floor-${Date.now()}`,
                  name: t('fuxiHall.fame.newFloor'),
                  capacity: 12
                })
              "
              >{{ t('fuxiHall.fame.addFloor') }}</ElButton
            >
          </fieldset>
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
            <div
              ><ElButton
                :loading="refreshing"
                :disabled="refreshing || savingCard"
                @click="refreshContributions"
                >{{ t('fuxiHall.contributions.refresh') }}</ElButton
              ><ElButton type="success" @click="openCreateCard">
                {{ t('fuxiHall.manage.addCard') }}
              </ElButton></div
            >
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
          <ElTableColumn :label="t('fuxiHall.fame.floor')" min-width="120"
            ><template #default="{ row }">{{
              pageForm.floors.find((floor) => floor.key === row.floor_key)?.name ||
              t(`fuxiHall.fame.floors.${row.floor_key}`)
            }}</template></ElTableColumn
          >
          <ElTableColumn prop="nickname" :label="t('fuxiHall.manage.nickname')" min-width="140" />
          <ElTableColumn
            prop="main_character_name"
            :label="t('fuxiHall.manage.mainCharacterName')"
            min-width="180"
          />
          <ElTableColumn :label="t('fuxiHall.manage.title')" min-width="220">
            <template #default="{ row }">
              <div class="fuxi-hall-manage__title-tags">
                <ElTag
                  v-for="tag in row.title_tags"
                  :key="`${row.id}-${tag}`"
                  size="small"
                  effect="plain"
                  round
                  :style="getTagStyle(row.accent_color)"
                >
                  {{ tag }}
                </ElTag>
              </div>
            </template>
          </ElTableColumn>
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

          <section v-for="floor in pageForm.floors" :key="floor.key">
            <h4
              >{{ floor.name || t(`fuxiHall.fame.floors.${floor.key}`) }} ·
              {{ previewCards.filter((card) => card.floor_key === floor.key).length }}/{{
                floor.capacity
              }}</h4
            >
            <div
              class="fuxi-hall-manage__preview-grid"
              :style="{
                maxWidth: `${Math.max(1, previewCards.filter((card) => card.floor_key === floor.key).length) * 396 - 16}px`
              }"
            >
              <FuxiHallMemberCard
                v-for="card in previewCards.filter((item) => item.floor_key === floor.key)"
                :key="card.id"
                :card="card"
              />
            </div>
          </section>

          <section v-if="!previewCards.length" class="fuxi-hall-manage__preview-empty">
            <span>{{ t('fuxiHall.manage.previewEmpty') }}</span>
          </section>
        </div>
      </ElCard>
    </section>

    <ElDialog
      v-model="cardDialogOpen"
      :title="cardDialogTitle"
      width="min(1180px, 94vw)"
      top="4vh"
      class="fuxi-hall-card-dialog"
      :close-on-click-modal="false"
      :before-close="requestCloseCard"
      destroy-on-close
      @opened="snapshotCardForm"
    >
      <ElForm label-position="top">
        <div class="fuxi-hall-manage__dialog-grid">
          <ElFormItem :label="t('fuxiHall.fame.floor')"
            ><ElSelect v-model="cardForm.floor_key"
              ><ElOption
                v-for="floor in pageForm.floors"
                :key="floor.key"
                :value="floor.key"
                :label="floor.name || t(`fuxiHall.fame.floors.${floor.key}`)" /></ElSelect
          ></ElFormItem>
          <ElFormItem :label="t('fuxiHall.fame.current')"
            ><ElSwitch v-model="cardForm.is_current_management"
          /></ElFormItem>
          <ElFormItem :label="t('fuxiHall.manage.nickname')" required>
            <ElInput v-model="cardForm.nickname" />
          </ElFormItem>
          <ElFormItem :label="t('fuxiHall.manage.mainCharacterName')" required>
            <ElInput v-model="cardForm.main_character_name" />
          </ElFormItem>
          <ElFormItem
            :label="t('fuxiHall.manage.title')"
            :required="cardForm.is_current_management"
          >
            <div class="fuxi-hall-manage__tag-editor">
              <ElInput
                v-model="titleTagInput"
                :placeholder="t('fuxiHall.manage.titleTagPlaceholder')"
                @keyup.enter="addTitleTag"
              />
              <ElButton type="primary" plain @click="addTitleTag">
                {{ t('fuxiHall.manage.addTitleTag') }}
              </ElButton>
            </div>
            <div class="fuxi-hall-manage__title-tags fuxi-hall-manage__title-tags--editor">
              <ElTag
                v-for="tag in cardForm.title_tags"
                :key="`form-${tag}`"
                size="small"
                effect="plain"
                closable
                round
                :style="getTagStyle(cardForm.accent_color)"
                @close="removeTitleTag(tag)"
              >
                {{ tag }}
              </ElTag>
            </div>
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
          <fieldset class="hall-offsets">
            <legend>{{ t('fuxiHall.contributions.offsets') }}</legend>
            <p>{{ t('fuxiHall.contributions.offsetHint') }}</p>
            <div
              ><ElFormItem
                v-for="key in contributionKeys"
                :key="key"
                :label="t(`fuxiHall.contributions.${key}`)"
                ><ElInputNumber
                  v-model="cardForm.contribution_offsets[key]"
                  :min="0"
                  :max="1000000000"
                  :precision="0" /></ElFormItem
            ></div>
          </fieldset>
        </div>

        <ElFormItem :label="t('fuxiHall.manage.cardDescription')">
          <ArtWangEditor v-model="cardForm.description_html" height="220px" />
        </ElFormItem>
      </ElForm>

      <template #footer>
        <ElButton @click="void requestCloseCard()">{{ t('common.cancel') }}</ElButton>
        <ElButton type="primary" :loading="savingCard" @click="void submitCard()">
          {{ t('common.save') }}
        </ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, onActivated, onMounted, reactive, ref } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import { useI18n } from 'vue-i18n'

  import {
    refreshFuxiHallContributions,
    createFuxiHallCard,
    deleteFuxiHallCard,
    fetchFuxiHallCards,
    fetchFuxiHallPage,
    reorderFuxiHallCards,
    updateFuxiHallCard,
    updateFuxiHallPage
  } from '@/api/fuxi-hall'
  import FuxiHallMemberCard from '../components/FuxiHallMemberCard.vue'

  const { t } = useI18n()

  const contributionKeys = ['fleets', 'welfare', 'srp', 'recruitment', 'mentoring'] as const
  const refreshing = ref(false)
  const loading = ref(false)
  const savingPage = ref(false)
  const savingCard = ref(false)
  const currentPageKey = ref<Api.FuxiHall.PageKey>('hall-of-fame')
  const pageConfig = ref<Api.FuxiHall.PageConfig | null>(null)
  const cards = ref<Api.FuxiHall.Card[]>([])

  const cardDialogOpen = ref(false)
  const editingCardId = ref<number | null>(null)
  const titleTagInput = ref('')

  type PageFormState = {
    title: string
    subtitle: string
    description_html: string
    floors: Api.FuxiHall.Floor[]
  }

  type CardFormState = {
    page_key: Api.FuxiHall.PageKey
    nickname: string
    main_character_name: string
    title_tags: string[]
    description_html: string
    accent_color: string
    avatar_shape: Api.FuxiHall.AvatarShape
    font_scale: number
    visible: boolean
    floor_key: string
    is_current_management: boolean
    contribution_offsets: Api.FuxiHall.Contributions
  }

  const pageForm = reactive<PageFormState>({
    title: '',
    subtitle: '',
    description_html: '',
    floors: []
  })

  const cardForm = reactive<CardFormState>({
    page_key: 'hall-of-fame',
    nickname: '',
    main_character_name: '',
    title_tags: [],
    description_html: '',
    accent_color: '#3b82f6',
    avatar_shape: 'circle',
    font_scale: 14,
    visible: true,
    floor_key: 'first',
    is_current_management: true,
    contribution_offsets: { fleets: 0, welfare: 0, srp: 0, recruitment: 0, mentoring: 0 }
  })

  const cardDialogTitle = computed(() =>
    t(editingCardId.value ? 'fuxiHall.manage.editCard' : 'fuxiHall.manage.addCard')
  )
  const avatarShapeOptions = computed(() =>
    ['circle', 'rounded', 'square'].map((value) => ({
      value,
      label: t(`fuxiHall.manage.avatarShape${value.charAt(0).toUpperCase() + value.slice(1)}`)
    }))
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
      main_character_id: 0,
      main_character_name: cardForm.main_character_name.trim(),
      title_tags: [...cardForm.title_tags],
      description_html: cardForm.description_html,
      accent_color: cardForm.accent_color,
      avatar_shape: cardForm.avatar_shape,
      font_scale: cardForm.font_scale,
      floor_key: cardForm.floor_key,
      is_current_management: cardForm.is_current_management,
      visible: cardForm.visible,
      contribution_offsets: { ...cardForm.contribution_offsets },
      qq: cards.value.find((card) => card.id === editingCardId.value)?.qq ?? '',
      contributions: contributionKeys.reduce<Api.FuxiHall.Contributions>(
        (totals, key) => {
          const existing = cards.value.find((card) => card.id === editingCardId.value)
          totals[key] =
            Math.max(
              0,
              (existing?.contributions?.[key] ?? 0) - (existing?.contribution_offsets?.[key] ?? 0)
            ) + cardForm.contribution_offsets[key]
          return totals
        },
        {
          corp_pap:
            cards.value.find((card) => card.id === editingCardId.value)?.contributions?.corp_pap ??
            0,
          fleets: 0,
          welfare: 0,
          srp: 0,
          recruitment: 0,
          mentoring: 0
        }
      ),
      contributions_updated_at: cards.value.find((card) => card.id === editingCardId.value)
        ?.contributions_updated_at,
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
  // Returning from the cached hall-of-fame tab picks up card edits made there. Only cards are
  // refreshed so unsaved page-form edits survive; a failed background refresh keeps the current list.
  onActivated(() => {
    if (pageConfig.value) loadCards().catch(() => undefined)
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
      pageForm.floors = nextPage.floors.map((floor) => ({ ...floor }))
      pageForm.title = nextPage.title
      pageForm.subtitle = nextPage.subtitle
      pageForm.description_html = nextPage.description_html
    } catch (error) {
      ElMessage.error(
        error instanceof Error
          ? error.message.startsWith('fuxiHall.')
            ? t(error.message)
            : error.message
          : t('fuxiHall.manage.loadFailed')
      )
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
        floors: pageForm.floors,
        description_html: pageForm.description_html || ''
      })
      pageConfig.value = saved
      ElMessage.success(t('fuxiHall.manage.saveSuccess'))
    } catch (error) {
      ElMessage.error(
        error instanceof Error
          ? error.message.startsWith('fuxiHall.')
            ? t(error.message)
            : error.message
          : t('fuxiHall.manage.saveFailed')
      )
    } finally {
      savingPage.value = false
    }
  }

  function resetCardForm() {
    cardForm.page_key = currentPageKey.value
    cardForm.floor_key = pageForm.floors.at(-1)?.key ?? 'first'
    cardForm.is_current_management = true
    cardForm.nickname = ''
    cardForm.main_character_name = ''
    cardForm.title_tags = []
    cardForm.description_html = ''
    cardForm.accent_color = '#3b82f6'
    cardForm.avatar_shape = 'circle'
    cardForm.font_scale = 14
    cardForm.visible = true
    cardForm.contribution_offsets = { fleets: 0, welfare: 0, srp: 0, recruitment: 0, mentoring: 0 }
    titleTagInput.value = ''
  }

  function openCreateCard() {
    editingCardId.value = null
    resetCardForm()
    cardDialogOpen.value = true
  }

  function openEditCard(card: Api.FuxiHall.Card) {
    editingCardId.value = card.id
    cardForm.page_key = card.page_key
    cardForm.floor_key = card.floor_key
    cardForm.is_current_management = card.is_current_management
    cardForm.nickname = card.nickname
    cardForm.main_character_name = card.main_character_name
    cardForm.title_tags = [...card.title_tags]
    cardForm.description_html = card.description_html
    cardForm.accent_color = card.accent_color
    cardForm.avatar_shape = card.avatar_shape
    cardForm.font_scale = card.font_scale
    cardForm.visible = card.visible
    cardForm.contribution_offsets = {
      fleets: 0,
      welfare: 0,
      srp: 0,
      recruitment: 0,
      mentoring: 0,
      ...card.contribution_offsets
    }
    titleTagInput.value = ''
    cardDialogOpen.value = true
  }

  async function submitCard() {
    if (
      !cardForm.nickname.trim() ||
      !cardForm.main_character_name.trim() ||
      (cardForm.is_current_management && cardForm.title_tags.length === 0)
    ) {
      ElMessage.warning(t('fuxiHall.manage.requiredFields'))
      return
    }

    const payload: Api.FuxiHall.CreateCardParams = {
      page_key: currentPageKey.value,
      nickname: cardForm.nickname.trim(),
      main_character_name: cardForm.main_character_name.trim(),
      title_tags: [...cardForm.title_tags],
      description_html: cardForm.description_html || '',
      accent_color: cardForm.accent_color,
      avatar_shape: cardForm.avatar_shape,
      font_scale: cardForm.font_scale,
      floor_key: cardForm.floor_key,
      is_current_management: cardForm.is_current_management,
      contribution_offsets: { ...cardForm.contribution_offsets },
      visible: cardForm.visible
    }

    savingCard.value = true
    try {
      if (editingCardId.value) {
        const updatePayload: Api.FuxiHall.UpdateCardParams = { ...payload }
        await updateFuxiHallCard(editingCardId.value, updatePayload)
      } else {
        await createFuxiHallCard(payload)
      }
      await loadCards()
      cardFormSnapshot = ''
      cardDialogOpen.value = false
      ElMessage.success(t('fuxiHall.manage.saveSuccess'))
    } catch (error) {
      ElMessage.error(
        error instanceof Error
          ? error.message.startsWith('fuxiHall.')
            ? t(error.message)
            : error.message
          : t('fuxiHall.manage.saveFailed')
      )
    } finally {
      savingCard.value = false
    }
  }

  async function refreshContributions() {
    refreshing.value = true
    try {
      await refreshFuxiHallContributions()
      await loadCards()
      ElMessage.success(t('fuxiHall.contributions.refreshed'))
    } catch (error) {
      ElMessage.error(
        error instanceof Error
          ? error.message.startsWith('fuxiHall.')
            ? t(error.message)
            : error.message
          : t('fuxiHall.errors.refresh')
      )
    } finally {
      refreshing.value = false
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
      ElMessage.error(
        error instanceof Error
          ? error.message.startsWith('fuxiHall.')
            ? t(error.message)
            : error.message
          : t('fuxiHall.manage.saveFailed')
      )
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
      ElMessage.error(
        error instanceof Error
          ? error.message.startsWith('fuxiHall.')
            ? t(error.message)
            : error.message
          : t('fuxiHall.manage.sortFailed')
      )
    }
  }

  function moveFloor(index: number, delta: number) {
    const floors = pageForm.floors
    ;[floors[index], floors[index + delta]] = [floors[index + delta], floors[index]]
  }

  // Snapshot after the dialog has opened: the rich-text editor normalizes its HTML on creation,
  // so an earlier snapshot would make every untouched card look modified.
  let cardFormSnapshot = ''
  const serializeCardForm = () => JSON.stringify([cardForm, titleTagInput.value])
  function snapshotCardForm() {
    cardFormSnapshot = serializeCardForm()
  }
  async function requestCloseCard(done: () => void = () => (cardDialogOpen.value = false)) {
    if (savingCard.value) return
    if (cardFormSnapshot && serializeCardForm() !== cardFormSnapshot) {
      try {
        await ElMessageBox.confirm(t('fuxiHall.manage.discardConfirm'), t('common.tips'), {
          type: 'warning',
          confirmButtonText: t('common.confirm'),
          cancelButtonText: t('common.cancel')
        })
      } catch {
        return
      }
    }
    cardFormSnapshot = ''
    done()
  }

  function addTitleTag() {
    const nextTag = titleTagInput.value.trim()
    if (!nextTag) {
      return
    }
    if (!cardForm.title_tags.includes(nextTag)) {
      cardForm.title_tags.push(nextTag)
    }
    titleTagInput.value = ''
  }

  function removeTitleTag(tag: string) {
    cardForm.title_tags = cardForm.title_tags.filter((item) => item !== tag)
  }

  function getTagStyle(accentColor: string) {
    return {
      color: accentColor,
      borderColor: accentColor,
      backgroundColor: 'transparent'
    }
  }
</script>

<style scoped>
  .hall-offsets {
    grid-column: 1 / -1;
  }
  .hall-offsets > p {
    font-size: 12px;
    color: var(--el-text-color-secondary);
    margin: 8px 0 12px;
  }
  .hall-offsets > div {
    display: flex;
    flex-wrap: wrap;
    gap: 16px;
  }
  .hall-floors {
    margin-bottom: 20px;
  }
  .hall-floor-row {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    margin: 12px 0;
  }
  .hall-floor-row .el-input {
    flex: 1;
    min-width: 120px;
  }
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

  .fuxi-hall-manage__title-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
  }

  .fuxi-hall-manage__title-tags--editor {
    margin-top: 8px;
  }

  .fuxi-hall-manage__dialog-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
    gap: 0 16px;
  }

  .fuxi-hall-manage__tag-editor {
    display: flex;
    gap: 8px;
    align-items: center;
  }

  .fuxi-hall-manage__stats {
    display: flex;
    flex-direction: column;
    gap: 2px;
    color: var(--el-text-color-regular);
    font-size: 13px;
    line-height: 1.4;
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
    width: 100%;
    margin-inline: auto;
    grid-template-columns: repeat(auto-fit, minmax(min(100%, 300px), 1fr));
    gap: 16px;
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

    .fuxi-hall-manage__tag-editor {
      flex-direction: column;
      align-items: stretch;
    }
  }
</style>

<style>
  /* The dialog is teleported to <body>, so this block is unscoped. It keeps the whole dialog
     within one screen and scrolls only the form, leaving the save button always visible. */
  .fuxi-hall-card-dialog {
    display: flex;
    flex-direction: column;
    max-height: 92vh;
    margin-bottom: 0;
  }
  .fuxi-hall-card-dialog .el-dialog__body {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
  }
</style>
