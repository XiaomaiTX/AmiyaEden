import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const scriptDirectory = path.dirname(fileURLToPath(import.meta.url))
const localeDirectory = path.resolve(scriptDirectory, '../src/locales/langs')

function loadMessages(locale) {
  const filePath = path.join(localeDirectory, `${locale}.json`)
  return JSON.parse(fs.readFileSync(filePath, 'utf8').replace(/^\uFEFF/, ''))
}

function flattenKeys(value, prefix = '') {
  return Object.entries(value).flatMap(([key, child]) => {
    const fullKey = prefix ? `${prefix}.${key}` : key
    return child && typeof child === 'object' && !Array.isArray(child)
      ? flattenKeys(child, fullKey)
      : [fullKey]
  })
}

const zhKeys = new Set(flattenKeys(loadMessages('zh')))
const enKeys = new Set(flattenKeys(loadMessages('en')))
const errors = []

for (const key of zhKeys) {
  if (!enKeys.has(key)) errors.push(`Missing en translation: ${key}`)
}
for (const key of enKeys) {
  if (!zhKeys.has(key)) errors.push(`Missing zh translation: ${key}`)
}

if (errors.length > 0) {
  console.error(errors.join('\n'))
  process.exit(1)
}

console.log(`i18n check passed (${zhKeys.size} localized keys)`)
