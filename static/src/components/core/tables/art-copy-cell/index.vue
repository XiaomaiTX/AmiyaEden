<!-- 紧凑型可复制文本单元格 -->
<template>
  <span class="art-copy-cell inline-flex min-w-0 items-center gap-0.5 align-middle">
    <span class="art-copy-cell__text min-w-0 truncate">
      {{ displayText }}
    </span>
    <button
      v-if="canCopy"
      type="button"
      class="art-copy-cell__trigger"
      :aria-label="t('common.copy')"
      :title="t('common.copy')"
      @click="handleCopy"
    >
      <ElIcon :size="14">
        <CopyDocument />
      </ElIcon>
    </button>
  </span>
</template>

<script setup lang="ts">
  import { ElIcon, ElMessage } from 'element-plus'
  import { CopyDocument } from '@element-plus/icons-vue'
  import { useI18n } from 'vue-i18n'
  import { copyToClipboard } from '@/utils/clipboard'

  defineOptions({ name: 'ArtCopyCell' })

  const props = withDefaults(
    defineProps<{
      text?: string | number | null
      emptyText?: string
    }>(),
    {
      emptyText: '-'
    }
  )

  const { t } = useI18n()

  const hasText = computed(
    () => props.text !== undefined && props.text !== null && String(props.text) !== ''
  )

  const displayText = computed(() => (hasText.value ? String(props.text) : props.emptyText))
  const canCopy = computed(() => hasText.value)

  const handleCopy = async () => {
    if (!hasText.value) return
    const ok = await copyToClipboard(String(props.text))
    if (ok) {
      ElMessage.success(t('common.copied'))
      return
    }
    ElMessage.error(t('common.copyFailed'))
  }
</script>

<style scoped lang="scss">
  .art-copy-cell__trigger {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 18px;
    height: 18px;
    flex: none;
    padding: 0;
    border: 0;
    border-radius: 4px;
    background: transparent;
    color: var(--el-text-color-placeholder);
    cursor: pointer;
    transition:
      color 0.15s ease,
      background-color 0.15s ease;
  }

  .art-copy-cell__trigger:hover,
  .art-copy-cell__trigger:focus-visible {
    color: var(--el-color-primary);
    background: var(--el-fill-color-light);
    outline: none;
  }
</style>
