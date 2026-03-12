<!-- 商品管理面板 -->
<template>
  <ElCard class="art-table-card" shadow="never">
    <ArtTableHeader v-model:columns="columnChecks" :loading="loading" @refresh="refreshData">
      <template #left>
        <div class="flex items-center gap-2">
          <ElButton type="success" :icon="Plus" @click="openCreateDialog">
            {{ $t('shopAdmin.products.createProduct') }}
          </ElButton>
          <ElInput
            v-model="nameFilter"
            :placeholder="$t('shopAdmin.products.searchNamePlaceholder')"
            clearable
            style="width: 160px"
            @keyup.enter="handleSearch"
          />
          <ElSelect
            v-model="typeFilter"
            :placeholder="$t('shopAdmin.products.typePlaceholder')"
            clearable
            style="width: 140px"
            @change="handleSearch"
          >
            <ElOption :label="$t('shopAdmin.products.typeNormal')" value="normal" />
            <ElOption :label="$t('shopAdmin.products.typeRedeem')" value="redeem" />
          </ElSelect>
          <ElSelect
            v-model="statusFilter"
            :placeholder="$t('shopAdmin.products.statusPlaceholder')"
            clearable
            style="width: 120px"
            @change="handleSearch"
          >
            <ElOption :label="$t('shopAdmin.products.statusOnSale')" :value="1" />
            <ElOption :label="$t('shopAdmin.products.statusOffSale')" :value="0" />
          </ElSelect>
          <ElButton type="primary" @click="handleSearch">{{ $t('common.search') }}</ElButton>
          <ElButton @click="handleReset">{{ $t('common.reset') }}</ElButton>
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

  <!-- 商品编辑对话框 -->
  <ElDialog
    v-model="dialogVisible"
    :title="
      editingProduct ? $t('shopAdmin.products.dialogEdit') : $t('shopAdmin.products.dialogCreate')
    "
    width="560px"
    destroy-on-close
  >
    <ElForm ref="formRef" :model="formData" :rules="formRules" label-width="100px">
      <ElFormItem :label="$t('shopAdmin.products.fields.name')" prop="name">
        <ElInput
          v-model="formData.name"
          :placeholder="$t('shopAdmin.products.placeholders.name')"
        />
      </ElFormItem>
      <ElFormItem :label="$t('shopAdmin.products.fields.description')">
        <ElInput
          v-model="formData.description"
          type="textarea"
          :rows="3"
          :placeholder="$t('shopAdmin.products.placeholders.description')"
        />
      </ElFormItem>
      <ElFormItem :label="$t('shopAdmin.products.fields.image')">
        <ElInput
          v-model="formData.image"
          :placeholder="$t('shopAdmin.products.placeholders.image')"
        />
      </ElFormItem>
      <ElFormItem :label="$t('shopAdmin.products.fields.price')" prop="price">
        <ElInputNumber
          v-model="formData.price"
          :min="0.01"
          :precision="2"
          :step="10"
          style="width: 200px"
        />
      </ElFormItem>
      <ElFormItem :label="$t('shopAdmin.products.fields.type')" prop="type">
        <ElSelect v-model="formData.type" style="width: 200px">
          <ElOption :label="$t('shopAdmin.products.typeNormal')" value="normal" />
          <ElOption :label="$t('shopAdmin.products.typeRedeemService')" value="redeem" />
        </ElSelect>
      </ElFormItem>
      <ElFormItem :label="$t('shopAdmin.products.fields.stock')">
        <ElInputNumber v-model="formData.stock" :min="-1" style="width: 200px" />
        <span class="ml-2 text-xs text-gray-400">{{ $t('shopAdmin.products.hints.stock') }}</span>
      </ElFormItem>
      <ElFormItem :label="$t('shopAdmin.products.fields.maxPerUser')">
        <ElInputNumber v-model="formData.max_per_user" :min="0" style="width: 200px" />
        <span class="ml-2 text-xs text-gray-400">{{
          $t('shopAdmin.products.hints.maxPerUser')
        }}</span>
      </ElFormItem>
      <ElFormItem :label="$t('shopAdmin.products.fields.needApproval')">
        <ElSwitch v-model="formData.need_approval" />
      </ElFormItem>
      <ElFormItem :label="$t('shopAdmin.products.fields.status')">
        <ElSelect v-model="formData.status" style="width: 200px">
          <ElOption :label="$t('shopAdmin.products.statusOnSale')" :value="1" />
          <ElOption :label="$t('shopAdmin.products.statusOffSale')" :value="0" />
        </ElSelect>
      </ElFormItem>
      <ElFormItem :label="$t('shopAdmin.products.fields.sort')">
        <ElInputNumber v-model="formData.sort_order" :min="0" style="width: 200px" />
        <span class="ml-2 text-xs text-gray-400">{{ $t('shopAdmin.products.hints.sort') }}</span>
      </ElFormItem>
    </ElForm>
    <template #footer>
      <ElButton @click="dialogVisible = false">{{ $t('common.cancel') }}</ElButton>
      <ElButton type="primary" :loading="submitLoading" @click="handleSubmit">{{
        $t('common.confirm')
      }}</ElButton>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import {
    ElTag,
    ElButton,
    ElInput,
    ElSelect,
    ElOption,
    ElSwitch,
    ElMessage,
    ElMessageBox
  } from 'element-plus'
  import type { FormInstance, FormRules } from 'element-plus'
  import { Plus } from '@element-plus/icons-vue'
  import { useI18n } from 'vue-i18n'
  import ArtButtonTable from '@/components/core/forms/art-button-table/index.vue'
  import {
    adminListProducts,
    adminCreateProduct,
    adminUpdateProduct,
    adminDeleteProduct
  } from '@/api/shop'
  import { useTable } from '@/hooks/core/useTable'

  defineOptions({ name: 'ManageProducts' })
  const { t } = useI18n()

  type Product = Api.Shop.Product

  // ─── 商品类型/状态映射 ───
  const PRODUCT_TYPE_CONFIG: Record<string, { label: string; type: string }> = {
    normal: { label: t('shopAdmin.products.values.normal'), type: 'primary' },
    redeem: { label: t('shopAdmin.products.values.redeem'), type: 'warning' }
  }

  const PRODUCT_STATUS_CONFIG: Record<number, { label: string; type: string }> = {
    1: { label: t('shopAdmin.products.statusOnSale'), type: 'success' },
    0: { label: t('shopAdmin.products.statusOffSale'), type: 'danger' }
  }

  const formatISK = (v: number) =>
    v.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })

  // ─── 搜索过滤状态 ───
  const nameFilter = ref('')
  const typeFilter = ref('')
  const statusFilter = ref<number | undefined>(undefined)

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
      apiFn: adminListProducts,
      apiParams: { current: 1, size: 20 },
      immediate: false,
      columnsFactory: () => [
        { type: 'index', width: 60, label: '#' },
        {
          prop: 'name',
          label: t('shopAdmin.products.table.name'),
          minWidth: 160,
          showOverflowTooltip: true
        },
        {
          prop: 'type',
          label: t('common.type'),
          width: 100,
          formatter: (row: Product) => {
            const cfg = PRODUCT_TYPE_CONFIG[row.type] ?? { label: row.type, type: 'info' }
            return h(
              ElTag,
              { type: cfg.type as any, size: 'small', effect: 'plain' },
              () => cfg.label
            )
          }
        },
        {
          prop: 'price',
          label: t('shopAdmin.products.table.price'),
          width: 130,
          formatter: (row: Product) =>
            h('span', { class: 'font-medium text-orange-600' }, formatISK(row.price))
        },
        {
          prop: 'stock',
          label: t('shopAdmin.products.table.stock'),
          width: 90,
          formatter: (row: Product) => {
            if (row.stock < 0)
              return h('span', { class: 'text-gray-400' }, t('shopAdmin.products.values.unlimited'))
            return h(
              'span',
              { class: row.stock === 0 ? 'text-red-500 font-bold' : '' },
              String(row.stock)
            )
          }
        },
        {
          prop: 'max_per_user',
          label: t('shopAdmin.products.table.maxPerUser'),
          width: 80,
          formatter: (row: Product) =>
            h(
              'span',
              {},
              row.max_per_user > 0
                ? String(row.max_per_user)
                : t('shopAdmin.products.values.noLimit')
            )
        },
        {
          prop: 'need_approval',
          label: t('shopAdmin.products.table.needApproval'),
          width: 90,
          formatter: (row: Product) =>
            h(
              ElTag,
              { type: row.need_approval ? 'warning' : 'info', size: 'small', effect: 'plain' },
              () => (row.need_approval ? t('common.yes') : t('common.no'))
            )
        },
        {
          prop: 'status',
          label: t('common.status'),
          width: 90,
          formatter: (row: Product) => {
            const cfg = PRODUCT_STATUS_CONFIG[row.status] ?? {
              label: String(row.status),
              type: 'info'
            }
            return h(
              ElTag,
              { type: cfg.type as any, size: 'small', effect: 'plain' },
              () => cfg.label
            )
          }
        },
        {
          prop: 'sort_order',
          label: t('shopAdmin.products.fields.sort'),
          width: 80
        },
        {
          prop: 'actions',
          label: t('common.operation'),
          width: 120,
          fixed: 'right',
          formatter: (row: Product) =>
            h('div', { class: 'flex gap-1' }, [
              h(ArtButtonTable, { type: 'edit', onClick: () => openEditDialog(row) }),
              h(ArtButtonTable, { type: 'delete', onClick: () => handleDelete(row) })
            ])
        }
      ]
    }
  })

  function handleSearch() {
    Object.assign(searchParams, {
      name: nameFilter.value || undefined,
      type: typeFilter.value || undefined,
      status: statusFilter.value,
      current: 1
    })
    getData()
  }

  function handleReset() {
    nameFilter.value = ''
    typeFilter.value = ''
    statusFilter.value = undefined
    resetSearchParams()
  }

  // ─── 对话框状态 ───
  const dialogVisible = ref(false)
  const submitLoading = ref(false)
  const formRef = ref<FormInstance>()
  const editingProduct = ref<Product | null>(null)

  const formData = reactive({
    name: '',
    description: '',
    image: '',
    price: 0,
    type: 'normal' as 'normal' | 'redeem',
    stock: -1,
    max_per_user: 0,
    need_approval: false,
    status: 1 as number,
    sort_order: 0
  })

  const formRules: FormRules = {
    name: [
      { required: true, message: t('shopAdmin.products.messages.nameRequired'), trigger: 'blur' }
    ],
    price: [
      { required: true, message: t('shopAdmin.products.messages.priceRequired'), trigger: 'blur' }
    ],
    type: [
      { required: true, message: t('shopAdmin.products.messages.typeRequired'), trigger: 'change' }
    ]
  }

  function resetForm() {
    Object.assign(formData, {
      name: '',
      description: '',
      image: '',
      price: 0,
      type: 'normal',
      stock: -1,
      max_per_user: 0,
      need_approval: false,
      status: 1,
      sort_order: 0
    })
    editingProduct.value = null
  }

  function openCreateDialog() {
    resetForm()
    dialogVisible.value = true
  }

  function openEditDialog(row: Product) {
    editingProduct.value = row
    Object.assign(formData, {
      name: row.name,
      description: row.description,
      image: row.image,
      price: row.price,
      type: row.type,
      stock: row.stock,
      max_per_user: row.max_per_user,
      need_approval: row.need_approval,
      status: row.status,
      sort_order: row.sort_order
    })
    dialogVisible.value = true
  }

  async function handleSubmit() {
    if (!formRef.value) return
    await formRef.value.validate()
    submitLoading.value = true
    try {
      if (editingProduct.value) {
        await adminUpdateProduct({ id: editingProduct.value.id, ...formData })
        ElMessage.success(t('shopAdmin.products.messages.updateSuccess'))
      } else {
        await adminCreateProduct({ ...formData })
        ElMessage.success(t('shopAdmin.products.messages.createSuccess'))
      }
      dialogVisible.value = false
      refreshData()
    } catch (e: any) {
      ElMessage.error(e?.message ?? t('walletAdmin.messages.actionFailed'))
    } finally {
      submitLoading.value = false
    }
  }

  async function handleDelete(row: Product) {
    await ElMessageBox.confirm(
      t('shopAdmin.products.messages.deleteConfirm', { name: row.name }),
      t('common.tips'),
      {
        confirmButtonText: t('common.delete'),
        cancelButtonText: t('common.cancel'),
        type: 'warning'
      }
    )
    try {
      await adminDeleteProduct(row.id)
      ElMessage.success(t('shopAdmin.products.messages.deleteSuccess'))
      refreshData()
    } catch (e: any) {
      ElMessage.error(e?.message ?? t('shopAdmin.products.messages.deleteFailed'))
    }
  }

  defineExpose({ load: getData, refresh: refreshData })
</script>
