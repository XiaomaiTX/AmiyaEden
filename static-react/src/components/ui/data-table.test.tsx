import { fireEvent, render, screen } from '@testing-library/react'
import { DataTable, type ColumnDef } from '@/components/ui/data-table'

type Row = { id: number; name: string }
const columns: ColumnDef<Row, unknown>[] = [{ accessorKey: 'name', header: 'Name' }]

describe('DataTable', () => {
  test('renders rows and delegates server pagination', () => {
    const onPageChange = vi.fn()
    render(
      <DataTable
        columns={columns}
        data={[{ id: 1, name: 'Amiya' }]}
        getRowId={(row) => String(row.id)}
        loadingText="Loading"
        emptyText="Empty"
        pagination={{
          page: 2,
          pageSize: 10,
          total: 30,
          onPageChange,
          onPageSizeChange: vi.fn(),
          previousLabel: 'Previous',
          nextLabel: 'Next',
          pageSizeLabel: 'Page size',
        }}
      />
    )

    expect(screen.getByText('Amiya')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Next' }))
    expect(onPageChange).toHaveBeenCalledWith(3)
  })

  test('prioritizes loading and error states over the empty state', () => {
    const { rerender } = render(
      <DataTable
        columns={columns}
        data={[]}
        loading
        error="Failed"
        loadingText="Loading"
        emptyText="Empty"
      />
    )
    expect(screen.getByText('Loading')).toBeInTheDocument()

    rerender(
      <DataTable
        columns={columns}
        data={[]}
        error="Failed"
        loadingText="Loading"
        emptyText="Empty"
      />
    )
    expect(screen.getByText('Failed')).toBeInTheDocument()
  })

  test('delegates sorting changes for sortable server-side columns', () => {
    const onSortingChange = vi.fn()
    render(
      <DataTable
        columns={columns}
        data={[{ id: 1, name: 'Amiya' }]}
        loadingText="Loading"
        emptyText="Empty"
        sorting={[]}
        onSortingChange={onSortingChange}
      />
    )

    fireEvent.click(screen.getByRole('button', { name: 'Name' }))
    expect(onSortingChange).toHaveBeenCalledOnce()
  })

  test('uses the ledger page-size preset when no page options are supplied', () => {
    render(
      <DataTable
        columns={columns}
        data={[]}
        loadingText="Loading"
        emptyText="Empty"
        variant="ledger"
        pagination={{
          page: 1,
          pageSize: 200,
          total: 0,
          onPageChange: vi.fn(),
          onPageSizeChange: vi.fn(),
          previousLabel: 'Previous',
          nextLabel: 'Next',
          pageSizeLabel: 'Page size',
        }}
      />
    )

    expect(screen.getByRole('button', { name: /Page size/ })).toHaveTextContent('200')
  })
})
