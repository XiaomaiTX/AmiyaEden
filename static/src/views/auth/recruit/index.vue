<template>
  <div class="recruit-landing">
    <ElCard shadow="never" class="recruit-landing__card">
      <div class="recruit-landing__hero">
        <p class="recruit-landing__eyebrow">{{ t('newbro.recruitLink.title') }}</p>
        <h1 class="recruit-landing__title">{{ t('recruit.title') }}</h1>
        <p class="recruit-landing__subtitle">{{ t('recruit.subtitle') }}</p>
      </div>

      <template v-if="!submitted">
        <ElForm ref="formRef" :model="form" :rules="rules" label-position="top">
          <ElFormItem :label="t('recruit.qqLabel')" prop="qq">
            <ElInput
              v-model="form.qq"
              :placeholder="t('recruit.qqPlaceholder')"
              maxlength="20"
              :disabled="loading"
              @keyup.enter="handleSubmit"
            />
          </ElFormItem>

          <ElButton
            type="primary"
            :loading="loading"
            class="recruit-landing__submit"
            @click="handleSubmit"
          >
            {{ t('recruit.submitBtn') }}
          </ElButton>
        </ElForm>

        <ElAlert
          v-if="submitError"
          :title="submitError"
          type="error"
          show-icon
          :closable="false"
          class="mt-4"
        />
      </template>

      <ElResult
        v-else
        icon="success"
        :title="t('recruit.successTitle')"
        :sub-title="t('recruit.successSubtitle')"
      >
        <template #extra>
          <ElButton type="primary" @click="handleRedirect">{{ t('recruit.goToQQ') }}</ElButton>
        </template>
      </ElResult>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
  import { computed, reactive, ref } from 'vue'
  import { useRoute } from 'vue-router'
  import { useI18n } from 'vue-i18n'
  import {
    ElAlert,
    ElButton,
    ElCard,
    ElForm,
    ElFormItem,
    ElInput,
    ElResult,
    type FormInstance,
    type FormRules
  } from 'element-plus'
  import { submitRecruitQQ } from '@/api/newbro'

  defineOptions({ name: 'RecruitLanding' })

  const { t } = useI18n()
  const route = useRoute()
  const code = computed(() => String(route.params.code ?? ''))

  const formRef = ref<FormInstance>()
  const loading = ref(false)
  const submitted = ref(false)
  const submitError = ref('')
  const qqUrl = ref('')

  const form = reactive<Api.Newbro.SubmitQQRequest>({
    qq: ''
  })

  const rules: FormRules<Api.Newbro.SubmitQQRequest> = {
    qq: [
      {
        required: true,
        message: t('recruit.qqRequired'),
        trigger: 'blur'
      },
      {
        pattern: /^\d{5,20}$/,
        message: t('recruit.qqInvalid'),
        trigger: 'blur'
      }
    ]
  }

  const handleSubmit = async () => {
    const valid = await formRef.value?.validate().catch(() => false)
    if (!valid) {
      return
    }

    loading.value = true
    submitError.value = ''
    try {
      const data = await submitRecruitQQ(code.value, { qq: form.qq })
      qqUrl.value = data.qq_url
      submitted.value = true
    } catch (error) {
      submitError.value = (error as Error)?.message || t('recruit.submitError')
    } finally {
      loading.value = false
    }
  }

  const handleRedirect = () => {
    if (!qqUrl.value) {
      return
    }
    window.open(qqUrl.value, '_blank', 'noopener,noreferrer')
  }
</script>

<style scoped>
  .recruit-landing {
    min-height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 24px;
    background:
      radial-gradient(circle at top, var(--el-color-primary-light-8), transparent 42%),
      linear-gradient(180deg, var(--el-fill-color-lighter), var(--el-bg-color-page));
  }

  .recruit-landing__card {
    width: min(100%, 520px);
    border: 1px solid var(--el-border-color-light);
    background: var(--el-bg-color);
    box-shadow: 0 18px 50px rgba(15, 23, 42, 0.08);
  }

  .recruit-landing__hero {
    margin-bottom: 20px;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .recruit-landing__eyebrow {
    font-size: 12px;
    font-weight: 700;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--el-color-primary);
  }

  .recruit-landing__title {
    margin: 0;
    font-size: 30px;
    line-height: 1.1;
    color: var(--el-text-color-primary);
  }

  .recruit-landing__subtitle {
    margin: 0;
    font-size: 14px;
    line-height: 1.6;
    color: var(--el-text-color-secondary);
  }

  .recruit-landing__submit {
    width: 100%;
  }

  @media (max-width: 640px) {
    .recruit-landing {
      padding: 16px;
    }

    .recruit-landing__title {
      font-size: 26px;
    }
  }
</style>
