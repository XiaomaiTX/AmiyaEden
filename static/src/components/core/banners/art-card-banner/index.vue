<!-- 卡片横幅组件 -->
<template>
  <div class="art-card-sm flex-c flex-col pb-6" :style="{ height: height }">
    <div class="flex-c flex-col gap-4 text-center">
      <div class="w-45">
        <img :src="image" :alt="title" class="w-full h-full object-contain" />
      </div>
      <div class="box-border px-4">
        <p class="mb-2 text-lg font-semibold text-g-800">{{ title }}</p>
        <p class="m-0 text-sm text-g-600">{{ description }}</p>
      </div>
      <div class="flex-c gap-3">
        <div
          v-if="cancelButton?.show"
          class="inline-block h-9 px-3 text-sm/9 c-p select-none rounded-md border border-g-300"
          :style="{
            backgroundColor: cancelButton?.color,
            color: cancelButton?.textColor
          }"
          @click="handleCancel"
        >
          {{ cancelButtonText }}
        </div>
        <div
          v-if="button?.show"
          class="inline-block h-9 px-3 text-sm/9 c-p select-none rounded-md"
          :style="{ backgroundColor: button?.color, color: button?.textColor }"
          @click="handleClick"
        >
          {{ buttonText }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  // 导入默认图标
  import { computed } from 'vue'
  import { useI18n } from 'vue-i18n'
  import defaultIcon from '@imgs/3d/icon1.webp'

  defineOptions({ name: 'ArtCardBanner' })
  const { t } = useI18n()

  // 定义卡片横幅组件的属性接口
  interface CardBannerProps {
    /** 高度 */
    height?: string
    /** 图片路径 */
    image?: string
    /** 标题文本 */
    title: string
    /** 描述文本 */
    description: string
    /** 主按钮配置 */
    button?: {
      /** 是否显示 */
      show?: boolean
      /** 按钮文本 */
      text?: string
      /** 背景颜色 */
      color?: string
      /** 文字颜色 */
      textColor?: string
    }
    /** 取消按钮配置 */
    cancelButton?: {
      /** 是否显示 */
      show?: boolean
      /** 按钮文本 */
      text?: string
      /** 背景颜色 */
      color?: string
      /** 文字颜色 */
      textColor?: string
    }
  }

  // 定义组件属性默认值
  const props = withDefaults(defineProps<CardBannerProps>(), {
    height: '24rem',
    image: defaultIcon,
    title: '',
    description: '',
    // 主按钮默认配置
    button: () => ({
      show: true,
      text: '',
      color: 'var(--theme-color)',
      textColor: '#fff'
    }),
    // 取消按钮默认配置
    cancelButton: () => ({
      show: false,
      text: '',
      color: '#f5f5f5',
      textColor: '#666'
    })
  })

  // 定义组件事件
  const emit = defineEmits<{
    (e: 'click'): void // 主按钮点击事件
    (e: 'cancel'): void // 取消按钮点击事件
  }>()

  // 主按钮点击处理函数
  const handleClick = () => {
    emit('click')
  }

  const buttonText = computed(() => props.button?.text || t('banner.card.viewDetails'))
  const cancelButtonText = computed(() => props.cancelButton?.text || t('common.cancel'))

  // 取消按钮点击处理函数
  const handleCancel = () => {
    emit('cancel')
  }
</script>
