<template>
  <ArtSearchBar
    ref="searchBarRef"
    v-model="formData"
    :items="formItems"
    :rules="rules"
    @reset="handleReset"
    @search="handleSearch"
  >
  </ArtSearchBar>
</template>

<script setup lang="ts">
  import { useI18n } from 'vue-i18n'

  interface Props {
    modelValue: Record<string, any>
  }
  interface Emits {
    (e: 'update:modelValue', value: Record<string, any>): void
    (e: 'search', params: Record<string, any>): void
    (e: 'reset'): void
  }
  const props = defineProps<Props>()
  const emit = defineEmits<Emits>()
  const { t } = useI18n()

  // 表单数据双向绑定
  const searchBarRef = ref()
  const formData = computed({
    get: () => props.modelValue,
    set: (val) => emit('update:modelValue', val)
  })

  // 校验规则
  const rules = {}

  // 状态选项
  const statusOptions = [
    { label: t('userAdmin.status.active'), value: 1 },
    { label: t('userAdmin.status.disabled'), value: 0 }
  ]

  // 表单配置
  const formItems = computed(() => [
    {
      label: t('userAdmin.search.keyword'),
      key: 'keyword',
      type: 'input',
      props: {
        placeholder: t('userAdmin.search.keywordPlaceholder'),
        clearable: true,
        onKeyup: (e: KeyboardEvent) => {
          if (e.key === 'Enter') handleSearch()
        }
      }
    },
    {
      label: t('common.status'),
      key: 'status',
      type: 'select',
      props: {
        placeholder: t('userAdmin.search.statusPlaceholder'),
        options: statusOptions,
        clearable: true
      }
    }
  ])

  // 事件
  function handleReset() {
    emit('reset')
  }

  async function handleSearch() {
    await searchBarRef.value.validate()
    emit('search', formData.value)
  }
</script>
