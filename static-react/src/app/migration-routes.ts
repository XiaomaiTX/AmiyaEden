import type { RouteAccessMeta } from '@/app/route-access'
import type { BadgeCounts } from '@/types/api/badge'

export type MigrationBatch = 'A' | 'B' | 'C' | 'D' | 'Tail'
export type AppPageType =
  | 'dashboard-console'
  | 'dashboard-characters'
  | 'dashboard-npc-kills'
  | 'dashboard-corporation-structures'
  | 'dashboard-fuel-officer-structures'
  | 'dashboard-galaxy-registry'
  | 'fuxi-hall-leadership'
  | 'fuxi-hall-contributors'
  | 'fuxi-hall-manage'
  | 'info-wallet'
  | 'info-skill'
  | 'info-npc-kills'
  | 'info-ships'
  | 'info-implants'
  | 'info-fittings'
  | 'info-assets'
  | 'info-contracts'
  | 'info-esi-check'
  | 'info-tool-bookmarks'
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
  | 'operation-fleets'
  | 'operation-fleet-detail'
  | 'operation-fleet-configs'
  | 'operation-corporation-pap'
  | 'shop-browse'
  | 'shop-manage'
  | 'shop-order-manage'
  | 'shop-wallet'
  | 'system-audit'
  | 'system-pap-exchange'
  | 'system-pap'
  | 'system-auto-role'
  | 'system-user-center'
  | 'system-webhook'
  | 'system-basic-config'
  | 'system-qq-governance'
  | 'srp-apply'
  | 'srp-manage'
  | 'srp-prices'
  | 'system-user'
  | 'system-task-manager'
  | 'system-wallet'

export interface AppRouteSpec {
  path: string
  titleKey: string
  pageType: AppPageType
  batch?: MigrationBatch
  menuGroup?: string
  menuIcon?: string
  menuHidden?: boolean
  badgeKey?: keyof BadgeCounts
  meta?: RouteAccessMeta
}

