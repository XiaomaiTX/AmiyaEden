import { createHashRouter, Navigate, type RouteObject } from 'react-router-dom'
import { appRouteSpecs } from '@/app/migration-routes'
import { RouteAccessGate } from '@/auth'
import { RouterRuntimeBridge } from '@/app/router-runtime-bridge'
import { AppShell } from '@/layout'
import { AdminDemoPage } from '@/pages/admin-demo-page'
import { AuthCallbackPage } from '@/pages/auth-callback-page'
import { DashboardCharactersPage } from '@/pages/dashboard-characters-page'
import { DashboardConsolePage } from '@/pages/dashboard-console-page'
import { DashboardNpcKillsPage } from '@/pages/dashboard-npc-kills-page'
import { DashboardCorporationStructuresPage } from '@/pages/dashboard-corporation-structures-page'
import { ForbiddenPage } from '@/pages/forbidden-page'
import { HomePage } from '@/pages/home-page'
import { InfoNpcKillsPage } from '@/pages/info-npc-kills-page'
import { InfoWalletPage } from '@/pages/info-wallet-page'
import { InfoSkillPage } from '@/pages/info-skill-page'
import { InfoShipsPage } from '@/pages/info-ships-page'
import { InfoImplantsPage } from '@/pages/info-implants-page'
import { InfoFittingsPage } from '@/pages/info-fittings-page'
import { InfoAssetsPage } from '@/pages/info-assets-page'
import { InfoContractsPage } from '@/pages/info-contracts-page'
import { InfoEsiCheckPage } from '@/pages/info-esi-check-page'
import { OperationCorporationPapPage } from '@/pages/operation-corporation-pap-page'
import { OperationFleetConfigsPage } from '@/pages/operation-fleet-configs-page'
import { OperationFleetDetailPage } from '@/pages/operation-fleet-detail-page'
import { OperationFleetsPage } from '@/pages/operation-fleets-page'
import { SystemTaskManagerPage } from '@/pages/system-task-manager-page'
import { SystemUserPage } from '@/pages/system-user-page'
import { SystemWalletPage } from '@/pages/system-wallet-page'
import { TicketCreatePage } from '@/pages/ticket-create-page'
import { TicketManagementPage } from '@/pages/ticket-management-page'
import { TicketCategoriesPage } from '@/pages/ticket-categories-page'
import { TicketStatisticsPage } from '@/pages/ticket-statistics-page'
import { TicketAdminDetailPage } from '@/pages/ticket-admin-detail-page'
import { TicketDetailPage } from '@/pages/ticket-detail-page'
import { TicketMyTicketsPage } from '@/pages/ticket-my-tickets-page'
import { NewbroSelectCaptainPage } from '@/pages/newbro-select-captain-page'
import { NewbroSelectMentorPage } from '@/pages/newbro-select-mentor-page'
import { NewbroCaptainPage } from '@/pages/newbro-captain-page'
import { NewbroMentorPage } from '@/pages/newbro-mentor-page'
import { NewbroManagePage } from '@/pages/newbro-manage-page'
import { NewbroMentorManagePage } from '@/pages/newbro-mentor-manage-page'
import { NewbroRecruitLinkPage } from '@/pages/newbro-recruit-link-page'
import { OperationJoinPage } from '@/pages/operation-join-page'
import { OperationPapPage } from '@/pages/operation-pap-page'
import { PersonalSkillPlansPage } from '@/pages/personal-skill-plans-page'
import { SystemAuditPage } from '@/pages/system-audit-page'
import { SystemAutoRolePage } from '@/pages/system-auto-role-page'
import { SystemBasicConfigPage } from '@/pages/system-basic-config-page'
import { SkillPlanCompletionCheckPage } from '@/pages/skill-plan-completion-check-page'
import { SkillPlansPage } from '@/pages/skill-plans-page'
import { SystemPAPExchangePage } from '@/pages/system-pap-exchange-page'
import { SystemPAPPage } from '@/pages/system-pap-page'
import { SystemUserCenterPage } from '@/pages/system-user-center-page'
import { SystemWebhookPage } from '@/pages/system-webhook-page'
import { WelfareMyPage } from '@/pages/welfare-my-page'
import { WelfareApprovalPage } from '@/pages/welfare-approval-page'
import { WelfareSettingsPage } from '@/pages/welfare-settings-page'
import { ShopBrowsePage } from '@/pages/shop-browse-page'
import { ShopManagePage } from '@/pages/shop-manage-page'
import { ShopOrderManagePage } from '@/pages/shop-order-manage-page'
import { ShopWalletPage } from '@/pages/shop-wallet-page'
import { SrpApplyPage } from '@/pages/srp-apply-page'
import { SrpManagePage } from '@/pages/srp-manage-page'
import { SrpPricesPage } from '@/pages/srp-prices-page'
import { LoginPage } from '@/pages/login-page'
import { MigrationStubPage } from '@/pages/migration-stub-page'
import { NotFoundPage } from '@/pages/not-found-page'
import { ServerErrorPage } from '@/pages/server-error-page'

