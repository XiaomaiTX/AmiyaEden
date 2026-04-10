type TranslateFn = (key: string) => string

function translateWithFallback(t: TranslateFn, key: string, fallback: string) {
  const translated = t(key)
  return translated === key ? fallback : translated
}

export function getTaskDisplayName(t: TranslateFn, taskName: string, fallback: string) {
  return translateWithFallback(t, `taskManager.tasks.${taskName}.name`, fallback)
}

export function getTaskDisplayDescription(t: TranslateFn, taskName: string, fallback: string) {
  return translateWithFallback(t, `taskManager.tasks.${taskName}.description`, fallback)
}