const unorderedAppRouteSpecs: AppRouteSpec[] = [
  {
    path: 'characters',
    titleKey: 'nav.dashboard.characters',
    pageType: 'dashboard-characters',
    menuGroup: 'nav.group.characters',
    menuIcon: 'characters',
    meta: { jwt: true },
  },
  {
    path: 'dashboard/console',
    titleKey: 'nav.dashboard.console',
    pageType: 'dashboard-console',
    menuGroup: 'nav.group.dashboard',
    menuIcon: 'dashboard',
    meta: { login: true, corpCapabilitiesAny: ['menu.dashboard'] },
  },
  {
    path: 'dashboard/npc-kills',
    titleKey: 'nav.dashboard.npcKills',
    pageType: 'dashboard-npc-kills',
    batch: 'A',
    menuGroup: 'nav.group.dashboard',
    menuIcon: 'dashboard',
    meta: {
      roles: ['super_admin', 'admin'],
      login: true,
      corpCapabilitiesAll: ['menu.dashboard', 'dashboard.npc_kills.corp', 'info.npc_kills.corp'],
    },
  },
  {
    path: 'dashboard/corporation-structures',
    titleKey: 'nav.dashboard.corporationStructures',
    pageType: 'dashboard-corporation-structures',
    batch: 'A',
    menuGroup: 'nav.group.dashboard',
    menuIcon: 'dashboard',
    badgeKey: 'corporation_structures_attention',
    meta: { roles: ['super_admin', 'admin'], login: true, corpCapabilitiesAny: ['menu.dashboard'] },
  },
  {
    path: 'dashboard/fuel-officer-structures',
    titleKey: 'nav.dashboard.fuelOfficerStructures',
    pageType: 'dashboard-fuel-officer-structures',
    menuGroup: 'nav.group.dashboard',
    menuIcon: 'dashboard',
    meta: {
      roles: ['super_admin', 'fuel_officer'],
      login: true,
      corpCapabilitiesAny: ['menu.dashboard'],
    },
  },
  {
    path: 'dashboard/galaxy-registry',
    titleKey: 'nav.dashboard.galaxyRegistry',
    pageType: 'dashboard-galaxy-registry',
    menuGroup: 'nav.group.dashboard',
    menuIcon: 'dashboard',
    meta: {
      roles: ['super_admin', 'admin', 'captain', 'user'],
      login: true,
      corpCapabilitiesAny: ['menu.dashboard'],
    },
  },
  {
    path: 'fuxi-hall/leadership',
    titleKey: 'nav.fuxiHall.leadership',
    pageType: 'fuxi-hall-leadership',
    menuGroup: 'nav.group.fuxiHall',
    menuIcon: 'fuxiHall',
    meta: { login: true, corpCapabilitiesAny: ['menu.fuxi_hall'] },
  },
  {
    path: 'fuxi-hall/contributors',
    titleKey: 'nav.fuxiHall.contributors',
    pageType: 'fuxi-hall-contributors',
    menuGroup: 'nav.group.fuxiHall',
    menuIcon: 'fuxiHall',
    meta: { login: true, corpCapabilitiesAny: ['menu.fuxi_hall'] },
  },
  {
    path: 'fuxi-hall/manage',
    titleKey: 'nav.fuxiHall.manage',
    pageType: 'fuxi-hall-manage',
    menuGroup: 'nav.group.fuxiHall',
    menuIcon: 'fuxiHall',
    meta: { roles: ['super_admin', 'admin'], login: true, corpCapabilitiesAny: ['menu.fuxi_hall'] },
  },

  {
    path: 'info/wallet',
    titleKey: 'nav.info.wallet',
    pageType: 'info-wallet',
    menuGroup: 'nav.group.info',
    menuIcon: 'info',
    meta: { login: true, corpCapabilitiesAll: ['menu.info', 'info.wallet.read'] },
  },
  {
    path: 'info/skill',
    titleKey: 'nav.info.skill',
    pageType: 'info-skill',
    menuGroup: 'nav.group.info',
    menuIcon: 'info',
    meta: { login: true, corpCapabilitiesAll: ['menu.info', 'info.skills.read'] },
  },
  {
    path: 'info/npc-kills',
    titleKey: 'nav.info.npcKills',
    pageType: 'info-npc-kills',
    batch: 'A',
    menuGroup: 'nav.group.info',
    menuIcon: 'info',
    meta: { login: true, corpCapabilitiesAll: ['menu.info', 'info.npc_kills.self'] },
  },
  {
    path: 'info/ships',
    titleKey: 'nav.info.ships',
    pageType: 'info-ships',
    batch: 'A',
    menuGroup: 'nav.group.info',
    menuIcon: 'info',
    meta: { login: true, corpCapabilitiesAny: ['menu.info'] },
  },
  {
    path: 'info/implants',
    titleKey: 'nav.info.implants',
    pageType: 'info-implants',
    batch: 'A',
    menuGroup: 'nav.group.info',
    menuIcon: 'info',
    meta: { login: true, corpCapabilitiesAny: ['menu.info'] },
  },
  {
    path: 'info/fittings',
    titleKey: 'nav.info.fittings',
    pageType: 'info-fittings',
    batch: 'A',
    menuGroup: 'nav.group.info',
    menuIcon: 'info',
    meta: { login: true, corpCapabilitiesAll: ['menu.info', 'info.fittings.manage'] },
  },
  {
    path: 'info/assets',
    titleKey: 'nav.info.assets',
    pageType: 'info-assets',
    batch: 'A',
    menuGroup: 'nav.group.info',
    menuIcon: 'info',
    meta: { login: true, corpCapabilitiesAll: ['menu.info', 'info.assets.read'] },
  },
  {
    path: 'info/contracts',
    titleKey: 'nav.info.contracts',
    pageType: 'info-contracts',
    batch: 'A',
    menuGroup: 'nav.group.info',
    menuIcon: 'info',
    meta: { login: true, corpCapabilitiesAll: ['menu.info', 'info.contracts.read'] },
  },
  {
    path: 'info/esi-check',
    titleKey: 'nav.info.esiCheck',
    pageType: 'info-esi-check',
    batch: 'A',
    menuGroup: 'nav.group.info',
    menuIcon: 'info',
    meta: { login: true, corpCapabilitiesAny: ['menu.info'] },
  },
  {
    path: 'info/tool-bookmarks',
    titleKey: 'nav.info.toolBookmarks',
    pageType: 'info-tool-bookmarks',
    menuGroup: 'nav.group.info',
    menuIcon: 'info',
    meta: { login: true, corpCapabilitiesAny: ['menu.info'] },
  },

  {
    path: 'operation/fleets',
    titleKey: 'nav.operation.fleets',
    pageType: 'operation-fleets',
    batch: 'D',
    menuGroup: 'nav.group.operation',
    menuIcon: 'operation',
    meta: {
      roles: ['super_admin', 'admin', 'fc', 'senior_fc'],
      login: true,
      corpCapabilitiesAny: ['menu.operation'],
    },
  },
  {
    path: 'operation/fleet-configs',
    titleKey: 'nav.operation.fleetConfigs',
    pageType: 'operation-fleet-configs',
    batch: 'D',
    menuGroup: 'nav.group.operation',
    menuIcon: 'operation',
    meta: { login: true, corpCapabilitiesAny: ['menu.operation'] },
  },
  {
    path: 'operation/fleet-detail/:id',
    titleKey: 'nav.operation.fleetDetail',
    pageType: 'operation-fleet-detail',
    batch: 'D',
    menuHidden: true,
    meta: {
      roles: ['super_admin', 'admin', 'fc', 'senior_fc'],
      login: true,
      corpCapabilitiesAny: ['menu.operation'],
    },
  },
  {
    path: 'operation/corporation-pap',
    titleKey: 'nav.operation.corporationPap',
    pageType: 'operation-corporation-pap',
    batch: 'D',
    menuGroup: 'nav.group.operation',
    menuIcon: 'operation',
    meta: { login: true, corpCapabilitiesAny: ['menu.operation'] },
  },
  {
    path: 'operation/pap',
    titleKey: 'nav.operation.pap',
    pageType: 'operation-pap',
    batch: 'C',
    menuGroup: 'nav.group.operation',
    menuIcon: 'operation',
    meta: { login: true, corpCapabilitiesAny: ['menu.operation'] },
  },
  {
    path: 'operation/join',
    titleKey: 'nav.operation.join',
    pageType: 'operation-join',
    batch: 'C',
    menuHidden: true,
    meta: { login: true, corpCapabilitiesAny: ['menu.operation'] },
  },

  {
    path: 'skill-planning/completion-check',
    titleKey: 'nav.skillPlanning.completionCheck',
    pageType: 'skill-plan-completion-check',
    batch: 'C',
    menuGroup: 'nav.group.skillPlanning',
    menuIcon: 'skillPlanning',
    meta: { login: true, corpCapabilitiesAny: ['menu.skill_planning'] },
  },
  {
    path: 'skill-planning/skill-plans',
    titleKey: 'nav.skillPlanning.skillPlans',
    pageType: 'skill-plans',
    batch: 'C',
    menuGroup: 'nav.group.skillPlanning',
    menuIcon: 'skillPlanning',
    meta: { login: true, corpCapabilitiesAny: ['menu.skill_planning'] },
  },
  {
    path: 'skill-planning/personal-skill-plans',
    titleKey: 'nav.skillPlanning.personalSkillPlans',
    pageType: 'personal-skill-plans',
    batch: 'C',
    menuGroup: 'nav.group.skillPlanning',
    menuIcon: 'skillPlanning',
    meta: { login: true, corpCapabilitiesAny: ['menu.skill_planning'] },
  },

  {
    path: 'welfare/my',
    titleKey: 'nav.welfare.my',
    pageType: 'welfare-my',
    batch: 'B',
    menuGroup: 'nav.group.welfare',
    menuIcon: 'welfare',
    badgeKey: 'welfare_eligible',
    meta: { login: true, corpCapabilitiesAny: ['welfare.user'] },
  },
  {
    path: 'welfare/approval',
    titleKey: 'nav.welfare.approval',
    pageType: 'welfare-approval',
    batch: 'B',
    menuGroup: 'nav.group.welfare',
    menuIcon: 'welfare',
    badgeKey: 'welfare_pending',
    meta: {
      roles: ['super_admin', 'admin', 'welfare'],
      login: true,
      corpCapabilitiesAny: ['welfare.approval'],
    },
  },
  {
    path: 'welfare/settings',
    titleKey: 'nav.welfare.settings',
    pageType: 'welfare-settings',
    batch: 'B',
    menuGroup: 'nav.group.welfare',
    menuIcon: 'welfare',
    meta: {
      roles: ['super_admin', 'admin', 'welfare'],
      login: true,
      corpCapabilitiesAny: ['welfare.settings'],
    },
  },

  {
    path: 'newbro/select-captain',
    titleKey: 'nav.newbro.selectCaptain',
    pageType: 'newbro-select-captain',
    batch: 'B',
    menuGroup: 'nav.group.newbro',
    menuIcon: 'newbro',
    meta: {
      login: true,
      requiresNewbro: true,
      corpCapabilitiesAny: ['menu.newbro'],
    },
  },
  {
    path: 'newbro/select-mentor',
    titleKey: 'nav.newbro.selectMentor',
    pageType: 'newbro-select-mentor',
    batch: 'B',
    menuGroup: 'nav.group.newbro',
    menuIcon: 'newbro',
    meta: {
      login: true,
      requiresMentorMenteeEligibility: true,
      corpCapabilitiesAny: ['menu.newbro'],
    },
  },
  {
    path: 'newbro/captain',
    titleKey: 'nav.newbro.captain',
    pageType: 'newbro-captain',
    batch: 'B',
    menuGroup: 'nav.group.newbro',
    menuIcon: 'newbro',
    meta: {
      roles: ['super_admin', 'captain'],
      login: true,
      corpCapabilitiesAny: ['menu.newbro'],
    },
  },
  {
    path: 'newbro/mentor',
    titleKey: 'nav.newbro.mentor',
    pageType: 'newbro-mentor',
    batch: 'B',
    menuGroup: 'nav.group.newbro',
    menuIcon: 'newbro',
    badgeKey: 'mentor_pending_applications',
    meta: {
      roles: ['super_admin', 'mentor'],
      login: true,
      corpCapabilitiesAny: ['menu.newbro'],
    },
  },
  {
    path: 'newbro/manage',
    titleKey: 'nav.newbro.manage',
    pageType: 'newbro-manage',
    batch: 'B',
    menuGroup: 'nav.group.newbro',
    menuIcon: 'newbro',
    meta: {
      roles: ['super_admin', 'admin', 'captain'],
      login: true,
      corpCapabilitiesAny: ['menu.newbro'],
    },
  },
  {
    path: 'newbro/mentor-manage',
    titleKey: 'nav.newbro.mentorManage',
    pageType: 'newbro-mentor-manage',
    batch: 'B',
    menuGroup: 'nav.group.newbro',
    menuIcon: 'newbro',
    meta: {
      roles: ['super_admin', 'admin'],
      login: true,
      corpCapabilitiesAny: ['menu.newbro'],
    },
  },
  {
    path: 'newbro/recruit-link',
    titleKey: 'nav.newbro.recruitLink',
    pageType: 'newbro-recruit-link',
    batch: 'B',
    menuGroup: 'nav.group.newbro',
    menuIcon: 'newbro',
    meta: { login: true, corpCapabilitiesAny: ['menu.newbro'] },
  },

  {
    path: 'shop/browse',
    titleKey: 'nav.shop.browse',
    pageType: 'shop-browse',
    batch: 'C',
    menuGroup: 'nav.group.shop',
    menuIcon: 'shop',
    meta: { login: true, corpCapabilitiesAll: ['menu.shop', 'shop.order.read_self'] },
  },
  {
    path: 'shop/manage',
    titleKey: 'nav.shop.manage',
    pageType: 'shop-manage',
    batch: 'C',
    menuGroup: 'nav.group.shop',
    menuIcon: 'shop',
    meta: {
      roles: ['super_admin', 'admin'],
      login: true,
      corpCapabilitiesAll: ['menu.shop', 'shop.manage', 'shop.admin.product.manage'],
      authList: [
        { titleKey: 'authActions.shop.addProduct', authMark: 'add_product' },
        { titleKey: 'authActions.shop.editProduct', authMark: 'edit_product' },
        { titleKey: 'authActions.shop.deleteProduct', authMark: 'delete_product' },
      ],
    },
  },
  {
    path: 'shop/order-manage',
    titleKey: 'nav.shop.orderManage',
    pageType: 'shop-order-manage',
    batch: 'C',
    menuGroup: 'nav.group.shop',
    menuIcon: 'shop',
    badgeKey: 'order_pending',
    meta: {
      roles: ['super_admin', 'admin', 'shop_order_manage'],
      login: true,
      corpCapabilitiesAll: ['menu.shop', 'shop.manage', 'shop.admin.order.manage'],
      authList: [{ titleKey: 'authActions.shop.approveOrder', authMark: 'approve_order' }],
    },
  },
  {
    path: 'shop/wallet',
    titleKey: 'nav.shop.wallet',
    pageType: 'shop-wallet',
    batch: 'C',
    menuGroup: 'nav.group.shop',
    menuIcon: 'shop',
    meta: { login: true, corpCapabilitiesAll: ['menu.shop', 'shop.wallet.read'] },
  },

  {
    path: 'srp/srp-apply',
    titleKey: 'nav.srp.apply',
    pageType: 'srp-apply',
    batch: 'B',
    menuGroup: 'nav.group.srp',
    menuIcon: 'srp',
    meta: { login: true, corpCapabilitiesAny: ['srp.user'] },
  },
  {
    path: 'srp/srp-manage',
    titleKey: 'nav.srp.manage',
    pageType: 'srp-manage',
    batch: 'B',
    menuGroup: 'nav.group.srp',
    menuIcon: 'srp',
    badgeKey: 'srp_pending',
    meta: {
      roles: ['super_admin', 'admin', 'senior_fc', 'srp'],
      login: true,
      corpCapabilitiesAny: ['srp.manage'],
      authList: [{ titleKey: 'authActions.srp.approve', authMark: 'approve' }],
    },
  },
  {
    path: 'srp/srp-prices',
    titleKey: 'nav.srp.prices',
    pageType: 'srp-prices',
    batch: 'B',
    menuGroup: 'nav.group.srp',
    menuIcon: 'srp',
    meta: {
      roles: ['super_admin', 'admin', 'senior_fc', 'srp'],
      login: true,
      corpCapabilitiesAny: ['srp.manage'],
    },
  },

  {
    path: 'ticket/my-tickets',
    titleKey: 'nav.ticket.myTickets',
    pageType: 'ticket-my-tickets',
    batch: 'B',
    menuGroup: 'nav.group.ticket',
    menuIcon: 'ticket',
    meta: { login: true, corpCapabilitiesAny: ['menu.ticket'] },
  },
  {
    path: 'ticket/create',
    titleKey: 'nav.ticket.create',
    pageType: 'ticket-create',
    batch: 'B',
    menuGroup: 'nav.group.ticket',
    menuIcon: 'ticket',
    meta: { login: true, corpCapabilitiesAll: ['menu.ticket', 'ticket.user.create'] },
  },
  {
    path: 'ticket/detail/:id',
    titleKey: 'nav.ticket.detail',
    pageType: 'ticket-detail',
    batch: 'B',
    menuHidden: true,
    meta: { login: true, corpCapabilitiesAny: ['menu.ticket'] },
  },
  {
    path: 'ticket/management',
    titleKey: 'nav.ticket.management',
    pageType: 'ticket-management',
    batch: 'B',
    menuGroup: 'nav.group.ticket',
    menuIcon: 'ticket',
    badgeKey: 'ticket_attention',
    meta: {
      roles: ['super_admin', 'admin'],
      login: true,
      corpCapabilitiesAll: ['menu.ticket', 'ticket.manage', 'ticket.admin.read'],
    },
  },
  {
    path: 'ticket/categories',
    titleKey: 'nav.ticket.categories',
    pageType: 'ticket-categories',
    batch: 'B',
    menuGroup: 'nav.group.ticket',
    menuIcon: 'ticket',
    meta: {
      roles: ['super_admin', 'admin'],
      login: true,
      corpCapabilitiesAll: ['menu.ticket', 'ticket.manage', 'ticket.admin.read'],
    },
  },
  {
    path: 'ticket/statistics',
    titleKey: 'nav.ticket.statistics',
    pageType: 'ticket-statistics',
    batch: 'B',
    menuGroup: 'nav.group.ticket',
    menuIcon: 'ticket',
    meta: {
      roles: ['super_admin', 'admin'],
      login: true,
      corpCapabilitiesAll: ['menu.ticket', 'ticket.manage', 'ticket.admin.read'],
    },
  },
  {
    path: 'ticket/admin-detail/:id',
    titleKey: 'nav.ticket.adminDetail',
    pageType: 'ticket-admin-detail',
    batch: 'B',
    menuHidden: true,
    meta: {
      roles: ['super_admin', 'admin'],
      login: true,
      corpCapabilitiesAll: ['menu.ticket', 'ticket.manage', 'ticket.admin.read'],
    },
  },

  {
    path: 'system/user',
    titleKey: 'nav.system.user',
    pageType: 'system-user',
    batch: 'D',
    menuGroup: 'nav.group.system',
    menuIcon: 'system',
    meta: {
      roles: ['super_admin', 'admin'],
      login: true,
      corpCapabilitiesAny: ['system.manage'],
      authList: [
        { titleKey: 'authActions.system.deleteUser', authMark: 'delete_user' },
        { titleKey: 'authActions.system.assignRole', authMark: 'assign_role' },
      ],
    },
  },
  {
    path: 'system/task-manager',
    titleKey: 'nav.system.taskManager',
    pageType: 'system-task-manager',
    batch: 'D',
    menuGroup: 'nav.group.system',
    menuIcon: 'system',
    meta: {
      roles: ['super_admin', 'admin'],
      login: true,
      corpCapabilitiesAny: ['system.task.read'],
      authList: [
        { titleKey: 'authActions.system.executeTask', authMark: 'execute_task' },
        { titleKey: 'authActions.system.updateSchedule', authMark: 'update_schedule' },
      ],
    },
  },
  {
    path: 'system/wallet',
    titleKey: 'nav.system.wallet',
    pageType: 'system-wallet',
    batch: 'D',
    menuGroup: 'nav.group.system',
    menuIcon: 'system',
    meta: {
      roles: ['super_admin', 'admin'],
      login: true,
      corpCapabilitiesAny: ['system.wallet.read'],
      authList: [
        { titleKey: 'authActions.system.adjustBalance', authMark: 'adjust_balance' },
        { titleKey: 'authActions.system.viewLog', authMark: 'view_log' },
      ],
    },
  },
  {
    path: 'system/audit',
    titleKey: 'nav.system.audit',
    pageType: 'system-audit',
    batch: 'D',
    menuGroup: 'nav.group.system',
    menuIcon: 'system',
    meta: {
      roles: ['super_admin', 'admin'],
      login: true,
      corpCapabilitiesAny: ['system.audit.read'],
      authList: [{ titleKey: 'authActions.system.viewAuditDetail', authMark: 'view_audit_detail' }],
    },
  },
  {
    path: 'system/pap-exchange',
    titleKey: 'nav.system.papExchange',
    pageType: 'system-pap-exchange',
    batch: 'D',
    menuGroup: 'nav.group.system',
    menuIcon: 'system',
    meta: {
      roles: ['super_admin', 'admin'],
      login: true,
      corpCapabilitiesAny: ['system.manage'],
      authList: [
        { titleKey: 'authActions.system.editExchangeRate', authMark: 'edit_exchange_rate' },
      ],
    },
  },
  {
    path: 'system/pap',
    titleKey: 'nav.system.pap',
    pageType: 'system-pap',
    batch: 'D',
    menuGroup: 'nav.group.system',
    menuIcon: 'system',
    meta: {
      roles: ['super_admin', 'admin'],
      login: true,
      corpCapabilitiesAny: ['system.manage'],
      authList: [{ titleKey: 'authActions.system.manualFetch', authMark: 'manual_fetch' }],
    },
  },
  {
    path: 'system/auto-role',
    titleKey: 'nav.system.autoRole',
    pageType: 'system-auto-role',
    batch: 'D',
    menuGroup: 'nav.group.system',
    menuIcon: 'system',
    meta: { roles: ['super_admin'] },
  },
  {
    path: 'system/user-center',
    titleKey: 'nav.system.userCenter',
    pageType: 'system-user-center',
    batch: 'D',
    menuHidden: true,
    meta: {
      roles: ['super_admin', 'admin'],
      corpCapabilitiesAny: [
        'system.manage',
        'system.task.read',
        'system.wallet.read',
        'system.audit.read',
        'system.basic_config.read',
      ],
    },
  },
  {
    path: 'system/webhook',
    titleKey: 'nav.system.webhook',
    pageType: 'system-webhook',
    batch: 'D',
    menuGroup: 'nav.group.system',
    menuIcon: 'system',
    meta: { roles: ['super_admin'], login: true, corpCapabilitiesAny: ['system.manage'] },
  },
  {
    path: 'system/basic-config',
    titleKey: 'nav.system.basicConfig',
    pageType: 'system-basic-config',
    batch: 'D',
    menuGroup: 'nav.group.system',
    menuIcon: 'system',
    meta: {
      roles: ['super_admin'],
      login: true,
      corpCapabilitiesAny: ['system.basic_config.read'],
    },
  },
  {
    path: 'system/qq-governance',
    titleKey: 'nav.system.qqGovernance',
    pageType: 'system-qq-governance',
    menuGroup: 'nav.group.system',
    menuIcon: 'system',
    meta: {
      roles: ['super_admin'],
      login: true,
      corpCapabilitiesAny: ['system.manage'],
    },
  },
]