function renderShellPage(route: (typeof appRouteSpecs)[number]) {
  switch (route.pageType) {
    case 'home':
      return <HomePage />
    case 'dashboard-console':
      return <DashboardConsolePage />
    case 'dashboard-characters':
      return <DashboardCharactersPage />
    case 'dashboard-npc-kills':
      return <DashboardNpcKillsPage />
    case 'dashboard-corporation-structures':
      return <DashboardCorporationStructuresPage />
    case 'info-wallet':
      return <InfoWalletPage />
    case 'info-skill':
      return <InfoSkillPage />
    case 'info-npc-kills':
      return <InfoNpcKillsPage />
    case 'info-ships':
      return <InfoShipsPage />
    case 'info-implants':
      return <InfoImplantsPage />
    case 'info-fittings':
      return <InfoFittingsPage />
    case 'info-assets':
      return <InfoAssetsPage />
    case 'info-contracts':
      return <InfoContractsPage />
    case 'info-esi-check':
      return <InfoEsiCheckPage />
    case 'ticket-my-tickets':
      return <TicketMyTicketsPage />
    case 'ticket-create':
      return <TicketCreatePage />
    case 'ticket-detail':
      return <TicketDetailPage />
    case 'ticket-management':
      return <TicketManagementPage />
    case 'ticket-categories':
      return <TicketCategoriesPage />
    case 'ticket-statistics':
      return <TicketStatisticsPage />
    case 'ticket-admin-detail':
      return <TicketAdminDetailPage />
    case 'welfare-my':
      return <WelfareMyPage />
    case 'welfare-approval':
      return <WelfareApprovalPage />
    case 'welfare-settings':
      return <WelfareSettingsPage />
    case 'newbro-select-captain':
      return <NewbroSelectCaptainPage />
    case 'newbro-select-mentor':
      return <NewbroSelectMentorPage />
    case 'newbro-captain':
      return <NewbroCaptainPage />
    case 'newbro-mentor':
      return <NewbroMentorPage />
    case 'newbro-manage':
      return <NewbroManagePage />
    case 'newbro-mentor-manage':
      return <NewbroMentorManagePage />
    case 'newbro-recruit-link':
      return <NewbroRecruitLinkPage />
    case 'skill-plan-completion-check':
      return <SkillPlanCompletionCheckPage />
    case 'skill-plans':
      return <SkillPlansPage />
    case 'personal-skill-plans':
      return <PersonalSkillPlansPage />
    case 'operation-join':
      return <OperationJoinPage />
    case 'operation-pap':
      return <OperationPapPage />
    case 'operation-fleets':
      return <OperationFleetsPage />
    case 'operation-fleet-detail':
      return <OperationFleetDetailPage />
    case 'operation-fleet-configs':
      return <OperationFleetConfigsPage />
    case 'operation-corporation-pap':
      return <OperationCorporationPapPage />
    case 'shop-browse':
      return <ShopBrowsePage />
    case 'shop-manage':
      return <ShopManagePage />
    case 'shop-order-manage':
      return <ShopOrderManagePage />
    case 'shop-wallet':
      return <ShopWalletPage />
    case 'system-task-manager':
      return <SystemTaskManagerPage />
    case 'system-user':
      return <SystemUserPage />
    case 'system-wallet':
      return <SystemWalletPage />
    case 'system-audit':
      return <SystemAuditPage />
    case 'system-pap-exchange':
      return <SystemPAPExchangePage />
    case 'system-pap':
      return <SystemPAPPage />
    case 'system-auto-role':
      return <SystemAutoRolePage />
    case 'system-user-center':
      return <SystemUserCenterPage />
    case 'system-webhook':
      return <SystemWebhookPage />
    case 'system-basic-config':
      return <SystemBasicConfigPage />
    case 'srp-apply':
      return <SrpApplyPage />
    case 'srp-manage':
      return <SrpManagePage />
    case 'srp-prices':
      return <SrpPricesPage />
    case 'admin-demo':
      return <AdminDemoPage />
    case 'stub':
    default:
      return (
        <MigrationStubPage
          title={route.stubTitle ?? route.path}
          path={`/${route.path}`}
          batch={route.batch ?? 'Tail'}
        />
      )
  }
}

const appShellChildren: RouteObject[] = appRouteSpecs.map((route) => {
  if (route.path === '') {
    return {
      index: true,
      element: <RouteAccessGate meta={route.meta}>{renderShellPage(route)}</RouteAccessGate>,
    }
  }

  return {
    path: route.path,
    element: <RouteAccessGate meta={route.meta}>{renderShellPage(route)}</RouteAccessGate>,
  }
})

export const appRoutes: RouteObject[] = [
  {
    element: <RouterRuntimeBridge />,
    children: [
      {
        path: '/login',
        element: <Navigate to="/auth/login" replace />,
      },
      {
        path: '/auth/login',
        element: <LoginPage />,
      },
      {
        path: '/auth/callback',
        element: <AuthCallbackPage />,
      },
      {
        path: '/403',
        element: <ForbiddenPage />,
      },
      {
        path: '/500',
        element: <ServerErrorPage />,
      },
      {
        path: '/',
        element: (
          <RouteAccessGate meta={{ login: true }}>
            <AppShell />
          </RouteAccessGate>
        ),
        children: appShellChildren,
      },
      {
        path: '*',
        element: <NotFoundPage />,
      },
    ],
  },
]

export const router = createHashRouter(appRoutes)
