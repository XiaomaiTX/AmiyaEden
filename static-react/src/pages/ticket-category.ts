import type { TicketCategory } from '@/types/api/ticket'

export function getCategoryName(category: TicketCategory, locale: string) {
  if (locale === 'en-US' && category.name_en.trim()) {
    return category.name_en
  }

  return category.name
}
