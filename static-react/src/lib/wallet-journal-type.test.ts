import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import zh from '@/i18n/messages/zh-CN'
import en from '@/i18n/messages/en-US'
import { formatJournalTypeLabel } from './wallet-journal-type'

const root = resolve(import.meta.dirname, '../../..')
const readJson = (path: string) => JSON.parse(readFileSync(resolve(root, path), 'utf8'))

test('both frontends translate every wallet type in the ESI contract in both languages', () => {
  const schema = readJson('docs/reference/esi-openapi.json')
  const types: string[] =
    schema.components.schemas.CharactersCharacterIdWalletJournalGet.items.properties.ref_type.enum
  const vueZh = readJson('static/src/locales/langs/zh.json').info.wallet.refTypes
  const vueEn = readJson('static/src/locales/langs/en.json').info.wallet.refTypes
  const reactZh: Record<string, string> = zh.infoWallet.refTypes
  const reactEn: Record<string, string> = en.infoWallet.refTypes

  for (const type of types) {
    expect(reactZh[type], type).toMatch(/\p{Script=Han}/u)
    expect(reactEn[type], type).toBeTruthy()
    expect(vueZh[type], type).toBe(reactZh[type])
    expect(vueEn[type], type).toBe(reactEn[type])
    expect(formatJournalTypeLabel(type, (key) => reactZh[key.split('.').at(-1)!] ?? key)).toBe(
      reactZh[type]
    )
    expect(formatJournalTypeLabel(type, (key) => reactEn[key.split('.').at(-1)!] ?? key)).toBe(
      reactEn[type]
    )
  }
})

test('unknown future types remain readable', () => {
  expect(formatJournalTypeLabel('future_wallet_type', (key) => key)).toBe('Future Wallet Type')
})
