import { getCategoryName } from '@/pages/ticket-category'

const category = {
  id: 10,
  name: '技术支持',
  name_en: 'Support',
  description: 'Support issues',
  sort_order: 1,
  enabled: true,
  created_at: '2026-05-01T00:00:00Z',
  updated_at: '2026-05-01T00:00:00Z',
}

describe('ticket category localization', () => {
  test('uses the English category name in en-US', () => {
    expect(getCategoryName(category, 'en-US')).toBe('Support')
  })

  test('falls back to the default category name when English is unavailable', () => {
    expect(getCategoryName({ ...category, name_en: '' }, 'en-US')).toBe('技术支持')
  })
})
