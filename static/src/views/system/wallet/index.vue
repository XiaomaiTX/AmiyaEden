<!-- 系统钱包管理页面 -->
<template>
  <div class="wallet-admin-page art-full-height">
    <ElCard class="art-table-card" shadow="never">
      <ElTabs v-model="activeTab">
        <!-- 钱包列表 -->
        <ElTabPane label="钱包列表" name="wallets">
          <WalletList
            ref="walletListRef"
            @adjust="handleAdjust"
            @view-transactions="handleViewTransactions"
          />
        </ElTabPane>

        <!-- 流水查询 -->
        <ElTabPane label="流水查询" name="transactions">
          <WalletTransactions ref="walletTxRef" />
        </ElTabPane>

        <!-- 操作日志 -->
        <ElTabPane label="操作日志" name="logs">
          <WalletLogs />
        </ElTabPane>
      </ElTabs>
    </ElCard>

    <!-- 调整余额弹窗 -->
    <ElDialog v-model="adjustDialogVisible" title="调整用户钱包" width="480px" destroy-on-close>
      <ElForm ref="adjustFormRef" :model="adjustForm" :rules="adjustRules" label-width="100px">
        <ElFormItem label="目标用户 ID" prop="target_uid">
          <ElInputNumber
            v-model="adjustForm.target_uid"
            :min="1"
            :controls="false"
            style="width: 100%"
          />
        </ElFormItem>
        <ElFormItem label="操作类型" prop="action">
          <ElRadioGroup v-model="adjustForm.action">
            <ElRadio value="add">增加</ElRadio>
            <ElRadio value="deduct">扣减</ElRadio>
            <ElRadio value="set">设为</ElRadio>
          </ElRadioGroup>
        </ElFormItem>
        <ElFormItem label="金额" prop="amount">
          <ElInputNumber
            v-model="adjustForm.amount"
            :min="0.01"
            :precision="2"
            :controls="false"
            style="width: 100%"
          />
        </ElFormItem>
        <ElFormItem label="操作原因" prop="reason">
          <ElInput
            v-model="adjustForm.reason"
            type="textarea"
            :rows="3"
            placeholder="请说明操作原因（必填）"
          />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="adjustDialogVisible = false">取消</ElButton>
        <ElButton type="primary" :loading="adjustLoading" @click="submitAdjust">确认</ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import {
    ElCard,
    ElTabs,
    ElTabPane,
    ElButton,
    ElDialog,
    ElForm,
    ElFormItem,
    ElInput,
    ElInputNumber,
    ElRadioGroup,
    ElRadio,
    ElMessage,
    type FormInstance,
    type FormRules
  } from 'element-plus'
  import { adminAdjustWallet } from '@/api/sys-wallet'
  import WalletList from './modules/wallet-list.vue'
  import WalletTransactions from './modules/wallet-transactions.vue'
  import WalletLogs from './modules/wallet-logs.vue'

  defineOptions({ name: 'SystemWallet' })

  // ── Tab ──
  const activeTab = ref('wallets')

  // ── 子模块 refs ──
  const walletListRef = ref<InstanceType<typeof WalletList>>()
  const walletTxRef = ref<InstanceType<typeof WalletTransactions>>()

  // ── 来自钱包列表的事件 ──
  const handleViewTransactions = (userId: number) => {
    activeTab.value = 'transactions'
    nextTick(() => walletTxRef.value?.filterByUser(userId))
  }

  const handleAdjust = (userId: number, action: 'add' | 'deduct' | 'set') => {
    showAdjustDialog(userId, action)
  }

  // ══════════════════════════════════════════
  //  调整余额弹窗
  // ══════════════════════════════════════════
  const adjustDialogVisible = ref(false)
  const adjustLoading = ref(false)
  const adjustFormRef = ref<FormInstance>()

  const adjustForm = reactive<Api.SysWallet.AdjustParams>({
    target_uid: 0,
    action: 'add',
    amount: 0,
    reason: ''
  })

  const adjustRules: FormRules = {
    target_uid: [{ required: true, message: '请输入目标用户 ID', trigger: 'blur' }],
    action: [{ required: true, message: '请选择操作类型', trigger: 'change' }],
    amount: [{ required: true, message: '请输入金额', trigger: 'blur' }],
    reason: [{ required: true, message: '请输入操作原因', trigger: 'blur' }]
  }

  const showAdjustDialog = (userId = 0, action: 'add' | 'deduct' | 'set' = 'add') => {
    adjustForm.target_uid = userId
    adjustForm.action = action
    adjustForm.amount = 0
    adjustForm.reason = ''
    adjustDialogVisible.value = true
  }

  const submitAdjust = async () => {
    if (!adjustFormRef.value) return
    await adjustFormRef.value.validate()

    adjustLoading.value = true
    try {
      await adminAdjustWallet(adjustForm)
      ElMessage.success('余额调整成功')
      adjustDialogVisible.value = false
      walletListRef.value?.refreshData()
    } catch (e: any) {
      ElMessage.error(e?.message ?? '操作失败')
    } finally {
      adjustLoading.value = false
    }
  }
</script>
