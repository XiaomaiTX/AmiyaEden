import userEvent from '@testing-library/user-event'
import { render, screen, waitFor } from '@testing-library/react'
import { RouterProvider, createMemoryRouter } from 'react-router-dom'
import { appRoutes } from '@/app/router'

describe('recruit landing page', () => {
  function setUp(code: string) {
    const router = createMemoryRouter(appRoutes, {
      initialEntries: [`/r/${code}`],
    })
    render(<RouterProvider router={router} />)
  }

  test('renders recruit form', () => {
    setUp('test-code')

    expect(screen.getByRole('heading', { name: '加入我们' })).toBeInTheDocument()
    expect(screen.getByRole('textbox')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '提交' })).toBeInTheDocument()
  })

  test('shows validation error when QQ is empty', async () => {
    setUp('test-code')

    await userEvent.click(screen.getByRole('button', { name: '提交' }))

    await waitFor(() => {
      expect(screen.getByText('请输入 QQ 号码')).toBeInTheDocument()
    })
  })

  test('shows validation error for invalid QQ format', async () => {
    setUp('test-code')

    await userEvent.type(screen.getByRole('textbox'), 'abc')
    await userEvent.click(screen.getByRole('button', { name: '提交' }))

    await waitFor(() => {
      expect(screen.getByText('QQ 号码格式不正确（5-20 位数字）')).toBeInTheDocument()
    })
  })

  test('shows validation error for too-short QQ', async () => {
    setUp('test-code')

    await userEvent.type(screen.getByRole('textbox'), '1234')
    await userEvent.click(screen.getByRole('button', { name: '提交' }))

    await waitFor(() => {
      expect(screen.getByText('QQ 号码格式不正确（5-20 位数字）')).toBeInTheDocument()
    })
  })

  test('submits successfully and shows redirect button', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(
      new Response(
        JSON.stringify({ code: 0, msg: 'ok', data: { qq_url: 'https://qun.qq.com/join' } }),
        {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        }
      )
    )

    setUp('test-code')

    await userEvent.type(screen.getByRole('textbox'), '123456789')
    await userEvent.click(screen.getByRole('button', { name: '提交' }))

    await waitFor(() => {
      expect(screen.getByText('提交成功')).toBeInTheDocument()
      expect(screen.getByText('感谢您的参与！点击下方按钮加入 QQ 群')).toBeInTheDocument()
      expect(screen.getByRole('button', { name: '加入 QQ 群' })).toBeInTheDocument()
    })
  })

  test('shows submit error on API failure', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(
      new Response(
        JSON.stringify({ code: 500, msg: 'server error', data: null }),
        {
          status: 500,
          headers: { 'Content-Type': 'application/json' },
        }
      )
    )

    setUp('test-code')

    await userEvent.type(screen.getByRole('textbox'), '123456789')
    await userEvent.click(screen.getByRole('button', { name: '提交' }))

    await waitFor(() => {
      expect(screen.getByText('server error')).toBeInTheDocument()
    })
  })

  test('redirect button calls window.open with qq_url', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(
      new Response(
        JSON.stringify({ code: 0, msg: 'ok', data: { qq_url: 'https://qun.qq.com/join' } }),
        {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        }
      )
    )

    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null)

    setUp('test-code')

    await userEvent.type(screen.getByRole('textbox'), '123456789')
    await userEvent.click(screen.getByRole('button', { name: '提交' }))

    await waitFor(() => {
      expect(screen.getByRole('button', { name: '加入 QQ 群' })).toBeInTheDocument()
    })

    await userEvent.click(screen.getByRole('button', { name: '加入 QQ 群' }))

    expect(openSpy).toHaveBeenCalledWith('https://qun.qq.com/join', '_blank', 'noopener,noreferrer')
  })

  test('clears validation error when user types', async () => {
    setUp('test-code')

    await userEvent.click(screen.getByRole('button', { name: '提交' }))

    await waitFor(() => {
      expect(screen.getByText('请输入 QQ 号码')).toBeInTheDocument()
    })

    await userEvent.type(screen.getByRole('textbox'), '1')

    expect(screen.queryByText('请输入 QQ 号码')).not.toBeInTheDocument()
  })
})
