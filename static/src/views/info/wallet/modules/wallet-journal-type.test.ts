import assert from 'node:assert/strict'
import test from 'node:test'

import { formatJournalTypeLabel, toTitleCaseRefType } from './wallet-journal-type'

const createTranslator = (dictionary: Record<string, string>) => (key: string) =>
  dictionary[key] ?? key

test('formatJournalTypeLabel resolves the closest layer of the i18n fallback chain', () => {
  const t = createTranslator({
    'npcKill.refTypes.bounty_prizes': '赏金',
    'walletAdmin.refTypes.bounty_prizes': '赏金（管理视图）',
    'walletAdmin.refTypes.market_transaction': '市场交易',
    'info.wallet.refTypes.player_donation': '玩家捐赠'
  })

  assert.equal(formatJournalTypeLabel('bounty_prizes', t), '赏金')
  assert.equal(formatJournalTypeLabel('market_transaction', t), '市场交易')
  assert.equal(formatJournalTypeLabel('player_donation', t), '玩家捐赠')
})

test('formatJournalTypeLabel falls back to title case when no layer matches', () => {
  const t = createTranslator({})

  assert.equal(formatJournalTypeLabel('ess_escrow_transfer', t), 'Ess Escrow Transfer')
  assert.equal(formatJournalTypeLabel('contract_price', t), 'Contract Price')
  assert.equal(toTitleCaseRefType('npc_bounty'), 'Npc Bounty')
})
