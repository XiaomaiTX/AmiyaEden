<!-- ESI 自动权限映射管理 -->
<template>
  <div class="art-full-height">
    <!-- 页头操作区 -->
    <ElCard class="mb-4" shadow="never">
      <div class="flex items-center justify-between">
        <div>
          <div class="text-base font-semibold">ESI 自动权限</div>
          <div class="mt-1 text-sm text-gray-500">
            根据角色的 EVE 军团职位和头衔，自动分配系统权限。<br />
            Director 始终自动映射为 admin 角色。
          </div>
        </div>
        <ElButton type="primary" :loading="syncLoading" @click="handleTriggerSync">
          手动触发同步
        </ElButton>
      </div>
    </ElCard>

    <ElTabs v-model="activeTab" type="border-card">
      <!-- ─── Tab 1：军团职位映射 ─── -->
      <ElTabPane label="军团职位映射" name="esi-role">
        <div class="flex items-center justify-between mb-3">
          <span class="text-sm text-gray-500">
            配置 EVE 军团职位（Corporation Role）与系统权限的对应关系。
          </span>
          <ElButton type="primary" :icon="Plus" @click="openEsiRoleDialog"> 新增映射 </ElButton>
        </div>

        <ElTable v-loading="esiRoleLoading" :data="esiRoleMappings" border stripe>
          <ElTableColumn label="序号" type="index" width="60" />
          <ElTableColumn label="EVE 军团职位" prop="esi_role" min-width="200">
            <template #default="{ row }">
              <ElTag size="small" type="warning" effect="plain">{{ row.esi_role }}</ElTag>
            </template>
          </ElTableColumn>
          <ElTableColumn label="映射系统角色" min-width="200">
            <template #default="{ row }">
              <ElTag size="small" :type="getRoleTagType(row.role_code)" effect="dark">
                {{ row.role_name || row.role_code }}
              </ElTag>
              <span class="ml-1 text-xs text-gray-400">{{ row.role_code }}</span>
            </template>
          </ElTableColumn>
          <ElTableColumn label="创建时间" prop="created_at" width="180">
            <template #default="{ row }">
              {{ formatDate(row.created_at) }}
            </template>
          </ElTableColumn>
          <ElTableColumn label="操作" width="100" fixed="right">
            <template #default="{ row }">
              <ElPopconfirm
                title="确认删除此映射？"
                confirm-button-text="删除"
                cancel-button-text="取消"
                @confirm="handleDeleteEsiRole(row.id)"
              >
                <template #reference>
                  <ElButton size="small" type="danger" plain>删除</ElButton>
                </template>
              </ElPopconfirm>
            </template>
          </ElTableColumn>
        </ElTable>
      </ElTabPane>

      <!-- ─── Tab 2：头衔映射 ─── -->
      <ElTabPane label="军团头衔映射" name="title">
        <div class="flex items-center justify-between mb-3">
          <span class="text-sm text-gray-500">
            配置军团头衔（Corporation Title）与系统权限的对应关系，需指定军团 ID。
          </span>
          <ElButton type="primary" :icon="Plus" @click="openTitleDialog"> 新增映射 </ElButton>
        </div>

        <ElTable v-loading="titleLoading" :data="titleMappings" border stripe>
          <ElTableColumn label="序号" type="index" width="60" />
          <ElTableColumn label="军团 ID" prop="corporation_id" width="160" />
          <ElTableColumn label="头衔 ID" prop="title_id" width="100" />
          <ElTableColumn label="头衔名称" prop="title_name" min-width="180" show-overflow-tooltip>
            <template #default="{ row }">
              <span v-if="row.title_name" class="text-orange-400">{{ row.title_name }}</span>
              <span v-else class="text-gray-400">—</span>
            </template>
          </ElTableColumn>
          <ElTableColumn label="映射系统角色" min-width="180">
            <template #default="{ row }">
              <ElTag size="small" :type="getRoleTagType(row.role_code)" effect="dark">
                {{ row.role_name || row.role_code }}
              </ElTag>
              <span class="ml-1 text-xs text-gray-400">{{ row.role_code }}</span>
            </template>
          </ElTableColumn>
          <ElTableColumn label="创建时间" prop="created_at" width="180">
            <template #default="{ row }">
              {{ formatDate(row.created_at) }}
            </template>
          </ElTableColumn>
          <ElTableColumn label="操作" width="100" fixed="right">
            <template #default="{ row }">
              <ElPopconfirm
                title="确认删除此映射？"
                confirm-button-text="删除"
                cancel-button-text="取消"
                @confirm="handleDeleteTitle(row.id)"
              >
                <template #reference>
                  <ElButton size="small" type="danger" plain>删除</ElButton>
                </template>
              </ElPopconfirm>
            </template>
          </ElTableColumn>
        </ElTable>
      </ElTabPane>
    </ElTabs>

    <!-- ─── 新增 ESI 角色映射对话框 ─── -->
    <ElDialog
      v-model="esiRoleDialogVisible"
      title="新增军团职位映射"
      width="460px"
      destroy-on-close
    >
      <ElForm
        ref="esiRoleFormRef"
        :model="esiRoleForm"
        :rules="esiRoleFormRules"
        label-width="100px"
      >
        <ElFormItem label="EVE 职位" prop="esi_role">
          <ElSelect
            v-model="esiRoleForm.esi_role"
            placeholder="选择军团职位"
            filterable
            style="width: 100%"
          >
            <ElOption
              v-for="role in allEsiRoles"
              :key="role"
              :label="role"
              :value="role"
            />
          </ElSelect>
        </ElFormItem>
        <ElFormItem label="系统角色" prop="role_id">
          <ElSelect
            v-model="esiRoleForm.role_id"
            placeholder="选择系统角色"
            filterable
            style="width: 100%"
          >
            <ElOption
              v-for="role in allSystemRoles"
              :key="role.id"
              :label="role.name"
              :value="role.id"
            >
              <span>{{ role.name }}</span>
              <span class="ml-2 text-xs text-gray-400">{{ role.code }}</span>
            </ElOption>
          </ElSelect>
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="esiRoleDialogVisible = false">取消</ElButton>
        <ElButton type="primary" :loading="esiRoleSubmitting" @click="handleCreateEsiRole">
          确定
        </ElButton>
      </template>
    </ElDialog>

    <!-- ─── 新增头衔映射对话框 ─── -->
    <ElDialog
      v-model="titleDialogVisible"
      title="新增军团头衔映射"
      width="480px"
      destroy-on-close
    >
      <ElForm
        ref="titleFormRef"
        :model="titleForm"
        :rules="titleFormRules"
        label-width="100px"
      >
        <ElFormItem label="选择头衔" prop="title_key">
          <ElSelect
            v-model="titleForm.title_key"
            filterable
            placeholder="选择军团头衔"
            style="width: 100%"
            @change="onTitleKeyChange"
          >
            <ElOption
              v-for="t in allCorpTitles"
              :key="`${t.corporation_id}_${t.title_id}`"
              :label="t.title_name || `头衔 ${t.title_id}`"
              :value="`${t.corporation_id}_${t.title_id}`"
            >
              <div class="flex items-center justify-between">
                <span>{{ t.title_name || `头衔 ${t.title_id}` }}</span>
                <span class="ml-3 text-xs text-gray-400">Corp {{ t.corporation_id }}</span>
              </div>
            </ElOption>
          </ElSelect>
        </ElFormItem>
        <ElFormItem label="系统角色" prop="role_id">
          <ElSelect
            v-model="titleForm.role_id"
            placeholder="选择系统角色"
            filterable
            style="width: 100%"
          >
            <ElOption
              v-for="role in allSystemRoles"
              :key="role.id"
              :label="role.name"
              :value="role.id"
            >
              <span>{{ role.name }}</span>
              <span class="ml-2 text-xs text-gray-400">{{ role.code }}</span>
            </ElOption>
          </ElSelect>
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="titleDialogVisible = false">取消</ElButton>
        <ElButton type="primary" :loading="titleSubmitting" @click="handleCreateTitle">
          确定
        </ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { ref, onMounted } from 'vue'
  import { Plus } from '@element-plus/icons-vue'
  import type { FormInstance, FormRules } from 'element-plus'
  import {
    fetchGetEsiRoleMappings,
    fetchCreateEsiRoleMapping,
    fetchDeleteEsiRoleMapping,
    fetchGetEsiTitleMappings,
    fetchCreateEsiTitleMapping,
    fetchDeleteEsiTitleMapping,
    fetchGetAllEsiRoles,
    fetchGetCorpTitles,
    fetchGetAllRoles,
    fetchTriggerAutoRoleSync
  } from '@/api/system-manage'

  defineOptions({ name: 'AutoRole' })

  type EsiRoleMapping = Api.SystemManage.EsiRoleMapping
  type EsiTitleMapping = Api.SystemManage.EsiTitleMapping
  type RoleItem = Api.SystemManage.RoleItem
  type CorpTitleInfo = Api.SystemManage.CorpTitleInfo

  // ─── Tab ───
  const activeTab = ref('esi-role')

  // ─── 角色标签颜色 ───
  const CODE_TYPE: Record<string, string> = {
    super_admin: 'danger',
    admin: 'warning',
    srp: '',
    fc: '',
    user: 'success',
    guest: 'info'
  }
  function getRoleTagType(code: string) {
    return (CODE_TYPE[code] ?? '') as any
  }

  // ─── 日期格式化 ───
  function formatDate(dateStr: string) {
    if (!dateStr) return '—'
    return new Date(dateStr).toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit'
    })
  }

  // ─── 基础数据 ───
  const allEsiRoles = ref<string[]>([])
  const allSystemRoles = ref<RoleItem[]>([])
  const allCorpTitles = ref<CorpTitleInfo[]>([])

  async function loadBaseData() {
    const [esiRoles, systemRoles, corpTitles] = await Promise.all([
      fetchGetAllEsiRoles(),
      fetchGetAllRoles(),
      fetchGetCorpTitles()
    ])
    allEsiRoles.value = esiRoles
    // 过滤掉 super_admin（文档规定不可映射）
    allSystemRoles.value = systemRoles.filter((r) => r.code !== 'super_admin')
    allCorpTitles.value = corpTitles
  }

  // ─── ESI 角色映射 ───
  const esiRoleMappings = ref<EsiRoleMapping[]>([])
  const esiRoleLoading = ref(false)

  async function loadEsiRoleMappings() {
    esiRoleLoading.value = true
    try {
      esiRoleMappings.value = await fetchGetEsiRoleMappings()
    } finally {
      esiRoleLoading.value = false
    }
  }

  async function handleDeleteEsiRole(id: number) {
    try {
      await fetchDeleteEsiRoleMapping(id)
      ElMessage.success('删除成功')
      await loadEsiRoleMappings()
    } catch (e: any) {
      ElMessage.error(e?.message ?? '删除失败')
    }
  }

  // ─── 新增 ESI 角色映射 ───
  const esiRoleDialogVisible = ref(false)
  const esiRoleSubmitting = ref(false)
  const esiRoleFormRef = ref<FormInstance>()
  const esiRoleForm = reactive({
    esi_role: '',
    role_id: undefined as number | undefined
  })
  const esiRoleFormRules: FormRules = {
    esi_role: [{ required: true, message: '请选择 EVE 军团职位', trigger: 'change' }],
    role_id: [{ required: true, message: '请选择系统角色', trigger: 'change' }]
  }

  function openEsiRoleDialog() {
    esiRoleForm.esi_role = ''
    esiRoleForm.role_id = undefined
    esiRoleDialogVisible.value = true
  }

  async function handleCreateEsiRole() {
    if (!esiRoleFormRef.value) return
    await esiRoleFormRef.value.validate()
    esiRoleSubmitting.value = true
    try {
      await fetchCreateEsiRoleMapping({
        esi_role: esiRoleForm.esi_role,
        role_id: esiRoleForm.role_id!
      })
      ElMessage.success('映射创建成功')
      esiRoleDialogVisible.value = false
      await loadEsiRoleMappings()
    } catch (e: any) {
      ElMessage.error(e?.message ?? '创建失败')
    } finally {
      esiRoleSubmitting.value = false
    }
  }

  // ─── 头衔映射 ───
  const titleMappings = ref<EsiTitleMapping[]>([])
  const titleLoading = ref(false)

  async function loadTitleMappings() {
    titleLoading.value = true
    try {
      titleMappings.value = await fetchGetEsiTitleMappings()
    } finally {
      titleLoading.value = false
    }
  }

  async function handleDeleteTitle(id: number) {
    try {
      await fetchDeleteEsiTitleMapping(id)
      ElMessage.success('删除成功')
      await loadTitleMappings()
    } catch (e: any) {
      ElMessage.error(e?.message ?? '删除失败')
    }
  }

  // ─── 新增头衔映射 ───
  const titleDialogVisible = ref(false)
  const titleSubmitting = ref(false)
  const titleFormRef = ref<FormInstance>()
  const titleForm = reactive({
    title_key: '',
    corporation_id: 0,
    title_id: 0,
    title_name: '',
    role_id: undefined as number | undefined
  })
  const titleFormRules: FormRules = {
    title_key: [{ required: true, message: '请选择头衔', trigger: 'change' }],
    role_id: [{ required: true, message: '请选择系统角色', trigger: 'change' }]
  }

  function onTitleKeyChange(key: string) {
    const t = allCorpTitles.value.find((t) => `${t.corporation_id}_${t.title_id}` === key)
    if (t) {
      titleForm.corporation_id = t.corporation_id
      titleForm.title_id = t.title_id
      titleForm.title_name = t.title_name
    }
  }

  function openTitleDialog() {
    titleForm.title_key = ''
    titleForm.corporation_id = 0
    titleForm.title_id = 0
    titleForm.title_name = ''
    titleForm.role_id = undefined
    titleDialogVisible.value = true
  }

  async function handleCreateTitle() {
    if (!titleFormRef.value) return
    await titleFormRef.value.validate()
    titleSubmitting.value = true
    try {
      await fetchCreateEsiTitleMapping({
        corporation_id: titleForm.corporation_id,
        title_id: titleForm.title_id,
        title_name: titleForm.title_name || undefined,
        role_id: titleForm.role_id!
      })
      ElMessage.success('映射创建成功')
      titleDialogVisible.value = false
      await loadTitleMappings()
    } catch (e: any) {
      ElMessage.error(e?.message ?? '创建失败')
    } finally {
      titleSubmitting.value = false
    }
  }

  // ─── 手动触发同步 ───
  const syncLoading = ref(false)

  async function handleTriggerSync() {
    syncLoading.value = true
    try {
      await fetchTriggerAutoRoleSync()
      ElMessage.success('同步任务已触发，将在后台执行')
    } catch (e: any) {
      ElMessage.error(e?.message ?? '触发失败')
    } finally {
      syncLoading.value = false
    }
  }

  // ─── 初始化 ───
  onMounted(() => {
    Promise.all([loadBaseData(), loadEsiRoleMappings(), loadTitleMappings()])
  })
</script>
