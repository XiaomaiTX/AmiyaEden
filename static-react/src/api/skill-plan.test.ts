import { beforeEach, describe, expect, test, vi } from 'vitest'

vi.mock('@/api/http-client', () => ({
  requestJson: vi.fn(),
}))

import { requestJson } from '@/api/http-client'
import { listPersonalSkillPlans, listSkillPlans } from '@/api/generated/client'

describe('skill plan api', () => {
  beforeEach(() => {
    vi.mocked(requestJson).mockReset()
  })

  test('listSkillPlans uses GET with query params', async () => {
    vi.mocked(requestJson).mockResolvedValueOnce({
      code: 200,
      data: { list: [], total: 0, page: 1, pageSize: 10 },
      msg: '',
    })

    await listSkillPlans({ current: 2, size: 20, keyword: 'logi' })

    expect(requestJson).toHaveBeenCalledWith(
      '/api/v1/skill-planning/skill-plans?current=2&size=20&keyword=logi',
      expect.objectContaining({
        method: 'GET',
      })
    )
  })

  test('listPersonalSkillPlans uses GET with explicit pagination', async () => {
    vi.mocked(requestJson).mockResolvedValueOnce({
      code: 200,
      data: { list: [], total: 0, page: 1, pageSize: 10 },
      msg: '',
    })

    await listPersonalSkillPlans({ current: 1, size: 10 })

    expect(requestJson).toHaveBeenCalledWith(
      '/api/v1/skill-planning/personal-skill-plans?current=1&size=10',
      expect.objectContaining({
        method: 'GET',
      })
    )
  })
})
