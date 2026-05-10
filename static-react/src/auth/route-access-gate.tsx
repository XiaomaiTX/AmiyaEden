import type { PropsWithChildren } from 'react'
import { useEffect } from 'react'
import { Navigate, useLocation } from 'react-router-dom'
import type { RouteAccessMeta } from '@/app/route-access'
import { hasRouteRolePermission } from '@/app/route-access'
import { useSessionStore } from '@/stores'

interface RouteAccessGateProps extends PropsWithChildren {
  meta?: RouteAccessMeta
}

export function RouteAccessGate({ meta, children }: RouteAccessGateProps) {
  const location = useLocation()
  const isLoggedIn = useSessionStore((state) => state.isLoggedIn)
  const roles = useSessionStore((state) => state.roles)
  const isCurrentlyNewbro = useSessionStore((state) => state.isCurrentlyNewbro)
  const isMentorMenteeEligible = useSessionStore((state) => state.isMentorMenteeEligible)
  const setRouteAuthList = useSessionStore((state) => state.setRouteAuthList)

  useEffect(() => {
    if (meta?.authList === undefined) {
      return
    }

    const authMarks = meta.authList.map((item) => item.authMark)
    setRouteAuthList(authMarks)
  }, [meta?.authList, setRouteAuthList])

  if (meta?.login && !isLoggedIn) {
    const redirect = `${location.pathname}${location.search}${location.hash}`
    return <Navigate to={`/auth/login?redirect=${encodeURIComponent(redirect)}`} replace />
  }

  if (!hasRouteRolePermission(roles, meta?.roles)) {
    return <Navigate to="/403" replace />
  }

  if (meta?.requiresNewbro && !isCurrentlyNewbro) {
    return <Navigate to="/403" replace />
  }

  if (meta?.requiresMentorMenteeEligibility && !isMentorMenteeEligible) {
    return <Navigate to="/403" replace />
  }

  return children
}

