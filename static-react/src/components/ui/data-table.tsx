import {
  flexRender,
  getCoreRowModel,
  useReactTable,
  type ColumnDef,
  type OnChangeFn,
  type RowSelectionState,
  type SortingState,
} from '@tanstack/react-table'
import { Button } from '@/components/ui/button'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'

export interface DataTablePagination {
  page: number
  pageSize: number
  total: number
  onPageChange: (page: number) => void
  onPageSizeChange: (pageSize: number) => void
  pageSizeOptions?: number[]
  previousLabel: string
  nextLabel: string
  pageSizeLabel: string
}

export const DEFAULT_LEDGER_PAGE_SIZE = 200

interface DataTableProps<TData> {
  columns: ColumnDef<TData, unknown>[]
  data: TData[]
  getRowId?: (row: TData) => string
  loading?: boolean
  error?: string | null
  loadingText: string
  emptyText: string
  pagination?: DataTablePagination
  sorting?: SortingState
  onSortingChange?: OnChangeFn<SortingState>
  rowSelection?: RowSelectionState
  onRowSelectionChange?: OnChangeFn<RowSelectionState>
  variant?: 'default' | 'ledger'
}

export function DataTable<TData>({
  columns,
  data,
  getRowId,
  loading = false,
  error,
  loadingText,
  emptyText,
  pagination,
  sorting,
  onSortingChange,
  rowSelection,
  onRowSelectionChange,
  variant = 'default',
}: DataTableProps<TData>) {
  // TanStack Table intentionally exposes mutable callback APIs; React Compiler skips this hook.
  // eslint-disable-next-line react-hooks/incompatible-library
  const table = useReactTable({
    data,
    columns,
    getRowId,
    getCoreRowModel: getCoreRowModel(),
    manualPagination: Boolean(pagination),
    manualSorting: Boolean(onSortingChange),
    enableSorting: Boolean(onSortingChange),
    onSortingChange,
    onRowSelectionChange,
    state: {
      sorting: sorting ?? [],
      rowSelection: rowSelection ?? {},
    },
  })

  const pageCount = pagination
    ? Math.max(1, Math.ceil(pagination.total / pagination.pageSize))
    : 1
  const pageSizeOptions = pagination?.pageSizeOptions ??
    (variant === 'ledger' ? [DEFAULT_LEDGER_PAGE_SIZE] : [10, 20, 50, 100, 200])

  return (
    <div className={variant === 'ledger' ? 'space-y-3' : 'space-y-4'}>
      <div className="rounded-md border">
        <Table>
          <TableHeader className="bg-muted/50">
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id}>
                {headerGroup.headers.map((header) => (
                  <TableHead
                    key={header.id}
                    className="whitespace-nowrap px-3 py-2 text-left font-medium"
                    style={{ width: header.getSize() }}
                  >
                    {header.isPlaceholder ? null : onSortingChange &&
                      header.column.getCanSort() ? (
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        className="-ml-2 h-7 gap-1 px-2"
                        onClick={header.column.getToggleSortingHandler()}
                      >
                        {flexRender(header.column.columnDef.header, header.getContext())}
                        <span aria-hidden="true">
                          {header.column.getIsSorted() === 'asc'
                            ? '↑'
                            : header.column.getIsSorted() === 'desc'
                              ? '↓'
                              : '↕'}
                        </span>
                      </Button>
                    ) : (
                      flexRender(header.column.columnDef.header, header.getContext())
                    )}
                  </TableHead>
                ))}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {loading || error || table.getRowModel().rows.length === 0 ? (
              <TableRow>
                <TableCell className="py-8 text-center text-muted-foreground" colSpan={columns.length}>
                  {loading ? loadingText : error || emptyText}
                </TableCell>
              </TableRow>
            ) : (
              table.getRowModel().rows.map((row) => (
                <TableRow key={row.id}>
                  {row.getVisibleCells().map((cell) => (
                    <TableCell key={cell.id}>
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      {pagination ? (
        <div className="flex flex-wrap items-center justify-between gap-3">
          <label className="flex items-center gap-2 text-sm text-muted-foreground">
            <span>{pagination.pageSizeLabel}</span>
            <NativeSelect
              value={pagination.pageSize}
              onChange={(event) => pagination.onPageSizeChange(Number(event.target.value))}
            >
              {pageSizeOptions.map((size) => (
                <NativeSelectOption key={size} value={size}>
                  {size}
                </NativeSelectOption>
              ))}
            </NativeSelect>
          </label>
          <div className="flex items-center gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={pagination.page <= 1 || loading}
              onClick={() => pagination.onPageChange(pagination.page - 1)}
            >
              {pagination.previousLabel}
            </Button>
            <span className="text-sm text-muted-foreground">
              {pagination.page} / {pageCount}
            </span>
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={pagination.page >= pageCount || loading}
              onClick={() => pagination.onPageChange(pagination.page + 1)}
            >
              {pagination.nextLabel}
            </Button>
          </div>
        </div>
      ) : null}
    </div>
  )
}

export type { ColumnDef, SortingState }
