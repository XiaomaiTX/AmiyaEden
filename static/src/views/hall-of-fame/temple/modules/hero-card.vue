<template>
  <article
    class="hero-card"
    :class="{ 'is-interactive': interactive, 'is-selected': selected }"
    :style="surfaceStyle"
  >
    <img v-if="rawStyle.frameSrc" :src="rawStyle.frameSrc" alt="" class="hero-card__frame" />

    <div class="hero-card__media">
      <div class="hero-card__avatar-shell">
        <img v-if="portraitUrl" :src="portraitUrl" :alt="card.name" class="hero-card__avatar" />
        <div v-else class="hero-card__avatar hero-card__avatar--placeholder">
          {{ card.name.slice(0, 1) || '?' }}
        </div>
      </div>

      <div v-if="card.badge_image" class="hero-card__badge-shell">
        <img :src="card.badge_image" :alt="card.name" class="hero-card__badge-image" />
      </div>
    </div>

    <div class="hero-card__content">
      <header class="hero-card__header">
        <h3 class="hero-card__name">{{ card.name }}</h3>
        <p v-if="card.title" class="hero-card__title">{{ card.title }}</p>
      </header>

      <p v-if="card.description" class="hero-card__description">{{ card.description }}</p>
    </div>
  </article>
</template>

<script setup lang="ts">
  import { computed } from 'vue'

  import { buildHallOfFamePortraitUrl } from '../../portrait.helpers'
  import { buildHeroCardStyle } from './temple-canvas.helpers'

  const props = withDefaults(
    defineProps<{
      card: Api.HallOfFame.Card
      interactive?: boolean
      selected?: boolean
    }>(),
    {
      interactive: false,
      selected: false
    }
  )

  const rawStyle = computed(() => buildHeroCardStyle(props.card))
  const portraitUrl = computed(() => buildHallOfFamePortraitUrl(props.card.character_id))

  const surfaceStyle = computed(() => ({
    width: rawStyle.value.width,
    minHeight: rawStyle.value.minHeight,
    background: rawStyle.value.background,
    backgroundClip: rawStyle.value.frameSrc ? 'border-box' : 'padding-box',
    color: rawStyle.value.color,
    borderColor: rawStyle.value.frameSrc ? 'transparent' : rawStyle.value.borderColor,
    '--hero-glow': rawStyle.value.boxShadowColor,
    '--hero-font-size': props.card.font_size > 0 ? `${props.card.font_size}px` : '13px',
    '--hero-title-color': rawStyle.value.titleColor
  }))
</script>

<style scoped>
  .hero-card {
    box-sizing: border-box;
    position: relative;
    display: grid;
    grid-template-columns: 72px minmax(0, 1fr);
    align-items: start;
    column-gap: 12px;
    border: 2px solid;
    border-radius: 24px;
    padding: 14px;
    box-shadow: 0 12px 30px rgba(0, 0, 0, 0.28);
    transition:
      transform 0.24s ease,
      box-shadow 0.24s ease,
      border-color 0.24s ease;
    overflow: hidden;
    background-clip: padding-box;
    backdrop-filter: blur(10px);
  }

  .hero-card__media {
    position: relative;
    z-index: 1;
    display: flex;
    flex-direction: column;
    gap: 12px;
    align-self: stretch;
  }

  .hero-card::before {
    content: '';
    position: absolute;
    inset: 2px;
    border-radius: 22px;
    background:
      radial-gradient(circle at top, rgba(255, 255, 255, 0.14), transparent 38%),
      linear-gradient(180deg, rgba(255, 255, 255, 0.1), transparent 45%);
    pointer-events: none;
  }

  .hero-card::after {
    content: none;
  }

  .hero-card__frame {
    position: absolute;
    inset: 0;
    z-index: 2;
    width: 100%;
    height: 100%;
    object-fit: fill;
    pointer-events: none;
  }

  .hero-card.is-interactive:hover {
    transform: translateY(-4px) scale(1.02);
    box-shadow:
      0 18px 42px rgba(0, 0, 0, 0.35),
      0 0 0 1px var(--hero-glow),
      0 0 28px var(--hero-glow);
  }

  .hero-card.is-selected {
    box-shadow:
      0 18px 42px rgba(0, 0, 0, 0.35),
      0 0 0 2px rgba(89, 161, 255, 0.85),
      0 0 0 6px rgba(89, 161, 255, 0.16);
  }

  .hero-card__avatar-shell {
    width: 72px;
    height: 72px;
    border-radius: 999px;
    padding: 3px;
    background: linear-gradient(180deg, rgba(255, 255, 255, 0.45), rgba(255, 255, 255, 0.08));
  }

  .hero-card__avatar {
    width: 100%;
    height: 100%;
    border-radius: 999px;
    object-fit: cover;
    display: block;
    background: rgba(255, 255, 255, 0.08);
  }

  .hero-card__avatar--placeholder {
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 22px;
    font-weight: 700;
    text-transform: uppercase;
  }

  .hero-card__header {
    position: relative;
    min-width: 0;
    text-align: left;
  }

  .hero-card__content {
    position: relative;
    z-index: 1;
    display: flex;
    min-width: 0;
    min-height: 0;
    flex-direction: column;
    gap: 0;
    align-self: stretch;
  }

  .hero-card__name {
    margin: 0;
    font-size: 18px;
    font-weight: 700;
    line-height: 1.2;
    letter-spacing: 0.03em;
    word-break: break-word;
  }

  .hero-card__title {
    margin: 0;
    color: var(--hero-title-color);
    font-size: 15px;
    font-weight: 700;
    line-height: 1.25;
    letter-spacing: 0.04em;
    word-break: break-word;
  }

  .hero-card__description {
    position: relative;
    min-height: 0;
    flex: 1 1 auto;
    margin: 8px 0;
    padding-right: 2px;
    font-size: var(--hero-font-size);
    line-height: 1.45;
    text-align: left;
    opacity: 0.92;
    overflow: auto;
    white-space: pre-wrap;
    word-break: break-word;
  }

  .hero-card__badge-shell {
    width: 72px;
    min-height: 52px;
    border-radius: 14px;
    overflow: hidden;
    background: rgba(255, 255, 255, 0.08);
    box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.1);
  }

  .hero-card__badge-image {
    display: block;
    width: 100%;
    height: 100%;
    min-height: 52px;
    object-fit: cover;
  }
</style>
