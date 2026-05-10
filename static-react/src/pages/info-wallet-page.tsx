import { useEffect, useMemo, useState } from 'react'
import { fetchMyCharacters } from '@/api/auth'
import { fetchInfoWallet } from '@/api/eve-info'
import { useI18n } from '@/i18n'
import type { EveCharacter } from '@/types/api/auth'
import type { WalletResponse } from '@/types/api/eve-info'

const PAGE_SIZE = 50

export function InfoWalletPage() {
  const { t } = useI18n()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [characters, setCharacters] = useState<EveCharacter[]>([])
  const [selectedCharacterId, setSelectedCharacterId] = useState<number | null>(null)
  const [selectedRefType, setSelectedRefType] = useState('')
  const [wallet, setWallet] = useState<WalletResponse | null>(null)

  const refTypeOptions = useMemo(() => wallet?.ref_types ?? [], [wallet?.ref_types])

  useEffect(() => {
    let cancelled = false

    const loadCharacters = async () => {
      setLoading(true)
      setError(null)
      try {
        const list = await fetchMyCharacters()
        if (cancelled) return
        setCharacters(list)
        if (list.length > 0) {
          setSelectedCharacterId(list[0].character_id)
        } else {
          setLoading(false)
        }
      } catch {
        if (!cancelled) {
          setError(t('infoWallet.loadCharactersFailed'))
          setLoading(false)
        }
      }
    }

    void loadCharacters()
    return () => {
      cancelled = true
    }
  }, [t])

  useEffect(() => {
    if (!selectedCharacterId) return
    let cancelled = false

    const loadWallet = async () => {
      setLoading(true)
      setError(null)
      try {
        const data = await fetchInfoWallet({
          character_id: selectedCharacterId,
          page: 1,
          page_size: PAGE_SIZE,
          ref_types: selectedRefType ? [selectedRefType] : undefined,
        })
        if (!cancelled) setWallet(data)
      } catch {
        if (!cancelled) setError(t('infoWallet.loadWalletFailed'))
      } finally {
        if (!cancelled) setLoading(false)
      }
    }

    void loadWallet()
    return () => {
      cancelled = true
    }
  }, [selectedCharacterId, selectedRefType, t])

  return (
    <section className="space-y-4">
      <h1 className="text-xl font-semibold">{t('infoWallet.title')}</h1>

      <div className="flex flex-wrap items-center gap-3 rounded-lg border bg-card p-4">
        <label className="text-sm text-muted-foreground" htmlFor="wallet-character">
          {t('infoWallet.selectCharacter')}
        </label>
        <select
          id="wallet-character"
          className="rounded border px-2 py-1 text-sm"
          value={selectedCharacterId ?? ''}
          onChange={(event) => setSelectedCharacterId(Number(event.target.value))}
        >
          {characters.map((character) => (
            <option key={character.character_id} value={character.character_id}>
              {character.character_name}
            </option>
          ))}
        </select>

        <label className="text-sm text-muted-foreground" htmlFor="wallet-ref-type">
          {t('infoWallet.refType')}
        </label>
        <select
          id="wallet-ref-type"
          className="rounded border px-2 py-1 text-sm"
          value={selectedRefType}
          onChange={(event) => setSelectedRefType(event.target.value)}
        >
          <option value="">{t('infoWallet.allRefTypes')}</option>
          {refTypeOptions.map((refType) => (
            <option key={refType} value={refType}>
              {refType}
            </option>
          ))}
        </select>
      </div>

      {loading ? <p className="text-sm">{t('infoWallet.loading')}</p> : null}
      {error ? <p className="text-sm text-destructive">{error}</p> : null}

      {wallet ? (
        <>
          <div className="rounded-lg border bg-card p-4">
            <p className="text-sm text-muted-foreground">{t('infoWallet.balance')}</p>
            <p className="mt-1 text-2xl font-semibold">{Intl.NumberFormat().format(wallet.balance)} ISK</p>
          </div>

          <div className="overflow-x-auto rounded-lg border bg-card">
            <table className="min-w-full text-sm">
              <thead>
                <tr className="border-b bg-muted/40 text-left">
                  <th className="px-3 py-2">{t('infoWallet.columns.date')}</th>
                  <th className="px-3 py-2">{t('infoWallet.columns.refType')}</th>
                  <th className="px-3 py-2">{t('infoWallet.columns.amount')}</th>
                  <th className="px-3 py-2">{t('infoWallet.columns.balance')}</th>
                  <th className="px-3 py-2">{t('infoWallet.columns.description')}</th>
                </tr>
              </thead>
              <tbody>
                {wallet.journals.map((row) => (
                  <tr key={row.id} className="border-b">
                    <td className="px-3 py-2">{row.date}</td>
                    <td className="px-3 py-2">{row.ref_type}</td>
                    <td className="px-3 py-2">{row.amount}</td>
                    <td className="px-3 py-2">{row.balance}</td>
                    <td className="px-3 py-2">{row.description}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      ) : null}
    </section>
  )
}
