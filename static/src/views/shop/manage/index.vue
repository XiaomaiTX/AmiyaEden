<!-- 商店管理页面（管理员） -->
<template>
  <div class="shop-admin-page art-full-height">
    <ManageProducts ref="productsRef" />
  </div>
</template>

<script setup lang="ts">
  import ManageProducts from './modules/manage-products.vue'

  defineOptions({ name: 'SystemShop' })

  const productsRef = ref<InstanceType<typeof ManageProducts>>()

  onMounted(() => {
    productsRef.value?.load()
  })
</script>
