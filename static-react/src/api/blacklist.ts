import { requestJson } from '@/api/http-client'
import { assertSuccess, type ApiResponse } from '@/api/response'
import type {
  BlacklistEntry,
  BlacklistListParams,
  BlacklistRefreshSummary,
} from '@/types/api/blacklist'
import type { PaginatedResponse } from '@/types/api/common'

// Errors are i18n keys (`blacklist.errors.*`); the page localizes them itself.
const base = '/api/v1/system/blacklist'
const fallback = 'blacklist.errors.internal'
// The server allows a refresh up to 90 s (sheet download plus ESI lookups).
const refreshTimeoutMs = 120_000

export async function fetchBlacklistEntries({
  current = 1,
  size = 20,
  kind,
  keyword,
}: BlacklistListParams) {
  const params = new URLSearchParams({ current: String(current), size: String(size) })
  if (kind) params.set('kind', kind)
  if (keyword) params.set('keyword', keyword)
  return assertSuccess(
    await requestJson<ApiResponse<PaginatedResponse<BlacklistEntry>>>(`${base}?${params}`),
    fallback
  )
}

export async function fetchBlacklistSummary() {
  return assertSuccess(
    await requestJson<ApiResponse<BlacklistRefreshSummary | null>>(`${base}/refresh`),
    fallback
  )
}

export async function refreshBlacklist() {
  return assertSuccess(
    await requestJson<ApiResponse<BlacklistRefreshSummary>>(`${base}/refresh`, {
      method: 'POST',
      signal: AbortSignal.timeout(refreshTimeoutMs),
    }),
    fallback
  )
}
