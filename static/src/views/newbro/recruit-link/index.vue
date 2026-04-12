<template>
  <div class="recruit-link-page">
    <ElCard shadow="never" class="mb-4">
      <div class="page-header">
        <div class="page-title">{{ t('newbro.recruitLink.title') }}</div>
        <div class="page-subtitle">{{ t('newbro.recruitLink.subtitle') }}</div>
      </div>
    </ElCard>

    <ElTabs v-model="activeTab">
      <ElTabPane :label="t('newbro.recruitLink.myLinksTab')" name="my">
        <ElCard shadow="never" class="mb-4 recruit-link-hero">
          <div class="recruit-link-hero__body">
            <div class="recruit-link-hero__content">
              <span class="recruit-link-hero__label">{{ t('newbro.recruitLink.yourLink') }}</span>

              <div v-if="currentLink" class="recruit-link-url">
                <span class="recruit-link-url__text">{{ fullLinkUrl(currentLink.code) }}</span>
                <ArtCopyButton
                  :text="fullLinkUrl(currentLink.code)"
                  :aria-label="t('common.copy')"
                />
              </div>

              <div v-else class="recruit-link-empty-hint">
                {{ t('newbro.recruitLink.noGeneratedLink') }}
              </div>
            </div>

            <ElButton type="primary" :loading="generating" @click="handleGenerate">
              {{ t('newbro.recruitLink.generateBtn') }}
            </ElButton>
          </div>
        </ElCard>

        <div v-loading="myLinksLoading">
          <ElEmpty
            v-if="!myLinksLoading && myLinks.length === 0"
            :description="t('newbro.recruitLink.empty')"
          />

          <div v-else class="recruit-link-list">
            <ElCard v-for="link in myLinks" :key="link.id" shadow="never" class="recruit-link-card">
              <template #header>
                <div class="recruit-link-card__header">
                  <div class="recruit-link-card__header-main">
                    <ElTag effect="plain" size="small">
                      {{ t(`newbro.recruitLink.source.${link.source}`) }}
                    </ElTag>
                  </div>
                  <div class="recruit-link-card__header-meta">
                    <span class="recruit-link-card__meta-label">
                      {{ t('newbro.recruitLink.colGeneratedAt') }}
                    </span>
                    <span class="recruit-link-card__meta-value">
                      {{ formatDateTime(link.generated_at) }}
                    </span>
                  </div>
                </div>
              </template>

              <div v-if="link.source === 'direct_referral'" class="recruit-link-card__link-row">
                <span class="recruit-link-card__link-label">
                  {{ t('newbro.recruitLink.directReferralRecord') }}
                </span>
                <div class="recruit-link-card__direct-referral-text">
                  {{ t('newbro.recruitLink.directReferralRecord') }}
                </div>
              </div>

              <div v-else class="recruit-link-card__link-row">
                <span class="recruit-link-card__link-label">{{
                  t('newbro.recruitLink.yourLink')
                }}</span>
                <div class="recruit-link-url">
                  <span class="recruit-link-url__text">{{ fullLinkUrl(link.code) }}</span>
                  <ArtCopyButton :text="fullLinkUrl(link.code)" :aria-label="t('common.copy')" />
                </div>
              </div>

              <ElTable v-if="link.entries.length > 0" :data="link.entries" stripe>
                <ElTableColumn prop="qq" :label="t('newbro.recruitLink.colQQ')" min-width="140" />
                <ElTableColumn
                  prop="entered_at"
                  :label="t('newbro.recruitLink.colEnteredAt')"
                  min-width="180"
                >
                  <template #default="{ row }">
                    {{ formatDateTime(row.entered_at) }}
                  </template>
                </ElTableColumn>
                <ElTableColumn prop="status" :label="t('newbro.recruitLink.colStatus')" width="120">
                  <template #default="{ row }">
                    <ElTag :type="statusTagType(row.status)" effect="plain" size="small">
                      {{ t(`newbro.recruitLink.status.${row.status}`) }}
                    </ElTag>
                  </template>
                </ElTableColumn>
                <ElTableColumn
                  prop="rewarded_at"
                  :label="t('newbro.recruitLink.colRewardedAt')"
                  min-width="180"
                >
                  <template #default="{ row }">
                    {{ row.rewarded_at ? formatDateTime(row.rewarded_at) : '-' }}
                  </template>
                </ElTableColumn>
              </ElTable>

              <ElEmpty v-else :description="t('newbro.recruitLink.noEntries')" />
            </ElCard>
          </div>
        </div>
      </ElTabPane>

      <ElTabPane v-if="isAdmin" :label="t('newbro.recruitLink.adminTab')" name="admin">
        <ElCard shadow="never" class="art-table-card">
          <ArtTableHeader
            v-model:columns="adminColumnChecks"
            :loading="adminLinksLoading"
            @refresh="loadAdminLinks"
          />

          <ArtTable
            :loading="adminLinksLoading"
            :data="adminLinks"
            :columns="adminTableColumns"
            :pagination="adminPagination"
            visual-variant="ledger"
            :pagination-options="{ pageSizes: [20, 50, 100] }"
            @pagination:size-change="handleAdminSizeChange"
            @pagination:current-change="handleAdminCurrentChange"
          />

          <ElEmpty
            v-if="!adminLinksLoading && adminLinks.length === 0"
            :description="t('newbro.recruitLink.empty')"
          />
        </ElCard>
      </ElTabPane>
    </ElTabs>
  </div>
