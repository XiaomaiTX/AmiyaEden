import { render, screen } from '@testing-library/react'
import { FuxiHallMemberCard } from '@/components/fuxi-hall-member-card'
import { I18nProvider } from '@/i18n'

const card = {
  id: 1,
  page_key: 'leadership' as const,
  nickname: 'Amiya',
  main_character_id: 1001,
  main_character_name: 'Amiya Prime',
  title_tags: ['FC'],
  description_html: '<p>Fleet commander</p>',
  accent_color: '#3b82f6',
  avatar_shape: 'circle' as const,
  font_scale: 14,
  visible: true,
  sort_order: 1,
  created_at: '',
  updated_at: '',
}

test('renders identity and only positive statistics', () => {
  render(
    <I18nProvider>
      <FuxiHallMemberCard
        card={{ ...card, fleet_led_count: 2, welfare_delivery_count: 0 }}
        showStats
      />
    </I18nProvider>
  )
  expect(screen.getByText('Amiya')).toBeInTheDocument()
  expect(screen.getByText('FC')).toBeInTheDocument()
  expect(screen.getByText(/带队次数: 2/)).toBeInTheDocument()
  expect(screen.queryByText(/福利发放次数/)).not.toBeInTheDocument()
})
