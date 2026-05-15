import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const pages = [
  new URL('./user-center/index.vue', import.meta.url),
  new URL('./webhook/index.vue', import.meta.url),
  new URL('./wallet/modules/wallet-logs.vue', import.meta.url)
]

function findHardcodedChinese(source: string) {
  const template = stripComments(source.match(/<template>([\s\S]*?)<\/template>/)?.[1] ?? '')
  const script = stripComments(source.match(/<script[^>]*>([\s\S]*?)<\/script>/)?.[1] ?? '')

  return [
    ...(template.match(/(?:label|placeholder|title)="[^"]*[\u4e00-\u9fff][^"]*"/g) ?? []),
    ...(template.match(/>[^<]*[\u4e00-\u9fff][^<]*</g) ?? []),
    ...(script.match(/['"][^'"\n]*[\u4e00-\u9fff][^'"\n]*['"]/g) ?? [])
  ]
}

function stripComments(source: string) {
  return source
    .replace(/<!--[\s\S]*?-->/g, '')
    .replace(/\/\*[\s\S]*?\*\//g, '')
    .replace(/\/\/.*$/gm, '')
}

test('system user-facing pages keep Chinese text in locale files', () => {
  const matches = pages.flatMap((page) =>
    findHardcodedChinese(readFileSync(page, 'utf8')).map((match) => ({
      page: page.pathname,
      match
    }))
  )

  assert.deepEqual(matches, [])
})
