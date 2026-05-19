import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'
import { parseFragment } from 'parse5'

const pages = [
  new URL('./user-center/index.vue', import.meta.url),
  new URL('./webhook/index.vue', import.meta.url),
  new URL('./wallet/modules/wallet-logs.vue', import.meta.url)
]

function findHardcodedChinese(source: string) {
  const template = stripComments(source.match(/<template>([\s\S]*?)<\/template>/i)?.[1] ?? '')
  const script = stripComments(
    source.match(/<script[^>]*>([\s\S]*?)<\/script(?:\s[^>]*)?>/i)?.[1] ?? ''
  )

  return [
    ...(template.match(/(?:label|placeholder|title)="[^"]*[\u4e00-\u9fff][^"]*"/g) ?? []),
    ...(template.match(/>[^<]*[\u4e00-\u9fff][^<]*</g) ?? []),
    ...(script.match(/['"][^'"\n]*[\u4e00-\u9fff][^'"\n]*['"]/g) ?? [])
  ]
}

type ParseNode = {
  nodeName?: string
  value?: string
  childNodes?: ParseNode[]
}

function collectText(node: ParseNode): string {
  if (node.nodeName === '#comment') return ''
  if (node.nodeName === '#text') return node.value ?? ''
  return (node.childNodes ?? []).map(collectText).join('')
}

function extractTextWithoutHtmlComments(source: string) {
  const fragment = parseFragment(source) as ParseNode
  return (fragment.childNodes ?? []).map(collectText).join('')
}

function stripComments(source: string) {
  let current = extractTextWithoutHtmlComments(source)
  let previous: string

  do {
    previous = current
    current = current.replace(/\/\*[\s\S]*?\*\//g, '').replace(/\/\/.*$/gm, '')
  } while (current !== previous)

  return current
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
