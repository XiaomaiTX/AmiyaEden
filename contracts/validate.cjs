const SwaggerParser = require('@apidevtools/swagger-parser')
const fs = require('node:fs')
const path = require('node:path')
const YAML = require('yaml')

function merge(target, entries, source) {
  for (const [key, value] of Object.entries(entries)) {
    if (Object.hasOwn(target, key)) throw new Error(`Duplicate contract key ${key} in ${source}`)
    Object.defineProperty(target, key, { value, enumerable: true, writable: true, configurable: true })
  }
}

function loadContract() {
  const spec = YAML.parse(fs.readFileSync('openapi.yaml', 'utf8'))
  for (const name of fs.readdirSync('modules').filter((name) => name.endsWith('.yaml')).sort()) {
    const fragment = YAML.parse(fs.readFileSync(path.join('modules', name), 'utf8'))
    if (Object.keys(fragment).some((key) => key !== 'paths' && key !== 'components')) {
      throw new Error(`Unexpected top-level property in ${name}`)
    }
    merge(spec.paths, fragment.paths ?? {}, name)
    for (const [section, entries] of Object.entries(fragment.components ?? {})) {
      spec.components[section] ??= {}
      merge(spec.components[section], entries, name)
    }
  }
  return spec
}
const methods = new Set(['get', 'put', 'post', 'delete', 'options', 'head', 'patch', 'trace'])
const role = /^[a-z][a-z0-9_]*$/
const capability = /^[a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*)+$/
const expectedJwtSchemes = new Set(['bearerAuth', 'queryToken'])
const bodyPolicyKeys = new Set(['maxBytes', 'allowEmpty', 'allowMalformedJSON', 'allowInvalidBinding', 'ignored'])

function validateJwtSchemes(api) {
  const { bearerAuth, queryToken } = api.components?.securitySchemes ?? {}
  if (bearerAuth?.type !== 'http' || bearerAuth.scheme !== 'bearer' ||
      queryToken?.type !== 'apiKey' || queryToken.in !== 'query' || queryToken.name !== 'token') {
    throw new Error('Gin jwt access requires bearerAuth (HTTP bearer) and queryToken (?token=) schemes')
  }
}

function validateGinAccess(api, operation, at) {
  const chain = operation['x-gin-access']
  if (!Array.isArray(chain)) throw new Error(`Missing or invalid x-gin-access array: ${at}`)

  if (chain.length > 0) {
    if (chain[0] !== 'jwt') throw new Error(`x-gin-access must start with jwt: ${at}`)
    for (const token of chain.slice(1)) {
      if (token === 'login') continue
      if (typeof token === 'string' && token.startsWith('role:')) {
        const roles = token.slice('role:'.length).split(',')
        if (roles.every((name) => role.test(name))) continue
      }
      if (typeof token === 'string' && token.startsWith('cap:') && capability.test(token.slice('cap:'.length))) continue
      throw new Error(`Invalid x-gin-access token ${JSON.stringify(token)}: ${at}`)
    }
  }

  const security = operation.security ?? api.security ?? []
  if (chain.length === 0) {
    if (security.length !== 0) throw new Error(`Public x-gin-access requires empty effective security: ${at}`)
    return
  }
  // JWTAuth accepts either the Authorization header or the token query parameter.
  // They must be separate alternatives (OR), never combined or optional.
  const schemes = new Set()
  for (const requirement of security) {
    const entries = Object.entries(requirement)
    if (entries.length !== 1) break
    const [[name, scopes]] = entries
    if (!expectedJwtSchemes.has(name) || !Array.isArray(scopes) || scopes.length !== 0) break
    schemes.add(name)
  }
  if (security.length !== expectedJwtSchemes.size || schemes.size !== expectedJwtSchemes.size) {
    throw new Error(`JWT x-gin-access requires bearerAuth OR queryToken without scopes: ${at}`)
  }
}

