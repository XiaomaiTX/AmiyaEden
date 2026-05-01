import { beforeEach, describe, expect, test, vi } from 'vitest'

vi.mock('@/api/http-client', () => ({
  requestJson: vi.fn(),
}))

import { requestJson } from '@/api/http-client'
import { fetchPersonalSkillPlanList, fetchSkillPlanList } from '@/api/skill-plan'

describe('skill plan api', () => {
  beforeEach(() => {
    vi.mocked(requestJson).mockReset()
  })

  test('fetchSkillPlanList uses GET with query params', async () => {
    vi.mocked(requestJson).mockResolvedValueOnce({
      code: 0,
      data: { list: [], total: 0, page: 1, pageSize: 10 },
      msg: '',
    })

    await fetchSkillPlanList({ current: 2, size: 20, keyword: 'logi' })

    expect(requestJson).toHaveBeenCalledWith(
      '/api/v1/skill-planning/skill-plans?current=2&size=20&keyword=logi',
      expect.objectContaining({
        method: 'GET',
      })
    )
  })

  test('fetchPersonalSkillPlanList uses GET with default pagination', async () => {
    vi.mocked(requestJson).mockResolvedValueOnce({
      code: 0,
      data: { list: [], total: 0, page: 1, pageSize: 10 },
      msg: '',
    })

    await fetchPersonalSkillPlanList()

    expect(requestJson).toHaveBeenCalledWith(
      '/api/v1/skill-planning/personal-skill-plans?current=1&size=10',
      expect.objectContaining({
        method: 'GET',
      })
    )
  })
})
