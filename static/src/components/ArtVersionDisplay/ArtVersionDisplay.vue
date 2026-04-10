<template>
  <div class="version-display"> v{{ version }} </div>
</template>

<script setup lang="ts">
  const version = __APP_VERSION__
</script>

<style scoped lang="scss">
  .version-display {
    padding: 12px 16px;
    font-size: 12px;
    color: var(--art-version-text-color, rgba(255, 255, 255, 0.5));
  }
</style>