function validateGinBody(operation, at) {
  if (!Object.hasOwn(operation, 'x-gin-body')) return
  const policy = operation['x-gin-body']
  if (!operation.requestBody || !policy || typeof policy !== 'object' || Array.isArray(policy) ||
      Object.keys(policy).some((key) => !bodyPolicyKeys.has(key))) {
    throw new Error(`Invalid x-gin-body policy or no request body: ${at}`)
  }
  if (Object.hasOwn(policy, 'maxBytes') && (!Number.isSafeInteger(policy.maxBytes) || policy.maxBytes < 1)) {
    throw new Error(`x-gin-body.maxBytes must be a positive safe integer: ${at}`)
  }
  for (const key of ['allowEmpty', 'allowMalformedJSON', 'allowInvalidBinding', 'ignored']) {
    if (Object.hasOwn(policy, key) && typeof policy[key] !== 'boolean') {
      throw new Error(`x-gin-body.${key} must be a boolean: ${at}`)
    }
  }
  if (policy.ignored && (operation.requestBody.required || Object.keys(policy).length !== 1)) {
    throw new Error(`An ignored body must be optional and cannot combine binding policies: ${at}`)
  }
  if ((policy.allowMalformedJSON || policy.allowInvalidBinding) && !Object.keys(operation.requestBody.content).some((type) =>
    type.toLowerCase() === 'application/json' || type.toLowerCase().endsWith('+json'))) {
    throw new Error(`x-gin-body JSON fallback requires a JSON request body: ${at}`)
  }
}

function validateGinQuery(operation, pathItem, at) {
  const effective = [...pathItem.parameters ?? [], ...operation.parameters ?? []]
  const parameters = new Map(effective.filter(parameter => parameter.in === 'query')
    .map(parameter => [parameter.name, parameter]))
  const names = operation['x-gin-ignore-invalid-query']
  if (Object.hasOwn(operation, 'x-gin-ignore-invalid-query') &&
      (!Array.isArray(names) || !names.length || new Set(names).size !== names.length ||
        names.some(name => !parameters.has(name) || parameters.get(name).required))) {
    throw new Error(`x-gin-ignore-invalid-query names must be distinct optional query parameters: ${at}`)
  }
  for (const parameter of effective) {
    const schema = parameter.schema
    if (Object.hasOwn(parameter, 'x-gin-trim-query') &&
        (parameter['x-gin-trim-query'] !== true || parameter.in !== 'query' || parameter.required ||
         (parameter.style ?? 'form') !== 'form' || schema?.type !== 'integer' ||
         Object.hasOwn(parameter, 'x-gin-legacy-atoi'))) {
      throw new Error(`x-gin-trim-query requires an optional integer query parameter: ${at}`)
    }
    if (!Object.hasOwn(parameter, 'x-gin-legacy-atoi')) continue
    if (parameter['x-gin-legacy-atoi'] !== true || parameter.in !== 'query' || parameter.required ||
        (parameter.style ?? 'form') !== 'form' || schema?.type !== 'integer' || schema.format ||
        schema['x-go-type'] || schema['x-go-type-skip-optional-pointer'] || names?.includes(parameter.name)) {
      throw new Error(`x-gin-legacy-atoi requires an optional native integer query parameter without another fallback: ${at}`)
    }
  }
}

function assertOperationLevelGinMetadata(value, at) {
  if (Object.hasOwn(value, 'x-gin-access') || Object.hasOwn(value, 'x-gin-body') || Object.hasOwn(value, 'x-gin-ignore-invalid-query')) {
    throw new Error(`x-gin-access and x-gin-body are operation-level only: ${at}`)
  }
}

async function main() {
  const spec = loadContract()
  const api = await SwaggerParser.validate(structuredClone(spec), { resolve: { external: false } })
  const operations = new Set()
  assertOperationLevelGinMetadata(api, 'OpenAPI root')
  validateJwtSchemes(api)
  for (const [path, item] of Object.entries(api.paths)) {
    assertOperationLevelGinMetadata(item, path)
    for (const [method, operation] of Object.entries(item)) {
      if (!methods.has(method)) continue
      const at = `${method.toUpperCase()} ${path}`
      if (!operation.operationId || operations.has(operation.operationId)) {
        throw new Error(`Missing or duplicate operationId: ${at}`)
      }
      operations.add(operation.operationId)
      validateGinAccess(api, operation, at)
      validateGinBody(operation, at)
      validateGinQuery(operation, item, at)
    }
  }
  fs.writeFileSync(process.argv[2], JSON.stringify(spec))
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
