import type * as api from '@/api/generated/client'
export type FittingReq = api.FleetConfigFittingReq & { fitting_name: string; srp_amount: number }
