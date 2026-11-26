import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const root = fileURLToPath(new URL('../src', import.meta.url))
const allowTests = /\.(test|spec)\.[jt]sx?$/
const allowContractStub = /src[\\/]+types[\\/]+api-contract\.d\.ts$/
const forbiddenPatterns = [
  /static\/src\/types\/api\/api/,
  /\bApi\./,
]

function walk(dir) {
  const entries = fs.readdirSync(dir, { withFileTypes: true })
  const files = []

  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name)
    if (entry.isDirectory()) {
      files.push(...walk(fullPath))
    } else if (/\.(ts|tsx|d\.ts)$/.test(entry.name)) {
      files.push(fullPath)
    }
  }

  return files
}

const violations = []

for (const file of walk(root)) {
  const normalized = file.replace(/\\/g, '/')
  if (allowTests.test(normalized) || allowContractStub.test(normalized)) {
    continue
  }

  const content = fs.readFileSync(file, 'utf8')
  if (forbiddenPatterns.some((pattern) => pattern.test(content))) {
    violations.push(`${normalized}: forbidden Vue API contract reference`)
  }
  if (!normalized.includes('/api/generated/') && /["'`][/]api[/]v1(?:[/"'`?]|$)/m.test(content)) {
    violations.push(`${normalized}: use the generated client for /api/v1 calls`)
  }
}

if (violations.length > 0) {
  console.error('API contract violations found:')
  for (const file of violations) {
    console.error(`- ${file}`)
  }
  process.exit(1)
}

