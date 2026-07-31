import fs from 'node:fs'
import path from 'node:path'
import ts from 'typescript'

const sourceRoot = path.resolve('src')
const auditedRoots = ['pages', 'layout', 'components']
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

    const ast = ts.createSourceFile(
      filePath,
      source,
      ts.ScriptTarget.Latest,
      true,
      ts.ScriptKind.TSX
    )
    const visit = (node) => {
      if (ts.isJsxOpeningElement(node) || ts.isJsxSelfClosingElement(node)) {
        if (node.tagName.getText(ast) === 'Button') {
          const attributes = node.attributes.properties
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
