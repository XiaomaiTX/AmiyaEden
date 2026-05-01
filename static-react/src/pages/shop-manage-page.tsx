import { useCallback, useEffect, useMemo, useState } from 'react'
import { adminCreateProduct, adminDeleteProduct, adminListProducts, adminUpdateProduct } from '@/api/shop'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useI18n } from '@/i18n'
import { useSessionStore } from '@/stores'
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

function hasAction(authList: string[], mark: string) {
  return authList.includes(mark)
}

export function ShopManagePage() {
  const { t } = useI18n()
  const authList = useSessionStore((state) => state.authList)
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

  const canCreate = hasAction(authList, 'add_product')
  const canEdit = hasAction(authList, 'edit_product')
  const canDelete = hasAction(authList, 'delete_product')

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
              <select
                className="h-10 rounded-md border border-input bg-background px-3 text-sm"
                value={statusFilter}
                onChange={(event) => setStatusFilter(event.target.value)}
              >
                <option value="">{t('shopManage.filterStatus')}</option>
                <option value="1">{t('shopManage.statusOnSale')}</option>
                <option value="0">{t('shopManage.statusOffSale')}</option>
              </select>
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
            <Button type="button" onClick={openCreateDialog} disabled={!canCreate}>
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
          <table className="min-w-full text-sm">
            <thead>
              <tr className="border-b bg-muted/40 text-left">
                <th className="px-3 py-2">{t('shopManage.columns.image')}</th>
                <th className="px-3 py-2">{t('shop.productName')}</th>
                <th className="px-3 py-2">{t('shopManage.columns.price')}</th>
                <th className="px-3 py-2">{t('shopManage.columns.stock')}</th>
                <th className="px-3 py-2">{t('shopManage.columns.limitPerUser')}</th>
                <th className="px-3 py-2">{t('shopManage.columns.limitPeriod')}</th>
                <th className="px-3 py-2">{t('shopManage.columns.status')}</th>
                <th className="px-3 py-2">{t('shopManage.columns.sort')}</th>
                <th className="px-3 py-2">{t('shopManage.columns.updatedAt')}</th>
                <th className="px-3 py-2">{t('shopManage.columns.actions')}</th>
              </tr>
            </thead>
            <tbody>
              {products.map((product) => (
                <tr key={product.id} className="border-b">
                  <td className="px-3 py-2">
                    {product.image ? (
                      <img alt={product.name} className="h-10 w-10 rounded object-cover" src={product.image} />
                    ) : (
                      <div className="h-10 w-10 rounded bg-muted" />
                    )}
                  </td>
                  <td className="px-3 py-2">
                    <div className="font-medium">{product.name}</div>
                    <div className="line-clamp-2 text-xs text-muted-foreground">{product.description || '-'}</div>
                  </td>
                  <td className="px-3 py-2 font-medium text-orange-600">
                    {formatCoin(product.price)} {t('shop.currency')}
                  </td>
                  <td className="px-3 py-2">
                    <span className={product.stock < 0 ? 'text-muted-foreground' : product.stock === 0 ? 'text-rose-600' : ''}>
                      {product.stock < 0 ? t('shopManage.stockUnlimited') : product.stock}
                    </span>
                  </td>
                  <td className="px-3 py-2">{product.max_per_user > 0 ? product.max_per_user : t('shopManage.stockUnlimited')}</td>
                  <td className="px-3 py-2">{getLimitPeriodLabel(t, product.limit_period)}</td>
                  <td className="px-3 py-2">
                    <ShopBadge className={productStatusClass(product.status)}>
                      {product.status === 1 ? t('shopManage.statusOnSale') : t('shopManage.statusOffSale')}
                    </ShopBadge>
                  </td>
                  <td className="px-3 py-2">{product.sort_order}</td>
                  <td className="px-3 py-2">{formatDateTime(product.updated_at)}</td>
                  <td className="px-3 py-2">
                    <div className="flex flex-wrap gap-2">
                      <Button
                        type="button"
                        size="sm"
                        variant="outline"
                        onClick={() => openEditDialog(product)}
                        disabled={!canEdit}
                      >
                        {t('common.edit')}
                      </Button>
                      <Button
                        type="button"
                        size="sm"
                        variant="outline"
                        onClick={() => void remove(product)}
                        disabled={!canDelete || deletingId === product.id}
                      >
                        {t('common.delete')}
                      </Button>
                    </div>
                  </td>
                </tr>
              ))}
              {!loading && products.length === 0 ? (
                <tr>
                  <td className="px-3 py-6 text-center text-muted-foreground" colSpan={10}>
                    {t('shopManage.empty')}
                  </td>
                </tr>
              ) : null}
            </tbody>
          </table>
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
          disabled={page <= 1}
        >
          {t('welfareMy.pagination.prev')}
        </Button>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => setPage((current) => current + 1)}
          disabled={products.length < pageSize || page * pageSize >= total}
        >
          {t('welfareMy.pagination.next')}
        </Button>
        <label className="flex items-center gap-2">
          <span>{t('welfareMy.pageSize')}</span>
          <select
            className="h-8 rounded-md border border-input bg-background px-2 text-sm"
            value={pageSize}
            onChange={(event) => {
              const nextSize = Number(event.target.value)
              setPageSize(nextSize)
              setPage(1)
            }}
          >
            {[10, 20, 50].map((size) => (
              <option key={size} value={size}>
                {size}
              </option>
            ))}
          </select>
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
            <Button type="button" variant="outline" onClick={() => setDialogOpen(false)} disabled={saving}>
              {t('common.cancel')}
            </Button>
            <Button type="button" onClick={() => void submit()} disabled={saving}>
              {saving ? t('shopManage.saving') : t('common.confirm')}
            </Button>
          </>
        }
      >
        <div className="grid gap-4 md:grid-cols-2">
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('shop.productName')}</span>
            <Input value={form.name} onChange={(event) => setForm((current) => ({ ...current, name: event.target.value }))} />
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('shopManage.fields.image')}</span>
            <Input value={form.image} onChange={(event) => setForm((current) => ({ ...current, image: event.target.value }))} />
          </label>
          <label className="space-y-2 md:col-span-2">
            <span className="text-sm text-muted-foreground">{t('shopManage.fields.description')}</span>
            <textarea
              className="min-h-24 w-full rounded-lg border border-input bg-background px-3 py-2 text-sm outline-none"
              value={form.description}
              onChange={(event) => setForm((current) => ({ ...current, description: event.target.value }))}
            />
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('shopManage.fields.price')}</span>
            <Input
              type="number"
              value={String(form.price)}
              onChange={(event) => setForm((current) => ({ ...current, price: Number(event.target.value) }))}
            />
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('shopManage.fields.stock')}</span>
            <Input
              type="number"
              value={String(form.stock)}
              onChange={(event) => setForm((current) => ({ ...current, stock: Number(event.target.value) }))}
            />
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('shopManage.fields.maxPerUser')}</span>
            <Input
              type="number"
              value={String(form.max_per_user)}
              onChange={(event) => setForm((current) => ({ ...current, max_per_user: Number(event.target.value) }))}
            />
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('shopManage.fields.sort')}</span>
            <Input
              type="number"
              value={String(form.sort_order)}
              onChange={(event) => setForm((current) => ({ ...current, sort_order: Number(event.target.value) }))}
            />
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('shopManage.fields.limitPeriod')}</span>
            <select
              className="h-10 rounded-md border border-input bg-background px-3 text-sm"
              value={form.limit_period}
              onChange={(event) =>
                setForm((current) => ({
                  ...current,
                  limit_period: event.target.value as ProductFormState['limit_period'],
                }))
              }
            >
              <option value="forever">{t('shopManage.periodForever')}</option>
              <option value="daily">{t('shopManage.periodDaily')}</option>
              <option value="weekly">{t('shopManage.periodWeekly')}</option>
              <option value="monthly">{t('shopManage.periodMonthly')}</option>
            </select>
          </label>
          <label className="space-y-2">
            <span className="text-sm text-muted-foreground">{t('shopManage.fields.status')}</span>
            <select
              className="h-10 rounded-md border border-input bg-background px-3 text-sm"
              value={String(form.status)}
              onChange={(event) => setForm((current) => ({ ...current, status: Number(event.target.value) }))}
            >
              <option value="1">{t('shopManage.statusOnSale')}</option>
              <option value="0">{t('shopManage.statusOffSale')}</option>
            </select>
          </label>
        </div>
      </ShopDialog>
    </section>
  )
}
