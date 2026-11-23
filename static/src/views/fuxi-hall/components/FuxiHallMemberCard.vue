<template>
  <article class="fx-card" :style="cardStyle">
    <header class="fx-card__head">
      <div class="fx-portrait" :class="`fx-portrait--${card.avatar_shape}`">
        <img
          v-if="card.main_character_id > 0 && failedPortrait !== card.main_character_id"
          loading="lazy"
          :src="buildEveCharacterPortraitUrl(card.main_character_id, 256)"
          :alt="card.nickname"
          @error="failedPortrait = card.main_character_id"
        />
      </div>
      <div class="fx-identity">
        <span v-if="card.is_current_management" class="fx-current">{{
          t('fuxiHall.fame.current')
        }}</span>
        <h3 :title="card.nickname">{{ card.nickname }}</h3>
        <div class="fx-character">
          <span :title="card.main_character_name">{{ card.main_character_name }}</span>
          <ElButton
            v-if="card.main_character_name"
            text
            native-type="button"
            class="fx-copy"
            :aria-label="t('fuxiHall.fame.copyCharacterName')"
            @click="copyText(card.main_character_name)"
          >
            <ArtSvgIcon icon="ri:file-copy-line" />
          </ElButton>
        </div>
      </div>
    </header>
    <div class="fx-card__body">
      <div v-if="card.title_tags.length" class="fx-roles">
        <span v-for="tag in card.title_tags" :key="tag">{{ tag }}</span>
      </div>
      <div class="fx-record">
        <ArtSvgIcon icon="ri:timer-line" aria-hidden="true" />
        <span>{{ t('fuxiHall.fame.tenure') }}</span>
        <strong>{{
          card.tenure_months == null
            ? t('fuxiHall.fame.unknown')
            : t('fuxiHall.fame.duration', {
                years: Math.floor(card.tenure_months / 12),
                months: card.tenure_months % 12
              })
        }}</strong>
      </div>
      <div v-if="qq" class="fx-contact">
        <span>{{ t('fuxiHall.fame.qq') }}</span
        ><strong>{{ qq }}</strong>
        <ElButton
          text
          native-type="button"
          class="fx-copy"
          :aria-label="t('fuxiHall.fame.copyQQ')"
          @click="copyText(qq)"
        >
          <ArtSvgIcon icon="ri:file-copy-line" />
        </ElButton>
      </div>
      <div v-if="card.description_html" class="fx-bio" v-html="card.description_html" />
    </div>
    <div
      v-if="visibleMetrics.length"
      class="fx-contributions"
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
          :class="`fx-metric--${key}`"
          :title="t(`fuxiHall.contributions.help.${key}`)"
        >
          <dt>{{ t(`fuxiHall.contributions.${key}`) }}</dt>
          <dd>{{
            card.contributions?.[key]?.toLocaleString(undefined, { maximumFractionDigits: 0 })
          }}</dd>
        </div>
      </dl>
    </div>
    <div v-if="$slots.action" class="fx-card__action"><slot name="action" /></div>
  </article>
</template>

<script setup lang="ts">
  import { computed, ref } from 'vue'
  import { useI18n } from 'vue-i18n'
  import { useClipboardCopy } from '@/hooks/core/useClipboardCopy'

  import { buildEveCharacterPortraitUrl } from '@/utils/eve-image'

  import '@/assets/fuxi-hall/hall.css'

  const failedPortrait = ref<number | null>(null)
  const props = defineProps<{
    card: Api.FuxiHall.Card
  }>()
  const metrics = ['corp_pap', 'fleets', 'welfare', 'srp', 'recruitment', 'mentoring'] as const
  // Values render without decimals, so filter on the rounded value to never show a "0" tag.
  const visibleMetrics = computed(() =>
    metrics.filter((key) => Math.round(props.card.contributions?.[key] ?? 0) > 0)
  )
  const { t } = useI18n()
  const { copyText } = useClipboardCopy()
  const qq = computed(() => props.card.qq?.trim() ?? '')

  const cardStyle = computed(() => ({
    '--fx-accent': props.card.accent_color,
    '--fx-size': `${props.card.font_scale}px`
  }))
</script>
