<template>
  <div class="info-esi-check-page">
    <ElCard class="art-card" shadow="never">
      <template #header>
        <span class="font-semibold">{{ $t('info.esiCheckOverview') }}</span>
      </template>
      <OverviewMatrix
        :scopes="scopes"
        :characters="characters"
        :loading="loading"
        @select-character="onSelectCharacter"
      />
    </ElCard>

    <ElCard class="art-card mt-4" shadow="never">
      <template #header>
        <span class="font-semibold">{{ $t('info.esiCheckDetail') }}</span>
      </template>
      <CharacterDetail
        :scopes="scopes"
        :characters="characters"
        :selected-character-id="selectedCharacterId"
        @update:selected-character-id="onSelectCharacter"
      />
    </ElCard>
  </div>
</template>

<script setup lang="ts">
  import { fetchEveSSOScopes, fetchMyCharacters } from '@/api/auth'
  import OverviewMatrix from './modules/overview-matrix.vue'
  import CharacterDetail from './modules/character-detail.vue'

  defineOptions({ name: 'EveInfoEsiCheck' })

  const scopes = ref<Api.Auth.RegisteredScope[]>([])
  const characters = ref<Api.Auth.EveCharacter[]>([])
  const loading = ref(true)
  const selectedCharacterId = ref<number>()

  const onSelectCharacter = (characterId: number) => {
    selectedCharacterId.value = characterId
  }

  const loadData = async () => {
    loading.value = true
    try {
      const [scopesRes, charactersRes] = await Promise.all([
        fetchEveSSOScopes(),
        fetchMyCharacters()
      ])
      scopes.value = scopesRes ?? []
      characters.value = charactersRes ?? []
      if (characters.value.length > 0 && !selectedCharacterId.value) {
        selectedCharacterId.value = characters.value[0].character_id
      }
    } catch {
      scopes.value = []
      characters.value = []
    } finally {
      loading.value = false
    }
  }

  onMounted(() => {
    loadData()
  })
</script>
