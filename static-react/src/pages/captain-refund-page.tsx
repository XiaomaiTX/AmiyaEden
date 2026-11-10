import { useCallback, useEffect, useState } from 'react'
import type { ColumnDef } from '@tanstack/react-table'
import {
  refundOptions,
  previewRefund,
  createRefund,
  listRefunds,
  getRefund,
  reviewRefund,
} from '@/api/captain-refund'
import type {
  RefundOptions,
  RefundApplication,
  RefundCalculation,
  RefundRequest,
} from '@/types/api/captain-refund'
import { useI18n } from '@/i18n'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { DataTable } from '@/components/ui/data-table'
import { formatTime } from '@/lib/format'
import { refundStatusKeys } from '@/pages/captain-refund-text'
import { RefundCalculationView } from '@/pages/captain-refund-calculation'

const LEDGER_PAGE_SIZE = 200

export function CaptainRefundPage({ admin = false }: { admin?: boolean }) {
  const { t } = useI18n()
  const [options, setOptions] = useState<RefundOptions | null>(null)
  const [rows, setRows] = useState<RefundApplication[]>([])
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [selected, setSelected] = useState<RefundApplication | null>(null)
  const [preview, setPreview] = useState<RefundCalculation | null>(null)
  const [captain, setCaptain] = useState('')
  const [system, setSystem] = useState('')
  const [names, setNames] = useState('')
  const [start, setStart] = useState('')
  const [end, setEnd] = useState('')
  const [reason, setReason] = useState('')
  const [busy, setBusy] = useState(false)
  const [historyLoading, setHistoryLoading] = useState(false)
  const [error, setError] = useState('')
  const [success, setSuccess] = useState('')

  const load = useCallback(async () => {
    setHistoryLoading(true)
    try {
      const data = await listRefunds(admin, page, LEDGER_PAGE_SIZE)
      setRows(data.list)
      setTotal(data.total)
    } finally {
      setHistoryLoading(false)
    }
  }, [admin, page])

  useEffect(() => {
    let active = true
    const timer = window.setTimeout(() => {
      void load().catch(() => {
        if (active) setError(t('refund.failed'))
      })
    }, 0)
    return () => {
      active = false
      window.clearTimeout(timer)
    }
  }, [load, t])

  useEffect(() => {
    if (admin) return
    let active = true
    void refundOptions()
      .then((value) => {
        if (active) {
          setOptions(value)
          setCaptain(String(value.characters[0]?.character_id ?? ''))
          setSystem(String(value.settings.allowed_system_ids[0] ?? ''))
        }
      })
      .catch(() => {
        if (active) setError(t('refund.failed'))
      })
    return () => {
      active = false
    }
  }, [admin, t])

  const run = async (fn: () => Promise<void>) => {
    if (busy) return
    setBusy(true)
    setError('')
    setSuccess('')
    try {
      await fn()
    } catch {
      setError(t('refund.failed'))
    } finally {
      setBusy(false)
    }
  }

  const invalidate = () => setPreview(null)
  const request = (): RefundRequest => ({
    captain_character_id: Number(captain),
    system_id: Number(system),
    member_names: names
      .split(/[,;\n]+/)
      .map((name) => name.trim())
      .filter(Boolean),
    start: new Date(start + 'Z').toISOString(),
    end: new Date(end + 'Z').toISOString(),
  })

  const openDetail = (row: RefundApplication) =>
    void run(async () => {
      setSelected(await getRefund(admin, row.id))
      setReason('')
    })

  const columns: ColumnDef<RefundApplication, unknown>[] = [
    { accessorKey: 'id', header: t('refund.application') },
    {
      id: 'captain',
      header: t('refund.captain'),
      cell: ({ row }) =>
        row.original.calculation.members.find((member) => member.is_captain)?.character_name ?? '—',
    },
    {
      accessorKey: 'status',
      header: t('common.status'),
      cell: ({ row }) => t(refundStatusKeys[row.original.status]),
    },
    {
      accessorKey: 'reward',
      header: t('refund.reward'),
      cell: ({ row }) => `${row.original.reward.toFixed(2)} ${t('refund.coin')}`,
    },
    {
      accessorKey: 'created_at',
      header: t('refund.submittedAt'),
      cell: ({ row }) => formatTime(row.original.created_at),
    },
    {
      id: 'operation',
      header: t('common.operation'),
      cell: ({ row }) => (
        <Button type="button" variant="outline" size="sm" isDisabled={busy} onClick={() => openDetail(row.original)}>
          {t('refund.details')}
        </Button>
      ),
    },
  ]

  return (
    <section className="space-y-4">
      <h1 className="text-xl font-semibold">{t(admin ? 'refund.manage' : 'refund.title')}</h1>
      {error && (
        <p role="alert" className="text-destructive">
          {error}
        </p>
      )}
      {success && <p role="status">{success}</p>}

      {!admin && (
        <div className="space-y-4 rounded-lg border bg-card p-5">
          <p>{t('refund.hint')}</p>
          <label className="block">
            {t('refund.captain')}
            <select
              className="ml-3 rounded border p-2"
              value={captain}
              onChange={(event) => {
                setCaptain(event.target.value)
                invalidate()
              }}
            >
              {options?.characters.map((character) => (
                <option key={character.character_id} value={character.character_id}>
                  {character.character_name}
                </option>
              ))}
            </select>
          </label>
          <label className="block">
            {t('refund.system')}
            <select
              className="ml-3 rounded border p-2"
              value={system}
              onChange={(event) => {
                setSystem(event.target.value)
                invalidate()
              }}
            >
              {options?.settings.allowed_system_ids.map((id) => (
                <option key={id} value={id}>
                  {id}
                </option>
              ))}
            </select>
          </label>
          {options?.settings.allowed_system_ids.length === 0 && (
            <p role="status">{t('refund.noSystems')}</p>
          )}
          <label className="block">
            {t('refund.members')}
            <Textarea
              value={names}
              onChange={(event) => {
                setNames(event.target.value)
                invalidate()
              }}
            />
          </label>
          <label className="block">
            {t('refund.start')}
            <Input
              type="datetime-local"
              step="1"
              value={start}
              onChange={(event) => {
                setStart(event.target.value)
                invalidate()
              }}
            />
          </label>
          <label className="block">
            {t('refund.end')}
            <Input
              type="datetime-local"
              step="1"
              value={end}
              onChange={(event) => {
                setEnd(event.target.value)
                invalidate()
              }}
            />
          </label>
          <Button
            isDisabled={busy || !captain || !system || !start || !end}
            onClick={() => void run(async () => setPreview(await previewRefund(request())))}
          >
            {t('refund.preview')}
          </Button>
          {preview && (
            <>
              <RefundCalculationView value={preview} />
              <Button
                isDisabled={busy || !preview.can_submit}
                onClick={() =>
                  void run(async () => {
                    const payload = request()
                    try {
                      const row = await createRefund(payload)
                      setSelected(row)
                      setPreview(null)
                      setSuccess(t('refund.submitted'))
                      await load()
                    } catch {
                      try {
                        const latest = await previewRefund(payload)
                        setPreview(latest)
                        setError(t(latest.conflicts.length > 0 ? 'refund.conflict' : 'refund.failed'))
                      } catch {
                        setError(t('refund.failed'))
                      }
                    }
                  })
                }
              >
                {t('refund.submit')}
              </Button>
            </>
          )}
        </div>
      )}

      <div className="space-y-3 rounded-lg border bg-card p-5">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <h2 className="text-base font-semibold">{t('refund.history')}</h2>
          <Button variant="outline" isDisabled={busy || historyLoading} onClick={() => void load()}>
            {t('common.refresh')}
          </Button>
        </div>
        <DataTable
          columns={columns}
          data={rows}
          getRowId={(row) => String(row.id)}
          loading={busy || historyLoading}
          loadingText={t('common.loading')}
          emptyText={t('common.empty')}
          variant="ledger"
          pagination={{
            page,
            pageSize: LEDGER_PAGE_SIZE,
            total,
            onPageChange: setPage,
            onPageSizeChange: (size) => {
              if (size !== LEDGER_PAGE_SIZE) return
              setPage(1)
            },
            previousLabel: t('common.previous'),
            nextLabel: t('common.next'),
            pageSizeLabel: t('common.pageSize'),
          }}
        />
      </div>

      {selected && (
        <div className="space-y-3 rounded-lg border bg-card p-5">
          <h2 className="text-base font-semibold">
            #{selected.id} · {t(refundStatusKeys[selected.status])}
          </h2>
          <RefundCalculationView value={selected.calculation} />
          <p>
            {t('refund.reviewed')}: {selected.reviewed_by || '—'} · {formatTime(selected.reviewed_at)}
          </p>
          {selected.reject_reason ? (
            <p>
              {t('refund.rejectReason')}: {selected.reject_reason}
            </p>
          ) : null}
          {admin && selected.status === 'pending' && (
            <>
              <label className="block">
                {t('refund.rejectReason')}
                <Textarea value={reason} onChange={(event) => setReason(event.target.value)} />
              </label>
              <div className="flex gap-3">
                <Button
                  isDisabled={busy}
                  onClick={() =>
                    void run(async () => {
                      await reviewRefund(selected.id, true, '')
                      setSelected(await getRefund(true, selected.id))
                      await load()
                    })
                  }
                >
                  {t('refund.approve')}
                </Button>
                <Button
                  variant="destructive"
                  isDisabled={busy || !reason.trim()}
                  onClick={() =>
                    void run(async () => {
                      await reviewRefund(selected.id, false, reason)
                      setSelected(await getRefund(true, selected.id))
                      await load()
                    })
                  }
                >
                  {t('refund.reject')}
                </Button>
              </div>
            </>
          )}
        </div>
      )}
    </section>
  )
}
