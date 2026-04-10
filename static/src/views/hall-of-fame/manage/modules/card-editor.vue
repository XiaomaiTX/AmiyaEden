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
        <img v-if="card.avatar" :src="card.avatar" :alt="card.name" class="card-editor__avatar" />
        <div v-else class="card-editor__avatar card-editor__avatar--placeholder">
          {{ card.name.slice(0, 1) || '?' }}
        </div>
        <ElButton @click="openAvatarPicker">{{ t('hallOfFame.manage.changeAvatar') }}</ElButton>
        <input
          ref="avatarInputRef"
          class="card-editor__file-input"
          type="file"
          accept="image/*"
          @change="handleAvatarChange"
        />
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
            <ElOption value="bronze" :label="t('hallOfFame.manage.bronze')" />
            <ElOption value="custom" :label="t('hallOfFame.manage.custom')" />
          </ElSelect>
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
            :max="24"
            :model-value="card.font_size || 14"
            @change="(value) => queueUpdate({ font_size: Number(value) })"
          />
        </label>

        <div class="card-editor__row">
          <label>
            <span>{{ t('hallOfFame.manage.zIndex') }}</span>
            <ElInputNumber
              :min="0"
              :max="999"
              :model-value="card.z_index"
              @update:model-value="handleLayerChange"
            />
          </label>
          <label>
            <span>{{ t('hallOfFame.manage.visible') }}</span>
            <ElSwitch
              :model-value="card.visible"
              @update:model-value="(value) => queueUpdate({ visible: Boolean(value) })"
            />
          </label>
        </div>
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
  import { onBeforeUnmount, ref } from 'vue'

  import {
    ElButton,
    ElColorPicker,
    ElInput,
    ElInputNumber,
    ElOption,
    ElPopconfirm,
    ElSelect,
    ElSlider,
    ElSwitch
  } from 'element-plus'
  import { useI18n } from 'vue-i18n'

  import { mergePendingCardUpdates } from './card-editor.helpers'

  const props = defineProps<{
    card: Api.HallOfFame.Card | null
  }>()

  const emit = defineEmits<{
    update: [id: number, updates: Api.HallOfFame.UpdateCardParams]
    'update-z-index': [id: number, value: number]
    delete: [id: number]
    'upload-avatar': [id: number, file: File]
  }>()

  const { t } = useI18n()
  const avatarInputRef = ref<HTMLInputElement | null>(null)
  const pendingUpdates = new Map<number, Api.HallOfFame.UpdateCardParams>()
  const pendingTimers = new Map<number, ReturnType<typeof setTimeout>>()

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

  function handleColorChange(
    field: 'custom_bg_color' | 'custom_text_color' | 'custom_border_color',
    value: string | null
  ) {
    queueUpdate({ [field]: value || '' })
  }

  function handleLayerChange(value?: number) {
    if (props.card && typeof value === 'number') {
      emit('update-z-index', props.card.id, value)
    }
  }

  function openAvatarPicker() {
    avatarInputRef.value?.click()
  }

  function handleAvatarChange(event: Event) {
    if (!props.card) {
      return
    }

    const target = event.target as HTMLInputElement
    const file = target.files?.[0]

    if (file) {
      emit('upload-avatar', props.card.id, file)
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

  .card-editor__row {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 12px;
  }

  .card-editor__danger-zone {
    margin-top: 22px;
    display: flex;
    justify-content: flex-end;
  }

  .card-editor__file-input {
    display: none;
  }

  @media (max-width: 960px) {
    .card-editor__row {
      grid-template-columns: 1fr;
    }
  }
</style>
