<template>
  <article class="fuxi-hall-member-card" :style="cardStyle">
    <div class="fuxi-hall-member-card__body">
      <div class="fuxi-hall-member-card__meta">
        <img
          v-if="card.main_character_id > 0"
          class="fuxi-hall-member-card__avatar"
          :class="`is-${card.avatar_shape}`"
          :src="buildEveCharacterPortraitUrl(card.main_character_id, 256)"
          :alt="card.nickname"
        />
        <div class="fuxi-hall-member-card__identity">
          <h3>{{ card.nickname }}</h3>
          <p class="fuxi-hall-member-card__main-name">{{ card.main_character_name }}</p>
          <div v-if="card.title_tags.length > 0" class="fuxi-hall-member-card__title-tags">
            <ElTag
              v-for="tag in card.title_tags"
              :key="`${card.id}-${tag}`"
              size="small"
              effect="plain"
              round
              :style="tagStyle"
            >
              {{ tag }}
            </ElTag>
          </div>
        </div>
      </div>
      <div class="fuxi-hall-member-card__description" v-html="card.description_html" />
    </div>
  </article>
</template>

<script setup lang="ts">
  import { computed } from 'vue'

  import { buildEveCharacterPortraitUrl } from '@/utils/eve-image'

  const props = defineProps<{
    card: Api.FuxiHall.Card
  }>()

  const cardStyle = computed(() => ({
    '--accent-color': props.card.accent_color,
    '--member-font-size': `${props.card.font_scale}px`
  }))

  const tagStyle = computed(() => ({
    color: props.card.accent_color,
    borderColor: props.card.accent_color,
    backgroundColor: 'transparent'
  }))
</script>

<style scoped>
  .fuxi-hall-member-card {
    border: 1px solid color-mix(in srgb, var(--accent-color), transparent 76%);
    border-radius: 16px;
    background: var(--el-bg-color);
    min-width: 0;
  }

  .fuxi-hall-member-card__body {
    padding: 14px;
    color: var(--el-text-color-regular);
    font-size: var(--member-font-size);
  }

  .fuxi-hall-member-card__meta {
    display: flex;
    align-items: flex-start;
    gap: 12px;
  }

  .fuxi-hall-member-card__avatar {
    width: 72px;
    height: 72px;
    object-fit: cover;
    flex-shrink: 0;
    border: 2px solid color-mix(in srgb, var(--accent-color), transparent 62%);
    display: block;
  }

  .fuxi-hall-member-card__avatar.is-circle {
    border-radius: 999px;
  }

  .fuxi-hall-member-card__avatar.is-rounded {
    border-radius: 12px;
  }

  .fuxi-hall-member-card__avatar.is-square {
    border-radius: 8px;
  }

  .fuxi-hall-member-card__identity h3 {
    margin: 0;
    color: var(--el-text-color-primary);
    font-size: calc(var(--member-font-size) + 3px);
  }

  .fuxi-hall-member-card__main-name {
    margin: 4px 0 0;
    color: var(--el-text-color-regular);
    font-style: italic;
  }

  .fuxi-hall-member-card__title-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    margin-top: 6px;
  }

  .fuxi-hall-member-card__description {
    margin-top: 8px;
    color: var(--el-text-color-secondary);
    line-height: 1.6;
    overflow-wrap: anywhere;
  }
</style>
