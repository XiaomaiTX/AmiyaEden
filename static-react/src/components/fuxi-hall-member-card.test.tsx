import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
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

test('shows QQ and copies it with success and failure feedback', async () => {
  const user = userEvent.setup()
  const write = vi.spyOn(navigator.clipboard, 'writeText').mockResolvedValueOnce(undefined)
  render(
    <I18nProvider>
      <FuxiHallMemberCard card={{ ...card, qq: '123456789' }} />
    </I18nProvider>
  )
  expect(screen.getByText('123456789')).toBeInTheDocument()
  await user.click(screen.getByRole('button', { name: '复制 QQ 号' }))
  expect(write).toHaveBeenCalledWith('123456789')
  expect(screen.getByRole('status')).toHaveTextContent('已复制')
  write.mockRejectedValueOnce(new Error('denied'))
  await user.click(screen.getByRole('button', { name: '复制 QQ 号' }))
  expect(screen.getByRole('status')).toHaveTextContent('复制失败')
})

test.each([undefined, '', '   '])('omits the QQ row when QQ is %j', (qq) => {
  render(
    <I18nProvider>
      <FuxiHallMemberCard card={{ ...card, qq }} />
    </I18nProvider>
  )
  expect(screen.queryByText(/QQ/)).not.toBeInTheDocument()
  expect(screen.queryByRole('button', { name: '复制 QQ 号' })).not.toBeInTheDocument()
})

test('renders compact contributions and hides zero metrics even after calculation', () => {
  render(
    <I18nProvider>
      <FuxiHallMemberCard
        card={{
          ...card,
          contributions: {
            corp_pap: 4.5,
            fleets: 25,
            welfare: 101,
            srp: 0,
            recruitment: 3,
            mentoring: 0,
          },
          contributions_updated_at: '2026-11-22T06:00:00Z',
        }}
      />
    </I18nProvider>
  )
  expect(screen.getByText('Amiya')).toBeInTheDocument()
  expect(screen.getByText('FC')).toBeInTheDocument()
  expect(screen.getByText('25')).toBeInTheDocument()
  expect(screen.getByText('101')).toBeInTheDocument()
  expect(screen.getByText('军团 PAP')).toBeInTheDocument()
  expect(screen.getByText('4.5')).toBeInTheDocument()
  expect(screen.getAllByRole('definition')).toHaveLength(4)
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
