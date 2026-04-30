import { createHashRouter, type RouteObject } from 'react-router-dom'
import { RouteAccessGate } from '@/auth'
import { RouterRuntimeBridge } from '@/app/router-runtime-bridge'
import { AppShell } from '@/layout'
import { AdminDemoPage } from '@/pages/admin-demo-page'
import { ForbiddenPage } from '@/pages/forbidden-page'
import { HomePage } from '@/pages/home-page'
import { LoginPage } from '@/pages/login-page'
import { NotFoundPage } from '@/pages/not-found-page'
import { ServerErrorPage } from '@/pages/server-error-page'

export const appRoutes: RouteObject[] = [
  {
    element: <RouterRuntimeBridge />,
    children: [
      {
        path: '/login',
        element: <LoginPage />,
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
        children: [
          {
            index: true,
            element: (
              <RouteAccessGate meta={{ authList: [] }}>
                <HomePage />
              </RouteAccessGate>
            ),
          },
          {
            path: 'admin-demo',
            element: (
              <RouteAccessGate
                meta={{
                  login: true,
                  roles: ['super_admin', 'admin'],
                  authList: [
                    { title: '审批订单', authMark: 'approve_order' },
                    { title: '编辑兑换率', authMark: 'edit_exchange_rate' },
                  ],
                }}
              >
                <AdminDemoPage />
              </RouteAccessGate>
            ),
          },
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
