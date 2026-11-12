import { fetchOperationSalaryRates } from '@/api/salary'
import { operationSalaryMessage } from './operation-salary'

vi.mock('@/api/salary', () => ({ fetchOperationSalaryRates: vi.fn() }))
const rates = { recruitment: 4.25, welfare_delivery: 18, shop_delivery: 6.5, srp_payout: 0.1 }
const t = (key: string, values?: Record<string, string | number>) =>
  values ? JSON.stringify(values) : key

beforeEach(() => {
  vi.resetAllMocks()
  vi.mocked(fetchOperationSalaryRates).mockResolvedValue(rates)
})
test.each(Object.keys(rates) as Array<keyof typeof rates>)(
  'uses the current %s rate',
  async (category) => {
    const count = category === 'srp_payout' ? 1 : 3
    const result = JSON.parse(await operationSalaryMessage(category, count, t))
    expect(result).toEqual({
      amount: (rates[category] * count).toFixed(2),
    })
  }
)
test('zero salary is explicit and failed loading never substitutes a default', async () => {
  vi.mocked(fetchOperationSalaryRates).mockResolvedValueOnce({ ...rates, recruitment: 0 })
  expect(JSON.parse(await operationSalaryMessage('recruitment', 1, t)).amount).toBe('0.00')
  vi.mocked(fetchOperationSalaryRates).mockRejectedValueOnce(new Error('offline'))
  await expect(operationSalaryMessage('recruitment', 1, t)).rejects.toThrow('offline')
})
test('invalid rates or counts cannot produce a confirmation', async () => {
  vi.mocked(fetchOperationSalaryRates).mockResolvedValue({ ...rates, recruitment: Number.NaN })
  await expect(operationSalaryMessage('recruitment', 1, t)).rejects.toThrow()
  await expect(operationSalaryMessage('srp_payout', 0, t)).rejects.toThrow()
})
