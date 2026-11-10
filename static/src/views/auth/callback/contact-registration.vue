<template>
  <form
    class="contact-registration"
    aria-labelledby="registration-title"
    :aria-busy="!ready || saving"
    @submit.prevent="submit"
  >
    <div class="registration-language" role="group" :aria-label="t('registration.language')">
      <button
        type="button"
        lang="zh-CN"
        :aria-pressed="locale === LanguageEnum.ZH"
        @click="changeLanguage(LanguageEnum.ZH)"
        >{{ t('registration.chinese') }}</button
      >
      <button
        type="button"
        lang="en"
        :aria-pressed="locale === LanguageEnum.EN"
        @click="changeLanguage(LanguageEnum.EN)"
        >{{ t('registration.english') }}</button
      >
    </div>
    <header class="registration-heading">
      <h1 id="registration-title">{{ t('registration.title') }}</h1>
      <p>{{ t('registration.intro') }}</p>
    </header>
    <div class="registration-identity">
      <div class="registration-portrait" aria-hidden="true">
        <span>{{ character.slice(0, 1) }}</span>
        <img
          v-if="characterId && !portraitFailed"
          :src="buildEveCharacterPortraitUrl(characterId)"
          alt=""
          @error="portraitFailed = true"
        />
      </div>
      <div class="registration-character">
        <span>{{ t('registration.character') }}</span>
        <strong>{{ character || t('common.loading') }}</strong>
      </div>
    </div>
    <div class="registration-field">
      <label for="registration-nickname"
        >{{ t('registration.nickname')
        }}<span class="registration-required" aria-hidden="true">*</span></label
      >
      <el-input
        id="registration-nickname"
        v-model="form.nickname"
        minlength="2"
        maxlength="20"
        required
        :placeholder="t('registration.nicknamePlaceholder')"
        aria-describedby="registration-nickname-hint"
      />
      <small id="registration-nickname-hint" class="registration-hint">{{
        t('registration.nicknameHint')
      }}</small>
    </div>
    <fieldset class="registration-contact-method">
      <legend
        >{{ t('registration.contactMethod')
        }}<span class="registration-required" aria-hidden="true">*</span></legend
      >
      <div class="registration-options">
        <label class="registration-option" :data-selected="contactMethod === 'qq'">
          <input
            v-model="contactMethod"
            type="radio"
            name="contact-method"
            value="qq"
            @change="error = ''"
          />
          <span>{{ t('registration.qq') }}</span>
        </label>
        <label class="registration-option" :data-selected="contactMethod === 'discord_id'">
          <input
            v-model="contactMethod"
            type="radio"
            name="contact-method"
            value="discord_id"
            @change="error = ''"
          />
          <span>{{ t('registration.discord') }}</span>
        </label>
      </div>
    </fieldset>
    <div class="registration-field">
      <label for="registration-contact"
        >{{ t(contactMethod === 'qq' ? 'registration.qq' : 'registration.discordId')
        }}<span class="registration-required" aria-hidden="true">*</span></label
      >
      <el-input
        id="registration-contact"
        :key="contactMethod"
        v-model="form[contactMethod]"
        maxlength="20"
        :inputmode="contactMethod === 'qq' ? 'numeric' : 'text'"
        :pattern="contactMethod === 'qq' ? '[0-9]*' : undefined"
        aria-required="true"
        :placeholder="
          t(
            contactMethod === 'qq'
              ? 'registration.qqPlaceholder'
              : 'registration.discordPlaceholder'
          )
        "
      />
    </div>
    <p v-if="error" role="alert" class="registration-error">{{
      error.startsWith('registration.') ? t(error) : error
    }}</p>
    <el-button
      class="registration-submit"
      native-type="submit"
      type="primary"
      :disabled="!ready"
      :loading="saving"
      >{{ t('registration.submit') }}</el-button
    >
    <el-button class="registration-login" text @click="emit('login')">{{
      t('login.callback.retry')
    }}</el-button>
  </form>
</template>

