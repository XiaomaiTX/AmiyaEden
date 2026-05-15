import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const files = [
  new URL('./useSrpManage.ts', import.meta.url),
  new URL('./useSrpWorkflow.ts', import.meta.url)
]

function stripComments(source: string) {
  return source.replace(/\/\*[\s\S]*?\*\//g, '').replace(/\/\/.*$/gm, '')
}

test('srp hooks keep Chinese user-facing output in locale files', () => {
  const matches = files.flatMap((file) => {
    const source = stripComments(readFileSync(file, 'utf8'))

    return (source.match(/['"][^'"\n]*[\u4e00-\u9fff][^'"\n]*['"]/g) ?? []).map((match) => ({
      file: file.pathname,
      match
    }))
  })

  assert.deepEqual(matches, [])
})
