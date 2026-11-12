import request from '@/utils/http'
import type { SalarySettings, OperationSalaryRates } from '@/types/api/salary'
export const fetchSalarySettings = () =>
  request.get<SalarySettings>({ url: '/api/v1/system/salary-settings' })
export const updateSalarySettings = (data: SalarySettings) =>
  request.put<SalarySettings>({ url: '/api/v1/system/salary-settings', data })
export const fetchOperationSalaryRates = () =>
  request.get<OperationSalaryRates>({ url: '/api/v1/operation-salary-rates' })
