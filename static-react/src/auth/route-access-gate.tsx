import type { PropsWithChildren } from 'react'
import { Navigate, Outlet, useLocation } from 'react-router-dom'
import type { RouteAccessMeta } from '@/app/route-access'
import { evaluateRouteAccess } from '@/app/route-access'
import { RouteLoadingFallback } from '@/app/route-loading-fallback'
import { getProfileLockReasons } from '@/auth/profile-lock'
import { RoutePermissionProvider } from '@/auth/permission-gates'
import { useSessionStore } from '@/stores'

interface RouteAccessGateProps extends PropsWithChildren {
  meta?: RouteAccessMeta
}

export function RouteAccessGate({ meta, children }: RouteAccessGateProps) {
  const location = useLocation()
  const isLoggedIn = useSessionStore((state) => state.isLoggedIn)
  const bootstrapRequired = useSessionStore((state) => state.bootstrapRequired)
  const roles = useSessionStore((state) => state.roles)
  const corpCapabilities = useSessionStore((state) => state.corpCapabilities)
  const isCurrentlyNewbro = useSessionStore((state) => state.isCurrentlyNewbro)
  const isMentorMenteeEligible = useSessionStore((state) => state.isMentorMenteeEligible)
  const profileComplete = useSessionStore((state) => state.profileComplete)
  const enforceCharacterESIRestriction = useSessionStore(
    (state) => state.enforceCharacterESIRestriction
  )
  const primaryCharacterId = useSessionStore((state) => state.primaryCharacterId)
  const characters = useSessionStore((state) => state.characters)

  if (bootstrapRequired) {
    return <RouteLoadingFallback />
  }

  const decision = evaluateRouteAccess(meta, {
    isLoggedIn,
    roles,
    corpCapabilities,
    isCurrentlyNewbro,
    isMentorMenteeEligible,
  })

  if (decision === 'login') {
    const redirect = `${location.pathname}${location.search}${location.hash}`
    return <Navigate to={`/auth/login?redirect=${encodeURIComponent(redirect)}`} replace />
  }

  if (decision === 'forbidden') {
    return <Navigate to="/403" replace />
  }

  if (
    isLoggedIn &&
    location.pathname !== '/characters'
  ) {
    const lockReasons = getProfileLockReasons({
      profileComplete,
      enforceCharacterESIRestriction,
      primaryCharacterId,
      characters,
    })
    if (lockReasons.length > 0) {
      return <Navigate to="/characters" replace state={{ profileLockReasons: lockReasons }} />
    }
  }

  const content = children ?? <Outlet />
  return (
    <RoutePermissionProvider permissions={meta?.authList?.map((item) => item.authMark) ?? []}>
      {content}
    </RoutePermissionProvider>
  )
}

