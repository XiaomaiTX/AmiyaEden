<!-- 舰队搜索栏 -->
<template>
  <ElCard class="art-search-card" shadow="never">
    <div class="flex items-center gap-3 flex-wrap">
      <ElSelect
        :model-value="modelValue.importance"
        :placeholder="$t('fleet.fields.importance')"
        clearable
        style="width: 140px"
        @update:model-value="(v) => emit('update:modelValue', { ...modelValue, importance: v })"
        @change="emit('search', { ...modelValue })"
      >
        <ElOption :label="$t('fleet.importance.strat_op')" value="strat_op" />
        <ElOption :label="$t('fleet.importance.cta')" value="cta" />
        <ElOption :label="$t('fleet.importance.skirmish')" value="skirmish" />
      </ElSelect>
      <ElButton @click="emit('reset')">
        {{ $t('table.searchBar.reset') }}
      </ElButton>
    </div>
  </ElCard>
</template>

<script setup lang="ts">
  defineOptions({ name: 'FleetSearch' })

  interface FleetSearchForm {
    importance: string | undefined
  }

  withDefaults(
    defineProps<{
      modelValue: FleetSearchForm
    }>(),
    {
      modelValue: () => ({ importance: undefined })
    }
  )

  const emit = defineEmits<{
    (e: 'update:modelValue', v: FleetSearchForm): void
    (e: 'search', params: FleetSearchForm): void
    (e: 'reset'): void
  }>()
</script>

<style scoped>
  .art-search-card {
    margin-bottom: 16px;
  }
</style>
