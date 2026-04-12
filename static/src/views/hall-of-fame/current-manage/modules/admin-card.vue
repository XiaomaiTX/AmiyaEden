<template>
  <div class="fuxi-admin-card" :style="cardStyle">
    <div class="fuxi-admin-card__portrait-wrap">
      <img
        v-if="portraitUrl"
        :src="portraitUrl"
        :alt="admin.nickname"
        class="fuxi-admin-card__portrait"
      />
      <div v-else class="fuxi-admin-card__portrait fuxi-admin-card__portrait--placeholder">
        {{ admin.nickname.slice(0, 1) || '?' }}
      </div>
    </div>

    <div class="fuxi-admin-card__body">
      <p class="fuxi-admin-card__name">{{ admin.nickname }}</p>
      <p v-if="admin.character_name" class="fuxi-admin-card__title">
        {{ admin.character_name }}
      </p>
      <p v-if="admin.description" class="fuxi-admin-card__description">{{ admin.description }}</p>
      <div class="fuxi-admin-card__contacts">
        <span v-if="admin.contact_qq" class="fuxi-admin-card__contact">
          <span class="fuxi-admin-card__contact-label">QQ</span>
          <span class="fuxi-admin-card__contact-value">{{ admin.contact_qq }}</span>
          <ArtCopyButton :text="admin.contact_qq" :aria-label="`${t('common.copy')} QQ`" />
        </span>
        <span v-if="admin.contact_discord" class="fuxi-admin-card__contact">
          <span class="fuxi-admin-card__contact-label">DC</span>
          <span class="fuxi-admin-card__contact-value">{{ admin.contact_discord }}</span>
          <ArtCopyButton
            :text="admin.contact_discord"
            :aria-label="`${t('common.copy')} Discord`"
          />
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
  import ArtCopyButton from '@/components/core/forms/art-copy-button/index.vue'
  import { buildHallOfFamePortraitUrl } from '@/views/hall-of-fame/portrait.helpers'

  const props = defineProps<{
    admin: Api.FuxiAdmin.Admin
    styleConfig: Api.FuxiAdmin.Config
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

  const cardStyle = computed(() => {
    const nameFontSize = props.styleConfig.base_font_size
    return {
      '--card-width': `${props.styleConfig.card_width}px`,
      '--card-name-font-size': `${nameFontSize}px`,
      '--card-title-font-size': `${Math.max(nameFontSize - 2, 8)}px`,
      '--card-description-font-size': `${Math.max(nameFontSize - 3, 8)}px`,
      '--card-contact-font-size': `${Math.max(nameFontSize - 3, 8)}px`,
      '--card-background-color': props.styleConfig.card_background_color,
      '--card-border-color': props.styleConfig.card_border_color,
      '--card-name-color': props.styleConfig.name_text_color,
      '--card-body-color': props.styleConfig.body_text_color
    }
  })
</script>

<style scoped>
  .fuxi-admin-card {
    flex: 0 0 var(--card-width, 240px);
    width: var(--card-width, 240px);
    max-width: 100%;
    position: relative;
    display: flex;
    flex-direction: column;
    align-items: stretch;
    gap: 12px;
    padding: 20px 18px 18px;
    border-radius: 16px;
    border: 1px solid var(--card-border-color, #d9a441);
    background:
      linear-gradient(180deg, rgba(255, 255, 255, 0.05), rgba(255, 255, 255, 0) 58%),
      var(--card-background-color, #1b324c);
    text-align: left;
    box-shadow: 0 20px 45px rgba(4, 10, 18, 0.2);
    transition:
      border-color 0.2s,
      transform 0.2s;
  }

  .fuxi-admin-card:hover {
    transform: translateY(-2px);
    box-shadow:
      0 24px 48px rgba(4, 10, 18, 0.24),
      0 0 0 1px var(--card-border-color, #d9a441);
  }

  .fuxi-admin-card__portrait-wrap {
    align-self: center;
    width: 80px;
    height: 80px;
    border-radius: 50%;
    overflow: hidden;
    border: 2px solid var(--card-border-color, #d9a441);
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
    background: rgba(255, 255, 255, 0.08);
    color: var(--card-border-color, #d9a441);
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
    color: var(--card-name-color, #fff7d6);
    font-size: var(--card-name-font-size, 14px);
    font-weight: 700;
    line-height: 1.3;
    overflow-wrap: anywhere;
  }

  .fuxi-admin-card__title {
    margin: 0;
    color: var(--card-body-color, #d7dfef);
    font-size: var(--card-title-font-size, 12px);
    line-height: 1.4;
    overflow-wrap: anywhere;
  }

  .fuxi-admin-card__description {
    margin: 2px 0 0;
    color: var(--card-body-color, #d7dfef);
    font-size: var(--card-description-font-size, 11px);
    line-height: 1.6;
    white-space: pre-wrap;
    overflow-wrap: anywhere;
  }

  .fuxi-admin-card__contacts {
    display: flex;
    flex-direction: column;
    gap: 6px;
    margin-top: 8px;
  }

  .fuxi-admin-card__contact {
    display: flex;
    align-items: center;
    justify-content: flex-start;
    gap: 4px;
    color: var(--card-body-color, #d7dfef);
    font-size: var(--card-contact-font-size, 10px);
    line-height: 1.4;
    overflow-wrap: anywhere;
  }

  .fuxi-admin-card__contact-value {
    min-width: 0;
    flex: 1;
    overflow-wrap: anywhere;
  }

  .fuxi-admin-card__contact-label {
    padding: 1px 4px;
    border-radius: 4px;
    border: 1px solid var(--card-border-color, #d9a441);
    background: rgba(255, 255, 255, 0.04);
    color: var(--card-border-color, #d9a441);
    font-size: 9px;
    font-weight: 600;
    letter-spacing: 0.04em;
    flex-shrink: 0;
  }

  .fuxi-admin-card__edit-overlay {
    display: flex;
    gap: 4px;
    opacity: 0;
    position: absolute;
    top: 10px;
    right: 10px;
    justify-content: flex-end;
    transition: opacity 0.2s;
  }

  .fuxi-admin-card:hover .fuxi-admin-card__edit-overlay {
    opacity: 1;
  }

  @media (max-width: 600px) {
    .fuxi-admin-card {
      flex-basis: 100%;
      width: 100%;
    }
  }
</style>
