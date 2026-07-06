export const formatTime = (v: string | null | undefined) =>
  v ? new Date(v).toLocaleString('en-GB', { hour12: false }) : '-'

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
