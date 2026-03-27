export interface FleetSearchForm {
  importance: string | undefined
}

export function buildFleetSearchForm(
  current: FleetSearchForm,
  importance: FleetSearchForm['importance']
): FleetSearchForm {
  return {
    ...current,
    importance
  }
}