</template>

<script setup lang="ts">
  import type { ColumnOption } from '@/types/component'
  import { computed, h, onMounted, ref, watch } from 'vue'
  import { useI18n } from 'vue-i18n'
  import { ElButton, ElCard, ElEmpty, ElMessage, ElTable, ElTableColumn, ElTag } from 'element-plus'
  import ArtCopyButton from '@/components/core/forms/art-copy-button/index.vue'
  import { fetchAdminRecruitLinks, fetchMyRecruitLinks, generateRecruitLink } from '@/api/newbro'
  import { useTable } from '@/hooks/core/useTable'
  import { useNewbroFormatters } from '@/hooks/newbro/useNewbroFormatters'
  import { useUserStore } from '@/store/modules/user'

  defineOptions({ name: 'NewbroRecruitLink' })

  const { t } = useI18n()
  const { formatDateTime } = useNewbroFormatters()
  const userStore = useUserStore()

  const activeTab = ref<'my' | 'admin'>('my')
  const generating = ref(false)
  const myLinksLoading = ref(false)
  const myLinks = ref<Api.Newbro.RecruitLink[]>([])
  const adminLoaded = ref(false)

  const roles = computed(() => userStore.getUserInfo?.roles ?? [])
  const isAdmin = computed(() =>
    roles.value.some((role) => ['admin', 'super_admin'].includes(role))
  )
  const currentLink = computed(() => myLinks.value.find((link) => link.source === 'link') ?? null)

  const fullLinkUrl = (code: string) =>
    `${window.location.origin}${window.location.pathname}#/r/${code}`

  const statusTagType = (status: Api.Newbro.RecruitEntry['status']) => {
    switch (status) {
      case 'valid':
        return 'success' as const
      case 'stalled':
        return 'info' as const
      default:
        return 'warning' as const
    }
  }

  const adminColumns = computed<ColumnOption<Api.Newbro.AdminRecruitLink>[]>(() => [
    {
      prop: 'user_id',
      label: t('newbro.recruitLink.colUserId'),
      width: 100
    },
    {
      prop: 'source',
      label: t('newbro.recruitLink.colSource'),
      minWidth: 140,
      formatter: (row) =>
        h(
          ElTag,
          {
            effect: 'plain',
            size: 'small'
          },
          () => t(`newbro.recruitLink.source.${row.source}`)
        )
    },
    {
      prop: 'code',
      label: t('newbro.recruitLink.colCode'),
      minWidth: 160,
      formatter: (row) => {
        if (row.source === 'direct_referral') {
          return h('span', { class: 'text-gray-400' }, t('newbro.recruitLink.directReferralRecord'))
        }

        return h('div', { class: 'recruit-code-cell' }, [
          h('span', { class: 'recruit-code-cell__text' }, row.code),
          h(ArtCopyButton, {
            text: fullLinkUrl(row.code),
            ariaLabel: t('common.copy')
          })
        ])
      }
    },
    {
      prop: 'generated_at',
      label: t('newbro.recruitLink.colGeneratedAt'),
      minWidth: 180,
      formatter: (row) => formatDateTime(row.generated_at)
    },
    {
      prop: 'entries',
      label: t('newbro.recruitLink.colEntries'),
      minWidth: 320,
      formatter: (row) => {
        if (row.entries.length === 0) {
          return h('span', { class: 'text-gray-400' }, '-')
        }

        return h(
          'div',
          { class: 'recruit-entry-summary' },
          row.entries.map((entry) =>
            h(
              ElTag,
              {
                key: entry.id,
                type: statusTagType(entry.status),
                effect: 'plain',
                size: 'small'
              },
              () => `${entry.qq} · ${t(`newbro.recruitLink.status.${entry.status}`)}`
            )
          )
        )
      }
    }
  ])

  const {
    columns: adminTableColumns,
    columnChecks: adminColumnChecks,
    data: adminLinks,
    loading: adminLinksLoading,
    pagination: adminPagination,
    handleSizeChange: handleAdminSizeChange,
    handleCurrentChange: handleAdminCurrentChange,
    getData: loadAdminLinks
  } = useTable({
    core: {
      apiFn: fetchAdminRecruitLinks,
      apiParams: { current: 1, size: 20 },
      immediate: false,
      columnsFactory: () => adminColumns.value
    },
    hooks: {
      onError: (error) => {
        ElMessage.error(error.message || t('httpMsg.requestFailed'))
      }
    }
  })

  const loadMyLinks = async () => {
    myLinksLoading.value = true
    try {
      myLinks.value = await fetchMyRecruitLinks()
    } catch (error) {
      myLinks.value = []
      ElMessage.error((error as Error)?.message || t('httpMsg.requestFailed'))
    } finally {
      myLinksLoading.value = false
    }
  }

  const handleGenerate = async () => {
    generating.value = true
    try {
      await generateRecruitLink()
      ElMessage.success(t('newbro.recruitLink.generateSuccess'))
      await loadMyLinks()
    } catch (error) {
      ElMessage.error((error as Error)?.message || t('newbro.recruitLink.generateFailed'))
    } finally {
      generating.value = false
    }
  }

  const ensureAdminLoaded = () => {
    if (!isAdmin.value || adminLoaded.value) {
      return
    }
    adminLoaded.value = true
    void loadAdminLinks()
  }

  onMounted(() => {
    void loadMyLinks()
  })

  watch(activeTab, (tab) => {
    if (tab === 'admin') {
      ensureAdminLoaded()
    }
  })
