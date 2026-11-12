<template>
  <p v-if="kind === 'alt'" class="alt-reminder">{{ t('admissions.altHint') }}</p>
  <I18nT v-else keypath="admissions.seatHint" tag="p" scope="global">
    <template #seat>
      <a href="https://seat.winterco.org" target="_blank" rel="noreferrer">{{
        t('admissions.seat')
      }}</a>
    </template>
  </I18nT>
</template>

<script setup lang="ts">
  import { I18nT, useI18n } from 'vue-i18n'
  import type { AdmissionKind } from '@/types/api/admissions'

  defineProps<{ kind: AdmissionKind }>()
  const { t } = useI18n()
</script>

<style scoped>
  a {
    color: var(--el-color-primary);
    text-decoration: underline;
  }
  .alt-reminder {
    color: var(--el-color-danger);
    font-weight: 600;
  }
</style>
