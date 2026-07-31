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
import { useEffect, useState } from 'react'
import { fetchInfoContractDetail, fetchInfoContracts } from '@/api/eve-info'
import { Button } from '@/components/ui/button'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { formatIskPlain } from '@/lib/isk'
import { useI18n } from '@/i18n'
import type { ContractBidItem, ContractItem, ContractItemDetail } from '@/types/api/eve-info'

type ContractDetailState = {
  items: ContractItemDetail[]
  bids: ContractBidItem[]
}

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof Error && error.message ? error.message : fallback
}

function formatTime(value: string | undefined) {
  if (!value) return '-'
  const parsed = new Date(value)
  return Number.isNaN(parsed.getTime()) ? value : parsed.toLocaleString()
}

function contractTypeTone(type: string) {
  switch (type) {
    case 'item_exchange':
      return 'bg-sky-100 text-sky-700 dark:bg-sky-500/15 dark:text-sky-300'
    case 'auction':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300'
    case 'courier':
      return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300'
    case 'loan':
      return 'bg-blue-100 text-blue-700 dark:bg-blue-500/15 dark:text-blue-300'
    default:
      return 'bg-muted text-muted-foreground'
  }
}

function contractStatusTone(status: string) {
  switch (status) {
    case 'outstanding':
      return 'bg-sky-100 text-sky-700 dark:bg-sky-500/15 dark:text-sky-300'
    case 'in_progress':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300'
    case 'finished':
      return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300'
    case 'rejected':
    case 'failed':
      return 'bg-red-100 text-red-700 dark:bg-red-500/15 dark:text-red-300'
    default:
      return 'bg-muted text-muted-foreground'
  }
}

function formatContractTypeLabel(t: ReturnType<typeof useI18n>['t'], type: string) {
  const key = `infoContracts.types.${type}`
  const translated = t(key)
  return translated === key ? type : translated
}

function formatContractStatusLabel(t: ReturnType<typeof useI18n>['t'], status: string) {
  const key = `infoContracts.statuses.${status}`
  const translated = t(key)
  return translated === key ? status : translated
}

function ContractMeta({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border bg-background px-3 py-2">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="mt-1 text-sm font-medium">{value}</div>
    </div>
  )
}

