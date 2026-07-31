import { Textarea } from '@/components/ui/textarea'
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
  adminCreateProduct,
  adminDeleteProduct,
  adminListProducts,
  adminUpdateProduct,
} from '@/api/shop'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { usePermission } from '@/hooks/use-permission'
import { useI18n } from '@/i18n'
import type { Product } from '@/types/api/shop'
import {
  formatCoin,
  formatDateTime,
  getErrorMessage,
  getLimitPeriodLabel,
  productStatusClass,
  ShopBadge,
  ShopDialog,
} from './shop-page-utils'

type ProductFormState = {
  id: number
  name: string
  description: string
  image: string
  price: number
  stock: number
  max_per_user: number
  limit_period: 'forever' | 'daily' | 'weekly' | 'monthly'
  status: number
  sort_order: number
}

const defaultFormState: ProductFormState = {
  id: 0,
  name: '',
  description: '',
  image: '',
  price: 0,
  stock: -1,
  max_per_user: 0,
  limit_period: 'forever',
  status: 1,
  sort_order: 0,
}

export function ShopManagePage() {
  const { t } = useI18n()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [products, setProducts] = useState<Product[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [nameFilter, setNameFilter] = useState('')
  const [statusFilter, setStatusFilter] = useState('')
  const [refreshSeed, setRefreshSeed] = useState(0)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [saving, setSaving] = useState(false)
  const [deletingId, setDeletingId] = useState<number | null>(null)
  const [form, setForm] = useState<ProductFormState>(defaultFormState)

  const canCreate = usePermission('add_product')
  const canEdit = usePermission('edit_product')
  const canDelete = usePermission('delete_product')

  const pageCount = useMemo(() => Math.max(1, Math.ceil(total / pageSize) || 1), [pageSize, total])

  const loadData = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const response = await adminListProducts({
        current: page,
        size: pageSize,
        name: nameFilter.trim() || undefined,
        status: statusFilter === '' ? undefined : Number(statusFilter),
      })
      setProducts(response.list ?? [])
      setTotal(response.total ?? 0)
      setPage(response.page ?? page)
      setPageSize(response.pageSize ?? pageSize)
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('shopManage.loadFailed')))
      setProducts([])
      setTotal(0)
    } finally {
      setLoading(false)
    }
  }, [nameFilter, page, pageSize, statusFilter, t])

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadData()
    }, 0)
    return () => window.clearTimeout(timer)
  }, [loadData, refreshSeed])

  const openCreateDialog = () => {
    setForm(defaultFormState)
    setDialogOpen(true)
  }

  const openEditDialog = (product: Product) => {
    setForm({
      id: product.id,
      name: product.name,
      description: product.description,
      image: product.image,
      price: product.price,
      stock: product.stock,
      max_per_user: product.max_per_user,
      limit_period: product.limit_period,
      status: product.status,
      sort_order: product.sort_order,
    })
    setDialogOpen(true)
  }

  const submit = async () => {
    if (!form.name.trim()) {
      setError(t('shopManage.requiredName'))
      return
    }

    if (form.price <= 0) {
      setError(t('shopManage.requiredPrice'))
      return
    }

    setSaving(true)
    setError(null)
    try {
      const payload = {
        name: form.name.trim(),
        description: form.description.trim() || undefined,
        image: form.image.trim() || undefined,
        price: form.price,
        stock: form.stock,
        max_per_user: form.max_per_user,
        limit_period: form.limit_period,
        type: 'normal' as const,
        status: form.status,
        sort_order: form.sort_order,
      }

      if (form.id > 0) {
        await adminUpdateProduct({ id: form.id, ...payload })
      } else {
        await adminCreateProduct(payload)
      }

      setDialogOpen(false)
      await loadData()
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('shopManage.operationFailed')))
    } finally {
      setSaving(false)
    }
  }

  const remove = async (product: Product) => {
    if (!window.confirm(t('shopManage.deleteConfirm', { name: product.name }))) {
      return
    }

    setDeletingId(product.id)
    setError(null)
    try {
      await adminDeleteProduct(product.id)
      await loadData()
    } catch (caughtError) {
      setError(getErrorMessage(caughtError, t('shopManage.deleteFailed')))
    } finally {
      setDeletingId(null)
    }
  }

  return (
    <section className="space-y-4">
      <div className="rounded-lg border bg-card p-5">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <h1 className="text-xl font-semibold">{t('shopManage.title')}</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t('shopManage.subtitle')}</p>
          </div>
          <div className="flex flex-wrap items-end gap-3">
            <label className="space-y-1">
              <span className="text-sm text-muted-foreground">{t('shopManage.filterName')}</span>
              <Input
                value={nameFilter}
                onChange={(event) => setNameFilter(event.target.value)}
                placeholder={t('shopManage.namePlaceholder')}
              />
            </label>
            <label className="space-y-1">
              <span className="text-sm text-muted-foreground">{t('shopManage.filterStatus')}</span>
              <Select
                selectedKey={String(statusFilter ?? '')}
                onSelectionChange={(key) => ((value) => setStatusFilter(value))(String(key))}
              >
                <SelectTrigger className="h-10 rounded-md border border-input bg-background px-3 text-sm">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem id="">{t('shopManage.filterStatus')}</SelectItem>
                  <SelectItem id="1">{t('shopManage.statusOnSale')}</SelectItem>
                  <SelectItem id="0">{t('shopManage.statusOffSale')}</SelectItem>
                </SelectContent>
              </Select>
            </label>
            <Button
              type="button"
              variant="outline"
              onClick={() => {
                setPage(1)
                setRefreshSeed((current) => current + 1)
              }}
            >
              {t('common.search')}
            </Button>
            <Button
              type="button"
              variant="outline"
              onClick={() => {
                setNameFilter('')
                setStatusFilter('')
                setPage(1)
                setRefreshSeed((current) => current + 1)
              }}
            >
              {t('common.reset')}
            </Button>
            <Button type="button" onClick={openCreateDialog} isDisabled={!canCreate}>
              {t('shopManage.createProduct')}
            </Button>
          </div>
        </div>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {loading ? <p className="text-sm text-muted-foreground">{t('shopManage.loading')}</p> : null}

      <div className="overflow-hidden rounded-lg border bg-card">
        <div className="border-b px-4 py-3 text-sm font-medium">
          {t('shopManage.title')} ({total})
        </div>
        <div className="overflow-x-auto">
          <Table className="min-w-full text-sm">
            <TableHeader>
              <TableRow className="border-b bg-muted/40 text-left">
                <TableHead className="px-3 py-2">{t('shopManage.columns.image')}</TableHead>
                <TableHead className="px-3 py-2">{t('shop.productName')}</TableHead>
                <TableHead className="px-3 py-2">{t('shopManage.columns.price')}</TableHead>
                <TableHead className="px-3 py-2">{t('shopManage.columns.stock')}</TableHead>
                <TableHead className="px-3 py-2">{t('shopManage.columns.limitPerUser')}</TableHead>
                <TableHead className="px-3 py-2">{t('shopManage.columns.limitPeriod')}</TableHead>
                <TableHead className="px-3 py-2">{t('shopManage.columns.status')}</TableHead>
                <TableHead className="px-3 py-2">{t('shopManage.columns.sort')}</TableHead>
                <TableHead className="px-3 py-2">{t('shopManage.columns.updatedAt')}</TableHead>
                <TableHead className="px-3 py-2">{t('shopManage.columns.actions')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {products.map((product) => (
                <TableRow key={product.id} className="border-b">
                  <TableCell className="px-3 py-2">
                    {product.image ? (
                      <img
                        alt={product.name}
                        className="h-10 w-10 rounded object-cover"
                        src={product.image}
                      />
                    ) : (
                      <div className="h-10 w-10 rounded bg-muted" />
                    )}
                  </TableCell>
                  <TableCell className="px-3 py-2">
                    <div className="font-medium">{product.name}</div>
                    <div className="line-clamp-2 text-xs text-muted-foreground">
                      {product.description || '-'}
                    </div>
                  </TableCell>
                  <TableCell className="px-3 py-2 font-medium text-orange-600">
                    {formatCoin(product.price)} {t('shop.currency')}
                  </TableCell>
                  <TableCell className="px-3 py-2">
                    <span
                      className={
                        product.stock < 0
                          ? 'text-muted-foreground'
                          : product.stock === 0
                            ? 'text-rose-600'
                            : ''
                      }
                    >
                      {product.stock < 0 ? t('shopManage.stockUnlimited') : product.stock}
                    </span>
                  </TableCell>
                  <TableCell className="px-3 py-2">
                    {product.max_per_user > 0
                      ? product.max_per_user
                      : t('shopManage.stockUnlimited')}
                  </TableCell>
                  <TableCell className="px-3 py-2">
                    {getLimitPeriodLabel(t, product.limit_period)}
                  </TableCell>
                  <TableCell className="px-3 py-2">
                    <ShopBadge className={productStatusClass(product.status)}>
                      {product.status === 1
                        ? t('shopManage.statusOnSale')
                        : t('shopManage.statusOffSale')}
                    </ShopBadge>
                  </TableCell>
                  <TableCell className="px-3 py-2">{product.sort_order}</TableCell>
                  <TableCell className="px-3 py-2">{formatDateTime(product.updated_at)}</TableCell>
                  <TableCell className="px-3 py-2">
                    <div className="flex flex-wrap gap-2">
                      <Button
                        type="button"
                        size="sm"
                        variant="outline"
                        onClick={() => openEditDialog(product)}
                        isDisabled={!canEdit}
                      >
                        {t('common.edit')}
                      </Button>
                      <Button
                        type="button"
                        size="sm"
                        variant="outline"
                        onClick={() => void remove(product)}
                        isDisabled={!canDelete || deletingId === product.id}
                      >
                        {t('common.delete')}
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
              {!loading && products.length === 0 ? (
                <TableRow>
                  <TableCell className="px-3 py-6 text-center text-muted-foreground" colSpan={10}>
                    {t('shopManage.empty')}
                  </TableCell>
                </TableRow>
              ) : null}
            </TableBody>
          </Table>
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-3 text-sm">
        <span>
          {page}/{pageCount}
        </span>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => setPage((current) => Math.max(1, current - 1))}
          isDisabled={page <= 1}
        >
          {t('welfareMy.pagination.prev')}
        </Button>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => setPage((current) => current + 1)}
          isDisabled={products.length < pageSize || page * pageSize >= total}
        >
          {t('welfareMy.pagination.next')}
        </Button>
        <label className="flex items-center gap-2">
          <span>{t('welfareMy.pageSize')}</span>
          <Select
            selectedKey={String(pageSize ?? '')}
            onSelectionChange={(key) => ((value) => {
              const nextSize = Number(value)
              setPageSize(nextSize)
              setPage(1)
            })(String(key))}
          >
            <SelectTrigger className="h-8 rounded-md border border-input bg-background px-2 text-sm">
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

      <ShopDialog
        open={dialogOpen}
        title={form.id > 0 ? t('shopManage.editProduct') : t('shopManage.createProduct')}
        widthClass="max-w-2xl"
        onClose={() => setDialogOpen(false)}
        closeLabel={t('common.close')}
        footer={
          <>
            <Button
              type="button"
              variant="outline"
              onClick={() => setDialogOpen(false)}
              isDisabled={saving}
            >
              {t('common.cancel')}
            </Button>
            <Button type="button" onClick={() => void submit()} isDisabled={saving}>
              {saving ? t('shopManage.saving') : t('common.confirm')}
            </Button>
          </>
        }
      >
        <div className="grid gap-4 md:grid-cols-2">
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('shop.productName')}</span>
            <Input
              value={form.name}
              onChange={(event) => setForm((current) => ({ ...current, name: event.target.value }))}
            />
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('shopManage.fields.image')}</span>
            <Input
              value={form.image}
              onChange={(event) =>
                setForm((current) => ({ ...current, image: event.target.value }))
              }
            />
          </label>
          <label className="space-y-2 md:col-span-2">
            <span className="text-sm text-muted-foreground">
              {t('shopManage.fields.description')}
            </span>
            <Textarea
              className="min-h-24 w-full rounded-lg border border-input bg-background px-3 py-2 text-sm outline-none"
              value={form.description}
              onChange={(event) =>
                setForm((current) => ({ ...current, description: event.target.value }))
              }
            />
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('shopManage.fields.price')}</span>
            <Input
              type="number"
              value={String(form.price)}
              onChange={(event) =>
                setForm((current) => ({ ...current, price: Number(event.target.value) }))
              }
            />
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('shopManage.fields.stock')}</span>
            <Input
              type="number"
              value={String(form.stock)}
              onChange={(event) =>
                setForm((current) => ({ ...current, stock: Number(event.target.value) }))
              }
            />
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">
              {t('shopManage.fields.maxPerUser')}
            </span>
            <Input
              type="number"
              value={String(form.max_per_user)}
              onChange={(event) =>
                setForm((current) => ({ ...current, max_per_user: Number(event.target.value) }))
              }
            />
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('shopManage.fields.sort')}</span>
            <Input
              type="number"
              value={String(form.sort_order)}
              onChange={(event) =>
                setForm((current) => ({ ...current, sort_order: Number(event.target.value) }))
              }
            />
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">
              {t('shopManage.fields.limitPeriod')}
            </span>
            <Select
              selectedKey={String(form.limit_period ?? '')}
              onSelectionChange={(key) => ((value) =>
                setForm((current) => ({
                  ...current,
                  limit_period: value as ProductFormState['limit_period'],
                })))(String(key))}
            >
              <SelectTrigger className="h-10 rounded-md border border-input bg-background px-3 text-sm">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem id="forever">{t('shopManage.periodForever')}</SelectItem>
                <SelectItem id="daily">{t('shopManage.periodDaily')}</SelectItem>
                <SelectItem id="weekly">{t('shopManage.periodWeekly')}</SelectItem>
                <SelectItem id="monthly">{t('shopManage.periodMonthly')}</SelectItem>
              </SelectContent>
            </Select>
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('shopManage.fields.status')}</span>
            <Select
              selectedKey={String(form.status ?? '')}
              onSelectionChange={(key) => ((value) =>
                setForm((current) => ({ ...current, status: Number(value) })))(String(key))}
            >
              <SelectTrigger className="h-10 rounded-md border border-input bg-background px-3 text-sm">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem id="1">{t('shopManage.statusOnSale')}</SelectItem>
                <SelectItem id="0">{t('shopManage.statusOffSale')}</SelectItem>
              </SelectContent>
            </Select>
          </label>
        </div>
      </ShopDialog>
    </section>
  )
}
