<template>
  <div class="info-assets-page art-full-height">
    <!-- 统计栏 -->
    <ElCard shadow="never" class="mb-2">
      <div class="flex items-center justify-between flex-wrap gap-4">
        <div class="flex items-center gap-4">
          <ElButton :loading="loading" size="small" @click="loadLocations">
            <el-icon class="mr-1"><Refresh /></el-icon>
            {{ $t('common.refresh') }}
          </ElButton>
        </div>
        <div v-if="locationsData" class="flex items-center gap-4 text-sm text-gray-500">
          <span>
            {{ $t('info.assetCount') }}:
            <strong class="text-blue-500">{{ locationsData.total_items }}</strong>
          </span>
          <span>
            {{ $t('info.locationName') }}:
            <strong class="text-blue-500">{{ locationsData.total_locations }}</strong>
          </span>
        </div>
      </div>
    </ElCard>

    <!-- 主体区域 -->
    <div v-loading="loading" class="assets-main">
      <!-- 搜索栏 -->
      <div class="filter-bar">
        <ElInput
          v-model="searchKeyword"
          :placeholder="$t('info.searchAsset')"
          clearable
          style="width: 280px"
          size="small"
          :prefix-icon="Search"
          @change="onSearchChange"
        />
      </div>

      <!-- 错误提示 -->
      <div v-if="errorMessage" class="asset-error">
        <ElAlert :title="errorMessage" type="error" show-icon :closable="false" />
        <ElButton size="small" type="primary" class="mt-4" @click="loadLocations">
          {{ $t('common.retry') }}
        </ElButton>
      </div>

      <!-- 空位置列表 -->
      <ElEmpty
        v-else-if="!loading && locationsData && locationsData.locations.length === 0"
        :description="$t('info.assetNoLocationData')"
        :image-size="60"
      />

      <!-- 按位置分组 -->
      <div v-else-if="locationsData && locationsData.locations.length > 0" class="assets-groups">
        <div v-for="loc in locationsData.locations" :key="loc.location_id" class="location-section">
          <!-- 位置标题 -->
          <div class="location-header" @click="toggleLocation(loc)">
            <span class="mg-arrow" :class="{ expanded: expandedLocations.has(loc.location_id) }"
              >▶</span
            >
            <span class="mg-title">{{ loc.location_name }}</span>
            <span class="mg-count">{{ loc.top_level_count }}</span>
            <span class="mg-char-count">{{ loc.character_count }} {{ $t('info.owner') }}</span>
          </div>

          <!-- 位置物品列表 -->
          <div v-if="expandedLocations.has(loc.location_id)" class="asset-items">
            <div v-if="locationErrors[loc.location_id]" class="section-error">
              <ElAlert
                :title="locationErrors[loc.location_id]"
                type="warning"
                show-icon
                :closable="false"
                class="mb-2"
              />
            </div>

            <div v-loading="locationLoadingMap[loc.location_id]" class="min-h-40">
              <template v-for="item in locationItemsMap[loc.location_id]" :key="item.item_id">
                <div
                  class="asset-item"
                  :class="{ clickable: item.has_children }"
                  @click="item.has_children ? toggleChildren(item) : undefined"
                >
                  <span
                    v-if="item.has_children"
                    class="mg-arrow child-toggle"
                    :class="{ expanded: expandedItems.has(item.item_id) }"
                    >▶</span
                  >
                  <span v-else class="mg-arrow-placeholder"></span>
                  <img
                    :src="getItemIcon(item)"
                    :alt="item.type_name"
                    class="asset-icon"
                    loading="lazy"
                  />
                  <div class="asset-info">
                    <span class="asset-type-name">{{ item.type_name }}</span>
                    <span v-if="item.asset_name" class="asset-name-tag">{{ item.asset_name }}</span>
                    <span v-if="item.child_count > 0" class="child-count-badge">{{
                      item.child_count
                    }}</span>
                  </div>
                  <span class="asset-group">{{ item.group_name }}</span>
                  <span class="asset-qty">
                    {{ item.quantity > 1 ? `x${item.quantity}` : '' }}
                  </span>
                  <span class="asset-owner">{{ item.character_name }}</span>
                </div>
                <!-- 子物品 -->
                <div
                  v-if="item.has_children && expandedItems.has(item.item_id)"
                  class="child-items"
                >
                  <div v-if="childrenErrors[item.item_id]" class="section-error">
                    <ElAlert
                      :title="childrenErrors[item.item_id]"
                      type="warning"
                      show-icon
                      :closable="false"
                      class="mb-2"
                    />
                  </div>
                  <div v-loading="childrenLoadingMap[item.item_id]" class="min-h-30">
                    <div
                      v-for="child in childrenMap[item.item_id]"
                      :key="child.item_id"
                      class="asset-item child"
                    >
                      <img
                        :src="getItemIcon(child)"
                        :alt="child.type_name"
                        class="asset-icon"
                        loading="lazy"
                      />
                      <div class="asset-info">
                        <span class="asset-type-name">{{ child.type_name }}</span>
                        <span v-if="child.asset_name" class="asset-name-tag">{{
                          child.asset_name
                        }}</span>
                        <span v-if="child.child_count > 0" class="child-count-badge">{{
                          child.child_count
                        }}</span>
                      </div>
                      <span class="asset-group">{{ child.group_name }}</span>
                      <span class="asset-qty">
                        {{ child.quantity > 1 ? `x${child.quantity}` : '' }}
                      </span>
                      <span class="asset-owner">{{ child.character_name }}</span>
                    </div>
                  </div>
                </div>
              </template>
            </div>

            <!-- 位置物品分页 -->
            <div
              v-if="
                locationPaginationMap[loc.location_id]?.total >
                locationPaginationMap[loc.location_id]?.pageSize
              "
              class="pagination-row"
            >
              <ElPagination
                small
                layout="prev, pager, next"
                :total="locationPaginationMap[loc.location_id].total"
                :page-size="locationPaginationMap[loc.location_id].pageSize"
                :current-page="locationPaginationMap[loc.location_id].page"
                @current-change="(p) => loadLocationItems(loc, p)"
              />
            </div>

            <ElEmpty
              v-if="
                !locationLoadingMap[loc.location_id] &&
                !locationErrors[loc.location_id] &&
                (locationItemsMap[loc.location_id]?.length ?? 0) === 0
              "
              :description="$t('info.assetNoItemsInLocation')"
              :image-size="40"
            />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { Refresh, Search } from '@element-plus/icons-vue'
  import { ElCard, ElButton, ElEmpty, ElInput, ElAlert, ElPagination } from 'element-plus'
  import {
    fetchInfoAssetLocations,
    fetchInfoAssetLocationItems,
    fetchInfoAssetChildren
  } from '@/api/eve-info'
  import { useUserStore } from '@/store/modules/user'

  import { useI18n } from 'vue-i18n'
  const { t } = useI18n()
  defineOptions({ name: 'EveInfoAssets' })

  const userStore = useUserStore()

  const loading = ref(false)
  const errorMessage = ref('')
  const searchKeyword = ref('')

  // 位置摘要列表
  const locationsData = ref<Api.EveInfo.AssetLocationsResponse | null>(null)
  const expandedLocations = ref(new Set<number>())

  // 每个位置的物品缓存
  const locationItemsMap = reactive<Record<number, Api.EveInfo.AssetListItemNode[]>>({})
  const locationLoadingMap = reactive<Record<number, boolean>>({})
  const locationErrors = reactive<Record<number, string>>({})
  const locationPaginationMap = reactive<
    Record<number, { page: number; pageSize: number; total: number }>
  >({})

  // 子物品缓存
  const expandedItems = ref(new Set<number>())
  const childrenMap = reactive<Record<number, Api.EveInfo.AssetListItemNode[]>>({})
  const childrenLoadingMap = reactive<Record<number, boolean>>({})
  const childrenErrors = reactive<Record<number, string>>({})

  /** 蓝图拷贝 categoryID=9 */
  const CATEGORY_BLUEPRINT = 9

  const getItemIcon = (item: {
    category_id: number
    type_id: number
    is_blueprint_copy?: boolean
  }) => {
    if (item.category_id === CATEGORY_BLUEPRINT) {
      const suffix = item.is_blueprint_copy ? 'bpc' : 'bp'
      return `https://images.evetech.net/types/${item.type_id}/${suffix}?size=32`
    }
    return `https://images.evetech.net/types/${item.type_id}/icon?size=32`
  }

  const onSearchChange = () => {
    loadLocations()
  }

  /** 加载位置摘要 */
  const loadLocations = async () => {
    loading.value = true
    errorMessage.value = ''
    try {
      locationsData.value = await fetchInfoAssetLocations({
        language: userStore.language,
        page: 1,
        page_size: 20,
        keyword: searchKeyword.value || undefined
      })
      // 清除所有展开缓存
      expandedLocations.value = new Set()
      Object.keys(locationItemsMap).forEach((k) => delete locationItemsMap[Number(k)])
      Object.keys(locationLoadingMap).forEach((k) => delete locationLoadingMap[Number(k)])
      Object.keys(locationErrors).forEach((k) => delete locationErrors[Number(k)])
      Object.keys(locationPaginationMap).forEach((k) => delete locationPaginationMap[Number(k)])
      // 清除子物品缓存
      expandedItems.value = new Set()
      Object.keys(childrenMap).forEach((k) => delete childrenMap[Number(k)])
      Object.keys(childrenLoadingMap).forEach((k) => delete childrenLoadingMap[Number(k)])
      Object.keys(childrenErrors).forEach((k) => delete childrenErrors[Number(k)])
    } catch (err: any) {
      locationsData.value = null
      errorMessage.value = err?.message || t('info.assetLoadFailed')
    } finally {
      loading.value = false
    }
  }

  /** 展开/折叠位置 → 加载根物品 */
  const toggleLocation = async (loc: Api.EveInfo.AssetLocationSummary) => {
    if (expandedLocations.value.has(loc.location_id)) {
      expandedLocations.value.delete(loc.location_id)
      expandedLocations.value = new Set(expandedLocations.value)
      return
    }
    expandedLocations.value.add(loc.location_id)
    expandedLocations.value = new Set(expandedLocations.value)

    if (!locationItemsMap[loc.location_id]) {
      await loadLocationItems(loc, 1)
    }
  }

  /** 加载指定位置的根物品 */
  const loadLocationItems = async (loc: Api.EveInfo.AssetLocationSummary, page: number) => {
    locationLoadingMap[loc.location_id] = true
    locationErrors[loc.location_id] = ''
    try {
      const result = await fetchInfoAssetLocationItems({
        language: userStore.language,
        location_id: loc.location_id,
        page,
        page_size: 50
      })
      locationItemsMap[loc.location_id] = result.items
      locationPaginationMap[loc.location_id] = {
        page,
        pageSize: 50,
        total: result.total_root_items
      }
    } catch (err: any) {
      locationErrors[loc.location_id] = err?.message || t('info.assetLocationLoadFailed')
    } finally {
      locationLoadingMap[loc.location_id] = false
    }
  }

  /** 展开/折叠容器 → 加载子物品 */
  const toggleChildren = async (item: Api.EveInfo.AssetListItemNode) => {
    if (expandedItems.value.has(item.item_id)) {
      expandedItems.value.delete(item.item_id)
      expandedItems.value = new Set(expandedItems.value)
      return
    }
    expandedItems.value.add(item.item_id)
    expandedItems.value = new Set(expandedItems.value)

    if (!childrenMap[item.item_id]) {
      childrenLoadingMap[item.item_id] = true
      childrenErrors[item.item_id] = ''
      try {
        const result = await fetchInfoAssetChildren({
          language: userStore.language,
          parent_item_id: item.item_id
        })
        childrenMap[item.item_id] = result.items
      } catch (err: any) {
        childrenErrors[item.item_id] = err?.message || t('info.assetChildrenLoadFailed')
      } finally {
        childrenLoadingMap[item.item_id] = false
      }
    }
  }

  onMounted(() => {
    loadLocations()
  })
