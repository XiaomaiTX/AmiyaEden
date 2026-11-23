<template>
  <header class="fx-banner" :class="{ 'fx-banner--preview': preview }">
    <div class="fx-banner__scene">
      <div class="fx-banner__signature">
        <img :src="insignia" width="30" height="30" alt="" />
        {{ SYSTEM_IDENTITY.displayName }}
      </div>
      <component :is="preview ? 'h3' : 'h1'" class="fx-banner__title">{{ title }}</component>
      <p v-if="subtitle" class="fx-banner__subtitle">{{ subtitle }}</p>
    </div>
    <slot />
  </header>
</template>
<script setup lang="ts">
  import { SYSTEM_IDENTITY } from '@/constants/system-identity'
  import insignia from '@/assets/fuxi-hall/legion.png'
  import '@/assets/fuxi-hall/hall.css'

  defineProps<{ title: string; subtitle?: string; preview?: boolean }>()
</script>
