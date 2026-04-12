<template>
  <div class="overview-matrix">
    <div v-if="loading" class="flex justify-center py-8">
      <ElIcon class="is-loading" :size="24"><Loading /></ElIcon>
    </div>
    <template v-else-if="scopes.length === 0 || characters.length === 0">
      <ElEmpty
        :description="
          characters.length === 0 ? $t('info.esiCheckNoCharacters') : $t('info.esiCheckNoData')
        "
      />
    </template>
    <template v-else>
      <div class="matrix-wrapper overflow-x-auto">
        <table class="matrix-table w-full border-collapse text-sm">
          <thead>
            <tr>
              <th
                class="matrix-corner sticky left-0 z-10 bg-white dark:bg-gray-900 border-b border-r p-2 text-left min-w-[240px]"
              >
                {{ $t('info.esiCheckScope') }}
              </th>
              <th
                v-for="char in characters"
                :key="char.character_id"
                class="border-b p-2 text-center min-w-[100px] cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800"
                @click="$emit('select-character', char.character_id)"
              >
                <div class="flex flex-col items-center gap-1">
                  <ElAvatar :src="buildEveCharacterPortraitUrl(char.character_id, 32)" :size="32" />
                  <span class="text-xs truncate max-w-[90px]" :title="char.character_name">
                    {{ char.character_name }}
                  </span>
                  <ElTag v-if="char.token_invalid" type="danger" size="small" effect="dark">
                    {{ $t('info.esiCheckTokenInvalid') }}
                  </ElTag>
                </div>
              </th>
            </tr>
          </thead>
          <tbody>
            <template v-for="(group, moduleName) in groupedScopes" :key="moduleName">
              <tr>
                <td
                  :colspan="characters.length + 1"
                  class="module-header sticky left-0 bg-gray-50 dark:bg-gray-800 font-semibold text-xs text-gray-500 dark:text-gray-400 px-2 py-1 border-b"
                >
                  {{ moduleName }}
                </td>
              </tr>
              <tr v-for="scope in group" :key="scope.scope">
                <td
                  class="scope-cell sticky left-0 z-10 bg-white dark:bg-gray-900 border-b border-r p-2 text-xs"
                  :title="scope.description"
                >
                  <div class="flex items-center gap-1">
                    <ElTag
                      v-if="scope.required"
                      size="small"
                      type="danger"
                      effect="plain"
                      class="shrink-0"
                    >
                      {{ $t('info.esiCheckRequired') }}
                    </ElTag>
                    <ElTag v-else size="small" type="info" effect="plain" class="shrink-0">
                      {{ $t('info.esiCheckOptional') }}
                    </ElTag>
                    <span class="truncate">{{ scope.scope }}</span>
                  </div>
                </td>
                <td
                  v-for="char in characters"
                  :key="char.character_id"
                  class="border-b p-2 text-center cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800"
                  :title="$t('info.esiCheckClickToDetail')"
                  @click="$emit('select-character', char.character_id)"
                >
                  <span v-if="hasScope(char, scope.scope)" class="text-green-500 text-lg"
                    >&#10003;</span
                  >
                  <span v-else class="text-red-400 text-lg">&#10007;</span>
                </td>
              </tr>
            </template>
          </tbody>
          <tfoot>
            <tr class="font-semibold">
              <td
                class="sticky left-0 z-10 bg-white dark:bg-gray-900 border-t border-r p-2 text-xs"
              >
                {{ $t('info.esiCheckCoverage').split('/')[0].replace('{granted}', '').trim() }}
              </td>
              <td
                v-for="char in characters"
                :key="char.character_id"
                class="border-t p-2 text-center text-xs"
              >
                {{ formatCoverage(char) }}
              </td>
            </tr>
          </tfoot>
        </table>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
  import { Loading } from '@element-plus/icons-vue'
  import { buildEveCharacterPortraitUrl } from '@/utils/eve-image'
  import { useI18n } from 'vue-i18n'

  defineOptions({ name: 'OverviewMatrix' })

  const props = defineProps<{
    scopes: Api.Auth.RegisteredScope[]
    characters: Api.Auth.EveCharacter[]
    loading: boolean
  }>()

  defineEmits<{
    (e: 'select-character', characterId: number): void
  }>()

  const { t } = useI18n()

  const groupedScopes = computed(() => {
    const groups: Record<string, Api.Auth.RegisteredScope[]> = {}
    for (const scope of props.scopes) {
      const mod = scope.module || 'Other'
      if (!groups[mod]) groups[mod] = []
      groups[mod].push(scope)
    }
    return groups
  })

  const parseScopeSet = (scopesStr: string): Set<string> => {
    if (!scopesStr) return new Set()
    return new Set(scopesStr.split(' ').filter(Boolean))
  }

  const hasScope = (char: Api.Auth.EveCharacter, scope: string): boolean => {
    if (char.token_invalid) return false
    return parseScopeSet(char.scopes).has(scope)
  }

  const formatCoverage = (char: Api.Auth.EveCharacter): string => {
    const scopeSet = parseScopeSet(char.scopes)
    const granted = char.token_invalid
      ? 0
      : props.scopes.filter((s) => scopeSet.has(s.scope)).length
    return t('info.esiCheckCoverage', { granted, total: props.scopes.length })
  }
</script>
