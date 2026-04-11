<template>
  <section class="fuxi-tier-section">
    <div class="fuxi-tier-section__header">
      <h3 class="fuxi-tier-section__title">{{ tier.name }}</h3>
      <div v-if="canEdit" class="fuxi-tier-section__header-actions">
        <ElButton size="small" text type="primary" @click="$emit('edit-tier')">
          {{ t('hallOfFame.currentManage.editTier') }}
        </ElButton>
        <ElButton size="small" text type="danger" @click="$emit('delete-tier')">
          {{ t('hallOfFame.currentManage.deleteTier') }}
        </ElButton>
        <ElButton size="small" type="primary" @click="$emit('add-admin')">
          {{ t('hallOfFame.currentManage.addAdmin') }}
        </ElButton>
      </div>
    </div>

    <div v-if="tier.admins.length > 0" class="fuxi-tier-section__cards">
      <AdminCard
        v-for="admin in tier.admins"
        :key="admin.id"
        :admin="admin"
        :base-font-size="baseFontSize"
        :can-edit="canEdit"
        @edit="$emit('edit-admin', admin)"
        @delete="$emit('delete-admin', admin)"
      />
    </div>

    <p v-else-if="canEdit" class="fuxi-tier-section__empty">
      {{ t('hallOfFame.currentManage.tierEmpty') }}
    </p>
  </section>
</template>

<script setup lang="ts">
  import { useI18n } from 'vue-i18n'
  import AdminCard from './admin-card.vue'

  defineProps<{
    tier: Api.FuxiAdmin.TierWithAdmins
    baseFontSize: number
    canEdit: boolean
  }>()

  defineEmits<{
    'edit-tier': []
    'delete-tier': []
    'add-admin': []
    'edit-admin': [admin: Api.FuxiAdmin.Admin]
    'delete-admin': [admin: Api.FuxiAdmin.Admin]
  }>()

  const { t } = useI18n()
</script>

<style scoped>
  .fuxi-tier-section {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .fuxi-tier-section__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding-bottom: 10px;
    border-bottom: 1px solid rgba(255, 215, 128, 0.18);
  }

  .fuxi-tier-section__title {
    margin: 0;
    color: #f8d26b;
    font-size: 15px;
    font-weight: 600;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }

  .fuxi-tier-section__header-actions {
    display: flex;
    gap: 6px;
    align-items: center;
    flex-shrink: 0;
  }

  .fuxi-tier-section__cards {
    display: flex;
    flex-wrap: wrap;
    gap: 16px;
    justify-content: center;
  }

  .fuxi-tier-section__cards > * {
    width: 160px;
  }

  .fuxi-tier-section__empty {
    margin: 0;
    color: rgba(255, 255, 255, 0.38);
    font-size: 13px;
    text-align: center;
    padding: 16px 0;
  }

  @media (max-width: 600px) {
    .fuxi-tier-section__header {
      flex-direction: column;
      align-items: flex-start;
    }

    .fuxi-tier-section__cards > * {
      width: 140px;
    }
  }
</style>
