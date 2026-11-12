import { canAccessAdmission } from '../../utils/admissions'
import type { AdmissionContext } from '../../types/api/admissions'
import type { AppRouteRecord } from '../../types/router'

export function admissionMenuVisibility(
  menu: AppRouteRecord[],
  context: AdmissionContext | null
): AppRouteRecord[] {
  let changed = false
  const result = menu.map((item) => {
    const children = item.children ? admissionMenuVisibility(item.children, context) : undefined
    const isHide = item.meta.admission
      ? !canAccessAdmission(context, item.meta.admission)
      : item.meta.isHide
    if (isHide === item.meta.isHide && children === item.children) return item
    changed = true
    return { ...item, meta: { ...item.meta, isHide }, ...(children ? { children } : {}) }
  })
  return changed ? result : menu
}
