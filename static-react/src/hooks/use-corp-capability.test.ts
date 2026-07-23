import { act, renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, test } from 'vitest'
import { useCorpCapability } from '@/hooks/use-corp-capability'
import { useSessionStore } from '@/stores'

describe('useCorpCapability', () => {
  beforeEach(() => {
    useSessionStore.getState().clearSession()
  })

  test('hasCapability returns true when capability is in session', () => {
    act(() => {
      useSessionStore.getState().setSessionSnapshot({
        roles: ['admin'],
        corpCapabilities: ['shop.order.create'],
      })
    })

    const { result } = renderHook(() => useCorpCapability())
    expect(result.current.hasCapability('shop.order.create')).toBe(true)
    expect(result.current.hasCapability('system.task.run')).toBe(false)
  })

  test('hasCapability short-circuits to true for super_admin', () => {
    act(() => {
      useSessionStore.getState().setSessionSnapshot({
        roles: ['super_admin'],
        corpCapabilities: [],
      })
    })

    const { result } = renderHook(() => useCorpCapability())
    expect(result.current.hasCapability('any.capability')).toBe(true)
  })

  test('hasAllCapabilities enforces AND semantics', () => {
    act(() => {
      useSessionStore.getState().setSessionSnapshot({
        roles: ['admin'],
        corpCapabilities: ['shop.manage', 'shop.admin.product.manage'],
      })
    })

    const { result } = renderHook(() => useCorpCapability())
    expect(
      result.current.hasAllCapabilities(['shop.manage', 'shop.admin.product.manage'])
    ).toBe(true)
    expect(result.current.hasAllCapabilities(['shop.manage', 'ticket.manage'])).toBe(false)
  })

  test('hasAnyCapability enforces OR semantics', () => {
    act(() => {
      useSessionStore.getState().setSessionSnapshot({
        roles: ['admin'],
        corpCapabilities: ['srp.user'],
      })
    })

    const { result } = renderHook(() => useCorpCapability())
    expect(result.current.hasAnyCapability(['srp.user', 'srp.manage'])).toBe(true)
    expect(result.current.hasAnyCapability(['srp.manage', 'ticket.manage'])).toBe(false)
  })

  test('empty capability lists return true', () => {
    act(() => {
      useSessionStore.getState().setSessionSnapshot({
        roles: ['admin'],
        corpCapabilities: [],
      })
    })

    const { result } = renderHook(() => useCorpCapability())
    expect(result.current.hasAllCapabilities([])).toBe(true)
    expect(result.current.hasAnyCapability([])).toBe(true)
  })
})
