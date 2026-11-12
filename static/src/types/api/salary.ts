export interface SalarySettings {
  recruitment: number
  welfare_delivery: number
  shop_delivery: number
  srp_payout: number
  fc_salary: number
  fc_salary_monthly_limit: number
}
export type OperationSalaryRates = Pick<
  SalarySettings,
  'recruitment' | 'welfare_delivery' | 'shop_delivery' | 'srp_payout'
>
export type OperationSalaryCategory = keyof OperationSalaryRates
