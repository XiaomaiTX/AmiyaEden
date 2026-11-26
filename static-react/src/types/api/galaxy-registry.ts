import type * as api from '@/api/generated/client'

export type GalaxyRegistryEntryListParams = api.ListAdminEntriesParams & {
  status?: api.GalaxyRegistryEntry['status'] | ''
  validation_status?: api.GalaxyRegistryEntry['validation_status'] | ''
}
