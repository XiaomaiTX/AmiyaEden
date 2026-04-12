<template>
  <ElDialog
    :model-value="modelValue"
    :title="
      admin ? t('hallOfFame.currentManage.editAdmin') : t('hallOfFame.currentManage.addAdmin')
    "
    width="480px"
    @update:model-value="$emit('update:modelValue', $event)"
    @closed="handleClosed"
  >
    <ElForm :model="form" label-width="90px">
      <ElFormItem :label="t('hallOfFame.currentManage.tierLabel')" required>
        <ElSelect
          v-model="form.tierId"
          :placeholder="t('hallOfFame.currentManage.tierPlaceholder')"
        >
          <ElOption v-for="tier in tiers" :key="tier.id" :label="tier.name" :value="tier.id" />
        </ElSelect>
      </ElFormItem>

      <ElFormItem :label="t('hallOfFame.currentManage.nameLabel')" required>
        <ElInput v-model="form.nickname" />
      </ElFormItem>

      <ElFormItem :label="t('hallOfFame.currentManage.titleLabel')">
        <ElInput v-model="form.characterName" />
      </ElFormItem>

      <ElFormItem :label="t('hallOfFame.currentManage.descriptionLabel')">
        <ElInput
          v-model="form.description"
          type="textarea"
          :rows="3"
          :placeholder="t('hallOfFame.currentManage.descriptionPlaceholder')"
        />
      </ElFormItem>

      <ElFormItem :label="t('hallOfFame.currentManage.characterIdLabel')">
        <ElInput
          v-model.number="form.characterId"
          type="number"
          :placeholder="t('hallOfFame.currentManage.characterIdPlaceholder')"
        />
        <div v-if="portraitPreviewUrl" class="admin-card-dialog__portrait-preview">
          <img :src="portraitPreviewUrl" alt="portrait preview" />
        </div>
      </ElFormItem>

      <ElFormItem :label="t('hallOfFame.currentManage.contactQqLabel')">
        <ElInput v-model="form.contactQq" />
      </ElFormItem>

      <ElFormItem :label="t('hallOfFame.currentManage.contactDiscordLabel')">
        <ElInput v-model="form.contactDiscord" />
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
  import { computed, reactive, ref, watch } from 'vue'
  import { useI18n } from 'vue-i18n'
  import { ElMessage } from 'element-plus'
  import { createFuxiAdmin, updateFuxiAdmin } from '@/api/fuxi-admins'
  import { buildHallOfFamePortraitUrl } from '@/views/hall-of-fame/portrait.helpers'

  const props = defineProps<{
    modelValue: boolean
    admin: Api.FuxiAdmin.Admin | null
    defaultTierId: number | null
    tiers: Api.FuxiAdmin.Tier[]
  }>()

  const emit = defineEmits<{
    'update:modelValue': [value: boolean]
    saved: [admin: Api.FuxiAdmin.Admin]
  }>()

  const { t } = useI18n()

  const saving = ref(false)
  const form = reactive({
    tierId: 0,
    nickname: '',
    characterName: '',
    description: '',
    characterId: 0,
    contactQq: '',
    contactDiscord: ''
  })

  const portraitPreviewUrl = computed(() =>
    form.characterId > 0 ? buildHallOfFamePortraitUrl(form.characterId, 64) : ''
  )

  watch(
    () => props.modelValue,
    (open) => {
      if (open) {
        form.tierId = props.admin?.tier_id ?? props.defaultTierId ?? props.tiers[0]?.id ?? 0
        form.nickname = props.admin?.nickname ?? ''
        form.characterName = props.admin?.character_name ?? ''
        form.description = props.admin?.description ?? ''
        form.characterId = props.admin?.character_id ?? 0
        form.contactQq = props.admin?.contact_qq ?? ''
        form.contactDiscord = props.admin?.contact_discord ?? ''
      }
    }
  )

  function handleClosed() {
    Object.assign(form, {
      tierId: 0,
      nickname: '',
      characterName: '',
      description: '',
      characterId: 0,
      contactQq: '',
      contactDiscord: ''
    })
  }

  async function handleSubmit() {
    if (!form.nickname.trim()) {
      ElMessage.warning(t('hallOfFame.currentManage.nameRequired'))
      return
    }
    if (!form.tierId) {
      ElMessage.warning(t('hallOfFame.currentManage.tierRequired'))
      return
    }
    saving.value = true
    try {
      const payload = {
        tier_id: form.tierId,
        nickname: form.nickname.trim(),
        character_name: form.characterName.trim(),
        description: form.description.trim(),
        character_id: form.characterId || 0,
        contact_qq: form.contactQq.trim(),
        contact_discord: form.contactDiscord.trim()
      }
      const saved = props.admin
        ? await updateFuxiAdmin(props.admin.id, payload)
        : await createFuxiAdmin(payload)
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

<style scoped>
  .admin-card-dialog__portrait-preview {
    margin-top: 8px;
  }

  .admin-card-dialog__portrait-preview img {
    width: 64px;
    height: 64px;
    border-radius: 50%;
    border: 2px solid rgba(255, 215, 128, 0.3);
    object-fit: cover;
  }
</style>
