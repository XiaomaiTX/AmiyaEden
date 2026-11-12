import { readFileSync, readdirSync } from 'node:fs'
import { resolve } from 'node:path'
import zh from './messages/zh-CN'
import en from './messages/en-US'
import { resolveLocaleText } from './context'
import { getAuditActionLabel, getAuditCategoryLabel } from '../lib/audit-labels'

const root = resolve(import.meta.dirname, '../../..')
const read = (path: string) => readFileSync(resolve(root, path), 'utf8')
const sources = (path: string) =>
  readdirSync(resolve(root, path))
    .filter((name) => name.endsWith('.go') && !name.endsWith('_test.go'))
    .map((name) => read(`${path}/${name}`))

test.each(['zh-CN', 'en-US'] as const)(
  '%s audit labels localize known values and retain unknown identifiers',
  (locale) => {
    const t = (key: string) => resolveLocaleText(locale, key)
    expect(getAuditActionLabel(t, 'audit_archive_daily')).not.toBe('audit_archive_daily')
    expect(getAuditActionLabel(t, 'create', 'corporation_admissions')).not.toBe('create')
    expect(getAuditCategoryLabel(t, 'system')).not.toBe('system')
    expect(getAuditActionLabel(t, 'future_event', 'corporation_admissions')).toBe('future_event')
    expect(getAuditActionLabel(t, 'create')).toBe('create')
    expect(getAuditCategoryLabel(t, 'future_category')).toBe('future_category')
  }
)

test('every backend task has a name and description in both frontend locales', () => {
  const jobs = sources('server/jobs').join('\n')
  const constants = new Map(
    [...jobs.matchAll(/\b(\w+)\s*=\s*"([a-z_]+)"/g)].map((match) => [match[1], match[2]])
  )
  const tasks = [...jobs.matchAll(/\bName:\s*(?:"([a-z_]+)"|(\w+))/g)].map((match) => {
    const name = match[1] || constants.get(match[2])
    expect(name, `unresolved task definition ${match[0]}`).toBeTruthy()
    return name!
  })
  const esiTasks = sources('server/pkg/eve/esi').flatMap((source) =>
    [...source.matchAll(/func \([^)]*\) Name\(\) string\s*\{\s*return "([a-z_]+)"/g)].map(
      (match) => match[1]
    )
  )
  expect(tasks.length).toBeGreaterThan(0)
  expect(esiTasks.length).toBeGreaterThan(0)
  const vueZh = JSON.parse(read('static/src/locales/langs/zh.json'))
  const vueEn = JSON.parse(read('static/src/locales/langs/en.json'))
  for (const [locale, dictionary] of [
    ['Vue zh', vueZh],
    ['Vue en', vueEn],
    ['React zh', zh],
    ['React en', en],
  ] as const) {
    for (const name of [...tasks, ...esiTasks]) {
      const catalog = dictionary.taskManager.tasks as Record<
        string,
        { name: string; description: string }
      >
      const entry = catalog[name]
      expect(entry?.name, `${locale}: ${name}.name`).toBeTruthy()
      expect(entry?.description, `${locale}: ${name}.description`).toBeTruthy()
      expect(entry?.name).not.toBe(name)
      if (locale.endsWith('zh')) {
        expect(entry.name).toMatch(/\p{Script=Han}/u)
        expect(entry.description).toMatch(/\p{Script=Han}/u)
      }
    }
  }
})

test('audit actions and categories emitted by services have translations in both frontends', () => {
  const serviceSources = sources('server/internal/service').filter((source) =>
    source.includes('AuditRecordInput')
  )
  const allSources = serviceSources.join('\n')
  const constants = new Map(
    [...allSources.matchAll(/\b(\w+)\s*=\s*"([a-z_]+)"/g)].map((match) => [match[1], match[2]])
  )
  const actions = new Set<string>()
  const categories = new Set<string>()
  for (const source of serviceSources) {
    for (const match of source.matchAll(/\bAction:\s*(?:"([a-z_]+)"|(\w+))/g)) {
      const action = match[1] || constants.get(match[2])
      if (action) actions.add(action)
    }
    for (const match of source.matchAll(
      /\.(?:record\w*Audit|audit)\("([a-z_]+)"|\baction\s*(?::=|=)\s*"([a-z_]+)"/g
    )) {
      const action = match[1] || match[2]
      actions.add(source.includes('recordAdmissionEvent') ? `admission_${action}` : action)
    }
    for (const match of source.matchAll(/\bCategory:\s*"([a-z_]+)"/g)) categories.add(match[1])
  }
  // Admission creation passes the action as the fourth argument rather than a named field.
  actions.add('admission_create')
  expect(actions.size).toBeGreaterThan(0)
  const vueZh = JSON.parse(read('static/src/locales/langs/zh.json'))
  const vueEn = JSON.parse(read('static/src/locales/langs/en.json'))
  for (const [locale, dictionary] of [
    ['Vue zh', vueZh],
    ['Vue en', vueEn],
    ['React zh', zh],
    ['React en', en],
  ] as const) {
    for (const [namespace, values] of [
      ['actionLabels', actions],
      ['categories', categories],
    ] as const) {
      const labels = dictionary.auditAdmin[namespace] as Record<string, string>
      for (const value of values) {
        expect(labels[value], `${locale}: ${namespace}.${value}`).toBeTruthy()
        if (locale.endsWith('zh')) expect(labels[value]).toMatch(/\p{Script=Han}/u)
      }
    }
  }
})
