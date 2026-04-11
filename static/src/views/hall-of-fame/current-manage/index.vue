<template>
  <div class="fuxi-directory art-full-height" v-loading="loading">
    <div class="fuxi-directory__inner">
      <!-- Header -->
      <header class="fuxi-directory__header">
        <div class="fuxi-directory__header-text">
          <p class="fuxi-directory__eyebrow">{{ t('menus.hallOfFame.currentManage') }}</p>
          <h1 class="fuxi-directory__title">{{ t('hallOfFame.currentManage.title') }}</h1>
        </div>

        <!-- Admin toolbar -->
        <div v-if="canEdit" class="fuxi-directory__toolbar">
          <ElInputNumber
            v-model="localFontSize"
            :min="8"
            :max="32"
            :label="t('hallOfFame.currentManage.baseFontSize')"
            size="small"
            @change="handleFontSizeChange"
          />
          <ElButton type="primary" @click="openAddTier">
            {{ t('hallOfFame.currentManage.addTier') }}
          </ElButton>
        </div>
      </header>

      <!-- Content -->
      <div v-if="directory && directory.tiers.length > 0" class="fuxi-directory__content">
        <TierSection
          v-for="tier in directory.tiers"
          :key="tier.id"
          :tier="tier"
          :base-font-size="localFontSize"
          :can-edit="canEdit"
          @edit-tier="openEditTier(tier)"
          @delete-tier="handleDeleteTier(tier)"
          @add-admin="openAddAdmin(tier)"
          @edit-admin="openEditAdmin"
          @delete-admin="handleDeleteAdmin"
        />
      </div>

      <div v-else-if="!loading" class="fuxi-directory__empty">
        <h2>{{ t('hallOfFame.currentManage.emptyTitle') }}</h2>
        <p>{{ t('hallOfFame.currentManage.emptySubtitle') }}</p>
      </div>
    </div>

    <!-- Dialogs -->
    <TierDialog
      v-model="tierDialogOpen"
      :tier="editingTier"
      @saved="handleTierSaved"
    />

    <AdminCardDialog
      v-model="adminDialogOpen"
      :admin="editingAdmin"
      :tiers="allTiers"
      @saved="handleAdminSaved"
    />
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, ref } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import { useI18n } from 'vue-i18n'

  import {
    fetchFuxiAdminDirectory,
    updateFuxiAdminConfig,
    deleteFuxiAdminTier,
    deleteFuxiAdmin
  } from '@/api/fuxi-admins'
  import { useUserStore } from '@/store/modules/user'

  import TierSection from './modules/tier-section.vue'
  import TierDialog from './modules/tier-dialog.vue'
  import AdminCardDialog from './modules/admin-card-dialog.vue'

  const { t } = useI18n()
  const userStore = useUserStore()

  const loading = ref(false)
  const directory = ref<Api.FuxiAdmin.DirectoryResponse | null>(null)
  const localFontSize = ref(14)

  const tierDialogOpen = ref(false)
  const editingTier = ref<Api.FuxiAdmin.Tier | null>(null)
  const adminDialogOpen = ref(false)
  const editingAdmin = ref<Api.FuxiAdmin.Admin | null>(null)
  const addingAdminToTierId = ref<number | null>(null)

  const canEdit = computed(() => {
    const roles = userStore.getUserInfo?.roles ?? []
    return roles.some((r) => ['super_admin', 'admin'].includes(r))
  })

  const allTiers = computed(() => directory.value?.tiers ?? [])

  onMounted(() => {
    void loadDirectory()
  })

  async function loadDirectory() {
    loading.value = true
    try {
      directory.value = await fetchFuxiAdminDirectory()
      localFontSize.value = directory.value.config.base_font_size
    } catch (error) {
      ElMessage.error(error instanceof Error ? error.message : t('hallOfFame.currentManage.saveFailed'))
    } finally {
      loading.value = false
    }
  }

  async function handleFontSizeChange(value: number) {
    try {
      await updateFuxiAdminConfig({ base_font_size: value })
    } catch (error) {
      ElMessage.error(error instanceof Error ? error.message : t('hallOfFame.currentManage.saveFailed'))
    }
  }

  // ─── Tier actions ───

  function openAddTier() {
    editingTier.value = null
    tierDialogOpen.value = true
  }

  function openEditTier(tier: Api.FuxiAdmin.TierWithAdmins) {
    editingTier.value = tier
    tierDialogOpen.value = true
  }

  function handleTierSaved(savedTier: Api.FuxiAdmin.Tier) {
    if (!directory.value) return

    const idx = directory.value.tiers.findIndex((t) => t.id === savedTier.id)
    if (idx >= 0) {
      // edit: update name only, keep admins
      directory.value.tiers[idx] = {
        ...directory.value.tiers[idx],
        name: savedTier.name,
        sort_order: savedTier.sort_order
      }
    } else {
      // create: new tier with empty admins
      directory.value.tiers = [
        ...directory.value.tiers,
        { ...savedTier, admins: [] }
      ]
    }
  }

  async function handleDeleteTier(tier: Api.FuxiAdmin.TierWithAdmins) {
    try {
      await ElMessageBox.confirm(
        t('hallOfFame.currentManage.deleteTierConfirm'),
        t('hallOfFame.currentManage.deleteTier'),
        { type: 'warning', confirmButtonText: t('common.confirm'), cancelButtonText: t('common.cancel') }
      )
    } catch {
      return
    }
    try {
      await deleteFuxiAdminTier(tier.id)
      if (directory.value) {
        directory.value.tiers = directory.value.tiers.filter((t) => t.id !== tier.id)
      }
      ElMessage.success(t('hallOfFame.currentManage.deleteSuccess'))
    } catch (error) {
      ElMessage.error(error instanceof Error ? error.message : t('hallOfFame.currentManage.saveFailed'))
    }
  }

  // ─── Admin actions ───

  function openAddAdmin(tier: Api.FuxiAdmin.TierWithAdmins) {
    editingAdmin.value = null
    addingAdminToTierId.value = tier.id
    adminDialogOpen.value = true
  }

  function openEditAdmin(admin: Api.FuxiAdmin.Admin) {
    editingAdmin.value = admin
    addingAdminToTierId.value = null
    adminDialogOpen.value = true
  }

  function handleAdminSaved(savedAdmin: Api.FuxiAdmin.Admin) {
    if (!directory.value) return

    // find which tier this admin belongs to
    for (const tier of directory.value.tiers) {
      const idx = tier.admins.findIndex((a) => a.id === savedAdmin.id)
      if (idx >= 0) {
        // edit in place (may have changed tier)
        if (savedAdmin.tier_id !== tier.id) {
          // moved to a different tier: remove from old tier
          tier.admins = tier.admins.filter((a) => a.id !== savedAdmin.id)
          // add to new tier
          const newTier = directory.value.tiers.find((t) => t.id === savedAdmin.tier_id)
          if (newTier) {
            newTier.admins = [...newTier.admins, savedAdmin]
          }
        } else {
          tier.admins[idx] = savedAdmin
        }
        return
      }
    }

    // new admin: add to its tier
    const targetTier = directory.value.tiers.find((t) => t.id === savedAdmin.tier_id)
    if (targetTier) {
      targetTier.admins = [...targetTier.admins, savedAdmin]
    }
  }

  async function handleDeleteAdmin(admin: Api.FuxiAdmin.Admin) {
    try {
      await ElMessageBox.confirm(
        t('hallOfFame.currentManage.deleteAdminConfirm'),
        t('hallOfFame.currentManage.deleteAdmin'),
        { type: 'warning', confirmButtonText: t('common.confirm'), cancelButtonText: t('common.cancel') }
      )
    } catch {
      return
    }
    try {
      await deleteFuxiAdmin(admin.id)
      if (directory.value) {
        for (const tier of directory.value.tiers) {
          tier.admins = tier.admins.filter((a) => a.id !== admin.id)
        }
      }
      ElMessage.success(t('hallOfFame.currentManage.deleteSuccess'))
    } catch (error) {
      ElMessage.error(error instanceof Error ? error.message : t('hallOfFame.currentManage.saveFailed'))
    }
  }
