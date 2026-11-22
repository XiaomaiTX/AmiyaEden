import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

import { formatJournalTypeLabel, toTitleCaseRefType } from './wallet-journal-type'

const createTranslator = (dictionary: Record<string, string>) => (key: string) =>
  dictionary[key] ?? key

test('formatJournalTypeLabel resolves the closest layer of the i18n fallback chain', () => {
  const t = createTranslator({
    'npcKill.refTypes.bounty_prizes': '赏金',
    'info.wallet.refTypes.ess_escrow_transfer': 'ESS 托管转账',
    'npcKill.refTypes.ess_escrow_transfer': 'ESS 托管',
    'walletAdmin.refTypes.bounty_prizes': '赏金（管理视图）',
    'walletAdmin.refTypes.market_transaction': '市场交易',
    'info.wallet.refTypes.player_donation': '玩家捐赠'
  })

  assert.equal(formatJournalTypeLabel('bounty_prizes', t), '赏金')
  assert.equal(formatJournalTypeLabel('ess_escrow_transfer', t), 'ESS 托管转账')
  assert.equal(formatJournalTypeLabel('market_transaction', t), '市场交易')
  assert.equal(formatJournalTypeLabel('player_donation', t), '玩家捐赠')
})

test('formatJournalTypeLabel falls back to title case when no layer matches', () => {
  const t = createTranslator({})

  assert.equal(formatJournalTypeLabel('ess_escrow_transfer', t), 'Ess Escrow Transfer')
  assert.equal(formatJournalTypeLabel('contract_price', t), 'Contract Price')
  assert.equal(toTitleCaseRefType('npc_bounty'), 'Npc Bounty')
})

test('real locale dictionaries translate all ESI wallet types through the page formatter', () => {
  const readJson = (relativePath: string) =>
    JSON.parse(readFileSync(new URL(relativePath, import.meta.url), 'utf8'))
  const schema = readJson('../../../../../../docs/reference/esi-openapi.json')
  const types: string[] =
    schema.components.schemas.CharactersCharacterIdWalletJournalGet.items.properties.ref_type.enum
  for (const language of ['zh', 'en']) {
    const dictionary = readJson(`../../../../locales/langs/${language}.json`)
    const t = (key: string): string => {
      const value = key.split('.').reduce((node, part) => node?.[part], dictionary)
      return typeof value === 'string' ? value : key
    }
    for (const type of types) {
      const label = formatJournalTypeLabel(type, t)
      assert.notEqual(t(`info.wallet.refTypes.${type}`), `info.wallet.refTypes.${type}`)
      assert.notEqual(label, type, `${language}: ${type}`)
      if (language === 'zh') assert.match(label, /\p{Script=Han}/u, type)
    }
  }
})
