<template>
  <div class="fuxi-admin-card" :style="cardStyle">
    <div class="fuxi-admin-card__portrait-wrap">
      <img
        v-if="portraitUrl"
        :src="portraitUrl"
        :alt="admin.name"
        class="fuxi-admin-card__portrait"
      />
      <div v-else class="fuxi-admin-card__portrait fuxi-admin-card__portrait--placeholder">
        {{ admin.name.slice(0, 1) || '?' }}
      </div>
    </div>

    <div class="fuxi-admin-card__body">
      <p class="fuxi-admin-card__name">{{ admin.name }}</p>
      <p v-if="admin.title" class="fuxi-admin-card__title">{{ admin.title }}</p>
      <div class="fuxi-admin-card__contacts">
        <span v-if="admin.contact_qq" class="fuxi-admin-card__contact">
          <span class="fuxi-admin-card__contact-label">QQ</span>
          {{ admin.contact_qq }}
        </span>
        <span v-if="admin.contact_discord" class="fuxi-admin-card__contact">
          <span class="fuxi-admin-card__contact-label">DC</span>
          {{ admin.contact_discord }}
        </span>
      </div>
    </div>

    <div v-if="canEdit" class="fuxi-admin-card__edit-overlay">
      <ElButton size="small" type="primary" text @click.stop="$emit('edit')">
        {{ t('hallOfFame.currentManage.editAdmin') }}
      </ElButton>
      <ElButton size="small" type="danger" text @click.stop="$emit('delete')">
        {{ t('hallOfFame.currentManage.deleteAdmin') }}
      </ElButton>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue'
  import { useI18n } from 'vue-i18n'
  import { buildHallOfFamePortraitUrl } from '@/views/hall-of-fame/portrait.helpers'

  const props = defineProps<{
    admin: Api.FuxiAdmin.Admin
    baseFontSize: number
    canEdit: boolean
  }>()

  defineEmits<{
    edit: []
    delete: []
  }>()

  const { t } = useI18n()

  const portraitUrl = computed(() =>
    props.admin.character_id > 0 ? buildHallOfFamePortraitUrl(props.admin.character_id, 128) : ''
  )

  const cardStyle = computed(() => ({
    '--card-font-size': `${props.baseFontSize}px`
  }))
</script>

<style scoped>
  .fuxi-admin-card {
    position: relative;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
    padding: 20px 16px 16px;
    border-radius: 16px;
    border: 1px solid rgba(255, 255, 255, 0.08);
    background: rgba(12, 22, 40, 0.72);
    text-align: center;
    transition: border-color 0.2s;
  }

  .fuxi-admin-card:hover {
    border-color: rgba(255, 215, 128, 0.28);
  }

  .fuxi-admin-card__portrait-wrap {
    width: 80px;
    height: 80px;
    border-radius: 50%;
    overflow: hidden;
    border: 2px solid rgba(255, 215, 128, 0.3);
    flex-shrink: 0;
  }

  .fuxi-admin-card__portrait {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .fuxi-admin-card__portrait--placeholder {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 100%;
    height: 100%;
    background: rgba(255, 215, 0, 0.12);
    color: #f8d26b;
    font-size: 28px;
    font-weight: 700;
  }

  .fuxi-admin-card__body {
    display: flex;
    flex-direction: column;
    gap: 4px;
    min-width: 0;
    width: 100%;
  }

  .fuxi-admin-card__name {
    margin: 0;
    color: #fff7d6;
    font-size: var(--card-font-size, 14px);
    font-weight: 600;
    line-height: 1.4;
  }

  .fuxi-admin-card__title {
    margin: 0;
    color: rgba(255, 255, 255, 0.72);
    font-size: calc(var(--card-font-size, 14px) - 2px);
    line-height: 1.4;
  }

  .fuxi-admin-card__contacts {
    display: flex;
    flex-direction: column;
    gap: 2px;
    margin-top: 4px;
  }

  .fuxi-admin-card__contact {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 4px;
    color: rgba(255, 255, 255, 0.55);
    font-size: calc(var(--card-font-size, 14px) - 3px);
  }

  .fuxi-admin-card__contact-label {
    padding: 1px 4px;
    border-radius: 4px;
    background: rgba(255, 255, 255, 0.08);
    color: rgba(255, 215, 128, 0.8);
    font-size: 10px;
    font-weight: 600;
    letter-spacing: 0.04em;
  }

  .fuxi-admin-card__edit-overlay {
    display: flex;
    gap: 4px;
    opacity: 0;
    position: absolute;
    bottom: 8px;
    left: 0;
    right: 0;
    justify-content: center;
    transition: opacity 0.2s;
  }

  .fuxi-admin-card:hover .fuxi-admin-card__edit-overlay {
    opacity: 1;
  }
</style>
