import type { ComponentType } from 'react'
import type { RouteObject } from 'react-router-dom'
import type { AppPageType } from '@/app/migration-routes'

type LazyRoute = NonNullable<RouteObject['lazy']>

function lazyNamed(
  importer: () => Promise<Record<string, unknown>>,
  exportName: string
): LazyRoute {
  return async () => {
    const module = await importer()
    return { Component: module[exportName] as ComponentType }
  }
}

export const appPageLoaders: Record<AppPageType, LazyRoute> = {
  'dashboard-console': lazyNamed(() => import('@/pages/dashboard-console-page'), 'DashboardConsolePage'),
  'dashboard-characters': lazyNamed(() => import('@/pages/dashboard-characters-page'), 'DashboardCharactersPage'),
  'dashboard-npc-kills': lazyNamed(() => import('@/pages/dashboard-npc-kills-page'), 'DashboardNpcKillsPage'),
  'dashboard-corporation-structures': lazyNamed(
    () => import('@/pages/dashboard-corporation-structures-page'),
    'DashboardCorporationStructuresPage'
  ),
  'dashboard-fuel-officer-structures': lazyNamed(
    () => import('@/pages/dashboard-fuel-officer-structures-page'),
    'DashboardFuelOfficerStructuresPage'
  ),
  'dashboard-galaxy-registry': lazyNamed(
    () => import('@/pages/dashboard-galaxy-registry-page'),
    'DashboardGalaxyRegistryPage'
  ),
  'fuxi-hall-leadership': lazyNamed(() => import('@/pages/fuxi-hall-public-page'), 'FuxiHallLeadershipPage'),
  'fuxi-hall-contributors': lazyNamed(() => import('@/pages/fuxi-hall-public-page'), 'FuxiHallContributorsPage'),
  'fuxi-hall-manage': lazyNamed(() => import('@/pages/fuxi-hall-manage-page'), 'FuxiHallManagePage'),
  'info-wallet': lazyNamed(() => import('@/pages/info-wallet-page'), 'InfoWalletPage'),
  'info-skill': lazyNamed(() => import('@/pages/info-skill-page'), 'InfoSkillPage'),
  'info-npc-kills': lazyNamed(() => import('@/pages/info-npc-kills-page'), 'InfoNpcKillsPage'),
  'info-ships': lazyNamed(() => import('@/pages/info-ships-page'), 'InfoShipsPage'),
  'info-implants': lazyNamed(() => import('@/pages/info-implants-page'), 'InfoImplantsPage'),
  'info-fittings': lazyNamed(() => import('@/pages/info-fittings-page'), 'InfoFittingsPage'),
  'info-assets': lazyNamed(() => import('@/pages/info-assets-page'), 'InfoAssetsPage'),
  'info-contracts': lazyNamed(() => import('@/pages/info-contracts-page'), 'InfoContractsPage'),
  'info-esi-check': lazyNamed(() => import('@/pages/info-esi-check-page'), 'InfoEsiCheckPage'),
  'info-tool-bookmarks': lazyNamed(() => import('@/pages/info-tool-bookmarks-page'), 'InfoToolBookmarksPage'),
  'ticket-my-tickets': lazyNamed(() => import('@/pages/ticket-my-tickets-page'), 'TicketMyTicketsPage'),
  'ticket-create': lazyNamed(() => import('@/pages/ticket-create-page'), 'TicketCreatePage'),
  'ticket-detail': lazyNamed(() => import('@/pages/ticket-detail-page'), 'TicketDetailPage'),
  'ticket-management': lazyNamed(() => import('@/pages/ticket-management-page'), 'TicketManagementPage'),
  'ticket-categories': lazyNamed(() => import('@/pages/ticket-categories-page'), 'TicketCategoriesPage'),
  'ticket-statistics': lazyNamed(() => import('@/pages/ticket-statistics-page'), 'TicketStatisticsPage'),
  'ticket-admin-detail': lazyNamed(() => import('@/pages/ticket-admin-detail-page'), 'TicketAdminDetailPage'),
  'welfare-my': lazyNamed(() => import('@/pages/welfare-my-page'), 'WelfareMyPage'),
  'welfare-approval': lazyNamed(() => import('@/pages/welfare-approval-page'), 'WelfareApprovalPage'),
  'welfare-settings': lazyNamed(() => import('@/pages/welfare-settings-page'), 'WelfareSettingsPage'),
  'newbro-select-captain': lazyNamed(() => import('@/pages/newbro-select-captain-page'), 'NewbroSelectCaptainPage'),
  'newbro-select-mentor': lazyNamed(() => import('@/pages/newbro-select-mentor-page'), 'NewbroSelectMentorPage'),
  'newbro-captain': lazyNamed(() => import('@/pages/newbro-captain-page'), 'NewbroCaptainPage'),
  'newbro-mentor': lazyNamed(() => import('@/pages/newbro-mentor-page'), 'NewbroMentorPage'),
  'newbro-manage': lazyNamed(() => import('@/pages/newbro-manage-page'), 'NewbroManagePage'),
  'newbro-mentor-manage': lazyNamed(() => import('@/pages/newbro-mentor-manage-page'), 'NewbroMentorManagePage'),
  'newbro-recruit-link': lazyNamed(() => import('@/pages/newbro-recruit-link-page'), 'NewbroRecruitLinkPage'),
  'skill-plan-completion-check': lazyNamed(
    () => import('@/pages/skill-plan-completion-check-page'),
    'SkillPlanCompletionCheckPage'
  ),
  'skill-plans': lazyNamed(() => import('@/pages/skill-plans-page'), 'SkillPlansPage'),
  'personal-skill-plans': lazyNamed(() => import('@/pages/personal-skill-plans-page'), 'PersonalSkillPlansPage'),
  'operation-join': lazyNamed(() => import('@/pages/operation-join-page'), 'OperationJoinPage'),
  'operation-pap': lazyNamed(() => import('@/pages/operation-pap-page'), 'OperationPapPage'),
  'operation-fleets': lazyNamed(() => import('@/pages/operation-fleets-page'), 'OperationFleetsPage'),
  'operation-fleet-detail': lazyNamed(() => import('@/pages/operation-fleet-detail-page'), 'OperationFleetDetailPage'),
  'operation-fleet-configs': lazyNamed(() => import('@/pages/operation-fleet-configs-page'), 'OperationFleetConfigsPage'),
  'operation-corporation-pap': lazyNamed(
    () => import('@/pages/operation-corporation-pap-page'),
    'OperationCorporationPapPage'
  ),
  'shop-browse': lazyNamed(() => import('@/pages/shop-browse-page'), 'ShopBrowsePage'),
  'shop-manage': lazyNamed(() => import('@/pages/shop-manage-page'), 'ShopManagePage'),
  'shop-order-manage': lazyNamed(() => import('@/pages/shop-order-manage-page'), 'ShopOrderManagePage'),
  'shop-wallet': lazyNamed(() => import('@/pages/shop-wallet-page'), 'ShopWalletPage'),
  'system-audit': lazyNamed(() => import('@/pages/system-audit-page'), 'SystemAuditPage'),
  'system-pap-exchange': lazyNamed(() => import('@/pages/system-pap-exchange-page'), 'SystemPAPExchangePage'),
  'system-pap': lazyNamed(() => import('@/pages/system-pap-page'), 'SystemPAPPage'),
  'system-auto-role': lazyNamed(() => import('@/pages/system-auto-role-page'), 'SystemAutoRolePage'),
  'system-user-center': lazyNamed(() => import('@/pages/system-user-center-page'), 'SystemUserCenterPage'),
  'system-webhook': lazyNamed(() => import('@/pages/system-webhook-page'), 'SystemWebhookPage'),
  'system-basic-config': lazyNamed(() => import('@/pages/system-basic-config-page'), 'SystemBasicConfigPage'),
  'system-qq-governance': lazyNamed(
    () => import('@/pages/system-qq-governance-page'),
    'SystemQQGovernancePage'
  ),
  'srp-apply': lazyNamed(() => import('@/pages/srp-apply-page'), 'SrpApplyPage'),
  'srp-manage': lazyNamed(() => import('@/pages/srp-manage-page'), 'SrpManagePage'),
  'srp-prices': lazyNamed(() => import('@/pages/srp-prices-page'), 'SrpPricesPage'),
  'system-user': lazyNamed(() => import('@/pages/system-user-page'), 'SystemUserPage'),
  'system-task-manager': lazyNamed(() => import('@/pages/system-task-manager-page'), 'SystemTaskManagerPage'),
  'system-wallet': lazyNamed(() => import('@/pages/system-wallet-page'), 'SystemWalletPage'),
}
