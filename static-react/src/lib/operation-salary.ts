import { fetchOperationSalaryRates } from '@/api/salary'
import type { OperationSalaryCategory } from '@/types/api/salary'

type Translate = (key: string, values?: Record<string, string | number>) => string

export async function operationSalaryMessage(
  category: OperationSalaryCategory,
  count: number,
  t: Translate
) {
  const rates = await fetchOperationSalaryRates()
  const rate = rates[category]
  if (!Number.isFinite(rate) || rate < 0 || !Number.isSafeInteger(count) || count < 1) {
    throw new Error(t('salaryManagement.loadFailed'))
  }
  return t('salaryManagement.confirmSalary', {
    count,
    unit: t(
      category === 'srp_payout' ? 'salaryManagement.operationUnit' : 'salaryManagement.itemUnit'
    ),
    rate: rate.toFixed(2),
    amount: ((Math.round(rate * 100) * count) / 100).toFixed(2),
  })
}
