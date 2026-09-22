<template>
  <div>
    <MumbleCredentialCard :canonical-name="primaryCharacterName" />
  </div>
</template>

<script setup lang="ts">
  import { fetchGetUserInfo } from '@/api/auth'
  import MumbleCredentialCard from '@/components/mumble-credential-card.vue'

  defineOptions({ name: 'MumbleCredentials' })

  const primaryCharacterName = ref('')

  onMounted(async () => {
    try {
      const info = await fetchGetUserInfo()
      const characters = info.characters ?? []
      const primary =
        characters.find((character) => character.character_id === info.primaryCharacterId) ??
        characters[0]
      primaryCharacterName.value = primary?.character_name ?? ''
    } catch {
      primaryCharacterName.value = ''
    }
  })
</script>
