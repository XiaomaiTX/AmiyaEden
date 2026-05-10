import { useEffect, useMemo, useState } from 'react'
import { fetchEveSSOScopes, fetchMyCharacters, getEveBindURL } from '@/api/auth'
import { Button } from '@/components/ui/button'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { useI18n } from '@/i18n'

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof Error && error.message ? error.message : fallback
}

function parseScopeSet(scopesText: string) {
  return new Set(scopesText.split(' ').filter(Boolean))
}

function portraitUrl(characterId: number) {
  return `https://images.evetech.net/characters/${characterId}/portrait?size=64`
}

function coverageText(t: ReturnType<typeof useI18n>['t'], granted: number, total: number) {
  return t('infoEsiCheck.coverage', { granted, total })
}

export function InfoEsiCheckPage() {
  const { t } = useI18n()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [scopes, setScopes] = useState<Api.Auth.RegisteredScope[]>([])
  const [characters, setCharacters] = useState<Api.Auth.EveCharacter[]>([])
  const [selectedCharacterId, setSelectedCharacterId] = useState<number>(0)
  const [reauthLoading, setReauthLoading] = useState(false)

  useEffect(() => {
    let cancelled = false

    const loadData = async () => {
      setLoading(true)
      setError(null)

      try {
        const [scopesData, charactersData] = await Promise.all([
          fetchEveSSOScopes(),
          fetchMyCharacters(),
        ])

        if (cancelled) {
          return
        }

        const nextScopes = scopesData ?? []
        const nextCharacters = charactersData ?? []
        setScopes(nextScopes)
        setCharacters(nextCharacters)
        setSelectedCharacterId((current) => {
          if (current > 0 && nextCharacters.some((character) => character.character_id === current)) {
            return current
          }
          return nextCharacters[0]?.character_id ?? 0
        })
      } catch (caughtError) {
        if (!cancelled) {
          setError(getErrorMessage(caughtError, t('infoEsiCheck.loadFailed')))
          setScopes([])
          setCharacters([])
          setSelectedCharacterId(0)
        }
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    }

    void loadData()

    return () => {
      cancelled = true
    }
  }, [t])

  const selectedCharacter = useMemo(
    () => characters.find((character) => character.character_id === selectedCharacterId) ?? null,
    [characters, selectedCharacterId]
  )

  const invalidCharacters = useMemo(() => {
    return characters.filter((character) => {
      if (character.token_invalid) {
        return true
      }
      const scopeSet = parseScopeSet(character.scopes)
      return scopes.some((scope) => scope.required && !scopeSet.has(scope.scope))
    })
  }, [characters, scopes])

  const scopeRows = useMemo(() => {
    if (!selectedCharacter) {
      return []
    }
    const authorizedScopes = selectedCharacter.token_invalid ? new Set<string>() : parseScopeSet(selectedCharacter.scopes)

    return scopes.map((scope) => ({
      ...scope,
      authorized: authorizedScopes.has(scope.scope),
    }))
  }, [scopes, selectedCharacter])

  const requiredScopes = useMemo(() => scopes.filter((scope) => scope.required), [scopes])
  const grantedRequiredCount = useMemo(() => {
    if (!selectedCharacter || selectedCharacter.token_invalid) {
      return 0
    }
    const scopeSet = parseScopeSet(selectedCharacter.scopes)
    return requiredScopes.filter((scope) => scopeSet.has(scope.scope)).length
  }, [requiredScopes, selectedCharacter])
  const hasMissingRequiredScopes = useMemo(
    () => scopeRows.some((row) => row.required && !row.authorized),
    [scopeRows]
  )

  const handleReauth = async () => {
    setReauthLoading(true)
    try {
      const url = await getEveBindURL()
      window.location.href = url
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('infoEsiCheck.reauthFailed')))
      setReauthLoading(false)
    }
  }

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <h1 className="text-xl font-semibold">{t('nav.info.esiCheck')}</h1>
        <p className="mt-1 text-sm text-muted-foreground">{t('infoEsiCheck.subtitle')}</p>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}

      <div className="rounded-lg border bg-card p-5">
        <div className="space-y-3">
          <h2 className="text-lg font-semibold">{t('infoEsiCheck.overview')}</h2>
          {loading ? (
            <p className="text-sm text-muted-foreground">{t('infoEsiCheck.loading')}</p>
          ) : characters.length === 0 ? (
            <p className="text-sm text-muted-foreground">{t('infoEsiCheck.noCharacters')}</p>
          ) : (
            <div className="flex flex-wrap items-center gap-3">
              <span className="text-sm text-muted-foreground">
                {t('infoEsiCheck.allCharactersCount', { count: characters.length })}
              </span>
              {invalidCharacters.length === 0 ? (
                <span className="rounded-full border border-emerald-200 bg-emerald-50 px-3 py-1 text-sm text-emerald-700 dark:border-emerald-500/30 dark:bg-emerald-500/10 dark:text-emerald-300">
                  {t('infoEsiCheck.allValid')}
                </span>
              ) : (
                <>
                  <span className="rounded-full border border-red-200 bg-red-50 px-3 py-1 text-sm text-red-700 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-300">
                    {t('infoEsiCheck.invalidCount', { count: invalidCharacters.length })}
                  </span>
                  {invalidCharacters.map((character) => (
                    <button
                      key={character.character_id}
                      type="button"
                      className="rounded-full border border-amber-200 bg-amber-50 px-3 py-1 text-sm text-amber-700 hover:bg-amber-100 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-300"
                      onClick={() => setSelectedCharacterId(character.character_id)}
                    >
                      {character.character_name}
                    </button>
                  ))}
                </>
              )}
            </div>
          )}
        </div>
      </div>

      <div className="rounded-lg border bg-card p-5">
        <div className="space-y-4">
          <h2 className="text-lg font-semibold">{t('infoEsiCheck.detail')}</h2>
          {characters.length === 0 ? (
            <p className="text-sm text-muted-foreground">{t('infoEsiCheck.noCharacters')}</p>
          ) : (
            <>
              <div className="flex flex-wrap items-center gap-3">
                <label className="space-y-1">
                  <span className="text-sm text-muted-foreground">{t('infoEsiCheck.selectCharacter')}</span>
                  <select
                    className="h-10 min-w-[240px] rounded-md border border-input bg-background px-3 text-sm"
                    value={selectedCharacterId}
                    onChange={(event) => setSelectedCharacterId(Number(event.target.value))}
                  >
                    {characters.map((character) => (
                      <option key={character.character_id} value={character.character_id}>
                        {character.character_name}
                      </option>
                    ))}
                  </select>
                </label>

                {selectedCharacter ? (
                  <div className="flex items-center gap-3 rounded-lg border bg-background px-3 py-2">
                    <Avatar className="size-10">
                      <AvatarImage src={portraitUrl(selectedCharacter.character_id)} alt={selectedCharacter.character_name} />
                      <AvatarFallback>{selectedCharacter.character_name.slice(0, 2).toUpperCase()}</AvatarFallback>
                    </Avatar>
                    <div className="text-sm">
                      <div className="font-medium">{selectedCharacter.character_name}</div>
                      <div className="text-muted-foreground">
                        {coverageText(t, grantedRequiredCount, requiredScopes.length)}
                      </div>
                    </div>
                  </div>
                ) : null}

                <Button type="button" variant="outline" onClick={() => void handleReauth()} disabled={reauthLoading}>
                  {t('infoEsiCheck.reauth')}
                </Button>
              </div>

              {selectedCharacter?.token_invalid ? (
                <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-300">
                  <div className="font-semibold">{t('infoEsiCheck.tokenInvalid')}</div>
                  <div className="mt-1">{t('infoEsiCheck.tokenInvalidTip')}</div>
                </div>
              ) : null}

              {selectedCharacter && hasMissingRequiredScopes ? (
                <div className="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-300">
                  {t('infoEsiCheck.reauthTip')}
                </div>
              ) : null}

              <div className="overflow-hidden rounded-lg border">
                <table className="min-w-full text-sm">
                  <thead>
                    <tr className="border-b bg-muted/40 text-left">
                      <th className="px-3 py-2">{t('infoEsiCheck.scope')}</th>
                      <th className="px-3 py-2">{t('infoEsiCheck.description')}</th>
                      <th className="px-3 py-2">{t('infoEsiCheck.module')}</th>
                      <th className="px-3 py-2 text-center">{t('infoEsiCheck.required')}</th>
                      <th className="px-3 py-2 text-center">{t('infoEsiCheck.authorized')}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {scopeRows.map((row) => (
                      <tr key={row.scope} className="border-b">
                        <td className="px-3 py-2 font-mono text-xs">{row.scope}</td>
                        <td className="px-3 py-2">{row.description}</td>
                        <td className="px-3 py-2">{row.module}</td>
                        <td className="px-3 py-2 text-center">
                          <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${row.required ? 'bg-red-100 text-red-700 dark:bg-red-500/10 dark:text-red-300' : 'bg-slate-100 text-slate-700 dark:bg-slate-500/10 dark:text-slate-300'}`}>
                            {row.required ? t('infoEsiCheck.required') : t('infoEsiCheck.optional')}
                          </span>
                        </td>
                        <td className="px-3 py-2 text-center">
                          {row.authorized ? (
                            <span className="font-bold text-emerald-600">✓</span>
                          ) : (
                            <span className="font-bold text-destructive">✗</span>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </>
          )}
        </div>
      </div>
    </section>
  )
}
