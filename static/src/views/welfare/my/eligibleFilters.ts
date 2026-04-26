export interface EligibleFilters {
  roleFilter: string
  welfareNameFilter: string
}

export interface EligibleFilterRowLike {
  welfareName: string
  isNaturalPersonRow: boolean
  roleFilterValue: string
}

export interface EligibleFilterOption {
  label: string
  value: string
}

export function filterEligibleRows<T extends EligibleFilterRowLike>(
  rows: T[],
  filters: EligibleFilters
): T[] {
  return rows.filter((row) => {
    if (filters.roleFilter && row.roleFilterValue !== filters.roleFilter) {
      return false
    }

    if (filters.welfareNameFilter && row.welfareName !== filters.welfareNameFilter) {
      return false
    }

    return true
  })
}

export function paginateEligibleRows<T>(rows: T[], current: number, size: number): T[] {
  const start = Math.max(0, (current - 1) * size)
  const end = Math.max(start, current * size)
  return rows.slice(start, end)
}

export function buildRoleFilterOptions<T extends EligibleFilterRowLike>(
  rows: T[],
  naturalPersonLabel: string
): EligibleFilterOption[] {
  const characterRoles = Array.from(
    new Set(
      rows
        .filter((row) => !row.isNaturalPersonRow)
        .map((row) => row.roleFilterValue)
        .filter((value) => value)
    )
  )

  return [
    { label: naturalPersonLabel, value: naturalPersonLabel },
    ...characterRoles.map((value) => ({ label: value, value }))
  ]
}

export function buildWelfareNameFilterOptions<T extends EligibleFilterRowLike>(
  rows: T[]
): EligibleFilterOption[] {
  return Array.from(new Set(rows.map((row) => row.welfareName).filter((name) => name))).map(
    (value) => ({
      label: value,
      value
    })
  )
}