<script setup lang="ts">
  import { useI18n } from 'vue-i18n'
  import { completeRegistration, getRegistration } from '@/api/auth'
  import { HttpError } from '@/utils/http/error'
  import { buildEveCharacterPortraitUrl } from '@/utils/eve-image'
  import { LanguageEnum } from '@/enums/appEnum'
  import { useUserStore } from '@/store/modules/user'

  const props = defineProps<{ ticket: string }>()
  const emit = defineEmits<{ completed: [token: string]; login: [] }>()
  const { t, locale } = useI18n()
  const userStore = useUserStore()
  function changeLanguage(language: LanguageEnum) {
    locale.value = language
    userStore.setLanguage(language)
  }
  const form = reactive({ nickname: '', qq: '', discord_id: '' })
  const character = ref('')
  const characterId = ref(0)
  const portraitFailed = ref(false)
  const contactMethod = ref<'qq' | 'discord_id'>('qq')
  const ready = ref(false)
  const saving = ref(false)
  const error = ref('')

  onMounted(async () => {
    try {
      const info = await getRegistration(props.ticket)
      character.value = info.character_name
      characterId.value = info.character_id
      ready.value = true
    } catch {
      error.value = 'registration.expired'
    }
  })

  async function submit() {
    if (saving.value || !ready.value) return
    if (Array.from(form.nickname.trim()).length < 2) {
      error.value = 'registration.nicknameLength'
      return
    }
    if (!form[contactMethod.value].trim()) {
      error.value = 'registration.contactRequired'
      return
    }
    saving.value = true
    error.value = ''
    try {
      const result = await completeRegistration(props.ticket, {
        nickname: form.nickname,
        qq: contactMethod.value === 'qq' ? form.qq : '',
        discord_id: contactMethod.value === 'discord_id' ? form.discord_id : ''
      })
      emit('completed', result.token)
    } catch (err) {
      if (!(err instanceof HttpError && err.code === 409))
        error.value = err instanceof Error ? err.message : 'registration.failed'
    } finally {
      saving.value = false
    }
  }
</script>

