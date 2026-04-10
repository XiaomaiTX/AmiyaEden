<template>
  <div class="canvas-toolbar">
    <div class="canvas-toolbar__group">
      <ElButton type="primary" @click="emit('add-card')">
        {{ t('hallOfFame.manage.addCard') }}
      </ElButton>
      <ElButton @click="openFilePicker">
        {{ t('hallOfFame.manage.setBackground') }}
      </ElButton>
      <input
        ref="fileInputRef"
        class="canvas-toolbar__file-input"
        type="file"
        accept="image/*"
        @change="handleFileChange"
      />
    </div>

    <div class="canvas-toolbar__group canvas-toolbar__group--metrics">
      <label class="canvas-toolbar__field">
        <span>{{ t('hallOfFame.manage.canvasWidth') }}</span>
        <ElInputNumber
          :min="800"
          :max="7680"
          :model-value="canvasWidth"
          @update:model-value="handleWidthChange"
        />
      </label>
      <label class="canvas-toolbar__field">
        <span>{{ t('hallOfFame.manage.canvasHeight') }}</span>
        <ElInputNumber
          :min="600"
          :max="4320"
          :model-value="canvasHeight"
          @update:model-value="handleHeightChange"
        />
      </label>
      <label class="canvas-toolbar__field canvas-toolbar__field--zoom">
        <span>{{ t('hallOfFame.manage.canvasZoom') }}</span>
        <div class="canvas-toolbar__zoom">
          <ElButton @click="adjustZoom(-10)">{{ t('hallOfFame.manage.zoomOut') }}</ElButton>
          <ElSlider
            class="canvas-toolbar__zoom-slider"
            :min="40"
            :max="160"
            :step="10"
            :model-value="zoomPercent"
            @update:model-value="handleZoomChange"
          />
          <ElButton @click="adjustZoom(10)">{{ t('hallOfFame.manage.zoomIn') }}</ElButton>
          <ElButton @click="emit('update:zoomPercent', 100)">
            {{ t('hallOfFame.manage.resetZoom') }}
          </ElButton>
          <strong>{{ zoomPercent }}%</strong>
        </div>
      </label>
    </div>

    <div class="canvas-toolbar__group canvas-toolbar__group--actions">
      <ElButton :loading="saving" @click="emit('preview')">
        {{ t('hallOfFame.manage.preview') }}
      </ElButton>
      <ElButton type="success" :loading="saving" @click="emit('save-layout')">
        {{ t('hallOfFame.manage.saveLayout') }}
      </ElButton>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { ref } from 'vue'

  import { ElButton, ElInputNumber, ElSlider } from 'element-plus'
  import { useI18n } from 'vue-i18n'

  import { clampHallOfFameZoom } from '../../temple/modules/temple-canvas.helpers'

  const props = defineProps<{
    canvasWidth: number
    canvasHeight: number
    zoomPercent: number
    saving: boolean
  }>()

  const emit = defineEmits<{
    'add-card': []
    'upload-background': [file: File]
    'update:canvasWidth': [value: number]
    'update:canvasHeight': [value: number]
    'update:zoomPercent': [value: number]
    'save-layout': []
    preview: []
  }>()

  const { t } = useI18n()
  const fileInputRef = ref<HTMLInputElement | null>(null)

  function openFilePicker() {
    fileInputRef.value?.click()
  }

  function handleFileChange(event: Event) {
    const target = event.target as HTMLInputElement
    const file = target.files?.[0]

    if (file) {
      emit('upload-background', file)
    }

    target.value = ''
  }

  function handleWidthChange(value?: number) {
    if (typeof value === 'number') {
      emit('update:canvasWidth', value)
    }
  }

  function handleHeightChange(value?: number) {
    if (typeof value === 'number') {
      emit('update:canvasHeight', value)
    }
  }

  function handleZoomChange(value: number | number[]) {
    if (typeof value === 'number') {
      emit('update:zoomPercent', clampHallOfFameZoom(value))
    }
  }

  function adjustZoom(delta: number) {
    emit('update:zoomPercent', clampHallOfFameZoom(props.zoomPercent + delta))
  }
</script>

<style scoped>
  .canvas-toolbar {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: 12px 16px;
    border-radius: 24px;
    border: 1px solid rgba(255, 255, 255, 0.08);
    padding: 16px 18px;
    background: rgba(11, 18, 32, 0.88);
    box-shadow: 0 18px 40px rgba(0, 0, 0, 0.18);
  }

  .canvas-toolbar__group {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 12px;
  }

  .canvas-toolbar__group--metrics {
    flex: 1;
  }

  .canvas-toolbar__group--actions {
    margin-left: auto;
  }

  .canvas-toolbar__field {
    display: flex;
    align-items: center;
    gap: 10px;
    color: rgba(255, 255, 255, 0.78);
    font-size: 12px;
    letter-spacing: 0.06em;
    text-transform: uppercase;
  }

  .canvas-toolbar__field--zoom {
    flex: 1 1 360px;
    align-items: flex-start;
  }

  .canvas-toolbar__zoom {
    display: flex;
    flex: 1;
    flex-wrap: wrap;
    align-items: center;
    gap: 10px;
  }

  .canvas-toolbar__zoom-slider {
    flex: 1 1 180px;
    min-width: 140px;
  }

  .canvas-toolbar__file-input {
    display: none;
  }

  @media (max-width: 1024px) {
    .canvas-toolbar {
      align-items: stretch;
    }

    .canvas-toolbar__group--actions {
      margin-left: 0;
    }
  }
</style>
