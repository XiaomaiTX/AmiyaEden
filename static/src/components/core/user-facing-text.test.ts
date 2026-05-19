import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const files = [
  new URL('./banners/art-basic-banner/index.vue', import.meta.url),
  new URL('./banners/art-card-banner/index.vue', import.meta.url),
  new URL('./cards/art-data-list-card/index.vue', import.meta.url),
  new URL('./cards/art-image-card/index.vue', import.meta.url),
  new URL('./forms/art-excel-export/index.vue', import.meta.url),
  new URL('./forms/art-drag-verify/index.vue', import.meta.url),
  new URL('./layouts/art-chat-window/index.vue', import.meta.url),
  new URL('./layouts/art-screen-lock/index.vue', import.meta.url),
  new URL('./media/art-cutter-img/index.vue', import.meta.url),
  new URL('./tables/art-table/index.vue', import.meta.url),
  new URL('../../directives/business/highlight.ts', import.meta.url)
]

const templateAttributePattern = /(?:label|placeholder|title|alt)="[^"]*[\u4e00-\u9fff][^"]*"/g
const templateTextPattern = />[^<]*[\u4e00-\u9fff][^<]*</g
const scriptStringPattern = /['"][^'"\n]*[\u4e00-\u9fff][^'"\n]*['"]/g

function stripComments(source: string) {
  let previous: string
  let current = source

  do {
    previous = current
    current = current
      .replace(/[<>]/g, '')
      .replace(/\/\*[\s\S]*?\*\//g, '')
      .replace(/\/\/.*$/gm, '')
  } while (current !== previous)

  return current
}

test('core shared UI keeps Chinese user-facing text in locale files', () => {
  const matches = files.flatMap((file) => {
    const source = stripComments(readFileSync(file, 'utf8'))
    const template = source.match(/<template>([\s\S]*?)<\/template>/)?.[1] ?? ''
    const script = source.match(/<script\b[^>]*>([\s\S]*?)<\/script(?:\s+[^>]*)?>/i)?.[1] ?? source

    return [
      ...(template.match(templateAttributePattern) ?? []),
      ...(template.match(templateTextPattern) ?? []),
      ...(script.match(scriptStringPattern) ?? [])
    ].map((match) => ({ file: file.pathname, match }))
  })

  assert.deepEqual(matches, [])
})
