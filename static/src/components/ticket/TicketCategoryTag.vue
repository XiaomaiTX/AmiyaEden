<template>
  <ElTag type="info">{{ label }}</ElTag>
</template>

<script setup lang="ts">
  import { ElTag } from 'element-plus'

  defineProps<{
    label: string
  }>()
</script>
