import type { RouteAccessMeta } from '@/app/route-access'

export type MigrationBatch = 'A' | 'B' | 'C' | 'D' | 'Tail'
export type AppPageType =
  | 'home'
  | 'dashboard-console'
  | 'dashboard-characters'
  | 'dashboard-npc-kills'
  | 'dashboard-corporation-structures'
  | 'info-wallet'
  | 'info-skill'
  | 'info-npc-kills'
  | 'info-ships'
  | 'info-implants'
  | 'info-fittings'
  | 'info-assets'
  | 'info-contracts'
  | 'info-esi-check'
  | 'ticket-my-tickets'
  | 'ticket-create'
  | 'ticket-detail'
  | 'ticket-management'
  | 'ticket-categories'
  | 'ticket-statistics'
  | 'ticket-admin-detail'
  | 'welfare-my'
  | 'welfare-approval'
  | 'welfare-settings'
  | 'newbro-select-captain'
  | 'newbro-select-mentor'
  | 'newbro-captain'
  | 'newbro-mentor'
  | 'newbro-manage'
  | 'newbro-mentor-manage'
  | 'newbro-recruit-link'
  | 'skill-plan-completion-check'
  | 'skill-plans'
  | 'personal-skill-plans'
  | 'operation-join'
  | 'operation-pap'
  | 'shop-browse'
  | 'shop-manage'
  | 'shop-order-manage'
  | 'shop-wallet'
  | 'srp-apply'
  | 'srp-manage'
  | 'srp-prices'
  | 'stub'
  | 'admin-demo'

export interface AppRouteSpec {
  path: string
  titleKey: string
  pageType: AppPageType
  batch?: MigrationBatch
  stubTitle?: string
  menuGroup?: string
  menuIcon?: string
  menuHidden?: boolean
  meta?: RouteAccessMeta
}

