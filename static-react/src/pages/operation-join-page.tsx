import { useEffect, useMemo, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { fetchMyCharacters } from '@/api/auth'
import { joinFleet } from '@/api/fleet'
import { Button } from '@/components/ui/button'
import { useI18n } from '@/i18n'
import { buildEveCharacterPortraitUrl } from '@/lib/eve-image'
import type { EveCharacter } from '@/types/api/auth'
import { getErrorMessage } from './shop-page-utils'

export function OperationJoinPage() {
  const { t } = useI18n()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const inviteCode = searchParams.get('code') ?? ''

  const [characters, setCharacters] = useState<EveCharacter[]>([])
  const [characterId, setCharacterId] = useState<number | ''>('')
  const [loading, setLoading] = useState(false)
  const [submitLoading, setSubmitLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const selectedCharacter = useMemo(
    () => characters.find((character) => character.character_id === characterId) ?? null,
    [characterId, characters]
  )

  useEffect(() => {
    if (!inviteCode) return

    const timer = window.setTimeout(() => {
      setLoading(true)
      void fetchMyCharacters()
        .then((list) => {
          setCharacters(list ?? [])
          if ((list ?? []).length === 1) {
            setCharacterId(list[0].character_id)
          }
        })
        .catch((caughtError) => {
          setError(getErrorMessage(caughtError, t('httpMsg.requestFailed')))
          setCharacters([])
        })
        .finally(() => setLoading(false))
    }, 0)

    return () => window.clearTimeout(timer)
  }, [inviteCode, t])

  const handleJoin = async () => {
    if (!inviteCode || !characterId) {
      setError(t('fleet.join.selectCharacterPlaceholder'))
      return
    }

    setSubmitLoading(true)
    setError(null)
    try {
      await joinFleet({ code: inviteCode, character_id: Number(characterId) })
      navigate('/operation/pap', { replace: true })
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('fleet.join.invalidCode')))
    } finally {
      setSubmitLoading(false)
    }
  }

  return (
    <section className="mx-auto flex min-h-[60vh] max-w-xl items-center justify-center p-4">
      <div className="w-full rounded-lg border bg-card p-5">
        <div className="text-center">
          <h1 className="text-xl font-semibold">{t('fleet.join.title')}</h1>
        </div>

        {inviteCode ? (
          <div className="mt-4 rounded-lg border bg-muted/30 p-3 text-sm">
            <span className="text-muted-foreground">{t('fleet.invite.code')}：</span>
            <code className="break-all">{inviteCode}</code>
          </div>
        ) : null}

        {error ? <p className="mt-4 text-sm text-destructive">{error}</p> : null}

        {!inviteCode ? (
          <p className="mt-4 text-sm text-muted-foreground">{t('fleet.join.missingCode')}</p>
        ) : (
          <div className="mt-4 space-y-4">
            <label className="space-y-2">
              <span className="text-sm text-muted-foreground">{t('fleet.join.selectCharacter')}</span>
              <select
                className="h-10 w-full rounded-md border border-input bg-background px-3 text-sm"
                value={characterId}
                disabled={loading}
                onChange={(event) => setCharacterId(Number(event.target.value))}
              >
                <option value="">{t('fleet.join.selectCharacterPlaceholder')}</option>
                {characters.map((character) => (
                  <option key={character.character_id} value={character.character_id}>
                    {character.character_name}
                  </option>
                ))}
              </select>
            </label>

            {selectedCharacter ? (
              <div className="flex items-center gap-3 rounded-lg border bg-background p-3">
                <img
                  className="h-10 w-10 rounded-full object-cover"
                  alt={selectedCharacter.character_name}
                  src={buildEveCharacterPortraitUrl(selectedCharacter.character_id, 64)}
                />
                <div>
                  <div className="font-medium">{selectedCharacter.character_name}</div>
                  <div className="text-xs text-muted-foreground">{selectedCharacter.corporation_id}</div>
                </div>
              </div>
            ) : null}

            <div className="flex justify-end gap-2">
              <Button type="button" variant="outline" onClick={() => navigate(-1)}>
                {t('common.cancel')}
              </Button>
              <Button type="button" onClick={() => void handleJoin()} disabled={submitLoading || !characterId}>
                {submitLoading ? t('common.confirm') : t('common.confirm')}
              </Button>
            </div>
          </div>
        )}
      </div>
    </section>
  )
}
