// Static candidate inventory; a reference is not proof of a runtime call.
const fs = require('node:fs')
const path = require('node:path')
const YAML = require('yaml')
const exceptions = require('./inventory-exceptions.cjs')
const ts = require(require.resolve('typescript', { paths: [path.resolve(__dirname, '../static')] }))
const root = path.resolve(__dirname, '..')
const check = process.argv.includes('--check')
const findings = []
const read = (file) => fs.readFileSync(path.join(root, file), 'utf8')
function* files(dir, suffix) {
  for (const entry of fs.readdirSync(path.join(root, dir), { withFileTypes: true })) {
    const name = `${dir}/${entry.name}`
    if (entry.isDirectory()) yield* files(name, suffix)
    else if (suffix.test(name)) yield name
  }
}
const specs = ['contracts/openapi.yaml', ...files('contracts/modules', /\.yaml$/)]
const operations = new Map()
const routes = new Set()
for (const file of specs) {
  for (const [route, item] of Object.entries(YAML.parse(read(file)).paths ?? {})) {
    for (const [method, operation] of Object.entries(item)) {
      if (!/^(get|post|put|patch|delete|head|options|trace)$/.test(method)) continue
      const key = `${method.toUpperCase()} /api/v1${route}`
      const id = operation.operationId
      if (!id || operations.has(id) || routes.has(key)) throw new Error(`Invalid operation: ${id} ${key}`)
      operations.set(id, { route: key })
      routes.add(key)
    }
  }
}
const generatedGo = read('server/internal/apicontract/api.gen.go')
const handlers = [...files('server/internal/handler', /(?<!_test)\.go$/)].map((file) => [file, read(file)])
for (const match of generatedGo.matchAll(/^[\t ]+(\w+)\(ctx context\.Context, request \w+RequestObject\)/gm)) {
  const [, name] = match
  const id = [...operations.keys()].find((key) => key.toLowerCase() === name.toLowerCase())
  if (!id) throw new Error(`No contract operation for Go ${name}`)
  const implementations = handlers.filter(([, text]) => new RegExp(`func \\([^)]*\\) ${name}\\([^\\n]*apicontract\\.\\w+RequestObject`).test(text))
  if (implementations.length !== 1) throw new Error(`Expected one strict implementation for ${name}; got ${implementations.length}`)
  operations.get(id).go = implementations[0][0]
}
for (const [frontend, column] of [['static', 'vue'], ['static-react', 'react']]) {
  const generated = read(`${frontend}/src/api/generated/client.ts`)
  const sources = [...files(`${frontend}/src`, /\.(vue|[cm]?tsx?)$/)].filter((file) => !/\/generated\/|\.test\.|\.spec\.|\.d\.ts$/.test(file))
  for (const match of generated.matchAll(/^\s*export const (\w+) = \([^]*?return contractRequest<[^\n]*\n\s*\{url: `([^`]+)`, method: '([A-Z]+)'/gm)) {
    const [, name, route, method] = match
    const key = `${method} ${route.replaceAll('${', '{')}`
    const shape = (value) => value.replace(/\{[^}]+\}/g, '{}')
    const row = [...operations.values()].find((operation) => shape(operation.route) === shape(key))
    if (!row || row[column]) throw new Error(`Unexpected generated ${column} operation: ${name} ${key}`)
    row[column] = { name, candidates: [] }
  }
  const importsByFile = new Map()
  for (const file of sources) {
    const text = read(file)
    if (/["'`]\/api\/v1(?:[/?"'`]|$)/m.test(text)) findings.push(`${file}: use the generated client for /api/v1 calls`)
    if (!text.includes('import') && !text.includes('export')) continue
    const source = file.endsWith('.vue')
      ? [...text.matchAll(/<script\b[^>]*>([\s\S]*?)<\/script>/gi)].map((match) => match[1]).join('\n')
      : text
    const ast = ts.createSourceFile(file, source, ts.ScriptTarget.Latest, true,
      file.endsWith('.tsx') ? ts.ScriptKind.TSX : ts.ScriptKind.TS)
    const names = new Set()
    const namedBindings = new Map()
    const namespaces = new Set()
    const imports = new Set()
    importsByFile.set(file, imports)
    for (const statement of ast.statements) {
      if ((ts.isImportDeclaration(statement) && !statement.importClause?.isTypeOnly ||
           ts.isExportDeclaration(statement) && !statement.isTypeOnly) &&
          statement.moduleSpecifier && ts.isStringLiteral(statement.moduleSpecifier)) {
        const specifier = statement.moduleSpecifier.text
        const target = specifier.startsWith('@/') ? `${frontend}/src/${specifier.slice(2)}` :
          specifier.startsWith('.') ? path.posix.normalize(path.posix.join(path.posix.dirname(file), specifier)) : null
        const bindings = ts.isImportDeclaration(statement) ? statement.importClause?.namedBindings : statement.exportClause
        const hasRuntimeBinding = !bindings || !(ts.isNamedImports(bindings) || ts.isNamedExports(bindings)) ||
          bindings.elements.some((item) => !item.isTypeOnly)
        if (target && hasRuntimeBinding) imports.add(target.replace(/\.[cm]?[jt]sx?$/, ''))
      }
      if (!ts.isImportDeclaration(statement) || !ts.isStringLiteral(statement.moduleSpecifier) ||
          !statement.moduleSpecifier.text.endsWith('/generated/client') || statement.importClause?.isTypeOnly) continue
      const binding = statement.importClause?.namedBindings
      if (!binding) continue
      if (ts.isNamespaceImport(binding)) namespaces.add(binding.name.text)
      else for (const entry of binding.elements) {
        if (!entry.isTypeOnly) namedBindings.set(entry.name.text, (entry.propertyName ?? entry.name).text)
      }
    }
    function scan(node) {
      if (ts.isTypeNode(node) || ts.isImportDeclaration(node)) return
      if (ts.isIdentifier(node) && namedBindings.has(node.text)) names.add(namedBindings.get(node.text))
      if (ts.isExportDeclaration(node) && !node.isTypeOnly && node.moduleSpecifier?.text?.endsWith('/generated/client') &&
          node.exportClause && ts.isNamedExports(node.exportClause)) {
        for (const entry of node.exportClause.elements) if (!entry.isTypeOnly) names.add((entry.propertyName ?? entry.name).text)
      }
      if (ts.isPropertyAccessExpression(node) && ts.isIdentifier(node.expression) && namespaces.has(node.expression.text)) {
        names.add(node.name.text)
      }
      ts.forEachChild(node, scan)
    }
    scan(ast)
    for (const row of operations.values()) {
      const entry = row[column]
      if (entry && names.has(entry.name)) entry.candidates.push(file)
    }
  }
  for (const [id, row] of operations) {
    const entry = row[column]
    if (!entry) continue
    const isAdapter = (file) => file.startsWith(`${frontend}/src/api/`)
    const adapter = entry.candidates.some(isAdapter)
    const direct = entry.candidates.some((file) => !isAdapter(file))
    const reason = exceptions[column]?.workflow?.[id]
    const navigation = exceptions[column]?.navigation?.[id]
    if (reason && !adapter) throw new Error(`Stale ${column} workflow exception: ${id}`)
    if (reason && entry.candidates.filter(isAdapter).length > 1) {
      findings.push(`${column}: more than one workflow adapter for ${id}`)
    }
    if (navigation && entry.candidates.length) throw new Error(`Unexpected ${column} client call for navigation: ${id}`)
    entry.status = adapter ? (reason ? 'workflow' : 'review-adapter') :
      direct ? 'direct' : navigation ? 'browser-navigation' : 'no-client-reference'
    if (reason || navigation) entry.reason = reason || navigation
    if (entry.status === 'review-adapter') findings.push(`${column}: review adapter for ${id}`)
    // File-level reverse imports nominate consumers for manual symbol-level review.
    entry.apiModuleImporters = Object.fromEntries(entry.candidates
      .filter((file) => file.startsWith(`${frontend}/src/api/`))
      .map((file) => [file, [...importsByFile].filter(([, imports]) =>
        imports.has(file.replace(/\.[cm]?[jt]sx?$/, ''))).map(([from]) => from)]))
  }
}
for (const [id, row] of operations) {
  if (!row.go || !row.vue || !row.react) throw new Error(`Missing generated coverage for ${id}`)
}
if (check) {
  if (findings.length) {
    console.error(findings.join('\n'))
    process.exitCode = 1
  } else {
    console.log(`${operations.size} OpenAPI operations checked across Go, Vue and React`)
  }
} else {
  console.log(JSON.stringify(Object.fromEntries(operations), null, 2))
}
