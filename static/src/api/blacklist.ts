import request from '@/utils/http'

const base = '/api/v1/system/blacklist'
// Errors are i18n keys (`blacklist.errors.*`); the page localizes them itself.
const options = { showErrorMessage: false }

export function fetchBlacklistEntries(params: Api.Blacklist.ListParams) {
  return request.get<Api.Common.PaginatedResponse<Api.Blacklist.Entry>>({
    url: base,
    params,
    ...options
  })
}

export function fetchBlacklistSummary() {
  return request.get<Api.Blacklist.RefreshSummary | null>({ url: `${base}/refresh`, ...options })
}

// The server allows a refresh up to 90 s (sheet download plus ESI lookups).
export function refreshBlacklist() {
  return request.post<Api.Blacklist.RefreshSummary>({
    url: `${base}/refresh`,
    timeout: 120 * 1000,
    ...options
  })
}
