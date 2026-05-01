import { describe, expect, test } from 'vitest'
import { resolveLocaleText } from '@/i18n'

describe('resolveLocaleText', () => {
  test('resolves Vue-style interpolation tokens', () => {
    expect(resolveLocaleText('zh-CN', 'characters.setPrimarySuccess', { name: 'Amiya' })).toBe(
      '已将 Amiya 设为主人物'
    )
  })

  test('resolves Vue-style reference aliases', () => {
    expect(resolveLocaleText('zh-CN', 'search.exitKeydown')).toBe('关闭')
  })

  test('resolves top-level ticket messages in zh-CN', () => {
    expect(resolveLocaleText('zh-CN', 'ticketMyTickets.title')).toBe('我的工单')
    expect(resolveLocaleText('zh-CN', 'ticketCreate.title')).toBe('提交工单')
    expect(resolveLocaleText('zh-CN', 'ticket.detailTitle')).toBe('工单详情')
  })

  test('resolves top-level ticket messages in en-US', () => {
    expect(resolveLocaleText('en-US', 'ticketMyTickets.title')).toBe('My Tickets')
    expect(resolveLocaleText('en-US', 'ticketCreate.title')).toBe('Create Ticket')
    expect(resolveLocaleText('en-US', 'ticket.detailTitle')).toBe('Ticket Details')
  })

  test('falls back to the key when text is missing', () => {
    expect(resolveLocaleText('zh-CN', 'missing.path')).toBe('missing.path')
  })
})
