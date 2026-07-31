import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from '@/components/ui/table'
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
        <Select
          selectedKey={String(selectedCharacterId ?? '')}
          onSelectionChange={(key) => ((value) => setSelectedCharacterId(Number(value)))(String(key))}
        >
          <SelectTrigger id="wallet-character" className="h-8">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {characters.map((character) => (
              <SelectItem key={character.character_id} id={String(character.character_id ?? '')}>
                {character.character_name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <label className="text-sm text-muted-foreground" htmlFor="wallet-ref-type">
          {t('infoWallet.refType')}
        </label>
        <Select
          selectedKey={String(selectedRefType ?? '')}
          onSelectionChange={(key) => ((value) => setSelectedRefType(value))(String(key))}
        >
          <SelectTrigger id="wallet-ref-type" className="h-8">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem id="">{t('infoWallet.allRefTypes')}</SelectItem>
            {refTypeOptions.map((refType) => (
              <SelectItem key={refType} id={String(refType ?? '')}>
                {refType}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {loading ? <p className="text-sm">{t('infoWallet.loading')}</p> : null}
      {error ? <p className="text-sm text-destructive">{error}</p> : null}

      {wallet ? (
        <>
          <div className="rounded-lg border bg-card p-4">
            <p className="text-sm text-muted-foreground">{t('infoWallet.balance')}</p>
            <p className="mt-1 text-2xl font-semibold">
              {Intl.NumberFormat().format(wallet.balance)} {t('common.isk')}
            </p>
          </div>

          <div className="overflow-x-auto rounded-lg border bg-card">
            <Table className="min-w-full text-sm">
              <TableHeader>
                <TableRow className="border-b bg-muted/40 text-left">
                  <TableHead className="px-3 py-2">{t('infoWallet.columns.date')}</TableHead>
                  <TableHead className="px-3 py-2">{t('infoWallet.columns.refType')}</TableHead>
                  <TableHead className="px-3 py-2">{t('infoWallet.columns.amount')}</TableHead>
                  <TableHead className="px-3 py-2">{t('infoWallet.columns.balance')}</TableHead>
                  <TableHead className="px-3 py-2">{t('infoWallet.columns.description')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {wallet.journals.map((row) => (
                  <TableRow key={row.id} className="border-b">
                    <TableCell className="px-3 py-2">{row.date}</TableCell>
                    <TableCell className="px-3 py-2">{row.ref_type}</TableCell>
                    <TableCell className="px-3 py-2">{row.amount}</TableCell>
                    <TableCell className="px-3 py-2">{row.balance}</TableCell>
                    <TableCell className="px-3 py-2">{row.description}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </>
      ) : null}
    </section>
  )
}
