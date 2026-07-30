import { render, screen } from '@testing-library/react'
import { PermissionGate, RoleGate, RoutePermissionProvider } from '@/auth/permission-gates'
import { useSessionStore } from '@/stores'

describe('React permission gates', () => {
  beforeEach(() => {
    useSessionStore.getState().clearSession()
  })

  test('renders only permissions supplied by the active route provider', () => {
    render(
      <RoutePermissionProvider permissions={['edit_exchange_rate']}>
        <PermissionGate permission="edit_exchange_rate">
          <span>allowed</span>
        </PermissionGate>
        <PermissionGate permission="delete_user">
          <span>denied</span>
        </PermissionGate>
      </RoutePermissionProvider>
    )

    expect(screen.getByText('allowed')).toBeInTheDocument()
    expect(screen.queryByText('denied')).not.toBeInTheDocument()
  })

  test('matches any required role from the current session', () => {
    useSessionStore.getState().setSessionSnapshot({ roles: ['captain'] })
    render(
      <RoleGate roles={['admin', 'captain']}>
        <span>captain action</span>
      </RoleGate>
    )

    expect(screen.getByText('captain action')).toBeInTheDocument()
  })
})