</script>

<style scoped>
  .fuxi-directory {
    display: flex;
    min-height: 100%;
    min-width: 0;
    flex-direction: column;
    padding: 24px;
    background:
      radial-gradient(circle at top left, rgba(246, 206, 112, 0.16), transparent 24%),
      radial-gradient(circle at top right, rgba(104, 164, 255, 0.14), transparent 18%),
      linear-gradient(180deg, #07111f 0%, #0f1728 58%, #131c2c 100%);
    overflow-y: auto;
  }

  .fuxi-directory__inner {
    display: flex;
    flex-direction: column;
    gap: 32px;
    max-width: 1200px;
    width: 100%;
    margin: 0 auto;
  }

  .fuxi-directory__header {
    display: flex;
    align-items: flex-end;
    justify-content: space-between;
    gap: 16px;
    flex-wrap: wrap;
  }

  .fuxi-directory__eyebrow {
    margin: 0 0 6px;
    color: #f8d26b;
    font-size: 12px;
    font-weight: 600;
    letter-spacing: 0.3em;
    text-transform: uppercase;
  }

  .fuxi-directory__title {
    margin: 0;
    color: #fff7d6;
    font-size: clamp(22px, 3vw, 32px);
    letter-spacing: 0.04em;
  }

  .fuxi-directory__toolbar {
    display: flex;
    align-items: center;
    gap: 10px;
    flex-shrink: 0;
  }

  .fuxi-directory__content {
    display: flex;
    flex-direction: column;
    gap: 40px;
  }

  .fuxi-directory__empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 12px;
    min-height: 40vh;
    text-align: center;
    color: rgba(255, 255, 255, 0.82);
  }

  .fuxi-directory__empty h2 {
    margin: 0;
    color: #fff7d6;
    font-size: 24px;
  }

  .fuxi-directory__empty p {
    margin: 0;
    max-width: 400px;
    color: rgba(255, 255, 255, 0.55);
    line-height: 1.6;
  }

  @media (max-width: 768px) {
    .fuxi-directory {
      padding: 16px;
    }

    .fuxi-directory__header {
      flex-direction: column;
      align-items: flex-start;
    }
  }
</style>
