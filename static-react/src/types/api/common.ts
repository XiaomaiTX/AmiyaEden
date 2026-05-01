export interface PaginatedResponse<T> {
  list: T[]
  total: number
  page: number
  pageSize: number
}

export interface SnakePaginatedResponse<T> {
  items: T[]
  total: number
  page: number
  page_size: number
}

export type CommonSearchParams = Partial<{
  current: number
  size: number
}>

export type EnableStatus = '1' | '2'