</script>

<style scoped>
  /* ===== 主体 ===== */
  .assets-main {
    flex: 1;
    display: flex;
    flex-direction: column;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-light);
    border-radius: 6px;
    padding: 16px;
    overflow: hidden;
  }

  /* ===== 错误 ===== */
  .asset-error {
    padding: 20px 0;
    text-align: center;
  }

  /* ===== 筛选栏 ===== */
  .filter-bar {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 12px;
    flex-wrap: wrap;
  }

  /* ===== 分组 ===== */
  .assets-groups {
    flex: 1;
    overflow-y: auto;
    scrollbar-width: thin;
    scrollbar-color: transparent transparent;
  }

  .assets-groups:hover {
    scrollbar-color: rgba(144, 147, 153, 0.4) transparent;
  }

  .assets-groups::-webkit-scrollbar {
    width: 4px;
  }

  .assets-groups::-webkit-scrollbar-thumb {
    background: transparent;
    border-radius: 2px;
    transition: background 0.2s;
  }

  .assets-groups:hover::-webkit-scrollbar-thumb {
    background: rgba(144, 147, 153, 0.4);
  }

  .location-section {
    margin-bottom: 8px;
  }

  .location-header {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 12px;
    background: var(--el-fill-color-light);
    border-radius: 4px;
    cursor: pointer;
    user-select: none;
    font-weight: 600;
    font-size: 14px;
  }

  .location-header:hover {
    background: var(--el-fill-color);
  }

  .mg-arrow {
    font-size: 10px;
    transition: transform 0.15s;
    color: var(--el-text-color-secondary);
  }

  .mg-arrow.expanded {
    transform: rotate(90deg);
  }

  .mg-arrow-placeholder {
    width: 10px;
    flex-shrink: 0;
  }

  .mg-title {
    flex: 1;
  }

  .mg-count {
    font-size: 12px;
    color: var(--el-text-color-secondary);
    font-weight: 400;
  }

  .mg-char-count {
    font-size: 11px;
    color: var(--el-text-color-disabled);
    font-weight: 400;
    margin-left: 8px;
  }

  .pagination-row {
    display: flex;
    justify-content: center;
    padding: 8px 0;
  }

  .section-error {
    padding: 4px 12px;
  }

  /* ===== 物品列表 ===== */
  .asset-items {
    padding: 4px 0;
  }

  .min-h-40 {
    min-height: 40px;
  }
  .min-h-30 {
    min-height: 30px;
  }

  .asset-item {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 5px 12px;
    border-radius: 4px;
    transition: background 0.15s;
  }

  .asset-item.clickable {
    cursor: pointer;
  }

  .asset-item:hover {
    background: var(--el-fill-color-light);
  }

  .asset-icon {
    width: 28px;
    height: 28px;
    border-radius: 3px;
    border: 1px solid var(--el-border-color-lighter);
    flex-shrink: 0;
  }

  .asset-info {
    flex: 1;
    min-width: 0;
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .asset-type-name {
    font-size: 13px;
    color: var(--el-text-color-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .asset-name-tag {
    font-size: 11px;
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
    padding: 0 5px;
    border-radius: 3px;
    white-space: nowrap;
    flex-shrink: 0;
  }

  .child-count-badge {
    font-size: 10px;
    color: var(--el-text-color-secondary);
    background: var(--el-fill-color);
    padding: 0 4px;
    border-radius: 3px;
    flex-shrink: 0;
  }

  .asset-group {
    font-size: 12px;
    color: var(--el-text-color-secondary);
    white-space: nowrap;
    width: 120px;
    text-align: right;
    flex-shrink: 0;
  }

  .asset-qty {
    font-size: 12px;
    color: var(--el-text-color-secondary);
    font-weight: 500;
    width: 50px;
    text-align: right;
    flex-shrink: 0;
  }

  .asset-owner {
    font-size: 12px;
    color: var(--el-text-color-regular);
    width: 100px;
    text-align: right;
    flex-shrink: 0;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .child-toggle {
    cursor: pointer;
    padding: 2px 4px;
  }

  /* ===== 子物品 ===== */
  .child-items {
    padding-left: 28px;
    border-left: 2px solid var(--el-border-color-lighter);
    margin-left: 24px;
  }

  .asset-item.child {
    padding-left: 8px;
  }
</style>
