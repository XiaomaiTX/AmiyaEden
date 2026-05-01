import { render, screen, waitFor } from '@testing-library/react'
import { RouterProvider, createMemoryRouter } from 'react-router-dom'
import { appRoutes } from '@/app/router'
import { useSessionStore } from '@/stores'

describe('info contracts page', () => {
  beforeEach(() => {
    useSessionStore.getState().setSessionSnapshot({
      isLoggedIn: true,
      accessToken: 'token-123',
      characterId: 1001,
      characterName: 'Amiya',
      roles: ['admin'],
      authList: [],
    })
  })

  test('loads contracts and opens detail sheet', async () => {
    vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            code: 0,
            msg: 'ok',
            data: {
              list: [
                {
                  character_id: 1001,
                  character_name: 'Amiya',
                  contract_id: 2001,
                  acceptor_id: 0,
                  assignee_id: 0,
                  availability: 'public',
                  buyout: 5000000,
                  collateral: 0,
                  date_issued: '2026-05-01T01:02:03Z',
                  date_expired: '2026-05-08T01:02:03Z',
                  for_corporation: false,
                  issuer_corporation_id: 9001,
                  issuer_id: 6001,
                  price: 2000000,
                  reward: 100000,
                  status: 'auction',
                  title: 'Marketplace Run',
                  type: 'auction',
                },
              ],
              total: 1,
              page: 1,
              pageSize: 20,
            },
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } }
        )
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            code: 0,
            msg: 'ok',
            data: {
              items: [
                {
                  type_id: 34,
                  type_name: 'Tritanium',
                  group_name: 'Mineral',
                  category_id: 18,
                  quantity: 100,
                  is_included: true,
                  is_singleton: false,
                },
              ],
              bids: [
                {
                  amount: 2500000,
                  bid_id: 1,
                  bidder_id: 7001,
                  date_bid: '2026-05-02T03:04:05Z',
                },
              ],
            },
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } }
        )
      )

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/info/contracts'],
    })
    render(<RouterProvider router={router} />)

    await waitFor(() => {
      expect(screen.getByText('Marketplace Run')).toBeInTheDocument()
    })
    expect(screen.getByText('拍卖')).toBeInTheDocument()

    screen.getByRole('button', { name: '查看详情' }).click()

    await waitFor(() => {
      expect(screen.getByText('Tritanium')).toBeInTheDocument()
    })
    expect(screen.getByText('Tritanium')).toBeInTheDocument()
    expect(screen.getByText('2,500,000.00')).toBeInTheDocument()
  })
})
