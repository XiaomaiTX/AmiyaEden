<template>
  <div class="character-detail">
    <div v-if="characters.length === 0">
      <ElEmpty :description="$t('info.esiCheckNoCharacters')" />
    </div>
    <template v-else>
      <div class="flex items-center gap-4 mb-4 flex-wrap">
        <ElSelect
          :model-value="selectedCharacterId"
          :placeholder="$t('info.esiCheckSelectChar')"
          style="width: 240px"
          @update:model-value="$emit('update:selected-character-id', $event)"
        >
          <ElOption
            v-for="char in characters"
            :key="char.character_id"
            :value="char.character_id"
            :label="char.character_name"
          >
            <div class="flex items-center gap-2">
              <ElAvatar :src="buildEveCharacterPortraitUrl(char.character_id, 24)" :size="24" />
              <span>{{ char.character_name }}</span>
            </div>
          </ElOption>
        </ElSelect>
        <span v-if="selectedCharacter" class="text-sm text-gray-500">
          {{ formatCoverage(selectedCharacter) }}
        </span>
        <ElButton type="primary" plain :loading="reauthLoading" @click="handleReauth">
          {{ $t('info.esiCheckReauth') }}
        </ElButton>
      </div>

      <ElAlert
        v-if="selectedCharacter?.token_invalid"
        :title="$t('info.esiCheckTokenInvalid')"
        :description="$t('info.esiCheckTokenInvalidTip')"
        type="error"
        show-icon
        :closable="false"
        class="mb-4"
      />

      <div v-if="selectedCharacter && hasMissingRequiredScopes" class="mb-4">
        <ElAlert :title="$t('info.esiCheckReauthTip')" type="warning" show-icon :closable="false" />
      </div>

      <ElTable v-if="selectedCharacter" :data="scopeRows" stripe size="small">
        <ElTableColumn
          :label="$t('info.esiCheckScope')"
          prop="scope"
          min-width="260"
          show-overflow-tooltip
        />
        <ElTableColumn
          :label="$t('info.esiCheckDescription')"
          prop="description"
          min-width="180"
          show-overflow-tooltip
        />
        <ElTableColumn :label="$t('info.esiCheckModule')" prop="module" width="120" />
        <ElTableColumn :label="$t('info.esiCheckRequired')" width="80" align="center">
          <template #default="{ row }">
            <ElTag v-if="row.required" size="small" type="danger" effect="plain">
              {{ $t('info.esiCheckRequired') }}
            </ElTag>
            <ElTag v-else size="small" type="info" effect="plain">
              {{ $t('info.esiCheckOptional') }}
            </ElTag>
          </template>
        </ElTableColumn>
        <ElTableColumn :label="$t('info.esiCheckAuthorized')" width="100" align="center">
          <template #default="{ row }">
            <span v-if="row.authorized" class="text-green-500 font-bold">&#10003;</span>
            <span v-else class="text-red-400 font-bold">&#10007;</span>
          </template>
        </ElTableColumn>
      </ElTable>
    </template>
  </div>
</template>

<script setup lang="ts">
  import { buildEveCharacterPortraitUrl } from '@/utils/eve-image'
  import { getEveBindURL } from '@/api/auth'
  import { ElMessage } from 'element-plus'
  import { useI18n } from 'vue-i18n'

  defineOptions({ name: 'CharacterDetail' })

  const props = defineProps<{
    scopes: Api.Auth.RegisteredScope[]
    characters: Api.Auth.EveCharacter[]
    selectedCharacterId?: number
  }>()

  defineEmits<{
    (e: 'update:selected-character-id', characterId: number): void
  }>()

  const { t } = useI18n()

  const reauthLoading = ref(false)

  const handleReauth = async () => {
    reauthLoading.value = true
    try {
      const url = await getEveBindURL()
      window.location.href = url
    } catch {
      reauthLoading.value = false
      ElMessage.error(t('info.esiCheckReauthFailed'))
    }
  }

  const selectedCharacter = computed(
    () => props.characters.find((c) => c.character_id === props.selectedCharacterId) ?? null
  )

  const parseScopeSet = (scopesStr: string): Set<string> => {
    if (!scopesStr) return new Set()
    return new Set(scopesStr.split(' ').filter(Boolean))
  }

  const scopeRows = computed(() => {
    const char = selectedCharacter.value
    if (!char) return []
    const scopeSet = char.token_invalid ? new Set<string>() : parseScopeSet(char.scopes)
    return props.scopes.map((s) => ({
      scope: s.scope,
      description: s.description,
      module: s.module,
      required: s.required,
      authorized: scopeSet.has(s.scope)
    }))
  })

  const hasMissingRequiredScopes = computed(() =>
    scopeRows.value.some((r) => r.required && !r.authorized)
  )

  const formatCoverage = (char: Api.Auth.EveCharacter): string => {
    const requiredScopes = props.scopes.filter((s) => s.required)
    if (requiredScopes.length === 0) return ''
    const scopeSet = parseScopeSet(char.scopes)
    const granted = char.token_invalid
      ? 0
      : requiredScopes.filter((s) => scopeSet.has(s.scope)).length
    return t('info.esiCheckCoverage', { granted, total: requiredScopes.length })
  }
</script>
