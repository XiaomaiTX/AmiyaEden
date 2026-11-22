import { render, screen } from '@testing-library/react'
import { FuxiHallMemberCard } from '@/components/fuxi-hall-member-card'
import { I18nProvider } from '@/i18n'

const card = {
  id: 1,
  page_key: 'hall-of-fame' as const,
  floor_key: 'first',
  is_current_management: false,
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

test('renders compact contributions and hides zero metrics even after calculation', () => {
  render(
    <I18nProvider>
      <FuxiHallMemberCard
        card={{
          ...card,
          contributions: { fleets: 25, welfare: 101, srp: 0, recruitment: 3, mentoring: 0 },
          contributions_updated_at: '2026-11-22T06:00:00Z',
        }}
      />
    </I18nProvider>
  )
  expect(screen.getByText('Amiya')).toBeInTheDocument()
  expect(screen.getByText('FC')).toBeInTheDocument()
  expect(screen.getByText('25')).toBeInTheDocument()
  expect(screen.getByText('101')).toBeInTheDocument()
  expect(screen.getAllByRole('definition')).toHaveLength(3)
  expect(screen.queryByText('0')).not.toBeInTheDocument()
})

test.each([undefined, { fleets: 0, welfare: 0, srp: 0, recruitment: 0, mentoring: 0 }])(
  'omits the entire contribution region when no positive metrics exist',
  (contributions) => {
    render(
      <I18nProvider>
        <FuxiHallMemberCard
          card={{ ...card, contributions, contributions_updated_at: '2026-11-22T06:00:00Z' }}
        />
      </I18nProvider>
    )
    expect(screen.queryByRole('definition')).not.toBeInTheDocument()
    expect(screen.queryByRole('list')).not.toBeInTheDocument()
    expect(screen.queryByText('0')).not.toBeInTheDocument()
  }
)
