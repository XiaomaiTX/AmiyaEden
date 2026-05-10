import { useCallback, useEffect, useMemo, useState } from 'react'
import { fetchGetUserInfo, getEveBindURL, setPrimaryCharacter, unbindCharacter, updateMyProfile } from '@/api/auth'
import {
  checkDirectReferrerQQ,
  confirmDirectReferrer,
  fetchDirectReferralStatus,
} from '@/api/newbro'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { buildEveCharacterPortraitUrl } from '@/lib/eve-image'
import { useI18n } from '@/i18n'
import type { EveCharacter, UserInfo } from '@/types/api/auth'
import type { DirectReferralStatus, DirectReferrerCandidate } from '@/types/api/newbro'
import { useSessionStore } from '@/stores'

const CORP_KM_SCOPE = 'esi-killmails.read_corporation_killmails.v1'
const MAX_TEXT_LENGTH = 20

function getTextLength(value: string) {
  return Array.from(value.trim()).length
}

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof Error && error.message ? error.message : fallback
}

export function DashboardCharactersPage() {
  const { t } = useI18n()
  const setSessionSnapshot = useSessionStore((state) => state.setSessionSnapshot)
  const roles = useSessionStore((state) => state.roles)

  const [loading, setLoading] = useState(true)
  const [notice, setNotice] = useState<{ kind: 'error' | 'success'; text: string } | null>(null)
  const [characters, setCharacters] = useState<EveCharacter[]>([])
  const [primaryCharacterId, setPrimaryCharacterId] = useState(0)
  const [profileComplete, setProfileComplete] = useState(false)
  const [enforceCharacterESIRestriction, setEnforceCharacterESIRestriction] = useState(true)
  const [profileForm, setProfileForm] = useState({
    nickname: '',
    qq: '',
    discordId: '',
  })
  const [profileSaving, setProfileSaving] = useState(false)
  const [bindLoading, setBindLoading] = useState(false)
  const [switchingId, setSwitchingId] = useState<number | null>(null)
  const [unbindingId, setUnbindingId] = useState<number | null>(null)
  const [directReferralLoading, setDirectReferralLoading] = useState(false)
  const [directReferralStatus, setDirectReferralStatus] = useState<DirectReferralStatus>({
    show_card: false,
    needs_profile_qq: false,
  })
  const [directReferrerQQ, setDirectReferrerQQ] = useState('')
  const [checkedDirectReferrerQQ, setCheckedDirectReferrerQQ] = useState('')
  const [directReferrerCandidate, setDirectReferrerCandidate] =
    useState<DirectReferrerCandidate | null>(null)
  const [directReferralChecking, setDirectReferralChecking] = useState(false)
  const [directReferralConfirming, setDirectReferralConfirming] = useState(false)

  const canManageCorpKm = useMemo(
    () => roles.some((role) => role === 'super_admin' || role === 'admin'),
    [roles]
  )

  const hasInvalidCharacterToken = useMemo(
    () => characters.some((character) => character.token_invalid),
    [characters]
  )

  const hasInvalidPrimaryCharacterToken = useMemo(
    () =>
      characters.some(
        (character) => character.character_id === primaryCharacterId && character.token_invalid
      ),
    [characters, primaryCharacterId]
  )

  const showTokenHealthAlert =
    hasInvalidPrimaryCharacterToken || (enforceCharacterESIRestriction && hasInvalidCharacterToken)

  const profileValidation = useMemo(() => {
    const nickname = profileForm.nickname.trim()
    const qq = profileForm.qq.trim()
    const discordId = profileForm.discordId.trim()

    if (!nickname) {
      return t('characters.profile.validation.nicknameRequired')
    }
    if (getTextLength(nickname) > MAX_TEXT_LENGTH) {
      return t('characters.profile.validation.nicknameLength')
    }
    if (getTextLength(qq) > MAX_TEXT_LENGTH) {
      return t('characters.profile.validation.qqLength')
    }
    if (qq && !/^\d+$/.test(qq)) {
      return t('characters.profile.validation.qqDigits')
    }
    if (getTextLength(discordId) > MAX_TEXT_LENGTH) {
      return t('characters.profile.validation.discordLength')
    }
    if (!qq && !discordId) {
      return t('characters.profile.validation.contactRequired')
    }

    return ''
  }, [profileForm.discordId, profileForm.nickname, profileForm.qq, t])

  const hasCorpKmScope = (character: EveCharacter) =>
    character.scopes?.split(' ').includes(CORP_KM_SCOPE) ?? false

  const applyUserInfo = useCallback((userInfo: UserInfo) => {
    setSessionSnapshot({
      isLoggedIn: true,
      characterId: userInfo.primaryCharacterId ?? null,
      characterName: userInfo.userName,
      roles: userInfo.roles,
      isCurrentlyNewbro: userInfo.isCurrentlyNewbro === true,
      isMentorMenteeEligible: userInfo.isMentorMenteeEligible === true,
    })
    setCharacters(userInfo.characters ?? [])
    setPrimaryCharacterId(userInfo.primaryCharacterId ?? 0)
    setProfileComplete(userInfo.profileComplete)
    setEnforceCharacterESIRestriction(userInfo.enforceCharacterESIRestriction)
    setProfileForm({
      nickname: userInfo.nickname,
      qq: userInfo.qq,
      discordId: userInfo.discordId,
    })
  }, [setSessionSnapshot])

  const syncUserInfo = useCallback(async () => {
    const userInfo = await fetchGetUserInfo()
    applyUserInfo(userInfo)
    return userInfo
  }, [applyUserInfo])

  const loadDirectReferralStatus = useCallback(async () => {
    setDirectReferralLoading(true)
    try {
      const status = await fetchDirectReferralStatus()
      setDirectReferralStatus(status)
      if (!status.show_card || status.needs_profile_qq) {
        setDirectReferrerCandidate(null)
        setCheckedDirectReferrerQQ('')
        setDirectReferrerQQ('')
      }
    } catch {
      setDirectReferralStatus({ show_card: false, needs_profile_qq: false })
      setDirectReferrerCandidate(null)
      setCheckedDirectReferrerQQ('')
      setDirectReferrerQQ('')
    } finally {
      setDirectReferralLoading(false)
    }
  }, [])

  useEffect(() => {
    let cancelled = false

    const load = async () => {
      setLoading(true)
      setNotice(null)
      try {
        const userInfo = await fetchGetUserInfo()
        if (cancelled) return
        applyUserInfo(userInfo)
      } catch (error) {
        if (!cancelled) {
          setNotice({ kind: 'error', text: getErrorMessage(error, t('characters.loadFailed')) })
        }
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    }

    void load()
    return () => {
      cancelled = true
    }
  }, [applyUserInfo, t])

  useEffect(() => {
    void loadDirectReferralStatus()
  }, [loadDirectReferralStatus])

  useEffect(() => {
    if (directReferrerQQ.trim() === checkedDirectReferrerQQ) {
      return
    }
    setDirectReferrerCandidate(null)
    setCheckedDirectReferrerQQ('')
  }, [checkedDirectReferrerQQ, directReferrerQQ])

  const handleRefresh = async () => {
    setNotice(null)
    try {
      await syncUserInfo()
      await loadDirectReferralStatus()
    } catch (error) {
      setNotice({ kind: 'error', text: getErrorMessage(error, t('characters.loadFailed')) })
    }
  }

  const handleSaveProfile = async () => {
    if (profileValidation) {
      setNotice({ kind: 'error', text: profileValidation })
      return
    }

    setProfileSaving(true)
    setNotice(null)
    try {
      await updateMyProfile({
        nickname: profileForm.nickname.trim(),
        qq: profileForm.qq.trim(),
        discord_id: profileForm.discordId.trim(),
      })
      await syncUserInfo()
      await loadDirectReferralStatus()
      setNotice({ kind: 'success', text: t('characters.profile.saveSuccess') })
    } catch (error) {
      setNotice({ kind: 'error', text: getErrorMessage(error, t('characters.profile.saveFailed')) })
    } finally {
      setProfileSaving(false)
    }
  }

  const handleBind = async () => {
    setBindLoading(true)
    setNotice(null)
    try {
      const url = await getEveBindURL()
      window.location.assign(url)
    } catch (error) {
      setBindLoading(false)
      setNotice({ kind: 'error', text: getErrorMessage(error, t('characters.bindFailed')) })
    }
  }

  const handleEnableCorpKm = async () => {
    setNotice(null)
    try {
      const url = await getEveBindURL([CORP_KM_SCOPE])
      window.location.assign(url)
    } catch (error) {
      setNotice({ kind: 'error', text: getErrorMessage(error, t('characters.corpKm.enableFailed')) })
    }
  }

  const handleSetPrimary = async (character: EveCharacter) => {
    setSwitchingId(character.character_id)
    setNotice(null)
    try {
      await setPrimaryCharacter(character.character_id)
      await syncUserInfo()
      setNotice({ kind: 'success', text: t('characters.setPrimarySuccess', { name: character.character_name }) })
    } catch (error) {
      setNotice({ kind: 'error', text: getErrorMessage(error, t('characters.setPrimaryFailed')) })
    } finally {
      setSwitchingId(null)
    }
  }

  const handleUnbind = async (character: EveCharacter) => {
    const confirmed = window.confirm(t('characters.unbindConfirm', { name: character.character_name }))
    if (!confirmed) {
      return
    }

    setUnbindingId(character.character_id)
    setNotice(null)
    try {
      await unbindCharacter(character.character_id)
      await syncUserInfo()
      setNotice({ kind: 'success', text: t('characters.unbindSuccess', { name: character.character_name }) })
    } catch (error) {
      setNotice({ kind: 'error', text: getErrorMessage(error, t('characters.unbindFailed')) })
    } finally {
      setUnbindingId(null)
    }
  }

  const handleCheckDirectReferrer = async () => {
    const qq = directReferrerQQ.trim()
    if (!qq) {
      setNotice({ kind: 'error', text: t('characters.directReferral.qqRequired') })
      return
    }

    setDirectReferralChecking(true)
    setDirectReferrerCandidate(null)
    setCheckedDirectReferrerQQ('')
    setNotice(null)
    try {
      const candidate = await checkDirectReferrerQQ({ qq })
      setDirectReferrerCandidate(candidate)
      setCheckedDirectReferrerQQ(qq)
      setNotice({ kind: 'success', text: t('characters.directReferral.checkSuccess') })
    } catch (error) {
      setNotice({
        kind: 'error',
        text: getErrorMessage(error, t('characters.directReferral.checkFailed')),
      })
    } finally {
      setDirectReferralChecking(false)
    }
  }

  const handleConfirmDirectReferrer = async () => {
    if (!directReferrerCandidate) {
      return
    }

    setDirectReferralConfirming(true)
    setNotice(null)
    try {
      await confirmDirectReferrer({ referrer_user_id: directReferrerCandidate.user_id })
      setDirectReferrerQQ('')
      setDirectReferrerCandidate(null)
      setCheckedDirectReferrerQQ('')
      await loadDirectReferralStatus()
      setNotice({ kind: 'success', text: t('characters.directReferral.confirmSuccess') })
    } catch (error) {
      setNotice({
        kind: 'error',
        text: getErrorMessage(error, t('characters.directReferral.confirmFailed')),
      })
    } finally {
      setDirectReferralConfirming(false)
    }
  }

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div className="max-w-2xl">
            <h1 className="text-xl font-semibold">{t('characters.title')}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t('characters.subTitle')}</p>
          </div>
          <div className="flex items-center gap-2">
            <span
              className={[
                'rounded-full px-3 py-1 text-xs font-medium',
                profileComplete
                  ? 'bg-emerald-500/10 text-emerald-600'
                  : 'bg-amber-500/10 text-amber-600',
              ].join(' ')}
            >
              {profileComplete ? t('characters.profile.completed') : t('characters.profile.incomplete')}
            </span>
            <Button type="button" variant="outline" onClick={() => void handleRefresh()}>
              {t('common.refresh')}
            </Button>
          </div>
        </div>

        <div
          className={[
            'mt-4 rounded-lg border px-4 py-3 text-sm',
            profileComplete
              ? 'border-emerald-500/20 bg-emerald-500/5 text-emerald-700'
              : 'border-amber-500/20 bg-amber-500/5 text-amber-700',
          ].join(' ')}
        >
          {profileComplete ? t('characters.profile.completedHint') : t('characters.profile.requiredHint')}
        </div>

        {notice ? (
          <div
            className={[
              'mt-4 rounded-lg border px-4 py-3 text-sm',
              notice.kind === 'success'
                ? 'border-emerald-500/20 bg-emerald-500/5 text-emerald-700'
                : 'border-destructive/20 bg-destructive/5 text-destructive',
            ].join(' ')}
          >
            {notice.text}
          </div>
        ) : null}

        <div className="mt-6 grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          <label className="space-y-2">
            <span className="text-sm font-medium">{t('characters.profile.nickname')}</span>
            <Input
              value={profileForm.nickname}
              maxLength={MAX_TEXT_LENGTH}
              onChange={(event) =>
                setProfileForm((current) => ({ ...current, nickname: event.target.value }))
              }
              placeholder={t('characters.profile.nicknamePlaceholder')}
            />
          </label>

          <label className="space-y-2">
            <span className="text-sm font-medium">{t('characters.profile.qq')}</span>
            <Input
              value={profileForm.qq}
              maxLength={MAX_TEXT_LENGTH}
              onChange={(event) => setProfileForm((current) => ({ ...current, qq: event.target.value }))}
              placeholder={t('characters.profile.qqPlaceholder')}
            />
          </label>

          <label className="space-y-2">
            <span className="text-sm font-medium">{t('characters.profile.discordId')}</span>
            <Input
              value={profileForm.discordId}
              maxLength={MAX_TEXT_LENGTH}
              onChange={(event) =>
                setProfileForm((current) => ({ ...current, discordId: event.target.value }))
              }
              placeholder={t('characters.profile.discordPlaceholder')}
            />
          </label>
        </div>

        <div className="mt-4 flex justify-end">
          <Button type="button" onClick={() => void handleSaveProfile()} disabled={profileSaving}>
            {profileSaving ? t('characters.profile.saving') : t('characters.profile.save')}
          </Button>
        </div>
      </div>

      {directReferralStatus.show_card ? (
        <div className="rounded-lg border bg-card p-5">
          <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
            <div className="max-w-2xl">
              <h2 className="text-lg font-semibold">{t('characters.directReferral.title')}</h2>
              <p className="mt-1 text-sm text-muted-foreground">{t('characters.directReferral.subtitle')}</p>
            </div>
            <span className="rounded-full bg-sky-500/10 px-3 py-1 text-xs font-medium text-sky-600">
              {t('characters.directReferral.windowTag')}
            </span>
          </div>

          {directReferralLoading ? (
            <p className="mt-4 text-sm text-muted-foreground">{t('characters.directReferral.loading')}</p>
          ) : null}

          {directReferralStatus.needs_profile_qq ? (
            <div className="mt-4 rounded-lg border border-amber-500/20 bg-amber-500/5 px-4 py-3 text-sm text-amber-700">
              {t('characters.directReferral.fillProfileQQFirst')}
            </div>
          ) : (
            <>
              <div className="mt-6 flex flex-col gap-3 lg:flex-row lg:items-end">
                <label className="flex-1 space-y-2">
                  <span className="text-sm font-medium">{t('characters.directReferral.referrerQQ')}</span>
                  <Input
                    value={directReferrerQQ}
                    maxLength={MAX_TEXT_LENGTH}
                    onChange={(event) => setDirectReferrerQQ(event.target.value)}
                    placeholder={t('characters.directReferral.referrerQQPlaceholder')}
                  />
                </label>

                <Button
                  type="button"
                  variant="outline"
                  onClick={() => void handleCheckDirectReferrer()}
                  disabled={directReferralChecking}
                >
                  {directReferralChecking ? t('characters.directReferral.checking') : t('characters.directReferral.checkBtn')}
                </Button>
              </div>

              <p className="mt-3 text-sm text-muted-foreground">{t('characters.directReferral.confirmHint')}</p>

              {directReferrerCandidate ? (
                <div className="mt-4 rounded-lg border border-primary/20 bg-primary/5 p-4">
                  <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
                    <div className="flex items-center gap-4">
                      <img
                        src={buildEveCharacterPortraitUrl(directReferrerCandidate.primary_character_id, 64)}
                        alt={directReferrerCandidate.nickname}
                        className="size-14 rounded-full border object-cover"
                      />
                      <div className="min-w-0">
                        <div className="truncate text-base font-medium">{directReferrerCandidate.nickname}</div>
                        <div className="mt-1 truncate text-sm text-muted-foreground">
                          {t('characters.directReferral.mainCharacter')}: {directReferrerCandidate.primary_character_name}
                        </div>
                        <div className="mt-1 truncate text-sm text-muted-foreground">
                          {t('characters.directReferral.nickname')}: {directReferrerCandidate.nickname}
                        </div>
                      </div>
                    </div>

                    <Button type="button" onClick={() => void handleConfirmDirectReferrer()} disabled={directReferralConfirming}>
                      {directReferralConfirming ? t('characters.directReferral.confirming') : t('characters.directReferral.confirmBtn')}
                    </Button>
                  </div>
                </div>
              ) : null}
            </>
          )}
        </div>
      ) : null}

      <div className="rounded-lg border bg-card p-5">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div>
            <h2 className="text-lg font-semibold">{t('characters.listTitle')}</h2>
            <p className="mt-1 text-sm text-muted-foreground">{t('characters.listSubTitle')}</p>
          </div>
          <Button type="button" onClick={() => void handleBind()} disabled={bindLoading}>
            {bindLoading ? t('characters.binding') : t('characters.bind')}
          </Button>
        </div>

        {loading ? (
          <p className="mt-4 text-sm text-muted-foreground">{t('characters.loading')}</p>
        ) : null}

        {showTokenHealthAlert ? (
          <div className="mt-4 rounded-lg border border-destructive/20 bg-destructive/5 px-4 py-3 text-sm text-destructive">
            <div className="font-medium">{t('characters.tokenHealth.title')}</div>
            <div className="mt-1">{t('characters.tokenHealth.requiredHint')}</div>
          </div>
        ) : null}

        <div className="mt-4 grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
          {characters.map((character) => {
            const isPrimary = character.character_id === primaryCharacterId

            return (
              <article
                key={character.character_id}
                className={[
                  'relative flex gap-4 rounded-lg border p-4',
                  isPrimary ? 'border-primary bg-primary/5' : 'border-border bg-background',
                ].join(' ')}
              >
                {isPrimary ? (
                  <span className="absolute -right-2 -top-2 rounded-full bg-primary px-2 py-0.5 text-xs font-medium text-primary-foreground">
                    {t('characters.primary')}
                  </span>
                ) : null}

                <img
                  src={buildEveCharacterPortraitUrl(character.character_id, 128)}
                  alt={character.character_name}
                  className={[
                    'size-14 rounded-full border object-cover',
                    character.token_invalid ? 'border-red-500' : 'border-border',
                  ].join(' ')}
                />

                <div className="min-w-0 flex-1">
                  <h3 className="truncate text-base font-medium">
                    {character.character_name}
                    {character.token_invalid ? (
                      <span className="ml-1 text-sm text-destructive">{t('characters.tokenInvalid')}</span>
                    ) : null}
                  </h3>
                  <p className="mt-0.5 text-xs text-muted-foreground">ID: {character.character_id}</p>
                  <p className="mt-0.5 truncate text-xs text-muted-foreground" title={character.scopes}>
                    {character.scopes
                      ? `${character.scopes.split(' ').filter(Boolean).length} ${t('characters.scopeCount')}`
                      : t('characters.noScopes')}
                  </p>

                  <div className="mt-3 flex flex-wrap gap-2">
                    {!isPrimary ? (
                      <Button
                        type="button"
                        size="sm"
                        variant="outline"
                        onClick={() => void handleSetPrimary(character)}
                        disabled={switchingId === character.character_id}
                      >
                        {switchingId === character.character_id
                          ? t('characters.setPrimaryLoading')
                          : t('characters.setPrimary')}
                      </Button>
                    ) : null}

                    {characters.length > 1 ? (
                      <Button
                        type="button"
                        size="sm"
                        variant="destructive"
                        onClick={() => void handleUnbind(character)}
                        disabled={unbindingId === character.character_id}
                      >
                        {unbindingId === character.character_id ? t('characters.unbinding') : t('characters.unbind')}
                      </Button>
                    ) : null}

                    {canManageCorpKm ? (
                      hasCorpKmScope(character) ? (
                        <span className="rounded-full bg-emerald-500/10 px-2 py-1 text-xs font-medium text-emerald-600">
                          {t('characters.corpKm.enabled')}
                        </span>
                      ) : (
                        <Button type="button" size="sm" variant="outline" onClick={() => void handleEnableCorpKm()}>
                          {t('characters.corpKm.enable')}
                        </Button>
                      )
                    ) : null}
                  </div>
                </div>
              </article>
            )
          })}
        </div>

        {!loading && characters.length === 0 ? (
          <p className="mt-4 text-sm text-muted-foreground">{t('characters.empty')}</p>
        ) : null}
      </div>
    </section>
  )
}
