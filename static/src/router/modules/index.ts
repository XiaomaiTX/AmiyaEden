import { AppRouteRecord } from '@/types/router'
import { dashboardRoutes } from './dashboard'
import { systemRoutes } from './system'
import { operationRoutes } from './operation'
import { skillPlanningRoutes } from './skill-planning'
import { exceptionRoutes } from './exception'
import { srpRoutes } from './srp'
import { welfareRoutes } from './welfare'
import { shopRoutes } from './shop'
import { infoRoutes } from './info'
import { newbroRoutes } from './newbro'
import { hallOfFameRoutes } from './hall-of-fame'

/**
 * 导出所有模块化路由
 */
export const routeModules: AppRouteRecord[] = [
  dashboardRoutes,
  operationRoutes,
  skillPlanningRoutes,
  infoRoutes,
  welfareRoutes,
  newbroRoutes,
  shopRoutes,
  srpRoutes,
  hallOfFameRoutes,
  systemRoutes,
  exceptionRoutes
]
