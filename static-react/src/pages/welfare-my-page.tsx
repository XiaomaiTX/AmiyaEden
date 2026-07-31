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
import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  applyForWelfare,
  getEligibleWelfares,
  getMyApplications,
  uploadWelfareEvidence,
} from '@/api/welfare'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useI18n } from '@/i18n'
import type { EligibleWelfare, MyApplication } from '@/types/api/welfare'
import { ShopDialog } from './shop-page-utils'

type EligibleRow = {
  welfareId: number
  welfareName: string
  description: string
  distMode: 'per_user' | 'per_character'
  characterId?: number
  characterName?: string
  canApplyNow: boolean
  ineligibleReason?: string
  requireEvidence: boolean
  exampleEvidence: string
}

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof Error && error.message ? error.message : fallback
}

function formatTime(value: string) {
  const parsed = new Date(value)
  return Number.isNaN(parsed.getTime()) ? value : parsed.toLocaleString()
}

function flattenWelfares(welfares: EligibleWelfare[]) {
  const rows: EligibleRow[] = []
  for (const welfare of welfares) {
    const shared = {
      welfareId: welfare.id,
      welfareName: welfare.name,
      description: welfare.description,
      distMode: welfare.dist_mode,
      requireEvidence: welfare.require_evidence,
      exampleEvidence: welfare.example_evidence,
    }

    if (welfare.dist_mode === 'per_user') {
      rows.push({
        ...shared,
        canApplyNow: welfare.can_apply_now,
        ineligibleReason: welfare.ineligible_reason,
      })
      continue
    }

    for (const character of welfare.eligible_characters) {
      rows.push({
        ...shared,
        canApplyNow: character.can_apply_now,
        ineligibleReason: character.ineligible_reason,
        characterId: character.character_id,
        characterName: character.character_name,
      })
    }
  }

  return rows
}

