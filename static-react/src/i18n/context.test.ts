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

  test('falls back to the key when text is missing', () => {
    expect(resolveLocaleText('zh-CN', 'missing.path')).toBe('missing.path')
  })
})
