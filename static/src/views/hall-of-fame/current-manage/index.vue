<template>
  <div class="fuxi-directory art-full-height" :style="directoryStyle" v-loading="loading">
    <div class="fuxi-directory__inner">
      <div v-if="canEdit" class="fuxi-directory__admin-bar">
        <section class="fuxi-directory__settings-group">
          <p class="fuxi-directory__settings-title">
            {{ t('hallOfFame.currentManage.layoutSettings') }}
          </p>

          <label class="fuxi-directory__control">
            <span>{{ t('hallOfFame.currentManage.baseFontSize') }}</span>
            <ElInputNumber
              v-model="localFontSize"
              :min="8"
              :max="32"
              size="small"
              @change="handleBaseFontSizeChange"
            />
          </label>

          <label class="fuxi-directory__control">
            <span>{{ t('hallOfFame.currentManage.cardWidth') }}</span>
            <ElInputNumber
              v-model="cardWidth"
              :min="160"
              :max="420"
              size="small"
              @change="handleCardWidthChange"
            />
          </label>
        </section>

        <section class="fuxi-directory__settings-group">
          <p class="fuxi-directory__settings-title">
            {{ t('hallOfFame.currentManage.surfaceColors') }}
          </p>

          <label class="fuxi-directory__control">
            <span>{{ t('hallOfFame.currentManage.pageBackgroundColor') }}</span>
            <ElColorPicker
              v-model="pageBackgroundColor"
              @change="(value) => handleColorConfigChange('page_background_color', value)"
            />
          </label>

          <label class="fuxi-directory__control">
            <span>{{ t('hallOfFame.currentManage.cardBackgroundColor') }}</span>
            <ElColorPicker
              v-model="cardBackgroundColor"
              @change="(value) => handleColorConfigChange('card_background_color', value)"
            />
          </label>

          <label class="fuxi-directory__control">
            <span>{{ t('hallOfFame.currentManage.cardBorderColor') }}</span>
            <ElColorPicker
              v-model="cardBorderColor"
              @change="(value) => handleColorConfigChange('card_border_color', value)"
            />
          </label>
        </section>

        <section class="fuxi-directory__settings-group">
          <p class="fuxi-directory__settings-title">
            {{ t('hallOfFame.currentManage.textColors') }}
          </p>

          <label class="fuxi-directory__control">
            <span>{{ t('hallOfFame.currentManage.tierTitleColor') }}</span>
            <ElColorPicker
              v-model="tierTitleColor"
              @change="(value) => handleColorConfigChange('tier_title_color', value)"
            />
          </label>

          <label class="fuxi-directory__control">
            <span>{{ t('hallOfFame.currentManage.nameTextColor') }}</span>
            <ElColorPicker
              v-model="nameTextColor"
              @change="(value) => handleColorConfigChange('name_text_color', value)"
            />
          </label>

          <label class="fuxi-directory__control">
            <span>{{ t('hallOfFame.currentManage.bodyTextColor') }}</span>
            <ElColorPicker
              v-model="bodyTextColor"
              @change="(value) => handleColorConfigChange('body_text_color', value)"
            />
          </label>
        </section>

        <div class="fuxi-directory__admin-actions">
          <ElButton type="primary" @click="openAddTier">
            {{ t('hallOfFame.currentManage.addTier') }}
          </ElButton>
        </div>
      </div>

      <div v-if="directory && directory.tiers.length > 0" class="fuxi-directory__content">
        <TierSection
          v-for="tier in directory.tiers"
          :key="tier.id"
          :tier="tier"
          :style-config="styleConfig"
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

    <TierDialog v-model="tierDialogOpen" :tier="editingTier" @saved="handleTierSaved" />

    <AdminCardDialog
      v-model="adminDialogOpen"
      :admin="editingAdmin"
      :default-tier-id="addingAdminToTierId"
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
    deleteFuxiAdmin,
    deleteFuxiAdminTier,
    fetchFuxiAdminDirectory,
    updateFuxiAdminConfig
  } from '@/api/fuxi-admins'
  import { useUserStore } from '@/store/modules/user'

  import AdminCardDialog from './modules/admin-card-dialog.vue'
  import TierDialog from './modules/tier-dialog.vue'
  import TierSection from './modules/tier-section.vue'

  const { t } = useI18n()
  const userStore = useUserStore()

  const loading = ref(false)
  const directory = ref<Api.FuxiAdmin.DirectoryResponse | null>(null)
  const localFontSize = ref(14)
  const cardWidth = ref(240)
  const pageBackgroundColor = ref('#10243a')
  const cardBackgroundColor = ref('#1b324c')
  const cardBorderColor = ref('#d9a441')
  const tierTitleColor = ref('#f8d26b')
  const nameTextColor = ref('#fff7d6')
  const bodyTextColor = ref('#d7dfef')

  const tierDialogOpen = ref(false)
  const editingTier = ref<Api.FuxiAdmin.Tier | null>(null)
  const adminDialogOpen = ref(false)
  const editingAdmin = ref<Api.FuxiAdmin.Admin | null>(null)
  const addingAdminToTierId = ref<number | null>(null)
  let pendingConfigSnapshot: Api.FuxiAdmin.UpdateConfigParams | null = null
  let configSaveInFlight = false

  const canEdit = computed(() => {
    const roles = userStore.getUserInfo?.roles ?? []
    return roles.some((role) => ['super_admin', 'admin'].includes(role))
  })

  const allTiers = computed<Api.FuxiAdmin.Tier[]>(() => directory.value?.tiers ?? [])
  const styleConfig = computed<Api.FuxiAdmin.Config>(() => ({
    id: directory.value?.config.id ?? 1,
    base_font_size: localFontSize.value,
    card_width: cardWidth.value,
    page_background_color: pageBackgroundColor.value,
    card_background_color: cardBackgroundColor.value,
    card_border_color: cardBorderColor.value,
    tier_title_color: tierTitleColor.value,
    name_text_color: nameTextColor.value,
    body_text_color: bodyTextColor.value,
    created_at: directory.value?.config.created_at ?? '',
    updated_at: directory.value?.config.updated_at ?? ''
  }))
  const directoryStyle = computed(() => ({
    '--page-background-color': pageBackgroundColor.value
  }))

  onMounted(() => {
    void loadDirectory()
  })

  async function loadDirectory() {
    loading.value = true
    try {
      directory.value = await fetchFuxiAdminDirectory()
      syncLocalConfig(directory.value.config)
    } catch (error) {
      ElMessage.error(
        error instanceof Error ? error.message : t('hallOfFame.currentManage.saveFailed')
      )
    } finally {
      loading.value = false
    }
  }

  function syncLocalConfig(config: Api.FuxiAdmin.Config) {
    localFontSize.value = config.base_font_size
    cardWidth.value = config.card_width
    pageBackgroundColor.value = config.page_background_color
    cardBackgroundColor.value = config.card_background_color
    cardBorderColor.value = config.card_border_color
    tierTitleColor.value = config.tier_title_color
    nameTextColor.value = config.name_text_color
    bodyTextColor.value = config.body_text_color
  }

  function sortTiers<T extends Api.FuxiAdmin.Tier>(tiers: T[]) {
    return [...tiers].sort(
      (left, right) => left.sort_order - right.sort_order || left.id - right.id
    )
  }

  function buildConfigUpdateSnapshot(): Api.FuxiAdmin.UpdateConfigParams {
    return {
      base_font_size: localFontSize.value,
      card_width: cardWidth.value,
      page_background_color: pageBackgroundColor.value,
      card_background_color: cardBackgroundColor.value,
      card_border_color: cardBorderColor.value,
      tier_title_color: tierTitleColor.value,
      name_text_color: nameTextColor.value,
      body_text_color: bodyTextColor.value
    }
  }

  function queueConfigSave() {
    pendingConfigSnapshot = buildConfigUpdateSnapshot()
    if (!configSaveInFlight) {
      void flushConfigSaveQueue()
    }
  }

  async function flushConfigSaveQueue() {
    if (!directory.value || !pendingConfigSnapshot) {
      return
    }

    const snapshot = pendingConfigSnapshot
    pendingConfigSnapshot = null
    configSaveInFlight = true

    try {
      const savedConfig = await updateFuxiAdminConfig(snapshot)
      directory.value.config = savedConfig
      if (!pendingConfigSnapshot) {
        syncLocalConfig(savedConfig)
      }
    } catch (error) {
      if (!pendingConfigSnapshot) {
        syncLocalConfig(directory.value.config)
      }
      ElMessage.error(
        error instanceof Error ? error.message : t('hallOfFame.currentManage.saveFailed')
      )
    } finally {
      configSaveInFlight = false
    }

    if (pendingConfigSnapshot) {
      void flushConfigSaveQueue()
    }
  }

  function handleBaseFontSizeChange(value: number | undefined) {
    if (value == null) {
      return
    }
    queueConfigSave()
  }

  function handleCardWidthChange(value: number | undefined) {
    if (value == null) {
      return
    }
    queueConfigSave()
  }

  function handleColorConfigChange(
    key:
      | 'page_background_color'
      | 'card_background_color'
      | 'card_border_color'
      | 'tier_title_color'
      | 'name_text_color'
      | 'body_text_color',
    value: string | null
  ) {
    if (typeof value !== 'string') {
      return
    }

    switch (key) {
      case 'page_background_color':
        pageBackgroundColor.value = value
        break
      case 'card_background_color':
        cardBackgroundColor.value = value
        break
      case 'card_border_color':
        cardBorderColor.value = value
        break
      case 'tier_title_color':
        tierTitleColor.value = value
        break
      case 'name_text_color':
        nameTextColor.value = value
        break
      case 'body_text_color':
        bodyTextColor.value = value
        break
    }

    queueConfigSave()
  }

  function openAddTier() {
    editingTier.value = null
    tierDialogOpen.value = true
  }

  function openEditTier(tier: Api.FuxiAdmin.TierWithAdmins) {
    editingTier.value = tier
    tierDialogOpen.value = true
  }

  function handleTierSaved(savedTier: Api.FuxiAdmin.Tier) {
    if (!directory.value) {
      return
    }

    const idx = directory.value.tiers.findIndex((tier) => tier.id === savedTier.id)
    if (idx >= 0) {
      directory.value.tiers[idx] = {
        ...directory.value.tiers[idx],
        name: savedTier.name,
        sort_order: savedTier.sort_order
      }
    } else {
      directory.value.tiers = [...directory.value.tiers, { ...savedTier, admins: [] }]
    }

    directory.value.tiers = sortTiers(directory.value.tiers)
  }

  async function handleDeleteTier(tier: Api.FuxiAdmin.TierWithAdmins) {
    try {
      await ElMessageBox.confirm(
        t('hallOfFame.currentManage.deleteTierConfirm'),
        t('hallOfFame.currentManage.deleteTier'),
        {
          type: 'warning',
          confirmButtonText: t('common.confirm'),
          cancelButtonText: t('common.cancel')
        }
      )
    } catch {
      return
    }

    try {
      await deleteFuxiAdminTier(tier.id)
      if (directory.value) {
        directory.value.tiers = directory.value.tiers.filter((item) => item.id !== tier.id)
      }
      ElMessage.success(t('hallOfFame.currentManage.deleteSuccess'))
    } catch (error) {
      ElMessage.error(
        error instanceof Error ? error.message : t('hallOfFame.currentManage.saveFailed')
      )
    }
  }

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
    if (!directory.value) {
      return
    }

    for (const tier of directory.value.tiers) {
      const idx = tier.admins.findIndex((admin) => admin.id === savedAdmin.id)
      if (idx >= 0) {
        if (savedAdmin.tier_id !== tier.id) {
          tier.admins = tier.admins.filter((admin) => admin.id !== savedAdmin.id)
          const newTier = directory.value.tiers.find((item) => item.id === savedAdmin.tier_id)
          if (newTier) {
            newTier.admins = [...newTier.admins, savedAdmin]
          }
        } else {
          tier.admins[idx] = savedAdmin
        }
        addingAdminToTierId.value = null
        return
      }
    }

    const targetTier = directory.value.tiers.find((tier) => tier.id === savedAdmin.tier_id)
    if (targetTier) {
      targetTier.admins = [...targetTier.admins, savedAdmin]
    }
    addingAdminToTierId.value = null
  }

  async function handleDeleteAdmin(admin: Api.FuxiAdmin.Admin) {
    try {
      await ElMessageBox.confirm(
        t('hallOfFame.currentManage.deleteAdminConfirm'),
        t('hallOfFame.currentManage.deleteAdmin'),
        {
          type: 'warning',
          confirmButtonText: t('common.confirm'),
          cancelButtonText: t('common.cancel')
        }
      )
    } catch {
      return
    }

    try {
      await deleteFuxiAdmin(admin.id)
      if (directory.value) {
        for (const tier of directory.value.tiers) {
          tier.admins = tier.admins.filter((item) => item.id !== admin.id)
        }
      }
      ElMessage.success(t('hallOfFame.currentManage.deleteSuccess'))
    } catch (error) {
      ElMessage.error(
        error instanceof Error ? error.message : t('hallOfFame.currentManage.saveFailed')
      )
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
    background-color: var(--page-background-color, #10243a);
    background-image:
      radial-gradient(circle at top left, rgba(246, 206, 112, 0.16), transparent 24%),
      radial-gradient(circle at top right, rgba(104, 164, 255, 0.16), transparent 20%),
      linear-gradient(180deg, rgba(255, 255, 255, 0.04), rgba(255, 255, 255, 0) 60%);
    overflow-y: auto;
  }

  .fuxi-directory__inner {
    display: flex;
    flex-direction: column;
    gap: 28px;
    max-width: 1200px;
    width: 100%;
    margin: 0 auto;
  }

  .fuxi-directory__admin-bar {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
    align-items: start;
    gap: 12px;
    padding: 16px 18px;
    border-radius: 18px;
    border: 1px solid rgba(255, 255, 255, 0.1);
    background: rgba(8, 14, 24, 0.24);
    backdrop-filter: blur(16px);
  }

  .fuxi-directory__settings-group {
    display: flex;
    flex-direction: column;
    gap: 12px;
    min-width: 0;
    padding: 14px;
    border-radius: 14px;
    border: 1px solid rgba(255, 255, 255, 0.08);
    background: rgba(255, 255, 255, 0.03);
  }

  .fuxi-directory__settings-title {
    margin: 0;
    color: rgba(255, 247, 214, 0.86);
    font-size: 12px;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }

  .fuxi-directory__control {
    display: flex;
    flex-direction: column;
    gap: 8px;
    min-width: 0;
  }

  .fuxi-directory__control span {
    color: rgba(255, 247, 214, 0.82);
    font-size: 11px;
    font-weight: 600;
    letter-spacing: 0.08em;
    line-height: 1;
    text-transform: uppercase;
  }

  .fuxi-directory__content {
    display: flex;
    flex-direction: column;
    gap: 40px;
  }

  .fuxi-directory__admin-actions {
    display: flex;
    align-items: flex-end;
    justify-content: flex-end;
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

  :deep(.el-input-number) {
    width: 100%;
  }

  :deep(.el-color-picker) {
    width: 100%;
  }

  :deep(.el-color-picker__trigger) {
    width: 100%;
    border-color: rgba(255, 255, 255, 0.14);
    background: rgba(255, 255, 255, 0.04);
  }

  @media (max-width: 768px) {
    .fuxi-directory {
      padding: 16px;
    }

    .fuxi-directory__admin-bar {
      grid-template-columns: 1fr;
    }

    .fuxi-directory__control {
      min-width: 100%;
    }

    .fuxi-directory__admin-actions {
      justify-content: stretch;
    }

    .fuxi-directory__admin-actions :deep(.el-button) {
      width: 100%;
    }
  }
</style>
