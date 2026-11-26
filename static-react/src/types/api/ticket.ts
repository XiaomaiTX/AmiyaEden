import type * as contract from '@/api/generated/client'
export type UpsertCategoryParams = contract.TicketCategoryInput & { name: string; name_en: string }
