<!-- 舰队搜索栏 -->
<template>
  <ElCard class="art-search-card" shadow="never">
    <div class="flex items-center gap-3 flex-wrap">
      <ElSelect
        :model-value="modelValue.importance"
        :placeholder="$t('fleet.fields.importance')"
        clearable
        style="width: 140px"
        @update:model-value="handleImportanceChange"
      >
        <ElOption label="Strat Op" value="strat_op" />
        <ElOption label="CTA" value="cta" />
        <ElOption label="Other" value="other" />
      </ElSelect>
      <ElButton @click="handleReset">
        {{ $t('table.searchBar.reset') }}
      </ElButton>
    </div>
  </ElCard>
</template>

<script setup lang="ts">
  import { buildFleetSearchForm, type FleetSearchForm } from './fleet-search-form'

  defineOptions({ name: 'FleetSearch' })

  interface Props {
    modelValue: FleetSearchForm
  }

  interface Emits {
    (e: 'update:modelValue', v: FleetSearchForm): void
    (e: 'search', params: FleetSearchForm): void
    (e: 'reset'): void
  }

  const props = withDefaults(defineProps<Props>(), {
    modelValue: () => ({ importance: undefined })
  })
  const emit = defineEmits<Emits>()

  function handleImportanceChange(importance: FleetSearchForm['importance']) {
    const next = buildFleetSearchForm(props.modelValue, importance)
    emit('update:modelValue', next)
    emit('search', next)
  }

  function handleReset() {
    const next: FleetSearchForm = { importance: undefined }
    emit('update:modelValue', next)
    emit('reset')
  }
</script>

<style scoped>
  .art-search-card {
    margin-bottom: 16px;
  }
</style>
