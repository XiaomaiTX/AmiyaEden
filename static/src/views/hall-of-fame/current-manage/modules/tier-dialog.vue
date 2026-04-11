<template>
  <ElDialog
    :model-value="modelValue"
    :title="tier ? t('hallOfFame.currentManage.editTier') : t('hallOfFame.currentManage.addTier')"
    width="420px"
    @update:model-value="$emit('update:modelValue', $event)"
    @closed="handleClosed"
  >
    <ElForm :model="form" @submit.prevent="handleSubmit">
      <ElFormItem :label="t('hallOfFame.currentManage.tierName')" required>
        <ElInput
          v-model="form.name"
          :placeholder="t('hallOfFame.currentManage.tierNamePlaceholder')"
        />
      </ElFormItem>
    </ElForm>

    <template #footer>
      <ElButton @click="$emit('update:modelValue', false)">{{ t('common.cancel') }}</ElButton>
      <ElButton type="primary" :loading="saving" @click="handleSubmit">
        {{ t('common.save') }}
      </ElButton>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import { reactive, ref, watch } from 'vue'
  import { useI18n } from 'vue-i18n'
  import { ElMessage } from 'element-plus'
  import { createFuxiAdminTier, updateFuxiAdminTier } from '@/api/fuxi-admins'

  const props = defineProps<{
    modelValue: boolean
    tier: Api.FuxiAdmin.Tier | null
  }>()

  const emit = defineEmits<{
    'update:modelValue': [value: boolean]
    saved: [tier: Api.FuxiAdmin.Tier]
  }>()

  const { t } = useI18n()

  const saving = ref(false)
  const form = reactive({ name: '' })

  watch(
    () => props.modelValue,
    (open) => {
      if (open) {
        form.name = props.tier?.name ?? ''
      }
    }
  )

  function handleClosed() {
    form.name = ''
  }

  async function handleSubmit() {
    if (!form.name.trim()) {
      ElMessage.warning(t('hallOfFame.currentManage.tierNameRequired'))
      return
    }
    saving.value = true
    try {
      const saved = props.tier
        ? await updateFuxiAdminTier(props.tier.id, { name: form.name.trim() })
        : await createFuxiAdminTier({ name: form.name.trim() })
      emit('saved', saved)
      emit('update:modelValue', false)
    } catch (error) {
      ElMessage.error(
        error instanceof Error ? error.message : t('hallOfFame.currentManage.saveFailed')
      )
    } finally {
      saving.value = false
    }
  }
</script>
