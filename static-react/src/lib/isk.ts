const plainFormatter = new Intl.NumberFormat('en-US', {
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
})

export function formatIskPlain(value: number | null | undefined) {
  if (value == null) {
    return '-'
  }

  return plainFormatter.format(Number(value))
}
