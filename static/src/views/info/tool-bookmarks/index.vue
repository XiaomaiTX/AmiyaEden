<template>
  <div class="tool-bookmarks-page">
    <ElCard shadow="never" class="mb-4">
      <div class="tool-bookmarks-header">
        <div>
          <div class="tool-bookmarks-title">{{ t('info.toolBookmarksTitle') }}</div>
          <div class="tool-bookmarks-subtitle">{{ t('info.toolBookmarksSubtitle') }}</div>
        </div>
        <div v-if="isAdmin" class="tool-bookmarks-actions">
          <ElButton @click="loadData" :loading="loading">{{ t('common.refresh') }}</ElButton>
          <ElButton type="primary" @click="openCreateDialog">
            {{ t('info.toolBookmarksAdd') }}
          </ElButton>
        </div>
      </div>
    </ElCard>

    <ElCard shadow="never" v-loading="loading">
      <ElEmpty
        v-if="!loading && bookmarks.length === 0"
        :description="t('info.toolBookmarksEmpty')"
      />
      <div v-else class="bookmark-grid">
        <div v-for="item in bookmarks" :key="item.id" class="bookmark-card">
          <div class="bookmark-main">
            <div class="bookmark-logo-wrap">
              <img
                v-if="item.logo_url"
                :src="item.logo_url"
                :alt="item.name"
                class="bookmark-logo"
                loading="lazy"
              />
              <div v-else class="bookmark-logo-fallback">{{ item.name.slice(0, 1) }}</div>
            </div>
            <div class="bookmark-content">
              <div class="bookmark-title-row">
                <a
                  :href="item.url"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="bookmark-title-link"
                >
                  {{ item.name }}
                </a>
                <ElTag v-if="isAdmin && !item.is_enabled" type="info" size="small">
                  {{ t('info.toolBookmarksDisabled') }}
                </ElTag>
              </div>
              <div v-if="item.description" class="bookmark-description">{{ item.description }}</div>
              <div class="bookmark-url">{{ item.url }}</div>
            </div>
          </div>
          <div v-if="isAdmin" class="bookmark-admin-actions">
            <ElButton size="small" @click="openEditDialog(item)">{{ t('common.edit') }}</ElButton>
            <ElButton size="small" type="danger" @click="handleDelete(item.id)">
              {{ t('common.delete') }}
            </ElButton>
          </div>
        </div>
      </div>
    </ElCard>

    <ElDialog v-model="dialogVisible" :title="dialogTitle" width="520px">
      <ElForm label-position="top">
        <ElFormItem :label="t('info.toolBookmarksFormName')">
          <ElInput v-model="form.name" maxlength="128" />
        </ElFormItem>
        <ElFormItem :label="t('info.toolBookmarksFormURL')">
          <ElInput v-model="form.url" />
        </ElFormItem>
        <ElFormItem :label="t('info.toolBookmarksFormDescription')">
          <ElInput v-model="form.description" type="textarea" :rows="3" maxlength="1024" />
        </ElFormItem>
        <ElFormItem :label="t('info.toolBookmarksFormSortOrder')">
          <ElInputNumber v-model="form.sort_order" :min="0" :controls="false" />
        </ElFormItem>
        <ElFormItem>
          <ElSwitch v-model="form.is_enabled" />
          <span class="ml-2">{{ t('info.toolBookmarksFormEnabled') }}</span>
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="dialogVisible = false">{{ t('common.cancel') }}</ElButton>
        <ElButton type="primary" :loading="submitting" @click="submitForm">
          {{ t('common.confirm') }}
        </ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { ElMessage, ElMessageBox } from 'element-plus'
  import { useI18n } from 'vue-i18n'
  import {
    createToolBookmark,
    deleteToolBookmark,
    fetchAdminToolBookmarks,
    fetchVisibleToolBookmarks,
    updateToolBookmark
  } from '@/api/tool-bookmark'
  import { useUserStore } from '@/store/modules/user'

  defineOptions({ name: 'EveInfoToolBookmarks' })

  const { t } = useI18n()
  const userStore = useUserStore()
  const loading = ref(false)
  const submitting = ref(false)
  const bookmarks = ref<Api.ToolBookmark.Bookmark[]>([])
  const dialogVisible = ref(false)
  const editingID = ref<number | null>(null)

  const roles = computed(() => userStore.getUserInfo?.roles ?? [])
  const isAdmin = computed(
    () => roles.value.includes('admin') || roles.value.includes('super_admin')
  )
  const dialogTitle = computed(() =>
    editingID.value ? t('info.toolBookmarksEdit') : t('info.toolBookmarksAdd')
  )

  const form = reactive<Api.ToolBookmark.UpsertParams>({
    name: '',
    url: '',
    description: '',
    is_enabled: true,
    sort_order: 0
  })

  async function loadData() {
    loading.value = true
    try {
      bookmarks.value = isAdmin.value
        ? await fetchAdminToolBookmarks()
        : await fetchVisibleToolBookmarks()
    } catch (error) {
      ElMessage.error((error as Error).message || t('httpMsg.requestFailed'))
      bookmarks.value = []
    } finally {
      loading.value = false
    }
  }

  function openCreateDialog() {
    editingID.value = null
    form.name = ''
    form.url = ''
    form.description = ''
    form.is_enabled = true
    form.sort_order = bookmarks.value.length + 1
    dialogVisible.value = true
  }

  function openEditDialog(item: Api.ToolBookmark.Bookmark) {
    editingID.value = item.id
    form.name = item.name
    form.url = item.url
    form.description = item.description
    form.is_enabled = item.is_enabled
    form.sort_order = item.sort_order
    dialogVisible.value = true
  }

  async function submitForm() {
    if (!form.name?.trim() || !form.url?.trim()) {
      ElMessage.warning(t('info.toolBookmarksRequiredFields'))
      return
    }
    submitting.value = true
    try {
      if (editingID.value) {
        await updateToolBookmark(editingID.value, { ...form })
      } else {
        await createToolBookmark({ ...form })
      }
      ElMessage.success(t('info.toolBookmarksSaveSuccess'))
      dialogVisible.value = false
      await loadData()
    } catch (error) {
      ElMessage.error((error as Error).message || t('info.toolBookmarksSaveFailed'))
    } finally {
      submitting.value = false
    }
  }

  async function handleDelete(id: number) {
    try {
      await ElMessageBox.confirm(t('info.toolBookmarksDeleteConfirm'), t('common.warning'), {
        type: 'warning'
      })
      await deleteToolBookmark(id)
      ElMessage.success(t('info.toolBookmarksDeleteSuccess'))
      await loadData()
    } catch (error) {
      if (error !== 'cancel') {
        ElMessage.error((error as Error).message || t('info.toolBookmarksDeleteFailed'))
      }
    }
  }

  onMounted(() => {
    loadData()
  })
