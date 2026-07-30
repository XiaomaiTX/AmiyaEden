import { createHashRouter, Navigate, type RouteObject } from 'react-router-dom'
import { appRouteSpecs } from '@/app/migration-routes'
import { appPageLoaders } from '@/app/page-loaders'
import { RouteLoadingFallback } from '@/app/route-loading-fallback'
import { RouterRuntimeBridge } from '@/app/router-runtime-bridge'
import { RouteAccessGate } from '@/auth'
import { AppShell } from '@/layout'
import { AuthCallbackPage } from '@/pages/auth-callback-page'
import { ForbiddenPage } from '@/pages/forbidden-page'
import { IframePage } from '@/pages/iframe-page'
import { LoginPage } from '@/pages/login-page'
import { NotFoundPage } from '@/pages/not-found-page'
import { RecruitLandingPage } from '@/pages/recruit-landing-page'
import { ServerErrorPage } from '@/pages/server-error-page'

const appShellChildren: RouteObject[] = appRouteSpecs.map((route) => ({
  path: route.path,
  element: <RouteAccessGate meta={route.meta} />,
  children: [
    {
      index: true,
      lazy: appPageLoaders[route.pageType],
      hydrateFallbackElement: <RouteLoadingFallback />,
    },
  ],
}))

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
        path: '/r/:code',
        element: <RecruitLandingPage />,
      },
      {
        path: '/outside/iframe/*',
        element: <IframePage />,
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
          <RouteAccessGate meta={{ jwt: true }}>
            <AppShell />
          </RouteAccessGate>
        ),
        children: [
          {
            index: true,
            element: <Navigate to="/dashboard/console" replace />,
          },
          ...appShellChildren,
        ],
      },
      {
        path: '*',
        element: <NotFoundPage />,
      },
    ],
  },
]

export const router = createHashRouter(appRoutes)
