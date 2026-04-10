<template>
  <article
    class="hero-card"
    :class="{ 'is-interactive': interactive, 'is-selected': selected }"
    :style="surfaceStyle"
  >
    <div class="hero-card__avatar-shell">
      <img v-if="card.avatar" :src="card.avatar" :alt="card.name" class="hero-card__avatar" />
      <div v-else class="hero-card__avatar hero-card__avatar--placeholder">
        {{ card.name.slice(0, 1) || '?' }}
      </div>
    </div>

    <header class="hero-card__header">
      <h3 class="hero-card__name">{{ card.name }}</h3>
      <p v-if="card.title" class="hero-card__title">{{ card.title }}</p>
    </header>

    <p v-if="card.description" class="hero-card__description">{{ card.description }}</p>
  </article>
</template>

<script setup lang="ts">
  import { computed } from 'vue'

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

  const surfaceStyle = computed(() => ({
    width: rawStyle.value.width,
    minHeight: rawStyle.value.minHeight,
    background: rawStyle.value.background,
    color: rawStyle.value.color,
    borderColor: rawStyle.value.borderColor,
    '--hero-glow': rawStyle.value.boxShadowColor,
    '--hero-font-size': props.card.font_size > 0 ? `${props.card.font_size}px` : '14px'
  }))
</script>

<style scoped>
  .hero-card {
    position: relative;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
    border: 2px solid;
    border-radius: 28px;
    padding: 18px 18px 16px;
    box-shadow: 0 12px 30px rgba(0, 0, 0, 0.28);
    transition:
      transform 0.24s ease,
      box-shadow 0.24s ease,
      border-color 0.24s ease;
    overflow: hidden;
    backdrop-filter: blur(10px);
  }

  .hero-card::before {
    content: '';
    position: absolute;
    inset: 0;
    background:
      radial-gradient(circle at top, rgba(255, 255, 255, 0.14), transparent 38%),
      linear-gradient(180deg, rgba(255, 255, 255, 0.1), transparent 45%);
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
    position: relative;
    z-index: 1;
    width: 96px;
    height: 96px;
    border-radius: 999px;
    padding: 4px;
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
    font-size: 28px;
    font-weight: 700;
    text-transform: uppercase;
  }

  .hero-card__header {
    position: relative;
    z-index: 1;
    text-align: center;
  }

  .hero-card__name {
    margin: 0;
    font-size: 20px;
    font-weight: 700;
    letter-spacing: 0.04em;
  }

  .hero-card__title {
    margin: 6px 0 0;
    font-size: 12px;
    letter-spacing: 0.18em;
    text-transform: uppercase;
    opacity: 0.84;
  }

  .hero-card__description {
    position: relative;
    z-index: 1;
    margin: 0;
    font-size: var(--hero-font-size);
    line-height: 1.55;
    text-align: center;
    opacity: 0.92;
    display: -webkit-box;
    overflow: hidden;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 2;
  }
</style>