</script>

<style scoped>
  .page-header {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .page-title {
    font-size: 18px;
    font-weight: 600;
  }

  .page-subtitle {
    font-size: 13px;
    color: var(--el-text-color-secondary);
  }

  .recruit-link-hero__body {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    flex-wrap: wrap;
  }

  .recruit-link-hero__content {
    display: flex;
    flex-direction: column;
    gap: 10px;
    min-width: 0;
    flex: 1;
  }

  .recruit-link-hero__label,
  .recruit-link-card__link-label,
  .recruit-link-card__meta-label {
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  .recruit-link-list {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .recruit-link-empty-hint {
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }

  .recruit-link-card__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
  }

  .recruit-link-card__header-main,
  .recruit-link-card__header-meta {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .recruit-link-card__meta-value {
    font-size: 13px;
    color: var(--el-text-color-primary);
  }

  .recruit-link-card__link-row {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-bottom: 16px;
  }

  .recruit-link-url {
    display: flex;
    align-items: flex-start;
    gap: 8px;
    min-width: 0;
    padding: 10px 12px;
    border: 1px solid var(--el-border-color-light);
    border-radius: 12px;
    background: var(--el-fill-color-extra-light);
  }

  .recruit-link-card__direct-referral-text {
    padding: 10px 12px;
    border: 1px solid var(--el-border-color-light);
    border-radius: 12px;
    background: var(--el-fill-color-extra-light);
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }

  .recruit-link-url__text,
  .recruit-code-cell__text {
    min-width: 0;
    word-break: break-all;
    font-size: 13px;
    line-height: 1.5;
    font-family:
      ui-monospace,
      SFMono-Regular,
      SFMono-Regular,
      Menlo,
      Monaco,
      Consolas,
      Liberation Mono,
      Courier New,
      monospace;
  }

  .recruit-code-cell {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .recruit-entry-summary {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  @media (max-width: 768px) {
    .recruit-link-card__header {
      flex-direction: column;
      align-items: flex-start;
    }

    .recruit-link-card__header-main,
    .recruit-link-card__header-meta {
      flex-wrap: wrap;
    }

    .recruit-code-cell {
      align-items: flex-start;
    }
  }
</style>
