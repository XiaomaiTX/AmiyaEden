<template>
  <div class="overview-matrix">
    <div v-if="loading" class="flex justify-center py-8">
      <ElIcon class="is-loading" :size="24"><Loading /></ElIcon>
    </div>
    <template v-else-if="characters.length === 0">
      <ElEmpty :description="$t('info.esiCheckNoCharacters')" />
    </template>
    <template v-else>
      <div class="flex items-center gap-3 flex-wrap">
        <span class="text-sm text-gray-500">
          {{ $t('info.esiCheckAllCharactersCount', { count: characters.length }) }}
        </span>
        <ElTag v-if="invalidCharacters.length === 0" type="success" effect="plain">
          {{ $t('info.esiCheckAllValid') }}
        </ElTag>
        <template v-else>
          <ElTag type="danger" effect="dark">
            {{ $t('info.esiCheckInvalidCount', { count: invalidCharacters.length }) }}
          </ElTag>
          <ElTag
            v-for="char in invalidCharacters"
            :key="char.character_id"
            type="warning"
            effect="plain"
            class="cursor-pointer"
            @click="$emit('select-character', char.character_id)"
          >
            {{ char.character_name }}
          </ElTag>
        </template>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
  import { Loading } from '@element-plus/icons-vue'

  defineOptions({ name: 'OverviewMatrix' })

  const props = defineProps<{
    scopes: Api.Auth.RegisteredScope[]
    characters: Api.Auth.EveCharacter[]
    loading: boolean
  }>()

  defineEmits<{
    (e: 'select-character', characterId: number): void
  }>()

  const parseScopeSet = (scopesStr: string): Set<string> => {
    if (!scopesStr) return new Set()
    return new Set(scopesStr.split(' ').filter(Boolean))
  }

  const invalidCharacters = computed(() =>
    props.characters.filter((char) => {
      if (char.token_invalid) return true
      const scopeSet = parseScopeSet(char.scopes)
      return props.scopes.some((s) => s.required && !scopeSet.has(s.scope))
    })
  )
</script>
