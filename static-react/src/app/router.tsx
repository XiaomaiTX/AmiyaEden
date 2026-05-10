import { createHashRouter, Navigate, type RouteObject } from 'react-router-dom'
import { appRouteSpecs } from '@/app/migration-routes'
import { RouteAccessGate } from '@/auth'
import { RouterRuntimeBridge } from '@/app/router-runtime-bridge'
import { AppShell } from '@/layout'
import { AdminDemoPage } from '@/pages/admin-demo-page'
import { AuthCallbackPage } from '@/pages/auth-callback-page'
import { DashboardCharactersPage } from '@/pages/dashboard-characters-page'
import { DashboardConsolePage } from '@/pages/dashboard-console-page'
import { ForbiddenPage } from '@/pages/forbidden-page'
import { HomePage } from '@/pages/home-page'
import { InfoWalletPage } from '@/pages/info-wallet-page'
import { InfoSkillPage } from '@/pages/info-skill-page'
import { InfoShipsPage } from '@/pages/info-ships-page'
import { InfoImplantsPage } from '@/pages/info-implants-page'
import { InfoFittingsPage } from '@/pages/info-fittings-page'
import { InfoAssetsPage } from '@/pages/info-assets-page'
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
    case 'info-wallet':
      return <InfoWalletPage />
    case 'info-skill':
      return <InfoSkillPage />
    case 'info-ships':
      return <InfoShipsPage />
    case 'info-implants':
      return <InfoImplantsPage />
    case 'info-fittings':
      return <InfoFittingsPage />
    case 'info-assets':
      return <InfoAssetsPage />
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
