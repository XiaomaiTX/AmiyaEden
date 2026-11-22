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
          <h3 :title="card.nickname">{{ card.nickname }}</h3>
          <div class="fuxi-hall-member-card__main-name">
            <span :title="card.main_character_name">{{ card.main_character_name }}</span>
            <ElButton
              v-if="card.main_character_name"
              text
              size="small"
              class="hall-copy"
              :aria-label="t('fuxiHall.fame.copyCharacterName')"
              @click="copyText(card.main_character_name)"
            >
              <ArtSvgIcon icon="ri:file-copy-line" />
            </ElButton>
          </div>
        </div>
        <p v-if="card.is_current_management" class="hall-current"
          ><span aria-hidden="true" />{{ t('fuxiHall.fame.current') }}</p
        >
      </div>
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
      <div v-if="qq" class="hall-contact">
        <span
          >{{ t('fuxiHall.fame.qq') }}: <strong>{{ qq }}</strong></span
        >
        <ElButton
          text
          size="small"
          class="hall-copy"
          :aria-label="t('fuxiHall.fame.copyQQ')"
          @click="copyText(qq)"
        >
          <ArtSvgIcon icon="ri:file-copy-line" />
        </ElButton>
      </div>
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
          :class="`is-${key}`"
          :title="t(`fuxiHall.contributions.help.${key}`)"
        >
          <dt>{{ t(`fuxiHall.contributions.${key}`) }}</dt>
          <dd>{{ card.contributions?.[key]?.toLocaleString() }}</dd>
        </div>
      </dl>
    </div>
    <div v-if="$slots.action" class="fuxi-hall-member-card__action"><slot name="action" /></div>
  </article>
</template>

<script setup lang="ts">
  import { computed } from 'vue'
  import { useI18n } from 'vue-i18n'
  import { useClipboardCopy } from '@/hooks/core/useClipboardCopy'

  import { buildEveCharacterPortraitUrl } from '@/utils/eve-image'

  const props = defineProps<{
    card: Api.FuxiHall.Card
  }>()
  const metrics = ['corp_pap', 'fleets', 'welfare', 'srp', 'recruitment', 'mentoring'] as const
  const visibleMetrics = computed(() =>
    metrics.filter((key) => (props.card.contributions?.[key] ?? 0) > 0)
  )
  const { t } = useI18n()
  const { copyText } = useClipboardCopy()
  const qq = computed(() => props.card.qq?.trim() ?? '')

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
  .hall-contact {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 8px;
    margin-top: 8px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }
  .hall-contact strong {
    color: var(--el-text-color-primary);
    font-weight: 500;
  }
  .hall-copy {
    height: 20px;
    padding: 0 4px;
    flex-shrink: 0;
  }
  .hall-current {
    display: inline-flex;
    align-items: center;
    flex-shrink: 0;
    gap: 6px;
    margin: 0 0 0 auto;
    padding: 2px 8px;
    border-radius: 999px;
    background: var(--el-color-success-light-9);
    color: var(--el-color-success);
    font-size: 12px;
    white-space: nowrap;
  }
  .hall-current span {
    width: 8px;
    height: 8px;
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
    overflow: hidden;
    border: 1px solid var(--el-border-color-light);
    border-top: 2px solid var(--accent-color);
    border-radius: 12px;
    background: var(--el-bg-color);
    min-width: 0;
    /* Fill the grid row so cards in one row share a height and contributions sit at the bottom. */
    height: 100%;
  }

  .fuxi-hall-member-card__body {
    flex: 1;
    padding: 16px;
    color: var(--el-text-color-regular);
    font-size: var(--member-font-size);
  }

  .fuxi-hall-member-card__meta {
    display: flex;
    align-items: flex-start;
    gap: 12px;
  }

  .fuxi-hall-member-card__avatar {
    width: 48px;
    height: 48px;
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

  .fuxi-hall-member-card__identity h3,
  .fuxi-hall-member-card__main-name span {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .fuxi-hall-member-card__identity h3 {
    margin: 0;
    color: var(--el-text-color-primary);
    font-size: calc(var(--member-font-size) + 3px);
  }

  .fuxi-hall-member-card__main-name {
    display: flex;
    align-items: center;
    gap: 2px;
    min-width: 0;
    margin-top: 4px;
    color: var(--el-text-color-regular);
    font-size: 12px;
  }

  .fuxi-hall-member-card__title-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    margin-top: 10px;
  }

  .fuxi-hall-member-card__action {
    padding: 0 16px 16px;
  }

  .fuxi-hall-member-card__action :deep(.el-button) {
    width: 100%;
  }

  .fuxi-hall-member-card__description {
    margin-top: 8px;
    color: var(--el-text-color-secondary);
    line-height: 1.6;
    overflow-wrap: anywhere;
  }

  .fuxi-hall-member-card__identity {
    flex: 1;
    min-width: 0;
  }
  .hall-contributions {
    border-top: 1px solid var(--el-border-color-light);
    padding: 12px 16px;
  }
  .hall-contributions dl {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    margin: 0;
  }
  .hall-contributions dl > div {
    display: inline-flex;
    align-items: baseline;
    gap: 8px;
    padding: 4px 10px;
    border-radius: 6px;
    border: 1px solid color-mix(in srgb, var(--metric-color), transparent 80%);
    background: color-mix(in srgb, var(--metric-color), transparent 90%);
  }
  .is-corp_pap {
    --metric-color: #d97706;
  }
  .is-fleets {
    --metric-color: #0284c7;
  }
  .is-welfare {
    --metric-color: #059669;
  }
  .is-srp {
    --metric-color: #9333ea;
  }
  .is-recruitment {
    --metric-color: #0891b2;
  }
  .is-mentoring {
    --metric-color: #e11d48;
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
