import { render, screen, waitFor } from '@testing-library/react'
import { FuxiHallLeadershipPage } from '@/pages/fuxi-hall-public-page'
import { I18nProvider } from '@/i18n'
import { useSessionStore } from '@/stores'

beforeEach(() => useSessionStore.getState().setSessionSnapshot({ isLoggedIn: true, accessToken: 'token', characterId: 1, characterName: 'Amiya', roles: ['user'], corpCapabilities: ['menu.fuxi_hall'] }))

test('renders the public leadership page from its API response', async () => {
  vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(new Response(JSON.stringify({ code: 0, msg: 'ok', data: { page: { id: 1, page_key: 'leadership', title: '管理层', subtitle: '', description_html: '' }, cards: [] } }), { headers: { 'Content-Type': 'application/json' } }))
  render(<I18nProvider><FuxiHallLeadershipPage /></I18nProvider>)
  await waitFor(() => expect(screen.getByRole('heading', { name: '管理层' })).toBeInTheDocument())
  expect(screen.getByText('暂无成员')).toBeInTheDocument()
})
