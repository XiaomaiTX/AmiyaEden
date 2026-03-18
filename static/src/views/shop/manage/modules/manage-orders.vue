<!-- 订单管理面板 -->
<template>
  <ElCard class="art-table-card" shadow="never">
    <ArtTableHeader v-model:columns="columnChecks" :loading="loading" @refresh="refreshData">
      <template #left>
        <div class="flex items-center gap-4">
          <ElInput
            v-model="userIdFilter"
            placeholder="用户 ID"
            clearable
            style="width: 140px"
            @keyup.enter="handleSearch"
          />
          <ElSelect
            v-model="statusFilter"
            placeholder="订单状态"
            clearable
            style="width: 140px"
            @change="handleSearch"
          >
            <ElOption label="待审批" value="pending" />
            <ElOption label="已完成" value="completed" />
            <ElOption label="已拒绝" value="rejected" />
            <ElOption label="余额不足" value="insufficient_funds" />
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

  <!-- 审批备注对话框 -->
  <ElDialog
    v-model="reviewDialogVisible"
    :title="reviewAction === 'approve' ? '审批通过' : '拒绝订单'"
    width="400px"
    destroy-on-close
  >
    <ElForm label-width="80px">
      <ElFormItem label="订单号">
        <span class="font-medium">{{ reviewOrderNo }}</span>
      </ElFormItem>
      <ElFormItem label="审批备注">
        <ElInput v-model="reviewRemark" type="textarea" :rows="3" placeholder="审批备注（可选）" />
      </ElFormItem>
    </ElForm>
    <template #footer>
      <ElButton @click="reviewDialogVisible = false">取消</ElButton>
      <ElButton
        :type="reviewAction === 'approve' ? 'success' : 'danger'"
        :loading="reviewSubmitting"
        @click="submitReview"
      >
        {{ reviewAction === 'approve' ? '确认通过' : '确认拒绝' }}
      </ElButton>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import { ElTag, ElButton, ElInput, ElSelect, ElOption, ElMessage } from 'element-plus'
  import { adminListOrders, adminApproveOrder, adminRejectOrder } from '@/api/shop'
  import { useTable } from '@/hooks/core/useTable'

  defineOptions({ name: 'ManageOrders' })

  type Order = Api.Shop.Order

  // ─── 订单状态映射 ───
  const ORDER_STATUS_CONFIG: Record<string, { label: string; type: string }> = {
    pending: { label: '待审批', type: 'warning' },
    paid: { label: '已付款', type: 'success' },
    approved: { label: '已审批', type: 'success' },
    rejected: { label: '已拒绝', type: 'danger' },
    completed: { label: '已完成', type: 'success' },
    cancelled: { label: '已取消', type: 'info' },
    insufficient_funds: { label: '余额不足', type: 'danger' }
  }

  const formatISK = (v: number) =>
    v.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })

  const formatTime = (t: string) => new Date(t).toLocaleString()

  // ─── 搜索过滤状态 ───
  const userIdFilter = ref('')
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
      apiFn: adminListOrders,
      apiParams: { current: 1, size: 20 },
      immediate: false,
      columnsFactory: () => [
        {
          prop: 'order_no',
          label: '订单号',
          width: 200,
          showOverflowTooltip: true
        },
        {
          prop: 'user_id',
          label: '用户ID',
          width: 90
        },
        {
          prop: 'product_name',
          label: '商品',
          minWidth: 140,
          showOverflowTooltip: true
        },
        {
          prop: 'quantity',
          label: '数量',
          width: 70
        },
        {
          prop: 'total_price',
          label: '总价',
          width: 130,
          formatter: (row: Order) =>
            h('span', { class: 'font-medium text-orange-600' }, formatISK(row.total_price))
        },
        {
          prop: 'status',
          label: '状态',
          width: 120,
          formatter: (row: Order) => {
            const cfg = ORDER_STATUS_CONFIG[row.status] ?? { label: row.status, type: 'info' }
            return h(
              ElTag,
              { type: cfg.type as any, size: 'small', effect: 'plain' },
              () => cfg.label
            )
          }
        },
        {
          prop: 'remark',
          label: '用户备注',
          width: 140,
          showOverflowTooltip: true
        },
        {
          prop: 'review_remark',
          label: '审批备注',
          width: 140,
          showOverflowTooltip: true
        },
        {
          prop: 'created_at',
          label: '下单时间',
          width: 180,
          formatter: (row: Order) => h('span', {}, formatTime(row.created_at))
        },
        {
          prop: 'actions',
          label: '操作',
          width: 160,
          fixed: 'right',
          formatter: (row: Order) => {
            if (row.status !== 'pending') {
              return h('span', { class: 'text-gray-400 text-sm' }, '-')
            }
            return h('div', { class: 'flex gap-1' }, [
              h(
                ElButton,
                { size: 'small', type: 'success', onClick: () => openApproveDialog(row) },
                () => '通过'
              ),
              h(
                ElButton,
                { size: 'small', type: 'danger', onClick: () => openRejectDialog(row) },
                () => '拒绝'
              )
            ])
          }
        }
      ]
    }
  })

  function handleSearch() {
    Object.assign(searchParams, {
      status: statusFilter.value || undefined,
      user_id: userIdFilter.value ? Number(userIdFilter.value) : undefined,
      current: 1
    })
    getData()
  }

  function handleReset() {
    userIdFilter.value = ''
    statusFilter.value = ''
    resetSearchParams()
  }

  // ─── 审批对话框状态 ───
  const reviewDialogVisible = ref(false)
  const reviewAction = ref<'approve' | 'reject'>('approve')
  const reviewOrderId = ref(0)
  const reviewOrderNo = ref('')
  const reviewRemark = ref('')
  const reviewSubmitting = ref(false)

  function openApproveDialog(order: Order) {
    reviewAction.value = 'approve'
    reviewOrderId.value = order.id
    reviewOrderNo.value = order.order_no
    reviewRemark.value = ''
    reviewDialogVisible.value = true
  }

  function openRejectDialog(order: Order) {
    reviewAction.value = 'reject'
    reviewOrderId.value = order.id
    reviewOrderNo.value = order.order_no
    reviewRemark.value = ''
    reviewDialogVisible.value = true
  }

  async function submitReview() {
    reviewSubmitting.value = true
    try {
      const params: Api.Shop.OrderReviewParams = {
        order_id: reviewOrderId.value,
        remark: reviewRemark.value
      }
      if (reviewAction.value === 'approve') {
        await adminApproveOrder(params)
        ElMessage.success('审批通过')
      } else {
        await adminRejectOrder(params)
        ElMessage.success('已拒绝')
      }
      reviewDialogVisible.value = false
      refreshData()
    } catch (e: any) {
      ElMessage.error(e?.message ?? '操作失败')
    } finally {
      reviewSubmitting.value = false
    }
  }

  defineExpose({ load: getData, refresh: refreshData })
</script>