export function WelfareMyPage() {
  const { t } = useI18n()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [eligibleRows, setEligibleRows] = useState<EligibleRow[]>([])
  const [applications, setApplications] = useState<MyApplication[]>([])
  const [appPage, setAppPage] = useState(1)
  const [appPageSize, setAppPageSize] = useState(20)
  const [appTotal, setAppTotal] = useState(0)
  const [selectedRow, setSelectedRow] = useState<EligibleRow | null>(null)
  const [evidenceUrl, setEvidenceUrl] = useState('')
  const [evidenceUploading, setEvidenceUploading] = useState(false)
  const [applying, setApplying] = useState(false)

  const loadData = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [eligible, applicationsResponse] = await Promise.all([
        getEligibleWelfares(),
        getMyApplications({ current: appPage, size: appPageSize }),
      ])
      setEligibleRows(flattenWelfares(eligible ?? []))
      setApplications(applicationsResponse.list ?? [])
      setAppTotal(applicationsResponse.total ?? 0)
      setAppPage(applicationsResponse.page ?? appPage)
      setAppPageSize(applicationsResponse.pageSize ?? appPageSize)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('welfareMy.loadFailed')))
      setEligibleRows([])
      setApplications([])
      setAppTotal(0)
    } finally {
      setLoading(false)
    }
  }, [appPage, appPageSize, t])

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadData()
    }, 0)
    return () => window.clearTimeout(timer)
  }, [loadData])

  const appPageCount = useMemo(
    () => Math.max(1, Math.ceil(appTotal / appPageSize) || 1),
    [appPageSize, appTotal]
  )

  const handleEvidenceUpload = async (file: File) => {
    setEvidenceUploading(true)
    try {
      const url = await uploadWelfareEvidence(file)
      setEvidenceUrl(url)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('welfareMy.uploadFailed')))
    } finally {
      setEvidenceUploading(false)
    }
  }

  const submitApply = async (row: EligibleRow) => {
    setApplying(true)
    try {
      await applyForWelfare({
        welfare_id: row.welfareId,
        character_id: row.characterId,
        evidence_image: row.requireEvidence ? evidenceUrl : undefined,
      })
      setSelectedRow(null)
      setEvidenceUrl('')
      await loadData()
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('welfareMy.applyFailed')))
    } finally {
      setApplying(false)
    }
  }

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <h1 className="text-xl font-semibold">{t('welfareMy.title')}</h1>
        <p className="mt-1 text-sm text-muted-foreground">{t('welfareMy.subtitle')}</p>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {loading ? <p className="text-sm text-muted-foreground">{t('welfareMy.loading')}</p> : null}

      <div className="grid gap-4 xl:grid-cols-2">
        <div className="rounded-lg border bg-card p-5">
          <div className="flex items-center justify-between gap-3">
            <h2 className="text-lg font-semibold">{t('welfareMy.eligibleTitle')}</h2>
            <Button type="button" variant="outline" size="sm" onClick={() => void loadData()}>
              {t('common.refresh')}
            </Button>
          </div>

          <div className="mt-4 space-y-3">
            {eligibleRows.map((row) => (
              <article
                key={`${row.welfareId}-${row.characterId ?? 'user'}`}
                className="rounded-lg border bg-background p-4"
              >
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div className="space-y-1">
                    <h3 className="font-medium">{row.welfareName}</h3>
                    <p className="text-sm text-muted-foreground">{row.description}</p>
                    <p className="text-xs text-muted-foreground">
                      {t('welfareMy.deliveryMode')}: {row.distMode}
                    </p>
                    {row.characterName ? (
                      <p className="text-xs text-muted-foreground">
                        {t('welfareMy.characterName')}: {row.characterName}
                      </p>
                    ) : null}
                    {!row.canApplyNow && row.ineligibleReason ? (
                      <p className="text-xs text-amber-600">{row.ineligibleReason}</p>
                    ) : null}
                  </div>
                  <Button
                    type="button"
                    isDisabled={!row.canApplyNow}
                    onClick={() => {
                      setSelectedRow(row)
                      setEvidenceUrl('')
                    }}
                  >
                    {t('welfareMy.applyBtn')}
                  </Button>
                </div>
              </article>
            ))}
            {!loading && eligibleRows.length === 0 ? (
              <p className="text-sm text-muted-foreground">{t('welfareMy.noEligibleWelfares')}</p>
            ) : null}
          </div>
        </div>

        <div className="rounded-lg border bg-card p-5">
          <h2 className="text-lg font-semibold">{t('welfareMy.applicationsTitle')}</h2>
          <div className="mt-4 overflow-x-auto">
            <Table className="min-w-full text-sm">
              <TableHeader>
                <TableRow className="border-b bg-muted/40 text-left">
                  <TableHead className="px-3 py-2">{t('welfareMy.columns.welfare')}</TableHead>
                  <TableHead className="px-3 py-2">{t('welfareMy.columns.character')}</TableHead>
                  <TableHead className="px-3 py-2">{t('welfareMy.columns.status')}</TableHead>
                  <TableHead className="px-3 py-2">{t('welfareMy.columns.reviewer')}</TableHead>
                  <TableHead className="px-3 py-2">{t('welfareMy.columns.appliedAt')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {applications.map((application) => (
                  <TableRow key={application.id} className="border-b">
                    <TableCell className="px-3 py-2">{application.welfare_name}</TableCell>
                    <TableCell className="px-3 py-2">{application.character_name}</TableCell>
                    <TableCell className="px-3 py-2">{application.status}</TableCell>
                    <TableCell className="px-3 py-2">{application.reviewer_name || '-'}</TableCell>
                    <TableCell className="px-3 py-2">
                      {formatTime(application.created_at)}
                    </TableCell>
                  </TableRow>
                ))}
                {!loading && applications.length === 0 ? (
                  <TableRow>
                    <TableCell className="px-3 py-6 text-center text-muted-foreground" colSpan={5}>
                      {t('welfareMy.noApplications')}
                    </TableCell>
                  </TableRow>
                ) : null}
              </TableBody>
            </Table>
          </div>

          <div className="mt-4 flex flex-wrap items-center gap-3 text-sm">
            <span>
              {appPage}/{appPageCount}
            </span>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => setAppPage((current) => Math.max(1, current - 1))}
              isDisabled={appPage <= 1}
            >
              {t('welfareMy.pagination.prev')}
            </Button>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => setAppPage((current) => current + 1)}
              isDisabled={applications.length < appPageSize || appPage * appPageSize >= appTotal}
            >
              {t('welfareMy.pagination.next')}
            </Button>
            <label className="flex items-center gap-2">
              <span>{t('welfareMy.pageSize')}</span>
              <Select
                selectedKey={String(appPageSize ?? '')}
                onSelectionChange={(key) => ((value) => {
                  setAppPageSize(Number(value))
                  setAppPage(1)
                })(String(key))}
              >
                <SelectTrigger className="h-8">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {[10, 20, 50].map((size) => (
                    <SelectItem key={size} id={String(size ?? '')}>
                      {size}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </label>
          </div>
        </div>
      </div>

      {selectedRow ? (
        <ShopDialog
          open={selectedRow !== null}
          title={t('welfareMy.applyDialogTitle')}
          onClose={() => setSelectedRow(null)}
          closeLabel={t('common.close')}
        >
          <div className="space-y-3 text-sm">
            <p>
              <span className="font-medium">{t('welfareMy.welfareName')}:</span>{' '}
              {selectedRow.welfareName}
            </p>
            {selectedRow.characterName ? (
              <p>
                <span className="font-medium">{t('welfareMy.characterName')}:</span>{' '}
                {selectedRow.characterName}
              </p>
            ) : null}
            {selectedRow.requireEvidence ? (
              <label className="space-y-2 block">
                <span className="text-sm text-muted-foreground">
                  {t('welfareMy.evidenceImage')}
                </span>
                <Input
                  type="file"
                  accept="image/*"
                  disabled={evidenceUploading || applying}
                  onChange={(event) => {
                    const file = event.target.files?.[0]
                    if (file) void handleEvidenceUpload(file)
                  }}
                />
                {evidenceUrl ? (
                  <img src={evidenceUrl} alt="" className="max-h-40 rounded border" />
                ) : null}
              </label>
            ) : null}
            <div className="flex justify-end gap-3">
              <Button type="button" variant="outline" onClick={() => setSelectedRow(null)}>
                {t('common.cancel')}
              </Button>
              <Button
                type="button"
                onClick={() => void submitApply(selectedRow)}
                isDisabled={applying || (selectedRow.requireEvidence && !evidenceUrl)}
              >
                {applying ? t('welfareMy.applying') : t('welfareMy.confirmApply')}
              </Button>
            </div>
          </div>
        </ShopDialog>
      ) : null}
    </section>
  )
}
