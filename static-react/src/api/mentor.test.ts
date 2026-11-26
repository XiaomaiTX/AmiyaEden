import { render, screen } from '@testing-library/react'
import { createElement } from 'react'
import { I18nProvider } from '@/i18n'
import { NewbroMentorManagePage } from '@/pages/newbro-mentor-manage-page'
import { adminGetMentorRewardStages, getMentorDashboardRewardStages } from './generated/client'
import { unwrapContract } from './response'

it('keeps administrator and mentor stage operations on their own permission boundaries', async () => {
  const fetch = vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
    if (String(input) === '/api/v1/system/mentor/reward-stages') {
      return new Response(JSON.stringify({ code: 200, msg: 'success', data: [] }), {
        headers: { 'Content-Type': 'application/json' },
      })
    }
    return new Response(JSON.stringify({ code: 403, msg: 'not a mentor', data: null }), {
      status: 403,
      headers: { 'Content-Type': 'application/json' },
    })
  })
  try {
    await expect(
      unwrapContract(adminGetMentorRewardStages(), 'admin stages failed')
    ).resolves.toEqual([])
    await expect(
      unwrapContract(getMentorDashboardRewardStages(), 'mentor stages failed')
    ).rejects.toThrow('not a mentor')
    expect(fetch.mock.calls.map(([input]) => input)).toEqual([
      '/api/v1/system/mentor/reward-stages',
      '/api/v1/mentor/dashboard/reward-stages',
    ])
  } finally {
    fetch.mockRestore()
  }
})

it('loads the administrator page with only the administrator operations', async () => {
  const fetch = vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
    const path = new URL(String(input), 'https://contract.test').pathname
    const responses: Record<string, unknown> = {
      '/api/v1/system/mentor/relationships': { list: [] },
      '/api/v1/system/mentor/reward-distributions': { list: [] },
      '/api/v1/system/mentor/reward-stages': [
        { id: 3, name: 'Admin reward stage', reward_amount: 50, stage_order: 1, threshold: 10 },
      ],
    }
    if (!(path in responses)) return new Response(null, { status: 403 })
    return new Response(JSON.stringify({ code: 200, msg: 'success', data: responses[path] }), {
      headers: { 'Content-Type': 'application/json' },
    })
  })
  try {
    render(createElement(I18nProvider, { children: createElement(NewbroMentorManagePage) }))
    expect(await screen.findByText('Admin reward stage')).toBeInTheDocument()
    expect(fetch).toHaveBeenCalledTimes(3)
  } finally {
    fetch.mockRestore()
  }
})
