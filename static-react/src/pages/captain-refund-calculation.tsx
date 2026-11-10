import { useI18n } from '@/i18n'
import type { RefundCalculation } from '@/types/api/captain-refund'
import { formatTime } from '@/lib/format'
import { refundEligibilityReasonKey } from '@/pages/captain-refund-text'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

export function RefundCalculationView({ value }: { value: RefundCalculation }) {
  const { t } = useI18n()
  return (
    <div className="space-y-3">
      <p>
        {t('refund.bounty')}: {value.bounty.toFixed(2)} {t('refund.isk')} · {t('refund.taxRate')}:{' '}
        {value.settings.tax_rate}% · {t('refund.reward')}: {value.reward.toFixed(2)} {t('refund.coin')}
      </p>
      <p>
        {t('refund.system')}: {value.request.system_id} · {t('refund.period')}:{' '}
        {formatTime(value.request.start)} — {formatTime(value.request.end)}
      </p>
      <p>
        {t('refund.ignored')}: {value.ignored_names.join(', ') || '—'}
      </p>
      {!value.has_newbro && <p className="text-destructive">{t('refund.noNewbro')}</p>}
      <div className="overflow-x-auto">
        <Table aria-label={t('refund.memberDetails')}>
          <TableHeader>
            <TableRow>
              {[
                'refund.character',
                'refund.newbro',
                'refund.sp',
                'refund.birthday',
                'refund.bounty',
                'refund.sync',
              ].map((key) => (
                <TableHead key={key} className="whitespace-nowrap px-3 py-2 text-left">
                  {t(key)}
                </TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {value.members.map((member) => (
              <TableRow key={member.character_id}>
                <TableCell>
                  {member.character_name}
                  {member.is_captain ? ` (${t('refund.captain')})` : ''}
                </TableCell>
                <TableCell>
                  {t(member.eligibility.is_currently_newbro ? 'refund.yes' : 'refund.no')}
                  {member.eligibility.disqualified_reason
                    ? ` · ${t(refundEligibilityReasonKey(member.eligibility.disqualified_reason))}`
                    : ''}
                </TableCell>
                <TableCell>{member.eligibility.total_sp.toLocaleString()}</TableCell>
                <TableCell>{member.eligibility.oldest_birthday ?? '—'}</TableCell>
                <TableCell>
                  {member.bounty.toFixed(2)} {t('refund.isk')}
                </TableCell>
                <TableCell>
                  {member.wallet_updated_at
                    ? formatTime(member.wallet_updated_at)
                    : t('refund.unknownSync')}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
      {value.conflicts.map((conflict) => (
        <p className="text-destructive" key={conflict.character_id + ':' + conflict.journal_id}>
          {t('refund.conflict')}: {conflict.character_id} · {formatTime(conflict.journal_at)} · #
          {conflict.refund_id}
        </p>
      ))}
      <details>
        <summary>
          {t('refund.journals')} ({value.journals.length})
        </summary>
        <div className="overflow-x-auto">
          <Table aria-label={t('refund.journals')}>
            <TableHeader>
              <TableRow>
                {[
                  'refund.character',
                  'refund.journal',
                  'refund.time',
                  'refund.bounty',
                  'refund.match',
                ].map((key) => (
                  <TableHead key={key} className="whitespace-nowrap px-3 py-2 text-left">
                    {t(key)}
                  </TableHead>
                ))}
              </TableRow>
            </TableHeader>
            <TableBody>
              {value.journals.map((journal) => (
                <TableRow key={journal.character_id + ':' + journal.journal_id}>
                  <TableCell>{journal.character_id}</TableCell>
                  <TableCell>{journal.journal_id}</TableCell>
                  <TableCell>{formatTime(journal.date)}</TableCell>
                  <TableCell>
                    {journal.amount.toFixed(2)} {t('refund.isk')}
                  </TableCell>
                  <TableCell>
                    {journal.match_journal_id || '—'}{' '}
                    {journal.match_date ? formatTime(journal.match_date) : '—'}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </details>
    </div>
  )
}
