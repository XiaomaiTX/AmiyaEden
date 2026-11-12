<template>
  <p v-if="!eligible" role="alert">{{ t('admissions.denied') }}</p>
  <form v-else class="admission-form" @submit.prevent="submit(mustResubmit)">
    <p
      >{{ t('admissions.bindHint') }}
      <RouterLink to="/characters">{{ t('admissions.bind') }}</RouterLink></p
    >
    <AdmissionInstructions :kind="kind" />
    <div v-if="kind === 'alt'"
      ><h3>{{ t('admissions.contacts') }}</h3
      ><p
        ><span v-if="context.qq">QQ: {{ context.qq }}</span>
        <span v-if="context.discord_id">Discord: {{ context.discord_id }}</span></p
      ></div
    >
    <label class="target-field"
      ><span>{{ t('admissions.target') }}</span
      ><ElSelect
        v-model="target"
        :aria-label="t('admissions.target')"
        :disabled="busy"
        @change="selected = []"
        ><ElOption
          v-for="item in context.targets"
          :key="item.corporation_id"
          :value="item.corporation_id"
          :label="t(`admissions.corporations.${item.corporation_id}`)" /></ElSelect
    ></label>
    <fieldset
      ><legend>{{ t('admissions.characters') }}</legend
      ><ElCheckboxGroup v-model="selected" class="character-list"
        ><ElCheckbox
          v-for="char in context.characters"
          :key="char.character_id"
          :value="char.character_id"
          :label="char.character_id"
          :disabled="busy || char.corporation_id === target"
          border
          ><span
            >{{ char.character_name }} <small>{{ char.character_id }}</small></span
          ></ElCheckbox
        ></ElCheckboxGroup
      ></fieldset
    >
    <template v-if="kind === 'new_member'">
      <div class="evidence-grid"
        ><label v-for="item in evidenceKinds" :key="item" class="upload-field"
          ><span
            ><span class="required-marker" aria-hidden="true">*</span>
            {{ t(`admissions.${item}`) }} ({{ t('admissions.required') }})</span
          >
          <input
            type="file"
            accept="image/jpeg,image/png"
            :aria-label="t(`admissions.${item}`)"
            aria-required="true"
            :disabled="busy"
            @change="choose(item, $event)" /><EvidenceImage
            v-if="files[item]"
            :kind="item"
            :blob="files[item]" /></label
      ></div>
      <details
        ><summary>{{ t('admissions.qqGuide') }} / {{ t('admissions.doneGuide') }}</summary
        ><div class="evidence-grid"
          ><figure v-for="guide in guides" :key="guide.file"
            ><figcaption>{{ t(`admissions.${guide.label}`) }}</figcaption
            ><a :href="`/admissions/${guide.file}`" target="_blank" rel="noreferrer"
              ><img
                :src="`/admissions/${guide.file}`"
                :alt="t(`admissions.${guide.label}`)" /></a></figure></div
      ></details>
    </template>
    <p v-if="error" role="alert" class="error">{{ t(error) }}</p>
    <div class="actions"
      ><ElButton
        type="primary"
        native-type="submit"
        :loading="busy"
        :disabled="busy || !selected.length"
        >{{
          t(
            mustResubmit
              ? 'admissions.resubmit'
              : application
                ? 'admissions.save'
                : 'admissions.submit'
          )
        }}</ElButton
      ><ElButton
        v-if="application?.status === 'rejected' && !mustResubmit"
        :disabled="busy || !selected.length"
        @click="submit(true)"
        >{{ t('admissions.resubmit') }}</ElButton
      ></div
    >
  </form>
</template>
<script setup lang="ts">
  import { ref, computed } from 'vue'
  import { useI18n } from 'vue-i18n'
  import { saveAdmission } from '@/api/admissions'
  import { canAccessAdmission, compressAdmissionImage } from '@/utils/admissions'
  import type {
    AdmissionApplication,
    AdmissionContext,
    AdmissionKind,
    EvidenceKind
  } from '@/types/api/admissions'
  import EvidenceImage from './evidence-image.vue'
  import AdmissionInstructions from './admission-instructions.vue'
  const props = defineProps<{
    context: AdmissionContext
    kind: AdmissionKind
    application?: AdmissionApplication
  }>()
  const emit = defineEmits<{ saved: [id: number] }>()
  const { t } = useI18n()
  const target = ref(props.application?.target_corporation_id ?? 98824507)
  const selected = ref(props.application?.characters.map((char) => char.character_id) ?? [])
  const files = ref<Partial<Record<EvidenceKind, Blob>>>({})
  const busy = ref(false)
  const error = ref('')
  const eligible = computed(() => canAccessAdmission(props.context, props.kind))
  const mustResubmit = computed(
    () =>
      props.kind === 'new_member' &&
      props.application?.status === 'rejected' &&
      !!props.application.evidence_expires_at &&
      Date.parse(props.application.evidence_expires_at) <= Date.now()
  )
  const evidenceKinds: EvidenceKind[] = ['launcher', 'seat_profile']
  const guides = [
    { file: 'qq-binding.png', label: 'qqGuide' },
    { file: 'seat-completed.png', label: 'doneGuide' }
  ]
  function errorKey(value: unknown) {
    return value instanceof Error && value.message.startsWith('admissions.errors.')
      ? value.message
      : 'admissions.errors.internal'
  }
  async function choose(kind: EvidenceKind, event: Event) {
    const input = event.target as HTMLInputElement
    const file = input.files?.[0]
    input.value = ''
    if (!file) return
    busy.value = true
    error.value = ''
    try {
      files.value[kind] = await compressAdmissionImage(file)
    } catch (value) {
      error.value = errorKey(value)
    } finally {
      busy.value = false
    }
  }
  async function submit(resubmit: boolean) {
    if (busy.value) return
    busy.value = true
    error.value = ''
    try {
      const result = await saveAdmission(
        {
          kind: props.kind,
          target_corporation_id: target.value,
          character_ids: selected.value,
          expected_version: props.application?.version ?? 0
        },
        files.value,
        props.application?.id,
        resubmit
      )
      emit('saved', result.id)
    } catch (value) {
      error.value = errorKey(value)
    } finally {
      busy.value = false
    }
  }
</script>
<style scoped>
  .admission-form {
    display: grid;
    gap: 24px;
  }
  .target-field,
  .upload-field {
    display: grid;
    gap: 10px;
  }
  h3,
  legend {
    font-weight: 600;
  }
  fieldset {
    min-width: 0;
  }
  .character-list {
    display: grid;
    gap: 10px;
    margin-top: 12px;
  }
  .character-list :deep(.el-checkbox) {
    margin: 0;
    padding: 18px 12px;
  }
  small {
    margin-left: 12px;
    color: var(--el-text-color-secondary);
  }
  .required-marker,
  .error {
    color: var(--el-color-danger);
    font-weight: 600;
  }
  .evidence-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 20px;
  }
  .upload-field,
  details {
    padding: 16px;
    border: 1px solid var(--el-border-color);
    border-radius: 6px;
  }
  summary {
    cursor: pointer;
  }
  details .evidence-grid {
    margin-top: 16px;
  }
  img {
    max-width: 100%;
    max-height: 512px;
    object-fit: contain;
    object-position: left;
  }
  figcaption {
    margin-bottom: 10px;
  }
  .actions {
    display: flex;
    flex-wrap: wrap;
    gap: 12px;
  }
  a {
    color: var(--el-color-primary);
  }
  @media (max-width: 720px) {
    .evidence-grid {
      grid-template-columns: minmax(0, 1fr);
    }
  }
</style>
