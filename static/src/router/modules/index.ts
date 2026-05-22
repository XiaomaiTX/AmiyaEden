import { AppRouteRecord } from '@/types/router'
import { charactersRoutes } from './characters'
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
import { fuxiHallRoutes } from './fuxi-hall'
import { ticketRoutes } from './ticket'

/**
 * 导出所有模块化路由
 */
export const routeModules: AppRouteRecord[] = [
  charactersRoutes,
  dashboardRoutes,
  operationRoutes,
  skillPlanningRoutes,
  infoRoutes,
  welfareRoutes,
  newbroRoutes,
  shopRoutes,
  srpRoutes,
  fuxiHallRoutes,
  ticketRoutes,
  systemRoutes,
  exceptionRoutes
]
