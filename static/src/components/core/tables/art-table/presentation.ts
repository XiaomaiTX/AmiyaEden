export const LEDGER_TABLE_PAGE_SIZES = [200, 500, 1000] as const
export const LEDGER_TABLE_PAGINATION_LAYOUT = 'total, sizes, prev, pager, next, jumper'

export type ArtTableVisualVariant = 'default' | 'ledger'

export interface ArtTablePaginationOptions {
  pageSizes?: number[]
  align?: 'left' | 'center' | 'right'
  layout?: string
  background?: boolean
  hideOnSinglePage?: boolean
  size?: 'small' | 'default' | 'large'
  pagerCount?: number
}

interface ResolveArtTablePresentationInput {
  visualVariant?: ArtTableVisualVariant
  border?: boolean
  stripe?: boolean
  paginationOptions?: ArtTablePaginationOptions
  basePaginationOptions: ArtTablePaginationOptions
}

interface ResolveArtTablePresentationResult {
  border?: boolean
  stripe?: boolean
  paginationOptions: ArtTablePaginationOptions
  tableClassNames: string[]
  paginationClassNames: string[]
}

export function resolveArtTablePresentation({
  visualVariant = 'default',
  border,
  stripe,
  paginationOptions,
  basePaginationOptions
}: ResolveArtTablePresentationInput): ResolveArtTablePresentationResult {
  const isLedger = visualVariant === 'ledger'

  return {
    border: border ?? (isLedger ? true : undefined),
    stripe: stripe ?? (isLedger ? true : undefined),
    paginationOptions: {
      ...basePaginationOptions,
      ...(isLedger
        ? {
            pageSizes: [...LEDGER_TABLE_PAGE_SIZES],
            align: 'right' as const,
            layout: LEDGER_TABLE_PAGINATION_LAYOUT
          }
        : {}),
      ...paginationOptions
    },
    tableClassNames: isLedger ? ['art-table--ledger'] : [],
    paginationClassNames: isLedger ? ['pagination--ledger'] : []
  }
}
