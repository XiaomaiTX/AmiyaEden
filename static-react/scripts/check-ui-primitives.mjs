import fs from 'node:fs'
import path from 'node:path'
import ts from 'typescript'

const sourceRoot = path.resolve('src')
const auditedRoots = ['.']
const ignoredPaths = new Set(['components/ui'])
const errors = []

function walk(directory) {
  return fs
    .readdirSync(directory, { withFileTypes: true })
    .flatMap((entry) =>
      entry.isDirectory()
        ? walk(path.join(directory, entry.name))
        : [path.join(directory, entry.name)]
    )
}

for (const root of auditedRoots) {
  const directory = path.join(sourceRoot, root)
  for (const filePath of walk(directory).filter((file) => /\.(ts|tsx)$/.test(file))) {
    const relativePath = path.relative(sourceRoot, filePath).replaceAll('\\', '/')
    if ([...ignoredPaths].some((prefix) => relativePath.startsWith(prefix))) continue

    const source = fs.readFileSync(filePath, 'utf8')
    if (/<select\b/.test(source))
      errors.push(`${relativePath}: use shadcn Select or Combobox instead of <select>`)
    if (/NativeSelect/.test(source))
      errors.push(`${relativePath}: NativeSelect has been removed; use Select or Combobox`)

    for (const dialogBlock of source.matchAll(/<ShopDialog\b[\s\S]*?<\/ShopDialog>/g)) {
      if (/rounded-lg\s+border\s+bg-card|shadow-xl/.test(dialogBlock[0])) {
        errors.push(
          `${relativePath}: ShopDialog content must not add a second card surface or shadow`
        )
      }
    }

    const ast = ts.createSourceFile(
      filePath,
      source,
      ts.ScriptTarget.Latest,
      true,
      ts.ScriptKind.TSX
    )
    const inputNames = new Set(['Input'])

    const getAttribute = (attributes, name) =>
      attributes.find(
        (attribute) => ts.isJsxAttribute(attribute) && attribute.name.text === name
      )

    const getStaticAttributeValue = (attribute) => {
      if (!attribute || !ts.isJsxAttribute(attribute)) return null
      if (!attribute.initializer) return true
      if (ts.isStringLiteral(attribute.initializer)) return attribute.initializer.text
      if (
        ts.isJsxExpression(attribute.initializer) &&
        attribute.initializer.expression &&
        ts.isStringLiteral(attribute.initializer.expression)
      ) {
        return attribute.initializer.expression.text
      }
      return undefined
    }

    const visit = (node) => {
      if (ts.isJsxOpeningElement(node) || ts.isJsxSelfClosingElement(node)) {
        const tagName = node.tagName.getText(ast)
        const attributes = node.attributes.properties
        const typeAttribute = getAttribute(attributes, 'type')
        const typeValue = getStaticAttributeValue(typeAttribute)
        const isNativeInput = tagName === 'input'
        const isSharedInput = inputNames.has(tagName)

        if (
          (isNativeInput || isSharedInput) &&
          (typeValue === 'checkbox' || typeValue === 'radio' || typeValue === undefined)
        ) {
          const { line } = ast.getLineAndCharacterOfPosition(node.getStart(ast))
          errors.push(
            `${relativePath}:${line + 1}: use shadcn Checkbox or RadioGroup instead of native ${tagName} checkbox/radio controls`
          )
        }

        if (tagName === 'Button') {
          const variant = attributes.find(
            (attribute) => ts.isJsxAttribute(attribute) && attribute.name.text === 'variant'
          )
          const className = attributes.find(
            (attribute) => ts.isJsxAttribute(attribute) && attribute.name.text === 'className'
          )
          const usesDefaultVariant =
            !variant ||
            (ts.isJsxAttribute(variant) &&
              variant.initializer &&
              ts.isStringLiteral(variant.initializer) &&
              variant.initializer.text === 'default')
          const classText =
            className && ts.isJsxAttribute(className) && className.initializer
              ? className.initializer.getText(ast)
              : ''

          if (
            usesDefaultVariant &&
            /(?:^|[\s'"`])bg-(?:background|card|muted|primary\/\d+)(?:\/[\w.]+)?(?=$|[\s'"`])/.test(
              classText
            )
          ) {
            const { line } = ast.getLineAndCharacterOfPosition(node.getStart(ast))
            errors.push(
              `${relativePath}:${line + 1}: surface backgrounds on Button require an explicit non-default variant`
            )
          }
        }

        if (tagName === 'SelectTrigger' || tagName === 'Textarea') {
          const className = getAttribute(attributes, 'className')
          const classText =
            className && ts.isJsxAttribute(className) && className.initializer
              ? className.initializer.getText(ast)
              : ''
          if (
            /(?:rounded-(?:md|lg)|border(?:-input)?|bg-background|px-[23]|py-2|text-sm|outline-none|focus-visible:)/.test(
              classText
            )
          ) {
            const { line } = ast.getLineAndCharacterOfPosition(node.getStart(ast))
            errors.push(
              `${relativePath}:${line + 1}: ${tagName} should rely on shadcn defaults; keep only layout or size classes`
            )
          }
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

console.log('ui primitive check passed')
