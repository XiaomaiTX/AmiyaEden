<template>
  <aside class="card-editor">
    <div class="card-editor__header">
      <p class="card-editor__eyebrow">{{ t('menus.hallOfFame.title') }}</p>
      <h2>{{ t('hallOfFame.manage.cardEditor') }}</h2>
    </div>

    <div v-if="!card" class="card-editor__empty">
      <p class="card-editor__empty-title">{{ t('hallOfFame.manage.noCardSelected') }}</p>
      <p>{{ t('hallOfFame.manage.selectCard') }}</p>
    </div>

    <div v-else class="card-editor__body">
      <div class="card-editor__avatar-block">
        <img v-if="portraitUrl" :src="portraitUrl" :alt="card.name" class="card-editor__avatar" />
        <div v-else class="card-editor__avatar card-editor__avatar--placeholder">
          {{ card.name.slice(0, 1) || '?' }}
        </div>

        <div class="card-editor__badge-block">
          <img
            v-if="card.badge_image"
            :src="card.badge_image"
            :alt="t('hallOfFame.manage.badgeImage')"
            class="card-editor__badge-image"
          />
          <div v-else class="card-editor__badge-placeholder">
            {{ t('hallOfFame.manage.badgeImageHint') }}
          </div>

          <div class="card-editor__badge-actions">
            <ElButton size="small" @click="openBadgeImagePicker">
              {{
                card.badge_image
                  ? t('hallOfFame.manage.replaceBadgeImage')
                  : t('hallOfFame.manage.uploadBadgeImage')
              }}
            </ElButton>
            <ElButton
              v-if="card.badge_image"
              size="small"
              text
              @click="queueUpdate({ badge_image: '' })"
            >
              {{ t('hallOfFame.manage.removeBadgeImage') }}
            </ElButton>
          </div>

          <input
            ref="badgeInputRef"
            class="card-editor__badge-input"
            type="file"
            accept="image/*"
            @change="handleBadgeImageChange"
          />
        </div>
      </div>

      <div class="card-editor__fields">
        <label>
          <span>{{ t('hallOfFame.manage.name') }}</span>
          <ElInput
            :model-value="card.name"
            @update:model-value="(value) => queueUpdate({ name: String(value) })"
          />
        </label>

        <label>
          <span>{{ t('hallOfFame.manage.titleField') }}</span>
          <ElInput
            :model-value="card.title"
            @update:model-value="(value) => queueUpdate({ title: String(value) })"
          />
        </label>

        <label>
          <span>{{ t('hallOfFame.manage.characterId') }}</span>
          <ElInput
            :model-value="card.character_id > 0 ? String(card.character_id) : ''"
            inputmode="numeric"
            @update:model-value="handleCharacterIdChange"
          />
        </label>

        <label>
          <span>{{ t('hallOfFame.manage.description') }}</span>
          <ElInput
            type="textarea"
            :rows="5"
            :model-value="card.description"
            @update:model-value="(value) => queueUpdate({ description: String(value) })"
          />
        </label>

        <label>
          <span>{{ t('hallOfFame.manage.stylePreset') }}</span>
          <ElSelect :model-value="card.style_preset" @update:model-value="handlePresetChange">
            <ElOption value="gold" :label="t('hallOfFame.manage.gold')" />
            <ElOption value="silver" :label="t('hallOfFame.manage.silver')" />
            <ElOption value="darkred" :label="t('hallOfFame.manage.darkred')" />
            <ElOption value="yellow" :label="t('hallOfFame.manage.yellow')" />
            <ElOption value="bronze" :label="t('hallOfFame.manage.bronze')" />
            <ElOption value="rose" :label="t('hallOfFame.manage.rose')" />
            <ElOption value="jade" :label="t('hallOfFame.manage.jade')" />
            <ElOption value="midnight" :label="t('hallOfFame.manage.midnight')" />
            <ElOption value="custom" :label="t('hallOfFame.manage.custom')" />
          </ElSelect>
        </label>

        <label>
          <span>{{ t('hallOfFame.manage.borderStyle') }}</span>
          <ElSelect
            :model-value="card.border_style || 'none'"
            @update:model-value="handleBorderStyleChange"
          >
            <ElOption value="none" :label="t('hallOfFame.manage.borderNone')" />
            <ElOption value="gilded" :label="t('hallOfFame.manage.borderGilded')" />
            <ElOption value="imperial" :label="t('hallOfFame.manage.borderImperial')" />
            <ElOption value="neon-circuit" :label="t('hallOfFame.manage.borderNeonCircuit')" />
            <ElOption value="void-rift" :label="t('hallOfFame.manage.borderVoidRift')" />
            <ElOption value="amarr" :label="t('hallOfFame.manage.borderAmarr')" />
            <ElOption value="caldari" :label="t('hallOfFame.manage.borderCaldari')" />
            <ElOption value="minmatar" :label="t('hallOfFame.manage.borderMinmatar')" />
            <ElOption value="gallente" :label="t('hallOfFame.manage.borderGallente')" />
          </ElSelect>
        </label>

        <label>
          <span>{{ t('hallOfFame.manage.titleColor') }}</span>
          <ElColorPicker
            :model-value="card.title_color || defaultTitleColor"
            @change="(value) => handleColorChange('title_color', value)"
          />
        </label>

        <template v-if="card.style_preset === 'custom'">
          <label>
            <span>{{ t('hallOfFame.manage.bgColor') }}</span>
            <ElColorPicker
              :model-value="card.custom_bg_color || '#101820'"
              @change="(value) => handleColorChange('custom_bg_color', value)"
            />
          </label>
          <label>
            <span>{{ t('hallOfFame.manage.textColor') }}</span>
            <ElColorPicker
              :model-value="card.custom_text_color || '#f7f7f7'"
              @change="(value) => handleColorChange('custom_text_color', value)"
            />
          </label>
          <label>
            <span>{{ t('hallOfFame.manage.borderColor') }}</span>
            <ElColorPicker
              :model-value="card.custom_border_color || '#4ad295'"
              @change="(value) => handleColorChange('custom_border_color', value)"
            />
          </label>
        </template>

        <label>
          <span>{{ t('hallOfFame.manage.fontSize') }}</span>
          <ElSlider
            :min="12"
            :max="32"
            :show-input="true"
            :model-value="card.font_size || 14"
            @update:model-value="(value) => handleFontSizeChange(value)"
          />
        </label>
      </div>

      <div class="card-editor__danger-zone">
        <ElPopconfirm :title="t('hallOfFame.manage.deleteConfirm')" @confirm="handleDeleteClick">
          <template #reference>
            <ElButton type="danger" plain>
              {{ t('hallOfFame.manage.deleteCard') }}
            </ElButton>
          </template>
        </ElPopconfirm>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
  import { computed, onBeforeUnmount, ref } from 'vue'

  import {
    ElButton,
    ElColorPicker,
    ElInput,
    ElOption,
    ElPopconfirm,
    ElSelect,
    ElSlider
  } from 'element-plus'
  import { useI18n } from 'vue-i18n'

  import { buildHallOfFamePortraitUrl } from '../../portrait.helpers'
  import { buildHeroCardStyle } from '../../temple/modules/temple-canvas.helpers'
  import { mergePendingCardUpdates } from './card-editor.helpers'

  const props = defineProps<{
    card: Api.HallOfFame.Card | null
  }>()

  const emit = defineEmits<{
    update: [id: number, updates: Api.HallOfFame.UpdateCardParams]
    'upload-badge-image': [id: number, file: File]
    delete: [id: number]
  }>()

  const { t } = useI18n()
  const badgeInputRef = ref<HTMLInputElement | null>(null)
  const pendingUpdates = new Map<number, Api.HallOfFame.UpdateCardParams>()
  const pendingTimers = new Map<number, ReturnType<typeof setTimeout>>()
  const portraitUrl = computed(() =>
    props.card ? buildHallOfFamePortraitUrl(props.card.character_id) : ''
  )
  const defaultTitleColor = computed(() =>
    props.card ? buildHeroCardStyle(props.card).titleColor : '#ffd86a'
  )

  function queueUpdate(updates: Api.HallOfFame.UpdateCardParams) {
    if (!props.card) {
      return
    }

    const nextId = props.card.id
    const current = pendingUpdates.get(nextId) ?? {}

    pendingUpdates.set(nextId, mergePendingCardUpdates(current, updates))

    const activeTimer = pendingTimers.get(nextId)
    if (activeTimer) {
      clearTimeout(activeTimer)
    }

    pendingTimers.set(
      nextId,
      setTimeout(() => {
        flushPendingUpdate(nextId)
      }, 250)
    )
  }

  function handlePresetChange(value: Api.HallOfFame.CardStylePreset) {
    queueUpdate({ style_preset: value })
  }

  function handleBorderStyleChange(value: Api.HallOfFame.CardBorderStyle) {
    queueUpdate({ border_style: value })
  }

  function handleColorChange(
    field: 'custom_bg_color' | 'custom_text_color' | 'custom_border_color' | 'title_color',
    value: string | null
  ) {
    queueUpdate({ [field]: value || '' })
  }

  function handleFontSizeChange(value: number | number[]) {
    if (typeof value === 'number') {
      queueUpdate({ font_size: value })
    }
  }

  function handleCharacterIdChange(value: string | number) {
    const normalized = String(value ?? '').trim()
    if (!normalized) {
      queueUpdate({ character_id: 0 })
      return
    }

    const digitsOnly = normalized.replace(/\D+/g, '')
    if (!digitsOnly) {
      return
    }

    queueUpdate({ character_id: Number(digitsOnly) })
  }

  function openBadgeImagePicker() {
    badgeInputRef.value?.click()
  }

  function handleBadgeImageChange(event: Event) {
    if (!props.card) {
      return
    }

    const target = event.target as HTMLInputElement
    const file = target.files?.[0]

    if (file) {
      emit('upload-badge-image', props.card.id, file)
    }

    target.value = ''
  }

  function handleDeleteClick() {
    if (!props.card) {
      return
    }

    clearPendingUpdate(props.card.id)
    emit('delete', props.card.id)
  }

  function flushPendingUpdate(id: number) {
    const updates = pendingUpdates.get(id)
    if (!updates) {
      return
    }

    clearPendingUpdate(id)
    emit('update', id, updates)
  }

  function clearPendingUpdate(id: number) {
    pendingUpdates.delete(id)

    const activeTimer = pendingTimers.get(id)
    if (activeTimer) {
      clearTimeout(activeTimer)
      pendingTimers.delete(id)
    }
  }

  function flushPendingUpdates() {
    for (const id of Array.from(pendingUpdates.keys())) {
      flushPendingUpdate(id)
    }
  }

  defineExpose<{
    flushPendingUpdates: () => void
  }>({
    flushPendingUpdates
  })

  onBeforeUnmount(() => {
    for (const timer of pendingTimers.values()) {
      clearTimeout(timer)
    }
    pendingTimers.clear()
    pendingUpdates.clear()
  })