export function InfoContractsPage() {
  const { t } = useI18n()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [contracts, setContracts] = useState<ContractItem[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [filterType, setFilterType] = useState('')
  const [filterStatus, setFilterStatus] = useState('')
  const [refreshToken, setRefreshToken] = useState(0)
  const [detailOpen, setDetailOpen] = useState(false)
  const [selectedContract, setSelectedContract] = useState<ContractItem | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)
  const [detailError, setDetailError] = useState<string | null>(null)
  const [detailData, setDetailData] = useState<ContractDetailState | null>(null)

  useEffect(() => {
    let cancelled = false

    const loadData = async () => {
      setLoading(true)
      setError(null)

      try {
        const response = await fetchInfoContracts({
          current: page,
          size: pageSize,
          type: filterType || undefined,
          status: filterStatus || undefined,
          language: 'en',
        })

        if (cancelled) {
          return
        }

        setContracts(response.list)
        setTotal(response.total)
        setPage(response.page)
        setPageSize(response.pageSize)
      } catch (caughtError) {
        if (!cancelled) {
          setError(getErrorMessage(caughtError, t('infoContracts.loadFailed')))
          setContracts([])
          setTotal(0)
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
  }, [filterStatus, filterType, page, pageSize, refreshToken, t])

  useEffect(() => {
    if (!detailOpen || !selectedContract) {
      return
    }

    let cancelled = false

    const loadDetail = async () => {
      setDetailLoading(true)
      setDetailError(null)
      setDetailData(null)

      try {
        const response = await fetchInfoContractDetail({
          character_id: selectedContract.character_id,
          contract_id: selectedContract.contract_id,
          language: 'en',
        })

        if (!cancelled) {
          setDetailData(response)
        }
      } catch (caughtError) {
        if (!cancelled) {
          setDetailError(getErrorMessage(caughtError, t('infoContracts.detailLoadFailed')))
        }
      } finally {
        if (!cancelled) {
          setDetailLoading(false)
        }
      }
    }

    void loadDetail()

    return () => {
      cancelled = true
    }
  }, [detailOpen, selectedContract, t])

  const contractRows = contracts
  const sortedBids = [...(detailData?.bids ?? [])].sort((left, right) => right.amount - left.amount)

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <h1 className="text-xl font-semibold">{t('infoContracts.title')}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t('infoContracts.subtitle')}</p>
          </div>

          <div className="flex flex-wrap items-end gap-3">
            <label className="space-y-1">
              <span className="text-sm text-muted-foreground">
                {t('infoContracts.filters.type')}
              </span>
              <Select
                selectedKey={String(filterType ?? '')}
                onSelectionChange={(key) => ((value) => {
                  setFilterType(value)
                  setPage(1)
                })(String(key))}
              >
                <SelectTrigger className="h-10">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem id="">{t('infoContracts.allTypes')}</SelectItem>
                  <SelectItem id="item_exchange">
                    {t('infoContracts.types.item_exchange')}
                  </SelectItem>
                  <SelectItem id="auction">{t('infoContracts.types.auction')}</SelectItem>
                  <SelectItem id="courier">{t('infoContracts.types.courier')}</SelectItem>
                  <SelectItem id="loan">{t('infoContracts.types.loan')}</SelectItem>
                </SelectContent>
              </Select>
            </label>

            <label className="space-y-1">
              <span className="text-sm text-muted-foreground">
                {t('infoContracts.filters.status')}
              </span>
              <Select
                selectedKey={String(filterStatus ?? '')}
                onSelectionChange={(key) => ((value) => {
                  setFilterStatus(value)
                  setPage(1)
                })(String(key))}
              >
                <SelectTrigger className="h-10">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem id="">{t('infoContracts.allStatuses')}</SelectItem>
                  <SelectItem id="outstanding">
                    {t('infoContracts.statuses.outstanding')}
                  </SelectItem>
                  <SelectItem id="in_progress">
                    {t('infoContracts.statuses.in_progress')}
                  </SelectItem>
                  <SelectItem id="finished">{t('infoContracts.statuses.finished')}</SelectItem>
                  <SelectItem id="cancelled">{t('infoContracts.statuses.cancelled')}</SelectItem>
                  <SelectItem id="rejected">{t('infoContracts.statuses.rejected')}</SelectItem>
                  <SelectItem id="failed">{t('infoContracts.statuses.failed')}</SelectItem>
                  <SelectItem id="deleted">{t('infoContracts.statuses.deleted')}</SelectItem>
                  <SelectItem id="reversed">{t('infoContracts.statuses.reversed')}</SelectItem>
                </SelectContent>
              </Select>
            </label>

            <Button
              type="button"
              variant="outline"
              onClick={() => setRefreshToken((current) => current + 1)}
              isDisabled={loading}
            >
              {t('common.refresh')}
            </Button>
          </div>
        </div>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {loading ? (
        <p className="text-sm text-muted-foreground">{t('infoContracts.loading')}</p>
      ) : null}

      <div className="overflow-hidden rounded-lg border bg-card">
        <div className="border-b px-4 py-3 text-sm font-medium">
          {t('infoContracts.title')} ({total})
        </div>
        <div className="overflow-x-auto">
          <Table className="min-w-full text-sm">
            <TableHeader>
              <TableRow className="border-b bg-muted/40 text-left">
                <TableHead className="px-3 py-2">{t('infoContracts.columns.type')}</TableHead>
                <TableHead className="px-3 py-2">{t('infoContracts.columns.status')}</TableHead>
                <TableHead className="px-3 py-2">{t('infoContracts.columns.title')}</TableHead>
                <TableHead className="px-3 py-2">{t('infoContracts.columns.price')}</TableHead>
                <TableHead className="px-3 py-2">{t('infoContracts.columns.reward')}</TableHead>
                <TableHead className="px-3 py-2">{t('infoContracts.columns.owner')}</TableHead>
                <TableHead className="px-3 py-2">{t('infoContracts.columns.expiry')}</TableHead>
                <TableHead className="px-3 py-2">{t('infoContracts.columns.actions')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {contractRows.map((contract) => (
                <TableRow key={contract.contract_id} className="border-b">
                  <TableCell className="px-3 py-2">
                    <span
                      className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${contractTypeTone(contract.type)}`}
                    >
                      {formatContractTypeLabel(t, contract.type)}
                    </span>
                  </TableCell>
                  <TableCell className="px-3 py-2">
                    <span
                      className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${contractStatusTone(contract.status)}`}
                    >
                      {formatContractStatusLabel(t, contract.status)}
                    </span>
                  </TableCell>
                  <TableCell className="px-3 py-2">
                    <div className="font-medium">
                      {contract.title ?? `#${contract.contract_id}`}
                    </div>
                    <div className="text-xs text-muted-foreground">
                      {contract.for_corporation
                        ? t('infoContracts.forCorporation')
                        : t('infoContracts.forPersonal')}
                    </div>
                  </TableCell>
                  <TableCell className="px-3 py-2">
                    {contract.price == null ? '-' : formatIskPlain(contract.price)}
                  </TableCell>
                  <TableCell className="px-3 py-2">
                    {contract.reward == null ? '-' : formatIskPlain(contract.reward)}
                  </TableCell>
                  <TableCell className="px-3 py-2">
                    <div>{contract.character_name}</div>
                    <div className="text-xs text-muted-foreground">{contract.character_id}</div>
                  </TableCell>
                  <TableCell className="px-3 py-2">{formatTime(contract.date_expired)}</TableCell>
                  <TableCell className="px-3 py-2">
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      onClick={() => {
                        setSelectedContract(contract)
                        setDetailOpen(true)
                      }}
                    >
                      {t('infoContracts.actions.viewDetail')}
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
              {!loading && contractRows.length === 0 ? (
                <TableRow>
                  <TableCell className="px-3 py-6 text-center text-muted-foreground" colSpan={8}>
                    {t('infoContracts.empty')}
                  </TableCell>
                </TableRow>
              ) : null}
            </TableBody>
          </Table>
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-3 text-sm">
        <span>
          {t('infoContracts.pagination.page')} {page}/
          {Math.max(1, Math.ceil(total / pageSize) || 1)}
        </span>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => setPage((current) => Math.max(1, current - 1))}
          isDisabled={page <= 1}
        >
          -
        </Button>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => setPage((current) => current + 1)}
          isDisabled={contractRows.length < pageSize || page * pageSize >= total}
        >
          +
        </Button>
        <label className="flex items-center gap-2">
          <span>{t('infoContracts.pageSize')}</span>
          <Select
            selectedKey={String(pageSize ?? '')}
            onSelectionChange={(key) => ((value) => {
              setPageSize(Number(value))
              setPage(1)
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

      <Sheet isOpen={detailOpen} onOpenChange={setDetailOpen}>
        <SheetContent className="w-full sm:max-w-2xl">
          <SheetHeader>
            <SheetTitle>
              {selectedContract?.title ??
                (selectedContract
                  ? `#${selectedContract.contract_id}`
                  : t('infoContracts.detailTitle'))}
            </SheetTitle>
            <SheetDescription>
              {selectedContract
                ? `${formatContractTypeLabel(t, selectedContract.type)} · ${formatContractStatusLabel(t, selectedContract.status)}`
                : t('infoContracts.detailHint')}
            </SheetDescription>
          </SheetHeader>

          <div className="space-y-4 px-4 pb-4">
            {selectedContract ? (
              <div className="grid gap-3 sm:grid-cols-2">
                <ContractMeta
                  label={t('infoContracts.detail.character')}
                  value={`${selectedContract.character_name} (${selectedContract.character_id})`}
                />
                <ContractMeta
                  label={t('infoContracts.detail.owner')}
                  value={`${selectedContract.for_corporation ? t('infoContracts.detail.corporation') : t('infoContracts.detail.personal')}`}
                />
                <ContractMeta
                  label={t('infoContracts.detail.issued')}
                  value={formatTime(selectedContract.date_issued)}
                />
                <ContractMeta
                  label={t('infoContracts.detail.expires')}
                  value={formatTime(selectedContract.date_expired)}
                />
              </div>
            ) : null}

            {detailLoading ? (
              <p className="text-sm text-muted-foreground">{t('infoContracts.detailLoading')}</p>
            ) : null}
            {detailError ? <p className="text-sm text-destructive">{detailError}</p> : null}

            {!detailLoading && !detailError && detailData?.items?.length ? (
              <div className="space-y-2">
                <h3 className="text-sm font-semibold">{t('infoContracts.detail.items')}</h3>
                <div className="overflow-hidden rounded-lg border">
                  <Table className="min-w-full text-sm">
                    <TableHeader>
                      <TableRow className="border-b bg-muted/40 text-left">
                        <TableHead className="px-3 py-2" />
                        <TableHead className="px-3 py-2">
                          {t('infoContracts.detail.itemName')}
                        </TableHead>
                        <TableHead className="px-3 py-2">
                          {t('infoContracts.detail.itemGroup')}
                        </TableHead>
                        <TableHead className="px-3 py-2">
                          {t('infoContracts.detail.itemQty')}
                        </TableHead>
                        <TableHead className="px-3 py-2">
                          {t('infoContracts.detail.itemIncluded')}
                        </TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {detailData.items.map((item) => (
                        <TableRow key={`${item.type_id}-${item.type_name}`} className="border-b">
                          <TableCell className="px-3 py-2">
                            <img
                              alt=""
                              className="h-7 w-7"
                              loading="lazy"
                              src={`https://images.evetech.net/types/${item.type_id}/icon?size=32`}
                            />
                          </TableCell>
                          <TableCell className="px-3 py-2">{item.type_name}</TableCell>
                          <TableCell className="px-3 py-2">{item.group_name}</TableCell>
                          <TableCell className="px-3 py-2 text-right">{item.quantity}</TableCell>
                          <TableCell className="px-3 py-2">
                            {item.is_included ? '✓' : '✗'}
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              </div>
            ) : null}

            {!detailLoading &&
            !detailError &&
            selectedContract?.type === 'auction' &&
            sortedBids.length > 0 ? (
              <div className="space-y-2">
                <h3 className="text-sm font-semibold">{t('infoContracts.detail.bids')}</h3>
                <div className="overflow-hidden rounded-lg border">
                  <Table className="min-w-full text-sm">
                    <TableHeader>
                      <TableRow className="border-b bg-muted/40 text-left">
                        <TableHead className="px-3 py-2">
                          {t('infoContracts.detail.bidAmount')}
                        </TableHead>
                        <TableHead className="px-3 py-2">
                          {t('infoContracts.detail.bidder')}
                        </TableHead>
                        <TableHead className="px-3 py-2">
                          {t('infoContracts.detail.bidTime')}
                        </TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {sortedBids.map((bid) => (
                        <TableRow key={bid.bid_id} className="border-b">
                          <TableCell className="px-3 py-2 font-medium">
                            {formatIskPlain(bid.amount)}
                          </TableCell>
                          <TableCell className="px-3 py-2">{bid.bidder_id}</TableCell>
                          <TableCell className="px-3 py-2">{formatTime(bid.date_bid)}</TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              </div>
            ) : null}

            {!detailLoading &&
            !detailError &&
            detailData &&
            detailData.items.length === 0 &&
            !(selectedContract?.type === 'auction' && sortedBids.length > 0) ? (
              <p className="text-sm text-muted-foreground">{t('infoContracts.detail.empty')}</p>
            ) : null}
          </div>
        </SheetContent>
      </Sheet>
    </section>
  )
}
