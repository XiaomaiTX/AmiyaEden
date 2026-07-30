import { beforeEach, describe, expect, test, vi } from 'vitest'

vi.mock('@/api/http-client', () => ({ requestJson: vi.fn() }))

import { requestJson } from '@/api/http-client'
import { fetchQQGovernanceMembers, fetchQQGovernanceReviews, searchQQGovernanceCorporations } from '@/api/qq-governance'

describe('QQ governance API', () => {
  beforeEach(() => { vi.mocked(requestJson).mockReset() })

  test('uses server pagination filters and the corporation query parameter', async () => {
    vi.mocked(requestJson)
      .mockResolvedValueOnce({ code: 0, msg: '', data: { list: [], total: 0, page: 2, page_size: 20 } })
      .mockResolvedValueOnce({ code: 0, msg: '', data: { list: [], total: 0, page: 3, page_size: 50 } })
      .mockResolvedValueOnce({ code: 0, msg: '', data: [] })

    await fetchQQGovernanceMembers({ page: 2, page_size: 20, group_id: 1, qq: 2, status: 'review' })
    await fetchQQGovernanceReviews({ page: 3, page_size: 50, group_id: 1, qq: 2, decision: 'matched' })
    await searchQQGovernanceCorporations('Amiya Eden')

    expect(requestJson).toHaveBeenNthCalledWith(1, '/api/v1/system/qq-governance/members?page=2&page_size=20&group_id=1&qq=2&status=review')
    expect(requestJson).toHaveBeenNthCalledWith(2, '/api/v1/system/qq-governance/reviews?page=3&page_size=50&group_id=1&qq=2&decision=matched')
    expect(requestJson).toHaveBeenNthCalledWith(3, '/api/v1/system/qq-governance/corporations?query=Amiya%20Eden')
  })
})