</script>

<style scoped>
  .card-editor {
    display: flex;
    min-height: 0;
    flex-direction: column;
    border-radius: 28px;
    border: 1px solid rgba(255, 255, 255, 0.08);
    background: linear-gradient(180deg, rgba(10, 16, 28, 0.96), rgba(7, 12, 21, 0.98));
    box-shadow: 0 18px 40px rgba(0, 0, 0, 0.18);
    overflow: hidden;
  }

  .card-editor__header {
    padding: 20px 20px 14px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.06);
  }

  .card-editor__header h2,
  .card-editor__empty-title {
    margin: 0;
    color: #fff7d6;
    font-size: 20px;
  }

  .card-editor__eyebrow {
    margin: 0 0 8px;
    color: rgba(255, 255, 255, 0.58);
    font-size: 12px;
    letter-spacing: 0.22em;
    text-transform: uppercase;
  }

  .card-editor__empty,
  .card-editor__body {
    min-height: 0;
    overflow: auto;
    padding: 20px;
  }

  .card-editor__empty {
    display: flex;
    flex-direction: column;
    justify-content: center;
    gap: 10px;
    color: rgba(255, 255, 255, 0.74);
    text-align: center;
  }

  .card-editor__avatar-block {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
    margin-bottom: 18px;
  }

  .card-editor__avatar {
    width: 112px;
    height: 112px;
    border-radius: 999px;
    object-fit: cover;
    border: 3px solid rgba(255, 255, 255, 0.16);
    background: rgba(255, 255, 255, 0.08);
  }

  .card-editor__avatar--placeholder {
    display: flex;
    align-items: center;
    justify-content: center;
    color: #ffd86a;
    font-size: 34px;
    font-weight: 700;
    text-transform: uppercase;
  }

  .card-editor__badge-block {
    width: 100%;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 8px;
  }

  .card-editor__badge-image,
  .card-editor__badge-placeholder {
    width: 100%;
    max-width: 160px;
    min-height: 68px;
    border-radius: 16px;
  }

  .card-editor__badge-image {
    object-fit: cover;
    border: 1px solid rgba(255, 255, 255, 0.14);
    background: rgba(255, 255, 255, 0.06);
  }

  .card-editor__badge-placeholder {
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 12px;
    border: 1px dashed rgba(255, 255, 255, 0.18);
    background: rgba(255, 255, 255, 0.04);
    color: rgba(255, 255, 255, 0.56);
    font-size: 12px;
    line-height: 1.5;
    text-align: center;
  }

  .card-editor__badge-actions {
    display: flex;
    flex-wrap: wrap;
    justify-content: center;
    gap: 8px;
  }

  .card-editor__badge-input {
    display: none;
  }

  .card-editor__fields {
    display: flex;
    flex-direction: column;
    gap: 14px;
  }

  .card-editor__fields label {
    display: flex;
    flex-direction: column;
    gap: 8px;
    color: rgba(255, 255, 255, 0.78);
    font-size: 12px;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }

  .card-editor__danger-zone {
    margin-top: 22px;
    display: flex;
    justify-content: flex-end;
  }
</style>
