export function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof Error && error.message ? error.message : fallback
}

export function formatDateTime(value: string | null | undefined) {
  if (!value) {
    return '-'
  }

  const parsed = new Date(value)
  return Number.isNaN(parsed.getTime()) ? value : parsed.toLocaleString()
}

export function formatNumber(value: number | null | undefined) {
  if (value === null || value === undefined) {
    return '-'
  }

  return new Intl.NumberFormat().format(value)
}
