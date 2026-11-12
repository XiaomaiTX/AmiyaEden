type TranslateFn = (key: string) => string

function auditLabel(t: TranslateFn, kind: string, value: string, fallback = value) {
  const key = `auditAdmin.${kind}.${value}`
  const translated = t(key)
  return translated === key ? fallback : translated
}

export const getAuditCategoryLabel = (t: TranslateFn, category: string) =>
  auditLabel(t, 'categories', category)

export const getAuditActionLabel = (t: TranslateFn, action: string, category?: string) =>
  auditLabel(
    t,
    'actionLabels',
    category === 'corporation_admissions' ? `admission_${action}` : action,
    action
  )
