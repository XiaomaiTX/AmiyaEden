import { Outlet } from 'react-router-dom'
import { UnauthorizedBridge } from '@/auth'
import { FeedbackHost } from '@/feedback'

export function RouterRuntimeBridge() {
  return (
    <>
      <UnauthorizedBridge />
      <Outlet />
      <FeedbackHost />
    </>
  )
}
