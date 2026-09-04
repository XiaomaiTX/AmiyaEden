export const formatTime = (v: string | null | undefined) =>
  v ? new Date(v).toLocaleString('en-GB', { hour12: false }) : '-'

/**
 * 整月差（UTC 口径）：fuel_expires 所在月距当前月还有几个月。
 * 与后端 EstimateFuelToMonthEnd 的「耗尽所在自然月（EVE UTC）月底」保持同一时区基准，
 * 避免月末/月初时本地时区与 UTC 月份不一致导致徽标口径漂移。解析失败返回 null。
 */
export const fuelExpiryMonthOffset = (
  fuelExpires: string,
  now: Date = new Date()
): number | null => {
  const expiry = new Date(fuelExpires)
  if (Number.isNaN(expiry.getTime())) return null
  return (
    (expiry.getUTCFullYear() - now.getUTCFullYear()) * 12 +
    (expiry.getUTCMonth() - now.getUTCMonth())
  )
}

/**
 * Convert a browser-local naive datetime string (e.g. "YYYY-MM-DD HH:mm:ss",
 * as emitted by ElDatePicker's value-format) into a self-describing RFC3339
 * string carrying the browser timezone offset (e.g.
 * "2026-07-06T14:00:00+08:00"). Sending an offset frees the backend from
 * guessing the client timezone when parsing expected_end_at.
 */
export const toLocalOffsetISO = (localNaive: string): string => {
  const date = new Date(localNaive.replace(' ', 'T'))
  const pad = (n: number) => String(n).padStart(2, '0')
  const offsetMinutes = -date.getTimezoneOffset()
  const sign = offsetMinutes >= 0 ? '+' : '-'
  const absMinutes = Math.abs(offsetMinutes)
  const offsetHours = pad(Math.floor(absMinutes / 60))
  const offsetMins = pad(absMinutes % 60)
  return (
    `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}` +
    `T${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}` +
    `${sign}${offsetHours}:${offsetMins}`
  )
}
