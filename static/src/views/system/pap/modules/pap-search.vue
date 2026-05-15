<template>
  <ArtSearchBar
    ref="searchBarRef"
    v-model="formData"
    :items="formItems"
    @reset="handleReset"
    @search="handleSearch"
  >
    <template #fetchBtn>
      <ElButton type="primary" :loading="fetching" :icon="Download" @click="handleFetch">
        {{ t('alliancePap.fetchLatest') }}
      </ElButton>
      <ElButton type="primary" :loading="fetching" :icon="Download" @click="openImportSEATDialog">
        {{ t('alliancePap.importBtnSEAT') }}
      </ElButton>
      <ArtExcelImport @import-success="handleImportXLS">
        {{ t('alliancePap.importBtnXLS') }}
      </ArtExcelImport>
    </template>
  </ArtSearchBar>

  <!-- 从 SEAT 导入弹窗 -->
  <ElDialog
    v-model="dialogVisible"
    :title="$t('alliancePap.importBtnSEAT')"
    width="600px"
    destroy-on-close
  >
    <ElForm ref="formRef" :model="formDataSEAT" :rules="formRulesSEAT" label-width="150px">
      <ElFormItem
        :label="$t('alliancePap.importFormSEAT.fields.laravelSession')"
        prop="laravelSession"
      >
        <ElInput
          v-model="formDataSEAT.laravelSession"
          :placeholder="$t('alliancePap.importFormSEAT.fields.laravelSession')"
        />
      </ElFormItem>
      <ElFormItem :label="$t('alliancePap.importFormSEAT.fields.cfClearance')" prop="cfClearance">
        <ElInput
          v-model="formDataSEAT.cfClearance"
          :placeholder="$t('alliancePap.importFormSEAT.fields.cfClearance')"
        />
      </ElFormItem>
      <ElFormItem :label="$t('alliancePap.importFormSEAT.fields.UA')" prop="UA">
        <ElInput
          v-model="formDataSEAT.UA"
          :placeholder="$t('alliancePap.importFormSEAT.fields.UA')"
        />
      </ElFormItem>
    </ElForm>
    <template #footer>
      <ElButton @click="dialogVisible = false">{{ $t('common.cancel') }}</ElButton>
      <ElButton type="primary" :loading="submitLoading" @click="handleImportSEAT">
        {{ $t('common.confirm') }}
      </ElButton>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import { Download } from '@element-plus/icons-vue'
  import { ElButton, type FormInstance, type FormRules } from 'element-plus'
  import { useI18n } from 'vue-i18n'
  import { fetchSeatPapTracking } from '@/api/alliance-pap'

  interface Props {
    modelValue: Record<string, unknown>
    fetching?: boolean
  }
  interface Emits {
    (e: 'update:modelValue', value: Record<string, unknown>): void
    (e: 'search', params: Record<string, unknown>): void
    (e: 'reset'): void
    (e: 'fetch'): void
    (e: 'import', rows: Record<string, unknown>[]): void
  }

  const props = withDefaults(defineProps<Props>(), { fetching: false })
  const emit = defineEmits<Emits>()
  const { t } = useI18n()

  const searchBarRef = ref()
  const formData = computed({
    get: () => props.modelValue,
    set: (val) => emit('update:modelValue', val)
  })

  const formItems = computed(() => [
    {
      label: t('alliancePap.selectMonth'),
      key: 'month',
      type: 'date',
      props: {
        type: 'month',
        format: 'YYYY-MM',
        valueFormat: 'YYYY-MM',
        clearable: false
      }
    },
    {
      key: 'fetchBtn',
      label: ''
    }
  ])

  function handleReset() {
    emit('reset')
  }

  async function handleSearch() {
    emit('search', formData.value)
  }

  function handleFetch() {
    emit('fetch')
  }

  function handleImportXLS(rows: Record<string, unknown>[]) {
    emit('import', rows)
  }

  // ─── 从 SEAT 导入 ───
  const dialogVisible = ref(false)
  const submitLoading = ref(false)
  const formRef = ref<FormInstance>()

  const formDataSEAT = reactive({
    laravelSession: '',
    cfClearance: '',
    UA: ''
  })

  const formRulesSEAT: FormRules = {
    laravelSession: [
      {
        required: true,
        message: t('alliancePap.importFormSEAT.fields.laravelSession'),
        trigger: 'blur'
      }
    ],
    cfClearance: [
      {
        required: false,
        message: t('alliancePap.importFormSEAT.fields.cfClearance'),
        trigger: 'blur'
      }
    ],
    UA: [{ required: false, message: t('alliancePap.importFormSEAT.fields.UA'), trigger: 'blur' }]
  }

  function resetFormDataSEAT() {
    formDataSEAT.laravelSession = ''
    formDataSEAT.cfClearance = ''
    formDataSEAT.UA = ''
  }

  function openImportSEATDialog() {
    resetFormDataSEAT()
    dialogVisible.value = true
  }

  async function handleImportSEAT() {
    if (!formRef.value) return
    await formRef.value.validate()

    const rows: Record<string, unknown>[] = []

    submitLoading.value = true
    try {
      const response = await fetchSeatPapTracking({
        laravelSession: formDataSEAT.laravelSession,
        cfClearance: formDataSEAT.cfClearance,
        userAgent: formDataSEAT.UA
      })

      if (response.status == 200 && response.data.data) {
        for (const item of response.data.data) {
          const temp: Record<string, unknown> = {
            主人物: item.character,
            '月 PAP': item.pap_count,
            数据时间: item.logoff_date
          }
          rows.push(temp)
        }
      } else {
        throw new Error(t('alliancePap.importFormSEAT.fetchPapError', { status: response.status }))
      }

      dialogVisible.value = false
      emit('import', rows)
    } catch (e: unknown) {
      ElMessage.error(e instanceof Error ? e.message : t('common.error'))
    } finally {
      submitLoading.value = false
    }
  }
</script>
