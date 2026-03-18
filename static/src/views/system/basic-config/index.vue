<!-- 系统管理 - 基础配置 -->
<template>
  <div class="basic-config-page">
    <ElCard shadow="never">
      <template #header>
        <h2 class="section-title">{{ $t('system.basicConfig.title') }}</h2>
      </template>

      <ElForm
        ref="formRef"
        :model="form"
        label-width="120px"
        style="max-width: 680px"
        v-loading="loadingConfig"
      >
        <ElFormItem :label="$t('system.basicConfig.corpId')" prop="corp_id">
          <ElInputNumber
            v-model="form.corp_id"
            :min="1"
            :controls="false"
            style="width: 220px"
            :placeholder="$t('system.basicConfig.corpIdPlaceholder')"
          />
        </ElFormItem>

        <ElFormItem :label="$t('system.basicConfig.siteTitle')" prop="site_title">
          <ElInput
            v-model="form.site_title"
            clearable
            :placeholder="$t('system.basicConfig.siteTitlePlaceholder')"
          />
        </ElFormItem>

        <ElFormItem>
          <ElButton type="primary" :loading="saving" @click="handleSave">
            {{ $t('system.basicConfig.save') }}
          </ElButton>
        </ElFormItem>
      </ElForm>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
  import { useI18n } from 'vue-i18n'
  import {
    ElCard,
    ElForm,
    ElFormItem,
    ElInput,
    ElInputNumber,
    ElButton,
    ElMessage
  } from 'element-plus'
  import { useSysConfigStore } from '@/store/modules/sys-config'

  defineOptions({ name: 'BasicConfig' })

  const { t } = useI18n()
  const sysConfigStore = useSysConfigStore()

  const loadingConfig = ref(false)
  const saving = ref(false)

  const form = reactive<Api.SysConfig.BasicConfig>({
    corp_id: sysConfigStore.config.corp_id,
    site_title: sysConfigStore.config.site_title
  })

  const loadConfig = async () => {
    loadingConfig.value = true
    try {
      await sysConfigStore.ensureLoaded()
      form.corp_id = sysConfigStore.config.corp_id
      form.site_title = sysConfigStore.config.site_title
    } catch {
      /* empty */
    } finally {
      loadingConfig.value = false
    }
  }

  const handleSave = async () => {
    saving.value = true
    try {
      await sysConfigStore.updateConfig({
        corp_id: form.corp_id,
        site_title: form.site_title
      })
      ElMessage.success(t('system.basicConfig.saveSuccess'))
    } catch {
      /* empty */
    } finally {
      saving.value = false
    }
  }

  onMounted(() => {
    loadConfig()
  })
</script>

<style scoped>
  .section-title {
    font-size: 15px;
    font-weight: 600;
    margin: 0;
  }
</style>
