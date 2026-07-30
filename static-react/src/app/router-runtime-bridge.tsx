import { Outlet } from 'react-router-dom'
import { UnauthorizedBridge } from '@/auth'
import { FeedbackHost } from '@/feedback'
import { useGalaxyRegistryTimeoutNotification } from '@/hooks/use-galaxy-registry-timeout-notification'

export function RouterRuntimeBridge() {
  useGalaxyRegistryTimeoutNotification()
  return (
    <>
      <UnauthorizedBridge />
      <Outlet />
      <FeedbackHost />
    </>
  )
}
