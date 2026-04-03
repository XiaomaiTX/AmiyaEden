import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const walletListSource = readFileSync(new URL('./wallet-list.vue', import.meta.url), 'utf8')
const walletLogsSource = readFileSync(new URL('./wallet-logs.vue', import.meta.url), 'utf8')
const apiSource = readFileSync(new URL('../../../../api/sys-wallet.ts', import.meta.url), 'utf8')
const typeSource = readFileSync(new URL('../../../../types/api/api.d.ts', import.meta.url), 'utf8')
const docSource = readFileSync(
  new URL('../../../../../../docs/features/current/commerce.md', import.meta.url),
  'utf8'
)
const zhLocale = JSON.parse(
  readFileSync(new URL('../../../../locales/langs/zh.json', import.meta.url), 'utf8')
)
const enLocale = JSON.parse(
  readFileSync(new URL('../../../../locales/langs/en.json', import.meta.url), 'utf8')
)

test('wallet list supports keyword search without a bulk manual-adjust button', () => {
  assert.doesNotMatch(walletListSource, /walletAdmin\.adjustBalance/)
  assert.doesNotMatch(walletListSource, /emit\('adjust',\s*0,\s*'add'\)/)
  assert.match(walletListSource, /walletAdmin\.placeholders\.userKeywordFilter/)
  assert.match(walletListSource, /const userKeywordFilter = ref\(''\)/)
  assert.match(
    walletListSource,
    /user_keyword:\s*userKeywordFilter\.value\.trim\(\)\s*\|\|\s*undefined/
  )
  assert.match(
    apiSource,
    /export function adminListWallets\(data\?: Api\.SysWallet\.WalletSearchParams\)/
  )
  assert.match(typeSource, /type WalletSearchParams = Partial<\{[\s\S]*user_keyword: string/)
  assert.match(docSource, /管理端钱包列表支持按当前用户昵称或任意已绑定人物名搜索/)
})

test('wallet log amount column resolves through the shared common locale key', () => {
  assert.match(walletLogsSource, /label:\s*t\('common\.amount'\)/)
  assert.equal(zhLocale.common.amount, '金额')
  assert.equal(enLocale.common.amount, 'Amount')
})
