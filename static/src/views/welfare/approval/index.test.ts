import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./index.vue', import.meta.url), 'utf8')

test('welfare approval row description tooltip stays visible long enough after mouse leave', () => {
  assert.doesNotMatch(source, /utils\/ui\/tooltip/)
  assert.match(
    source,
    /const\s+ROW_DESCRIPTION_TOOLTIP_HIDE_DELAY_MS\s*=\s*800/,
    'expected welfare approval to own its local tooltip delay constant'
  )
  assert.match(
    source,
    /handleCellMouseLeave\(\)\s*\{[\s\S]*?ROW_DESCRIPTION_TOOLTIP_HIDE_DELAY_MS[\s\S]*?\)/,
    'expected handleCellMouseLeave to use the local tooltip hide delay constant'
  )
  assert.match(source, /effect="dark"/)
  assert.doesNotMatch(source, /:show-after="0"/)
})

test('welfare approval character rows use the shared copy button instead of page-local clipboard logic', () => {
  assert.match(
    source,
    /import ArtCopyButton from '@\/components\/core\/forms\/art-copy-button\/index.vue'/
  )
  assert.match(
    source,
    /prop:\s*'character_name'[\s\S]*h\(ArtCopyButton,[\s\S]*text:\s*row\.character_name/
  )
  assert.doesNotMatch(source, /const copyText = async \(text: string\)/)
})

test('welfare approval history shows a localized system reviewer label for auto-delivered claims', () => {
  assert.match(
    source,
    /import\s+\{\s*formatWelfareHistoryReviewerName\s*\}\s+from '\.\.\/reviewerName'/
  )
  assert.match(
    source,
    /formatWelfareHistoryReviewerName\(\{[\s\S]*systemLabel:\s*t\('welfareApproval\.systemReviewer'\)/
  )
})

test('welfare approval shows proof images only for pending rows', () => {
  assert.match(
    source,
    /buildBaseColumns\(\{ includeEvidenceImage: true \}\)\.filter\([\s\S]*?\(c\) => c\.prop !== 'reviewer_name'/,
    'expected pending table to keep the evidence image column'
  )
  assert.match(
    source,
    /buildBaseColumns\(\{ includeEvidenceImage: false \}\)/,
    'expected history table to omit the evidence image column'
  )
})

test('welfare approval handles delivery-time automatic rejection and refreshes both lists', () => {
  assert.match(source, /result\.outcome === 'auto_rejected'/)
  assert.match(source, /welfareApproval\.eligibilityReasons\./)
  assert.match(source, /Promise\.all\(\[loadPending\(\), loadHistory\(\)\]\)/)
})