const appRouteOrder = [
  'characters',
  'dashboard/console',
  'dashboard/npc-kills',
  'dashboard/corporation-structures',
  'dashboard/fuel-officer-structures',
  'dashboard/galaxy-registry',
  'operation/fleets',
  'operation/fleet-configs',
  'operation/fleet-detail/:id',
  'operation/corporation-pap',
  'operation/pap',
  'operation/join',
  'skill-planning/completion-check',
  'skill-planning/skill-plans',
  'skill-planning/personal-skill-plans',
  'info/wallet',
  'info/skill',
  'info/npc-kills',
  'info/ships',
  'info/implants',
  'info/fittings',
  'info/assets',
  'info/contracts',
  'info/esi-check',
  'info/tool-bookmarks',
  'welfare/my',
  'welfare/approval',
  'welfare/settings',
  'newbro/select-captain',
  'newbro/select-mentor',
  'newbro/captain',
  'newbro/mentor',
  'newbro/manage',
  'newbro/mentor-manage',
  'newbro/recruit-link',
  'shop/browse',
  'shop/manage',
  'shop/order-manage',
  'shop/wallet',
  'srp/srp-apply',
  'srp/srp-manage',
  'srp/srp-prices',
  'fuxi-hall/leadership',
  'fuxi-hall/contributors',
  'fuxi-hall/manage',
  'ticket/my-tickets',
  'ticket/create',
  'ticket/detail/:id',
  'ticket/management',
  'ticket/categories',
  'ticket/statistics',
  'ticket/admin-detail/:id',
  'system/user',
  'system/task-manager',
  'system/wallet',
  'system/audit',
  'system/pap-exchange',
  'system/pap',
  'system/auto-role',
  'system/user-center',
  'system/webhook',
  'system/qq-governance',
  'system/basic-config',
] as const

const routesByPath = new Map(unorderedAppRouteSpecs.map((route) => [route.path, route]))

export const appRouteSpecs: AppRouteSpec[] = appRouteOrder.map((path) => {
  const route = routesByPath.get(path)
  if (!route) {
    throw new Error(`Missing React route specification for ${path}`)
  }
  return route
})
