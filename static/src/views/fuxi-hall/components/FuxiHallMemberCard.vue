<template>
  <article class="fuxi-hall-member-card" :style="cardStyle">
    <div class="fuxi-hall-member-card__body">
      <div class="fuxi-hall-member-card__meta">
        <img
          loading="lazy"
          v-if="card.main_character_id > 0"
          class="fuxi-hall-member-card__avatar"
          :class="`is-${card.avatar_shape}`"
          :src="buildEveCharacterPortraitUrl(card.main_character_id, 256)"
          :alt="card.nickname"
        />
        <div class="fuxi-hall-member-card__identity">
          <h3>{{ card.nickname }}</h3>
          <p class="fuxi-hall-member-card__main-name">{{ card.main_character_name }}</p>
          <p v-if="card.is_current_management" class="hall-current"
            ><span aria-hidden="true" />{{ t('fuxiHall.fame.current') }}</p
          >
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
      <p class="hall-tenure"
        >{{ t('fuxiHall.fame.tenure') }}:
        {{
          card.tenure_months == null
            ? t('fuxiHall.fame.unknown')
            : t('fuxiHall.fame.duration', {
                years: Math.floor(card.tenure_months / 12),
                months: card.tenure_months % 12
              })
        }}</p
      >
      <div class="fuxi-hall-member-card__description" v-html="card.description_html" />
    </div>
    <div
      v-if="visibleMetrics.length"
      class="hall-contributions"
      :title="
        card.contributions_updated_at
          ? t('fuxiHall.contributions.updated', {
              date: new Date(card.contributions_updated_at).toLocaleDateString()
            })
          : t('fuxiHall.contributions.pending')
      "
    >
      <dl :aria-label="t('fuxiHall.contributions.title')">
        <div
          v-for="key in visibleMetrics"
          :key="key"
          :title="t(`fuxiHall.contributions.help.${key}`)"
        >
          <dt>{{ t(`fuxiHall.contributions.${key}`) }}</dt>
          <dd>{{ card.contributions?.[key]?.toLocaleString() }}</dd>
        </div>
      </dl>
    </div>
  </article>
</template>

<script setup lang="ts">
  import { computed } from 'vue'
  import { useI18n } from 'vue-i18n'

  import { buildEveCharacterPortraitUrl } from '@/utils/eve-image'

  const props = defineProps<{
    card: Api.FuxiHall.Card
  }>()
  const metrics = ['fleets', 'welfare', 'srp', 'recruitment', 'mentoring'] as const
  const visibleMetrics = computed(() =>
    metrics.filter((key) => (props.card.contributions?.[key] ?? 0) > 0)
  )
  const { t } = useI18n()

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
  .hall-current {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-top: 8px;
    color: var(--el-color-success);
    font-size: 12px;
  }
  .hall-current span {
    width: 10px;
    height: 10px;
    border-radius: 50%;
    background: var(--el-color-success);
    flex-shrink: 0;
  }
  .hall-tenure {
    margin-top: 8px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }
  .fuxi-hall-member-card {
    display: flex;
    flex-direction: column;
    height: 100%;
    overflow: hidden;
    border: 1px solid var(--el-border-color-light);
    border-top: 2px solid var(--accent-color);
    border-radius: 12px;
    background: var(--el-bg-color);
    min-width: 0;
  }

  .fuxi-hall-member-card__body {
    padding: 20px;
    flex: 1;
    color: var(--el-text-color-regular);
    font-size: var(--member-font-size);
  }

  .fuxi-hall-member-card__meta {
    display: flex;
    align-items: flex-start;
    gap: 12px;
  }

  .fuxi-hall-member-card__avatar {
    width: 56px;
    height: 56px;
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
    overflow-wrap: anywhere;
    color: var(--el-text-color-primary);
    font-size: calc(var(--member-font-size) + 3px);
  }

  .fuxi-hall-member-card__main-name {
    margin: 4px 0 0;
    color: var(--el-text-color-regular);
    font-size: 12px;
    overflow-wrap: anywhere;
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

  .fuxi-hall-member-card__identity {
    min-width: 0;
  }
  .hall-contributions {
    border-top: 1px solid var(--el-border-color-light);
    background: var(--el-fill-color-extra-light);
    padding: 12px 20px;
  }
  .hall-contributions dl {
    display: flex;
    flex-wrap: wrap;
    gap: 8px 20px;
    margin: 0;
  }
  .hall-contributions dl > div {
    display: inline-flex;
    align-items: baseline;
    gap: 6px;
  }
  .hall-contributions dt {
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }
  .hall-contributions dd {
    margin: 0;
    font-size: 14px;
    font-weight: 600;
    font-variant-numeric: tabular-nums;
    color: var(--el-text-color-primary);
  }
</style>
