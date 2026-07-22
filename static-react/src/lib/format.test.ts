import { describe, expect, it } from 'vitest'
import { formatIskPlain, formatIskSmart } from './isk'
import { formatTime, humanizeNumber } from './format'

describe('formatIskSmart', () => {
  it('applies unit thresholds and promotion', () => {
    expect(formatIskSmart(950)).toBe('950.00')
    expect(formatIskSmart(12_500)).toBe('12.50 K')
    expect(formatIskSmart(711_103_702.38)).toBe('711.10 M')
    expect(formatIskSmart(1_250_000_000)).toBe('1.25 B')
    expect(formatIskSmart(2_400_000_000_000)).toBe('2.40 T')
    expect(formatIskSmart(999_995)).toBe('1.00 M')
    expect(formatIskSmart(-12_500)).toBe('-12.50 K')
    expect(formatIskSmart(null)).toBe('-')
    expect(formatIskSmart(undefined)).toBe('-')
  })
})

describe('formatIskPlain', () => {
  it('returns dash for null or undefined', () => {
    expect(formatIskPlain(null)).toBe('-')
    expect(formatIskPlain(undefined)).toBe('-')
  })

  it('formats numbers with two decimals', () => {
    expect(formatIskPlain(1234.5)).toBe('1,234.50')
  })
})

describe('humanizeNumber', () => {
  it('returns dash for null or undefined', () => {
    expect(humanizeNumber(null)).toBe('-')
    expect(humanizeNumber(undefined)).toBe('-')
  })

  it('abbreviates large numbers', () => {
    expect(humanizeNumber(1_500)).toBe('1.50k')
    expect(humanizeNumber(2_500_000)).toBe('2.50m')
    expect(humanizeNumber(3_200_000_000)).toBe('3.20b')
    expect(humanizeNumber(4_100_000_000_000)).toBe('4.10t')
    expect(humanizeNumber(1_000_000)).toBe('1m')
  })

  it('returns plain number for small values', () => {
    expect(humanizeNumber(42)).toBe('42')
  })

  it('keeps sign for negative values', () => {
    expect(humanizeNumber(-2_500_000)).toBe('-2.50m')
  })
})

describe('formatTime', () => {
  it('returns dash for empty values', () => {
    expect(formatTime(null)).toBe('-')
    expect(formatTime(undefined)).toBe('-')
    expect(formatTime('')).toBe('-')
  })

  it('returns original value when the timestamp is invalid', () => {
    expect(formatTime('not-a-date')).toBe('not-a-date')
  })

  it('formats valid timestamps as localized strings', () => {
    const value = formatTime('2026-07-22T08:30:00Z')
    expect(typeof value).toBe('string')
    expect(value).toContain('2026')
  })
})
