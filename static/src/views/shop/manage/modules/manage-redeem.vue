<!-- 兑换码管理面板 -->
<template>
  <ElCard class="art-table-card" shadow="never">
    <ArtTableHeader v-model:columns="columnChecks" :loading="loading" @refresh="refreshData">
      <template #left>
        <div class="flex items-center gap-4">
          <ElInput
            v-model="productIdFilter"
            placeholder="商品 ID"
            clearable
            style="min-width: 140px"
            @keyup.enter="handleSearch"
          />
          <ElSelect
            v-model="statusFilter"
            placeholder="状态"
            clearable
            style="min-width: 120px"
            @change="handleSearch"
          >
            <ElOption label="未使用" value="unused" />
            <ElOption label="已使用" value="used" />
            <ElOption label="已过期" value="expired" />
          </ElSelect>
          <ElButton type="primary" @click="handleSearch">查询</ElButton>
          <ElButton @click="handleReset">重置</ElButton>
        </div>
      </template>
    </ArtTableHeader>

    <ArtTable
      :loading="loading"
      :data="data"
      :columns="columns"
      :pagination="pagination"
      @pagination:size-change="handleSizeChange"
      @pagination:current-change="handleCurrentChange"
    />
  </ElCard>
</template>

<script setup lang="ts">
  import { ElTag, ElInput, ElSelect, ElOption, ElButton } from 'element-plus'
  import { adminListRedeemCodes } from '@/api/shop'
  import { useTable } from '@/hooks/core/useTable'

  defineOptions({ name: 'ManageRedeem' })

  type RedeemCode = Api.Shop.RedeemCode

  // ─── 兑换码状态映射 ───
  const REDEEM_STATUS_CONFIG: Record<string, { label: string; type: string }> = {
    unused: { label: '未使用', type: 'success' },
    used: { label: '已使用', type: 'info' },
    expired: { label: '已过期', type: 'danger' }
  }

  const formatTime = (v: string | null) => (v ? new Date(v).toLocaleString() : '-')

  // ─── 搜索过滤状态 ───
  const productIdFilter = ref('')
  const statusFilter = ref('')

  const {
    columns,
    columnChecks,
    data,
    loading,
    pagination,
    searchParams,
    getData,
    resetSearchParams,
    handleSizeChange,
    handleCurrentChange,
    refreshData
  } = useTable({
    core: {
      apiFn: adminListRedeemCodes,
      apiParams: { current: 1, size: 20 },
      immediate: false,
      columnsFactory: () => [
        { type: 'index', width: 60, label: '#' },
        {
          prop: 'product_id',
          label: '商品ID',
          minWidth: 90
        },
        {
          prop: 'user_id',
          label: '用户ID',
          minWidth: 90
        },
        {
          prop: 'order_id',
          label: '订单ID',
          minWidth: 90
        },
        {
          prop: 'code',
          label: '兑换码',
          minWidth: 220,
          showOverflowTooltip: true,
          formatter: (row: RedeemCode) => h('code', { class: 'text-sm font-mono' }, row.code)
        },
        {
          prop: 'status',
          label: '状态',
          minWidth: 100,
          formatter: (row: RedeemCode) => {
            const cfg = REDEEM_STATUS_CONFIG[row.status] ?? { label: row.status, type: 'info' }
            return h(
              ElTag,
              { type: cfg.type as any, size: 'small', effect: 'plain' },
              () => cfg.label
            )
          }
        },
        {
          prop: 'created_at',
          label: '创建时间',
          minWidth: 180,
          formatter: (row: RedeemCode) => h('span', {}, formatTime(row.created_at))
        },
        {
          prop: 'expires_at',
          label: '过期时间',
          minWidth: 180,
          formatter: (row: RedeemCode) => h('span', {}, formatTime(row.expires_at))
        }
      ]
    }
  })

  function handleSearch() {
    Object.assign(searchParams, {
      status: statusFilter.value || undefined,
      product_id: productIdFilter.value ? Number(productIdFilter.value) : undefined,
      current: 1
    })
    getData()
  }

  function handleReset() {
    productIdFilter.value = ''
    statusFilter.value = ''
    resetSearchParams()
  }

  defineExpose({ load: getData, refresh: refreshData })
</script>