<style scoped>
  .contact-registration {
    --reg-surface: #fff;
    --reg-text: #203047;
    --reg-muted: #64748b;
    --reg-border: #e2e8f0;
    --reg-tint: #f5f7fc;
    --reg-accent: #4665d5;
    box-sizing: border-box;
    width: 100%;
    max-width: 460px;
    padding: 36px;
    border: 1px solid var(--reg-border);
    border-radius: 20px;
    background: var(--reg-surface);
    color: var(--reg-text);
    box-shadow: 0 16px 48px -20px rgb(32 48 71 / 18%);
    text-align: left;
    font-size: 14px;
    line-height: 1.5;
  }

  .registration-heading h1 {
    margin: 0;
    font-size: 25px;
    font-weight: 650;
    letter-spacing: -0.5px;
    line-height: 1.4;
  }

  .registration-heading p {
    margin: 8px 0 0;
    color: var(--reg-muted);
    line-height: 1.8;
  }

  .registration-identity {
    display: flex;
    align-items: center;
    gap: 14px;
    margin: 26px 0;
    padding: 14px;
    border-radius: 12px;
    background: var(--reg-tint);
  }

  .registration-portrait {
    position: relative;
    display: grid;
    flex: 0 0 52px;
    width: 52px;
    height: 52px;
    place-items: center;
    overflow: hidden;
    border-radius: 9px;
    background: var(--reg-border);
    color: var(--reg-muted);
    font-size: 22px;
  }

  .registration-portrait img {
    position: absolute;
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .registration-character {
    display: grid;
    min-width: 0;
    gap: 3px;
  }

  .registration-character > span {
    color: var(--reg-muted);
    font-size: 12px;
  }

  .registration-character strong {
    overflow-wrap: anywhere;
    font-size: 16px;
    font-weight: 600;
  }

  .registration-field {
    display: grid;
    gap: 9px;
  }

  .registration-field > span,
  .registration-field > label,
  .registration-contact-method legend {
    color: var(--reg-text);
    font-size: 13px;
    font-weight: 550;
  }

  .registration-contact-method {
    min-width: 0;
    margin: 23px 0 18px;
    padding: 0;
    border: 0;
  }

  .registration-contact-method legend {
    margin-bottom: 9px;
    padding: 0;
  }

  .registration-options {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 10px;
  }

  .registration-option {
    display: flex;
    align-items: center;
    gap: 9px;
    min-height: 44px;
    padding: 10px 14px;
    border: 1px solid var(--reg-border);
    border-radius: 8px;
    cursor: pointer;
  }

  .registration-option[data-selected='true'] {
    border-color: var(--reg-accent);
    background: var(--reg-tint);
    color: var(--reg-accent);
  }

  .registration-option:focus-within {
    outline: 3px solid rgb(70 101 213 / 16%);
    outline-offset: 2px;
  }

  .registration-option input {
    width: 16px;
    height: 16px;
    margin: 0;
    accent-color: var(--reg-accent);
  }

  .registration-error {
    margin: 14px 0 0;
    color: #c33838;
    font-size: 13px;
    line-height: 1.7;
  }

  .contact-registration .registration-submit {
    width: 100%;
    height: 44px;
    margin: 28px 0 0;
    border: 0;
    border-radius: 8px;
    background: var(--reg-accent);
    color: #fff;
    font-size: 14px;
    font-weight: 550;
    cursor: pointer;
  }

  .contact-registration .registration-submit:hover:not(:disabled) {
    background: #3854bd;
  }

  .contact-registration .registration-submit:disabled {
    opacity: 0.55;
    cursor: not-allowed;
  }

  .contact-registration .registration-login {
    display: flex;
    width: 100%;
    height: 36px;
    margin: 10px 0 0;
    border: 0;
    background: transparent;
    color: var(--reg-muted);
    font-size: 13px;
    cursor: pointer;
  }

  .contact-registration .registration-login:hover {
    background: var(--reg-tint);
    color: var(--reg-text);
  }

  :global(.dark) .contact-registration {
    --reg-surface: #1d2737;
    --reg-text: #e6edf7;
    --reg-muted: #a3b2c7;
    --reg-border: #344156;
    --reg-tint: #253146;
    --reg-accent: #7593ff;
  }

  @media (max-width: 520px) {
    .contact-registration {
      padding: 28px 24px;
      border-radius: 16px;
    }

    .registration-heading h1 {
      font-size: 23px;
    }
  }

  .registration-field :deep(.el-input__wrapper) {
    min-height: 44px;
    padding: 0 13px;
    border-radius: 8px;
    background: var(--reg-surface);
    box-shadow: 0 0 0 1px var(--reg-border) inset;
  }
  .registration-field :deep(.el-input__wrapper.is-focus) {
    box-shadow:
      0 0 0 1px var(--reg-accent) inset,
      0 0 0 3px rgb(70 101 213 / 12%);
  }
  .registration-field :deep(.el-input__inner) {
    height: 44px;
    color: var(--reg-text);
    font-size: 14px;
  }
  .registration-field :deep(.el-input__inner::placeholder) {
    color: var(--reg-muted);
  }

  .registration-language {
    display: flex;
    justify-content: flex-end;
    gap: 2px;
    margin: -10px -8px 16px 0;
  }
  .registration-language button {
    min-width: 38px;
    min-height: 30px;
    padding: 4px 8px;
    border: 0;
    border-radius: 6px;
    background: transparent;
    color: var(--reg-muted);
    font-size: 12px;
    cursor: pointer;
  }
  .registration-language button[aria-pressed='true'] {
    background: var(--reg-tint);
    color: var(--reg-accent);
    font-weight: 600;
  }
  .registration-language button:focus-visible {
    outline: 2px solid var(--reg-accent);
    outline-offset: 2px;
  }
  .registration-required {
    margin-left: 4px;
    color: #d3454f;
  }
  .registration-hint {
    color: var(--reg-muted);
    font-size: 12px;
  }

  :global(.dark) .contact-registration .registration-submit {
    color: #142039;
  }
  :global(.dark) .contact-registration .registration-submit:hover:not(:disabled) {
    background: #9ab0ff;
  }
</style>
