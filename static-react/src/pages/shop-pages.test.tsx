import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { RouterProvider, createMemoryRouter } from 'react-router-dom'
import { appRoutes } from '@/app/router'
import { usePreferenceStore, useSessionStore } from '@/stores'

function jsonResponse(data: unknown) {
  return new Response(
    JSON.stringify({
      code: 0,
      msg: 'ok',
      data,
    }),
    {
      status: 200,
      headers: {
        'Content-Type': 'application/json',
      },
    }
  )
}

function resetStores() {
  useSessionStore.setState({
    isLoggedIn: true,
    accessToken: 'token-123',
    characterId: 1001,
    characterName: 'Amiya',
    roles: ['admin'],
    authList: [],
    isCurrentlyNewbro: false,
    isMentorMenteeEligible: false,
    hydratedAt: null,
  })
  usePreferenceStore.setState({
    locale: 'zh-CN',
    sidebarCollapsed: false,
    theme: 'system',
  })
}

describe('shop pages', () => {
  beforeEach(() => {
    resetStores()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  test('shop browse page loads products and submits a purchase', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch')
    fetchMock
      .mockResolvedValueOnce(
        jsonResponse({
          id: 1,
          user_id: 1001,
          balance: 12345,
          updated_at: '2026-05-01T00:00:00Z',
          character_name: 'Amiya',
        })
      )
      .mockResolvedValueOnce(
        jsonResponse({
          list: [
            {
              id: 11,
              name: 'Pilot Pack',
              description: 'Starter goods',
              image: '',
              price: 1200,
              stock: 10,
              max_per_user: 2,
              limit_period: 'weekly',
              type: 'normal',
              status: 1,
              sort_order: 1,
              created_at: '2026-05-01T00:00:00Z',
              updated_at: '2026-05-01T00:00:00Z',
            },
          ],
          total: 1,
          page: 1,
          pageSize: 12,
        })
      )
      .mockResolvedValueOnce(
        jsonResponse({
          id: 88,
          order_no: 'ORD-1001',
          user_id: 1001,
          main_character_name: 'Amiya',
          nickname: 'Amiya',
          qq: '',
          discord_id: '',
          product_id: 11,
          product_name: 'Pilot Pack',
          product_type: 'normal',
          quantity: 2,
          unit_price: 1200,
          total_price: 2400,
          status: 'requested',
          transaction_id: null,
          remark: 'gift',
          reviewed_by: null,
          reviewed_at: null,
          reviewer_name: '',
          review_remark: '',
          created_at: '2026-05-01T00:00:00Z',
          updated_at: '2026-05-01T00:00:00Z',
        })
      )
      .mockResolvedValueOnce(
        jsonResponse({
          id: 1,
          user_id: 1001,
          balance: 10000,
          updated_at: '2026-05-01T00:10:00Z',
          character_name: 'Amiya',
        })
      )
      .mockResolvedValueOnce(
        jsonResponse({
          list: [
            {
              id: 11,
              name: 'Pilot Pack',
              description: 'Starter goods',
              image: '',
              price: 1200,
              stock: 8,
              max_per_user: 2,
              limit_period: 'weekly',
              type: 'normal',
              status: 1,
              sort_order: 1,
              created_at: '2026-05-01T00:00:00Z',
              updated_at: '2026-05-01T00:10:00Z',
            },
          ],
          total: 1,
          page: 1,
          pageSize: 12,
        })
      )

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/shop/browse'],
    })

    render(<RouterProvider router={router} />)

    await screen.findByText('Pilot Pack')
    expect(screen.getByText('12,345 伏羲币')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: '购买' }))

    const dialog = await screen.findByRole('dialog', { name: '购买商品' })
    fireEvent.change(within(dialog).getByRole('spinbutton'), {
      target: { value: '2' },
    })
    fireEvent.change(within(dialog).getByPlaceholderText('备注信息（可选）'), {
      target: { value: 'gift' },
    })
    fireEvent.click(within(dialog).getByRole('button', { name: '确认购买' }))

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledTimes(5)
    })

    const buyCall = fetchMock.mock.calls[2]
    expect(String(buyCall[0])).toContain('/api/v1/shop/buy')
    expect(JSON.parse(String(buyCall[1]?.body))).toEqual({
      product_id: 11,
      quantity: 2,
      remark: 'gift',
    })
  })

  test('shop manage page creates a product', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch')
    fetchMock
      .mockResolvedValueOnce(
        jsonResponse({
          list: [
            {
              id: 1,
              name: 'Old Product',
              description: 'old',
              image: '',
              price: 100,
              stock: 10,
              max_per_user: 1,
              limit_period: 'forever',
              type: 'normal',
              status: 1,
              sort_order: 1,
              created_at: '2026-05-01T00:00:00Z',
              updated_at: '2026-05-01T00:00:00Z',
            },
          ],
          total: 1,
          page: 1,
          pageSize: 20,
        })
      )
      .mockResolvedValueOnce(
        jsonResponse({
          id: 2,
          name: 'New Product',
          description: 'new desc',
          image: 'https://example.com/new.png',
          price: 500,
          stock: 30,
          max_per_user: 5,
          limit_period: 'daily',
          type: 'normal',
          status: 1,
          sort_order: 2,
          created_at: '2026-05-01T00:00:00Z',
          updated_at: '2026-05-01T00:10:00Z',
        })
      )
      .mockResolvedValueOnce(
        jsonResponse({
          list: [
            {
              id: 1,
              name: 'Old Product',
              description: 'old',
              image: '',
              price: 100,
              stock: 10,
              max_per_user: 1,
              limit_period: 'forever',
              type: 'normal',
              status: 1,
              sort_order: 1,
              created_at: '2026-05-01T00:00:00Z',
              updated_at: '2026-05-01T00:00:00Z',
            },
            {
              id: 2,
              name: 'New Product',
              description: 'new desc',
              image: 'https://example.com/new.png',
              price: 500,
              stock: 30,
              max_per_user: 5,
              limit_period: 'daily',
              type: 'normal',
              status: 1,
              sort_order: 2,
              created_at: '2026-05-01T00:00:00Z',
              updated_at: '2026-05-01T00:10:00Z',
            },
          ],
          total: 2,
          page: 1,
          pageSize: 20,
        })
      )

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/shop/manage'],
    })

    render(<RouterProvider router={router} />)

    await screen.findByText('Old Product')
    await waitFor(() => expect(screen.getByRole('button', { name: '新增商品' })).toBeEnabled())

    fireEvent.click(screen.getByRole('button', { name: '新增商品' }))

    const dialog = await screen.findByRole('dialog', { name: '新增商品' })
    const textboxes = within(dialog).getAllByRole('textbox')
    const spinbuttons = within(dialog).getAllByRole('spinbutton')

    fireEvent.change(textboxes[0], { target: { value: 'New Product' } })
    fireEvent.change(textboxes[1], { target: { value: 'https://example.com/new.png' } })
    fireEvent.change(textboxes[2], { target: { value: 'new desc' } })
    fireEvent.change(spinbuttons[0], { target: { value: '500' } })
    fireEvent.change(spinbuttons[1], { target: { value: '30' } })
    fireEvent.change(spinbuttons[2], { target: { value: '5' } })
    fireEvent.change(spinbuttons[3], { target: { value: '2' } })

    fireEvent.click(within(dialog).getByRole('button', { name: '确认' }))

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledTimes(3)
    })

    const createCall = fetchMock.mock.calls[1]
    expect(String(createCall[0])).toContain('/api/v1/system/shop/product/add')
    expect(JSON.parse(String(createCall[1]?.body))).toMatchObject({
      name: 'New Product',
      image: 'https://example.com/new.png',
      description: 'new desc',
      price: 500,
      stock: 30,
      max_per_user: 5,
      sort_order: 2,
      status: 1,
      type: 'normal',
    })

    expect(await screen.findByText('New Product')).toBeInTheDocument()
  })

  test('shop order manage page delivers an order', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch')
    fetchMock
      .mockResolvedValueOnce(
        jsonResponse({
          list: [
            {
              id: 7,
              order_no: 'ORD-001',
              user_id: 1001,
              main_character_name: 'Amiya',
              nickname: 'Amiya',
              qq: '123',
              discord_id: '',
              product_id: 11,
              product_name: 'Pilot Pack',
              product_type: 'normal',
              quantity: 2,
              unit_price: 1200,
              total_price: 2400,
              status: 'requested',
              transaction_id: null,
              remark: 'please handle',
              reviewed_by: null,
              reviewed_at: null,
              reviewer_name: '',
              review_remark: '',
              created_at: '2026-05-01T00:00:00Z',
              updated_at: '2026-05-01T00:00:00Z',
            },
          ],
          total: 1,
          page: 1,
          pageSize: 20,
        })
      )
      .mockResolvedValueOnce(
        jsonResponse({
          list: [
            {
              id: 8,
              order_no: 'ORD-002',
              user_id: 1002,
              main_character_name: 'Bamiya',
              nickname: 'Bamiya',
              qq: '',
              discord_id: '',
              product_id: 12,
              product_name: 'Old Pack',
              product_type: 'normal',
              quantity: 1,
              unit_price: 500,
              total_price: 500,
              status: 'delivered',
              transaction_id: 99,
              remark: 'done',
              reviewed_by: 1,
              reviewed_at: '2026-05-01T00:00:00Z',
              reviewer_name: 'Operator',
              review_remark: 'ok',
              created_at: '2026-05-01T00:00:00Z',
              updated_at: '2026-05-01T00:00:00Z',
            },
          ],
          total: 1,
          page: 1,
          pageSize: 20,
        })
      )
      .mockResolvedValueOnce(
        jsonResponse({
          id: 7,
          order_no: 'ORD-001',
          user_id: 1001,
          main_character_name: 'Amiya',
          nickname: 'Amiya',
          qq: '123',
          discord_id: '',
          product_id: 11,
          product_name: 'Pilot Pack',
          product_type: 'normal',
          quantity: 2,
          unit_price: 1200,
          total_price: 2400,
          status: 'delivered',
          transaction_id: 101,
          remark: 'please handle',
          reviewed_by: 1,
          reviewed_at: '2026-05-01T00:10:00Z',
          reviewer_name: 'Admin',
          review_remark: 'delivered',
          created_at: '2026-05-01T00:00:00Z',
          updated_at: '2026-05-01T00:10:00Z',
          mail_id: 88,
          mail_url: 'https://example.com/mail/88',
        })
      )
      .mockResolvedValueOnce(
        jsonResponse({
          list: [],
          total: 0,
          page: 1,
          pageSize: 20,
        })
      )
      .mockResolvedValueOnce(
        jsonResponse({
          list: [
            {
              id: 8,
              order_no: 'ORD-002',
              user_id: 1002,
              main_character_name: 'Bamiya',
              nickname: 'Bamiya',
              qq: '',
              discord_id: '',
              product_id: 12,
              product_name: 'Old Pack',
              product_type: 'normal',
              quantity: 1,
              unit_price: 500,
              total_price: 500,
              status: 'delivered',
              transaction_id: 99,
              remark: 'done',
              reviewed_by: 1,
              reviewed_at: '2026-05-01T00:00:00Z',
              reviewer_name: 'Operator',
              review_remark: 'ok',
              created_at: '2026-05-01T00:00:00Z',
              updated_at: '2026-05-01T00:00:00Z',
            },
            {
              id: 7,
              order_no: 'ORD-001',
              user_id: 1001,
              main_character_name: 'Amiya',
              nickname: 'Amiya',
              qq: '123',
              discord_id: '',
              product_id: 11,
              product_name: 'Pilot Pack',
              product_type: 'normal',
              quantity: 2,
              unit_price: 1200,
              total_price: 2400,
              status: 'delivered',
              transaction_id: 101,
              remark: 'please handle',
              reviewed_by: 1,
              reviewed_at: '2026-05-01T00:10:00Z',
              reviewer_name: 'Admin',
              review_remark: 'delivered',
              created_at: '2026-05-01T00:00:00Z',
              updated_at: '2026-05-01T00:10:00Z',
            },
          ],
          total: 2,
          page: 1,
          pageSize: 20,
        })
      )

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/shop/order-manage'],
    })

    render(<RouterProvider router={router} />)

    await screen.findByText('ORD-001')
    await waitFor(() => expect(screen.getByRole('button', { name: '发放' })).toBeEnabled())

    fireEvent.click(screen.getByRole('button', { name: '发放' }))

    const dialog = await screen.findByRole('dialog', { name: '发放订单' })
    fireEvent.change(within(dialog).getByPlaceholderText('发放备注（可选）'), {
      target: { value: 'delivered in time' },
    })
    fireEvent.click(within(dialog).getByRole('button', { name: '确认发放' }))

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledTimes(5)
    })

    const deliverCall = fetchMock.mock.calls[2]
    expect(String(deliverCall[0])).toContain('/api/v1/system/shop/order/deliver')
    expect(JSON.parse(String(deliverCall[1]?.body))).toEqual({
      order_id: 7,
      remark: 'delivered in time',
    })
  })

  test('shop wallet page loads balance and transactions', async () => {
    vi.spyOn(globalThis, 'fetch')
      .mockResolvedValueOnce(
        jsonResponse({
          id: 1,
          user_id: 1001,
          balance: 54321,
          updated_at: '2026-05-01T00:00:00Z',
          character_name: 'Amiya',
        })
      )
      .mockResolvedValueOnce(
        jsonResponse({
          list: [
            {
              id: 1,
              user_id: 1001,
              amount: -1200,
              reason: '商城购买',
              ref_type: 'shop_purchase',
              ref_id: 'ORD-1001',
              balance_after: 53121,
              operator_id: 0,
              created_at: '2026-05-01T00:00:00Z',
              character_name: 'Amiya',
              nickname: 'Amiya',
              operator_name: 'System',
            },
          ],
          total: 1,
          page: 1,
          pageSize: 20,
        })
      )

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/shop/wallet'],
    })

    render(<RouterProvider router={router} />)

    await screen.findByText('54,321 伏羲币')
    expect(screen.getByText('商城购买')).toBeInTheDocument()
    expect(screen.getByText('53,121 伏羲币')).toBeInTheDocument()
  })
})
