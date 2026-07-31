import fs from 'node:fs'
import path from 'node:path'
import ts from 'typescript'

const sourceRoot = path.resolve('src')
const messagesRoot = path.join(sourceRoot, 'i18n', 'messages')
const auditedRoots = ['pages', 'layout', 'components', 'feedback']
const ignoredPaths = new Set(['components/ui'])

function walk(directory) {
  return fs.readdirSync(directory, { withFileTypes: true }).flatMap((entry) =>
    entry.isDirectory() ? walk(path.join(directory, entry.name)) : [path.join(directory, entry.name)]
  )
}

function loadDictionary(filePath) {
  const source = fs
    .readFileSync(filePath, 'utf8')
    .replace(/^\uFEFF/, '')
    .replace(/^const \w+ = /, 'module.exports = ')
    .replace(/\s+as const\s*\n\s*export default \w+\s*;?\s*$/, '\n')
  const module = { exports: {} }
  new Function('module', source)(module)
  return module.exports
}

function leafKeys(value, prefix = '') {
  return Object.entries(value).flatMap(([key, child]) =>
    child && typeof child === 'object' ? leafKeys(child, `${prefix}${key}.`) : [`${prefix}${key}`]
  )
}

function jsxText(value) {
  return value.replace(/\s+/g, ' ').trim()
}

const zhKeys = new Set(leafKeys(loadDictionary(path.join(messagesRoot, 'zh-CN.ts'))))
const enKeys = new Set(leafKeys(loadDictionary(path.join(messagesRoot, 'en-US.ts'))))
const errors = []

for (const key of zhKeys) {
  if (!enKeys.has(key)) errors.push(`Missing en-US key: ${key}`)
}
for (const key of enKeys) {
  if (!zhKeys.has(key)) errors.push(`Missing zh-CN key: ${key}`)
}

for (const root of auditedRoots) {
  const directory = path.join(sourceRoot, root)
  for (const filePath of walk(directory).filter((file) => /\.(ts|tsx)$/.test(file) && !file.includes('.test.'))) {
    const relativePath = path.relative(sourceRoot, filePath).replaceAll('\\', '/')
    if ([...ignoredPaths].some((prefix) => relativePath.startsWith(prefix))) continue

    const source = fs.readFileSync(filePath, 'utf8')
    for (const match of source.matchAll(/\bt\(\s*['"]([^'"]+)['"]/g)) {
      const key = match[1]
      if (!zhKeys.has(key) || !enKeys.has(key)) errors.push(`${relativePath}: missing i18n key ${key}`)
    }

    const ast = ts.createSourceFile(filePath, source, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX)
    const visit = (node) => {
      if (ts.isJsxText(node)) {
        const value = jsxText(node.getText(ast))
        if (value && /[\p{L}]/u.test(value)) errors.push(`${relativePath}: hardcoded JSX text "${value}"`)
      }
      if (ts.isJsxAttribute(node) && node.initializer && ts.isStringLiteral(node.initializer)) {
        const name = node.name.text
        if (['title', 'placeholder', 'aria-label', 'alt'].includes(name) && /[\p{L}]/u.test(node.initializer.text)) {
          errors.push(`${relativePath}: hardcoded ${name} "${node.initializer.text}"`)
        }
      }
      ts.forEachChild(node, visit)
    }
    visit(ast)
  }
}

if (errors.length) {
  console.error(errors.join('\n'))
  process.exit(1)
}

console.log(`i18n check passed (${zhKeys.size} localized keys)`)
