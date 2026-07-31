import { RouterProvider as AriaRouterProvider } from 'react-aria-components'
import { Outlet, useNavigate } from 'react-router-dom'
import { UnauthorizedBridge } from '@/auth'
import { FeedbackHost } from '@/feedback'
import { useGalaxyRegistryTimeoutNotification } from '@/hooks/use-galaxy-registry-timeout-notification'

export function RouterRuntimeBridge() {
  useGalaxyRegistryTimeoutNotification()
  return (
    <AriaRouterBridge>
      <UnauthorizedBridge />
      <Outlet />
      <FeedbackHost />
    </AriaRouterBridge>
  )
}

function AriaRouterBridge({ children }: { children: React.ReactNode }) {
  const navigate = useNavigate()
  return (
    <AriaRouterProvider
      useHref={(target) => `#${target}`}
      navigate={(target) => navigate(target)}
    >
      {children}
    </AriaRouterProvider>
  )
}
