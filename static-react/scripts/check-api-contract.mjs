import fs from 'node:fs'
import path from 'node:path'

const root = path.resolve('src')
const targets = ['api', 'pages', 'types']
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

for (const target of targets) {
  const dir = path.join(root, target)
  if (!fs.existsSync(dir)) {
    continue
  }

  for (const file of walk(dir)) {
    const normalized = file.replace(/\\/g, '/')
    if (allowTests.test(normalized) || allowContractStub.test(normalized)) {
      continue
    }

    const content = fs.readFileSync(file, 'utf8')
    for (const pattern of forbiddenPatterns) {
      if (pattern.test(content)) {
        violations.push(normalized)
        break
      }
    }
  }
}

if (violations.length > 0) {
  console.error('Forbidden Vue API contract references found:')
  for (const file of violations) {
    console.error(`- ${file}`)
  }
  process.exit(1)
}

