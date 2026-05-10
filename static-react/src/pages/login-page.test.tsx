import userEvent from '@testing-library/user-event'
import { render, screen, waitFor } from '@testing-library/react'
import { RouterProvider, createMemoryRouter } from 'react-router-dom'
import { appRoutes } from '@/app/router'

describe('login page', () => {
  test('starts eve sso login flow', async () => {
    const fetchSpy = vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(
      new Response(JSON.stringify({ code: 0, msg: 'ok', data: { url: 'https://example.com/sso' } }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    )

    const router = createMemoryRouter(appRoutes, {
      initialEntries: ['/auth/login?redirect=%2Finfo%2Fwallet'],
    })

    render(<RouterProvider router={router} />)

    await userEvent.click(screen.getByRole('button', { name: /EVE SSO/i }))

    await waitFor(() => {
      expect(fetchSpy).toHaveBeenCalled()
    })
  })
})
