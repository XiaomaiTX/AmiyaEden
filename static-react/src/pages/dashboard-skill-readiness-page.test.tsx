import userEvent from '@testing-library/user-event'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { I18nProvider } from '@/i18n'
import { DashboardSkillReadinessPage } from '@/pages/dashboard-skill-readiness-page'
import { usePreferenceStore, useSessionStore } from '@/stores'
import {
  fetchSkillReadinessCharacterSkills,
  fetchSkillReadinessOptions,
  fetchSkillReadinessUserCharacters,
  querySkillReadiness,
} from '@/api/skill-readiness'

vi.mock('@/api/skill-readiness', () => ({
  fetchSkillReadinessOptions: vi.fn(),
  querySkillReadiness: vi.fn(),
  fetchSkillReadinessUserCharacters: vi.fn(),
  fetchSkillReadinessCharacterSkills: vi.fn(),
}))

const requestResult = {
  summary: {
    character_count: 3,
    user_count: 2,
    satisfied_user_count: 1,
    satisfied_character_count: 1,
    unknown_character_count: 1,
  },
  list: [
    {
      user_id: 31,
      nickname: 'Pilot A',
      character_count: 2,
      satisfied_character_count: 1,
      unknown_character_count: 0,
    },
  ],
  total: 1,
  page: 1,
  pageSize: 20,
}

describe('dashboard skill readiness page', () => {
  beforeEach(() => {
    vi.mocked(fetchSkillReadinessOptions)
      .mockReset()
      .mockResolvedValue({
        corporations: [{ corporation_id: 10, corporation_name: 'Corp X' }],
        plans: [{ id: 7, title: 'Logistics', description: '', ship_type_id: null }],
      })
    vi.mocked(querySkillReadiness).mockReset().mockResolvedValue(requestResult)
    vi.mocked(fetchSkillReadinessUserCharacters)
      .mockReset()
      .mockResolvedValue([
        {
          user_id: 31,
          character_id: 1001,
          character_name: 'Pilot One',
          corporation_id: 10,
          corporation_name: 'Corp X',
          status: 'satisfied',
          matched_skills: 2,
          total_skills: 2,
          skill_updated_at: 1710000000,
          token_invalid: false,
        },
        {
          user_id: 31,
          character_id: 1002,
          character_name: 'Pilot Two',
          corporation_id: 10,
          corporation_name: 'Corp X',
          status: 'unknown',
          matched_skills: 0,
          total_skills: 2,
          skill_updated_at: null,
          token_invalid: false,
        },
      ])
    vi.mocked(fetchSkillReadinessCharacterSkills)
      .mockReset()
      .mockResolvedValue({
        character: {
          user_id: 31,
          character_id: 1001,
          character_name: 'Pilot One',
          corporation_id: 10,
          corporation_name: 'Corp X',
          status: 'satisfied',
          matched_skills: 1,
          total_skills: 1,
          skill_updated_at: 1710000000,
          token_invalid: true,
        },
        requirements: [
          {
            skill_type_id: 3300,
            skill_name: 'Damage Control',
            group_name: 'Engineering',
            active_level: 4,
            trained_level: 4,
            skillpoints_in_skill: 1000,
            required_level: 4,
            current_level: 4,
            satisfied: true,
          },
          {
            skill_type_id: 3301,
            skill_name: 'Hull Upgrades',
            group_name: 'Mechanics',
            active_level: 2,
            trained_level: 2,
            skillpoints_in_skill: 200,
            required_level: 4,
            current_level: 2,
            satisfied: false,
          },
        ],
        skills: [
          {
            skill_type_id: 3300,
            skill_name: 'Damage Control',
            group_name: 'Engineering',
            active_level: 4,
            trained_level: 4,
            skillpoints_in_skill: 1000,
            current_level: 4,
          },
        ],
      })
    useSessionStore.getState().setSessionSnapshot({
      isLoggedIn: true,
      accessToken: 'token',
      characterId: 1001,
      characterName: 'Admin',
      roles: ['admin'],
      corpCapabilities: ['menu.dashboard'],
    })
    usePreferenceStore.getState().setLocale('zh-CN')
  })

  test('restores URL filters, loads users, and opens scoped character skill comparison', async () => {
    const user = userEvent.setup()
    render(
      <I18nProvider>
        <MemoryRouter initialEntries={['/dashboard/skill-readiness?plan=7&corporations=10']}>
          <Routes>
            <Route path="/dashboard/skill-readiness" element={<DashboardSkillReadinessPage />} />
          </Routes>
        </MemoryRouter>
      </I18nProvider>
    )

    expect(await screen.findByText('Pilot A')).toBeTruthy()
    expect(querySkillReadiness).toHaveBeenCalledWith({
      corporation_ids: [10],
      plan_id: 7,
      page: 1,
      page_size: 20,
      language: 'zh',
    })
    await user.click(screen.getByRole('button', { name: '查看角色' }))
    expect(await screen.findByText('Pilot One')).toBeTruthy()
    await user.click(screen.getAllByRole('button', { name: '技能详情' })[0]!)
    expect(await screen.findByText('Damage Control')).toBeTruthy()
    expect(screen.getByText('ESI 授权已失效；当前结果使用最近一次技能快照。')).toBeTruthy()
    expect(screen.getByText('Hull Upgrades')).toBeTruthy()

    await user.click(screen.getByLabelText('仅看未达标技能'))
    await waitFor(() => expect(screen.queryByText('Damage Control')).toBeNull())
    expect(screen.getByText('Hull Upgrades')).toBeTruthy()
    expect(fetchSkillReadinessUserCharacters).toHaveBeenCalledWith(
      31,
      expect.objectContaining({ corporation_ids: [10], plan_id: 7 })
    )
    expect(fetchSkillReadinessCharacterSkills).toHaveBeenCalledWith(
      1001,
      expect.objectContaining({ corporation_ids: [10], plan_id: 7 })
    )
  })
})