export const appRouteSpecs: AppRouteSpec[] = [
  { path: '', titleKey: 'nav.home', pageType: 'home', menuHidden: true, meta: { authList: [] } },
  {
    path: 'admin-demo',
    titleKey: 'nav.permissionDemo',
    pageType: 'admin-demo',
    menuHidden: true,
    meta: {
      login: true,
      roles: ['super_admin', 'admin'],
      authList: [
        { title: '审批订单', authMark: 'approve_order' },
        { title: '编辑兑换率', authMark: 'edit_exchange_rate' },
      ],
    },
  },
  {
    path: 'dashboard/console',
    titleKey: 'nav.dashboard.console',
    pageType: 'dashboard-console',
    menuGroup: 'nav.group.dashboard',
    menuIcon: 'dashboard',
    meta: { login: true },
  },
  {
    path: 'dashboard/characters',
    titleKey: 'nav.dashboard.characters',
    pageType: 'dashboard-characters',
    menuGroup: 'nav.group.dashboard',
    menuIcon: 'dashboard',
    meta: { login: true },
  },
  {
    path: 'dashboard/npc-kills',
    titleKey: 'nav.dashboard.npcKills',
    pageType: 'dashboard-npc-kills',
    batch: 'A',
    menuGroup: 'nav.group.dashboard',
    menuIcon: 'dashboard',
    meta: { roles: ['super_admin', 'admin'] },
  },
  {
    path: 'dashboard/corporation-structures',
    titleKey: 'nav.dashboard.corporationStructures',
    pageType: 'dashboard-corporation-structures',
    batch: 'A',
    menuGroup: 'nav.group.dashboard',
    menuIcon: 'dashboard',
    meta: { roles: ['super_admin', 'admin'] },
  },

  {
    path: 'info/wallet',
    titleKey: 'nav.info.wallet',
    pageType: 'info-wallet',
    menuGroup: 'nav.group.info',
    menuIcon: 'info',
    meta: { login: true },
  },
  {
    path: 'info/skill',
    titleKey: 'nav.info.skill',
    pageType: 'info-skill',
    menuGroup: 'nav.group.info',
    menuIcon: 'info',
    meta: { login: true },
  },
  {
    path: 'info/npc-kills',
    titleKey: 'nav.info.npcKills',
    pageType: 'info-npc-kills',
    batch: 'A',
    menuGroup: 'nav.group.info',
    menuIcon: 'info',
    meta: { login: true },
  },
  {
    path: 'info/ships',
    titleKey: 'nav.info.ships',
    pageType: 'info-ships',
    batch: 'A',
    menuGroup: 'nav.group.info',
    menuIcon: 'info',
    meta: { login: true },
  },
  {
    path: 'info/implants',
    titleKey: 'nav.info.implants',
    pageType: 'info-implants',
    batch: 'A',
    menuGroup: 'nav.group.info',
    menuIcon: 'info',
    meta: { login: true },
  },
  {
    path: 'info/fittings',
    titleKey: 'nav.info.fittings',
    pageType: 'info-fittings',
    batch: 'A',
    menuGroup: 'nav.group.info',
    menuIcon: 'info',
    meta: { login: true },
  },
  {
    path: 'info/assets',
    titleKey: 'nav.info.assets',
    pageType: 'info-assets',
    batch: 'A',
    menuGroup: 'nav.group.info',
    menuIcon: 'info',
    meta: { login: true },
  },
  {
    path: 'info/contracts',
    titleKey: 'nav.info.contracts',
    pageType: 'info-contracts',
    batch: 'A',
    menuGroup: 'nav.group.info',
    menuIcon: 'info',
    meta: { login: true },
  },
  {
    path: 'info/esi-check',
    titleKey: 'nav.info.esiCheck',
    pageType: 'info-esi-check',
    batch: 'A',
    menuGroup: 'nav.group.info',
    menuIcon: 'info',
    meta: { login: true },
  },

  {
    path: 'operation/fleets',
    titleKey: 'nav.operation.fleets',
    pageType: 'stub',
    stubTitle: 'Operation Fleets',
    batch: 'D',
    menuGroup: 'nav.group.operation',
    menuIcon: 'operation',
    meta: { roles: ['super_admin', 'admin', 'fc', 'senior_fc'] },
  },
  {
    path: 'operation/fleet-configs',
    titleKey: 'nav.operation.fleetConfigs',
    pageType: 'stub',
    stubTitle: 'Operation Fleet Configs',
    batch: 'D',
    menuGroup: 'nav.group.operation',
    menuIcon: 'operation',
    meta: { login: true },
  },
  {
    path: 'operation/fleet-detail/:id',
    titleKey: 'nav.operation.fleetDetail',
    pageType: 'stub',
    stubTitle: 'Operation Fleet Detail',
    batch: 'D',
    menuHidden: true,
    meta: { roles: ['super_admin', 'admin', 'fc', 'senior_fc'] },
  },
  {
    path: 'operation/corporation-pap',
    titleKey: 'nav.operation.corporationPap',
    pageType: 'stub',
    stubTitle: 'Operation Corporation PAP',
    batch: 'D',
    menuGroup: 'nav.group.operation',
    menuIcon: 'operation',
    meta: { login: true },
  },
  {
    path: 'operation/pap',
    titleKey: 'nav.operation.pap',
    pageType: 'operation-pap',
    batch: 'C',
    menuGroup: 'nav.group.operation',
    menuIcon: 'operation',
    meta: { login: true },
  },
  {
    path: 'operation/join',
    titleKey: 'nav.operation.join',
    pageType: 'operation-join',
    batch: 'C',
    menuHidden: true,
    meta: { login: true },
  },

  {
    path: 'skill-planning/completion-check',
    titleKey: 'nav.skillPlanning.completionCheck',
    pageType: 'skill-plan-completion-check',
    batch: 'C',
    menuGroup: 'nav.group.skillPlanning',
    menuIcon: 'skillPlanning',
    meta: { login: true },
  },
  {
    path: 'skill-planning/skill-plans',
    titleKey: 'nav.skillPlanning.skillPlans',
    pageType: 'skill-plans',
    batch: 'C',
    menuGroup: 'nav.group.skillPlanning',
    menuIcon: 'skillPlanning',
    meta: { login: true },
  },
  {
    path: 'skill-planning/personal-skill-plans',
    titleKey: 'nav.skillPlanning.personalSkillPlans',
    pageType: 'personal-skill-plans',
    batch: 'C',
    menuGroup: 'nav.group.skillPlanning',
    menuIcon: 'skillPlanning',
    meta: { login: true },
  },

  {
    path: 'welfare/my',
    titleKey: 'nav.welfare.my',
    pageType: 'welfare-my',
    batch: 'B',
    menuGroup: 'nav.group.welfare',
    menuIcon: 'welfare',
    meta: { login: true },
  },
  {
    path: 'welfare/approval',
    titleKey: 'nav.welfare.approval',
    pageType: 'welfare-approval',
    batch: 'B',
    menuGroup: 'nav.group.welfare',
    menuIcon: 'welfare',
    meta: { roles: ['super_admin', 'admin', 'welfare'] },
  },
  {
    path: 'welfare/settings',
    titleKey: 'nav.welfare.settings',
    pageType: 'welfare-settings',
    batch: 'B',
    menuGroup: 'nav.group.welfare',
    menuIcon: 'welfare',
    meta: { roles: ['super_admin', 'admin', 'welfare'] },
  },

  {
    path: 'newbro/select-captain',
    titleKey: 'nav.newbro.selectCaptain',
    pageType: 'newbro-select-captain',
    batch: 'B',
    menuGroup: 'nav.group.newbro',
    menuIcon: 'newbro',
    meta: { login: true, requiresNewbro: true },
  },
  {
    path: 'newbro/select-mentor',
    titleKey: 'nav.newbro.selectMentor',
    pageType: 'newbro-select-mentor',
    batch: 'B',
    menuGroup: 'nav.group.newbro',
    menuIcon: 'newbro',
    meta: { login: true, requiresMentorMenteeEligibility: true },
  },
  {
    path: 'newbro/captain',
    titleKey: 'nav.newbro.captain',
    pageType: 'newbro-captain',
    batch: 'B',
    menuGroup: 'nav.group.newbro',
    menuIcon: 'newbro',
    meta: { roles: ['captain'] },
  },
  {
    path: 'newbro/mentor',
    titleKey: 'nav.newbro.mentor',
    pageType: 'newbro-mentor',
    batch: 'B',
    menuGroup: 'nav.group.newbro',
    menuIcon: 'newbro',
    meta: { roles: ['mentor'] },
  },
  {
    path: 'newbro/manage',
    titleKey: 'nav.newbro.manage',
    pageType: 'newbro-manage',
    batch: 'B',
    menuGroup: 'nav.group.newbro',
    menuIcon: 'newbro',
    meta: { roles: ['super_admin', 'admin', 'captain'] },
  },
  {
    path: 'newbro/mentor-manage',
    titleKey: 'nav.newbro.mentorManage',
    pageType: 'newbro-mentor-manage',
    batch: 'B',
    menuGroup: 'nav.group.newbro',
    menuIcon: 'newbro',
    meta: { roles: ['super_admin', 'admin'] },
  },
  {
    path: 'newbro/recruit-link',
    titleKey: 'nav.newbro.recruitLink',
    pageType: 'newbro-recruit-link',
    batch: 'B',
    menuGroup: 'nav.group.newbro',
    menuIcon: 'newbro',
    meta: { login: true },
  },

  {
    path: 'shop/browse',
    titleKey: 'nav.shop.browse',
    pageType: 'shop-browse',
    batch: 'C',
    menuGroup: 'nav.group.shop',
    menuIcon: 'shop',
    meta: { login: true },
  },
  {
    path: 'shop/manage',
    titleKey: 'nav.shop.manage',
    pageType: 'shop-manage',
    batch: 'C',
    menuGroup: 'nav.group.shop',
    menuIcon: 'shop',
    meta: { roles: ['super_admin', 'admin'], authList: [
      { title: '新增商品', authMark: 'add_product' },
      { title: '编辑商品', authMark: 'edit_product' },
      { title: '删除商品', authMark: 'delete_product' },
    ] },
  },
  {
    path: 'shop/order-manage',
    titleKey: 'nav.shop.orderManage',
    pageType: 'shop-order-manage',
    batch: 'C',
    menuGroup: 'nav.group.shop',
    menuIcon: 'shop',
    meta: { roles: ['super_admin', 'admin', 'shop_order_manage'], authList: [{ title: '审批订单', authMark: 'approve_order' }] },
  },
  {
    path: 'shop/wallet',
    titleKey: 'nav.shop.wallet',
    pageType: 'shop-wallet',
    batch: 'C',
    menuGroup: 'nav.group.shop',
    menuIcon: 'shop',
    meta: { login: true },
  },

  {
    path: 'srp/srp-apply',
    titleKey: 'nav.srp.apply',
    pageType: 'srp-apply',
    batch: 'B',
    menuGroup: 'nav.group.srp',
    menuIcon: 'srp',
    meta: { login: true },
  },
  {
    path: 'srp/srp-manage',
    titleKey: 'nav.srp.manage',
    pageType: 'srp-manage',
    batch: 'B',
    menuGroup: 'nav.group.srp',
    menuIcon: 'srp',
    meta: {
      roles: ['super_admin', 'admin', 'senior_fc', 'srp'],
      authList: [{ title: '审批', authMark: 'approve' }],
    },
  },
  {
    path: 'srp/srp-prices',
    titleKey: 'nav.srp.prices',
    pageType: 'srp-prices',
    batch: 'B',
    menuGroup: 'nav.group.srp',
    menuIcon: 'srp',
    meta: { roles: ['super_admin', 'admin', 'senior_fc', 'srp'] },
  },

  {
    path: 'ticket/my-tickets',
    titleKey: 'nav.ticket.myTickets',
    pageType: 'ticket-my-tickets',
    batch: 'B',
    menuGroup: 'nav.group.ticket',
    menuIcon: 'ticket',
    meta: { login: true },
  },
  {
    path: 'ticket/create',
    titleKey: 'nav.ticket.create',
    pageType: 'ticket-create',
    batch: 'B',
    menuGroup: 'nav.group.ticket',
    menuIcon: 'ticket',
    meta: { login: true },
  },
  {
    path: 'ticket/detail/:id',
    titleKey: 'nav.ticket.detail',
    pageType: 'ticket-detail',
    batch: 'B',
    menuHidden: true,
    meta: { login: true },
  },
  {
    path: 'ticket/management',
    titleKey: 'nav.ticket.management',
    pageType: 'ticket-management',
    batch: 'B',
    menuGroup: 'nav.group.ticket',
    menuIcon: 'ticket',
    meta: { roles: ['super_admin', 'admin'] },
  },
  {
    path: 'ticket/categories',
    titleKey: 'nav.ticket.categories',
    pageType: 'ticket-categories',
    batch: 'B',
    menuGroup: 'nav.group.ticket',
    menuIcon: 'ticket',
    meta: { roles: ['super_admin', 'admin'] },
  },
  {
    path: 'ticket/statistics',
    titleKey: 'nav.ticket.statistics',
    pageType: 'ticket-statistics',
    batch: 'B',
    menuGroup: 'nav.group.ticket',
    menuIcon: 'ticket',
    meta: { roles: ['super_admin', 'admin'] },
  },
  {
    path: 'ticket/admin-detail/:id',
    titleKey: 'nav.ticket.adminDetail',
    pageType: 'ticket-admin-detail',
    batch: 'B',
    menuHidden: true,
    meta: { roles: ['super_admin', 'admin'] },
  },

  {
    path: 'system/user',
    titleKey: 'nav.system.user',
    pageType: 'stub',
    stubTitle: 'System User',
    batch: 'D',
    menuGroup: 'nav.group.system',
    menuIcon: 'system',
    meta: {
      roles: ['super_admin', 'admin'],
      authList: [
        { title: '删除用户', authMark: 'delete_user' },
        { title: '分配职权', authMark: 'assign_role' },
      ],
    },
  },
  {
    path: 'system/task-manager',
    titleKey: 'nav.system.taskManager',
    pageType: 'stub',
    stubTitle: 'System Task Manager',
    batch: 'D',
    menuGroup: 'nav.group.system',
    menuIcon: 'system',
    meta: {
      roles: ['super_admin', 'admin'],
      authList: [
        { title: '执行任务', authMark: 'execute_task' },
        { title: '修改调度', authMark: 'update_schedule' },
      ],
    },
  },
  {
    path: 'system/wallet',
    titleKey: 'nav.system.wallet',
    pageType: 'stub',
    stubTitle: 'System Wallet',
    batch: 'D',
    menuGroup: 'nav.group.system',
    menuIcon: 'system',
    meta: {
      roles: ['super_admin', 'admin'],
      authList: [
        { title: '调整余额', authMark: 'adjust_balance' },
        { title: '查看日志', authMark: 'view_log' },
      ],
    },
  },
  {
    path: 'system/audit',
    titleKey: 'nav.system.audit',
    pageType: 'stub',
    stubTitle: 'System Audit',
    batch: 'D',
    menuGroup: 'nav.group.system',
    menuIcon: 'system',
    meta: {
      roles: ['super_admin', 'admin'],
      authList: [{ title: '查看审计明细', authMark: 'view_audit_detail' }],
    },
  },
  {
    path: 'system/pap-exchange',
    titleKey: 'nav.system.papExchange',
    pageType: 'stub',
    stubTitle: 'System PAP Exchange',
    batch: 'D',
    menuGroup: 'nav.group.system',
    menuIcon: 'system',
    meta: {
      roles: ['super_admin', 'admin'],
      authList: [{ title: '编辑兑换率', authMark: 'edit_exchange_rate' }],
    },
  },
  {
    path: 'system/pap',
    titleKey: 'nav.system.pap',
    pageType: 'stub',
    stubTitle: 'System PAP',
    batch: 'D',
    menuGroup: 'nav.group.system',
    menuIcon: 'system',
    meta: {
      roles: ['super_admin', 'admin'],
      authList: [{ title: '手动拉取', authMark: 'manual_fetch' }],
    },
  },
  {
    path: 'system/auto-role',
    titleKey: 'nav.system.autoRole',
    pageType: 'stub',
    stubTitle: 'System Auto Role',
    batch: 'D',
    menuGroup: 'nav.group.system',
    menuIcon: 'system',
    meta: { roles: ['super_admin'] },
  },
  {
    path: 'system/user-center',
    titleKey: 'nav.system.userCenter',
    pageType: 'stub',
    stubTitle: 'System User Center',
    batch: 'D',
    menuHidden: true,
  },
  {
    path: 'system/webhook',
    titleKey: 'nav.system.webhook',
    pageType: 'stub',
    stubTitle: 'System Webhook',
    batch: 'D',
    menuGroup: 'nav.group.system',
    menuIcon: 'system',
    meta: { roles: ['super_admin'] },
  },
  {
    path: 'system/basic-config',
    titleKey: 'nav.system.basicConfig',
    pageType: 'stub',
    stubTitle: 'System Basic Config',
    batch: 'D',
    menuGroup: 'nav.group.system',
    menuIcon: 'system',
    meta: { roles: ['super_admin'] },
  },

  {
    path: 'hall-of-fame/temple',
    titleKey: 'nav.hallOfFame.temple',
    pageType: 'stub',
    stubTitle: 'Hall Of Fame Temple',
    batch: 'A',
    menuHidden: true,
    meta: { login: true },
  },
  {
    path: 'hall-of-fame/manage',
    titleKey: 'nav.hallOfFame.manage',
    pageType: 'stub',
    stubTitle: 'Hall Of Fame Manage',
    batch: 'Tail',
    menuHidden: true,
    meta: { roles: ['super_admin', 'admin'] },
  },
  {
    path: 'hall-of-fame/current-manage',
    titleKey: 'nav.hallOfFame.currentManage',
    pageType: 'stub',
    stubTitle: 'Hall Of Fame Current Manage',
    batch: 'Tail',
    menuHidden: true,
    meta: { login: true },
  },
]