</script>

<style scoped>
  .tool-bookmarks-header {
    display: flex;
    justify-content: space-between;
    gap: 12px;
    align-items: center;
  }

  .tool-bookmarks-title {
    font-size: 18px;
    font-weight: 600;
  }

  .tool-bookmarks-subtitle {
    margin-top: 4px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }

  .tool-bookmarks-actions {
    display: flex;
    gap: 8px;
  }

  .bookmark-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
    gap: 12px;
  }

  .bookmark-card {
    border: 1px solid var(--el-border-color-light);
    border-radius: 12px;
    padding: 12px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .bookmark-main {
    display: flex;
    gap: 12px;
  }

  .bookmark-logo-wrap {
    width: 36px;
    height: 36px;
    border-radius: 10px;
    overflow: hidden;
    border: 1px solid var(--el-border-color-lighter);
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--el-fill-color-extra-light);
  }

  .bookmark-logo {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .bookmark-logo-fallback {
    font-size: 14px;
    font-weight: 600;
    color: var(--el-text-color-secondary);
  }

  .bookmark-content {
    min-width: 0;
    flex: 1;
  }

  .bookmark-title-row {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .bookmark-title-link {
    font-size: 15px;
    font-weight: 600;
    color: var(--el-color-primary);
    text-decoration: none;
  }

  .bookmark-title-link:hover {
    text-decoration: underline;
  }

  .bookmark-description {
    margin-top: 6px;
    font-size: 13px;
    color: var(--el-text-color-primary);
    word-break: break-word;
  }

  .bookmark-url {
    margin-top: 6px;
    color: var(--el-text-color-secondary);
    font-size: 12px;
    word-break: break-all;
  }

  .bookmark-admin-actions {
    display: flex;
    gap: 8px;
    justify-content: flex-end;
  }
</style>
