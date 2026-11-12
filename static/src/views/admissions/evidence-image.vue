<template>
  <p v-if="error" role="alert">{{ t(error) }}</p>
  <ElImage
    v-else-if="url"
    :src="url"
    :alt="t(`admissions.${kind}`)"
    :preview-src-list="[url]"
    fit="contain"
    class="evidence-image"
  />
  <p v-else>{{ t('admissions.loading') }}</p>
</template>
<script setup lang="ts">
  import { ref, watch, onBeforeUnmount } from 'vue'
  import { useI18n } from 'vue-i18n'
  import { getAdmissionEvidence } from '@/api/admissions'
  import type { EvidenceKind } from '@/types/api/admissions'
  const props = defineProps<{ id?: number; kind: EvidenceKind; blob?: Blob }>()
  const { t } = useI18n()
  const url = ref('')
  const error = ref('')
  let generation = 0
  function release() {
    if (url.value) URL.revokeObjectURL(url.value)
    url.value = ''
  }
  watch(
    () => [props.id, props.kind, props.blob] as const,
    async () => {
      const current = ++generation
      release()
      error.value = ''
      try {
        const blob =
          props.blob ?? (props.id ? await getAdmissionEvidence(props.id, props.kind) : null)
        if (current === generation && blob) url.value = URL.createObjectURL(blob)
      } catch {
        if (current === generation) error.value = 'admissions.errors.evidence'
      }
    },
    { immediate: true }
  )
  onBeforeUnmount(() => {
    generation++
    release()
  })
</script>
<style scoped>
  .evidence-image {
    display: block;
    width: 100%;
    max-height: 420px;
  }
</style>
