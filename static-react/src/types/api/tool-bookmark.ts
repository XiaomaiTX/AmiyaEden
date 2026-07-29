export interface ToolBookmark {
  id: number
  name: string
  url: string
  description: string
  logo_url: string
  logo_source: string
  is_enabled: boolean
  sort_order: number
  created_by: number
  created_at: string
  updated_at: string
}

export interface ToolBookmarkUpsertRequest {
  name: string
  url: string
  description?: string
  is_enabled?: boolean
  sort_order?: number
}
