export function humanizeNumber(num: number | null | undefined): string {
  if (num == null) {
    return '-'
  }

  const value = Number(num)
  const abs = Math.abs(value)

  if (abs >= 1e12) {
    return `${(value / 1e12).toFixed(2).replace(/\.00$/, '')}t`
  }

  if (abs >= 1e9) {
    return `${(value / 1e9).toFixed(2).replace(/\.00$/, '')}b`
  }

  if (abs >= 1e6) {
    return `${(value / 1e6).toFixed(2).replace(/\.00$/, '')}m`
  }

  if (abs >= 1e3) {
    return `${(value / 1e3).toFixed(2).replace(/\.00$/, '')}k`
  }

  return value.toString()
}

export function formatTime(value: string | null | undefined): string {
  if (!value) {
    return '-'
  }

  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) {
    return value
  }

  return parsed.toLocaleString('en-GB', { hour12: false })
}

// 整月差（UTC 口径）：fuel_expires 所在月距当前月还有几个月。
// 与后端 EstimateFuelToMonthEnd 的「耗尽所在自然月（EVE UTC）月底」保持同一时区基准，
// 避免月末/月初时本地时区与 UTC 月份不一致导致徽标口径漂移。解析失败返回 null。
export function fuelExpiryMonthOffset(
  fuelExpires: string,
  now: Date = new Date()
): number | null {
  const expiry = new Date(fuelExpires)
  if (Number.isNaN(expiry.getTime())) {
    return null
  }
  return (
    (expiry.getUTCFullYear() - now.getUTCFullYear()) * 12 +
    (expiry.getUTCMonth() - now.getUTCMonth())
  )
}
