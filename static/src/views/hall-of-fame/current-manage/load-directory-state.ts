import { ApiStatus } from '@/utils/http/status'

type FuxiAdminDirectoryState =
  | Api.FuxiAdmin.DirectoryResponse
  | Api.FuxiAdmin.ManageDirectoryResponse

export interface LoadFuxiAdminDirectoryStateResult {
  directory: FuxiAdminDirectoryState | null
  loadErrorMessage: string | null
  showErrorToast: boolean
  isAuthDenied: boolean
}

export interface ResolveManageAccessOptions {
  hadAccess: boolean
  hasRole: boolean
  gotDirectory: boolean
  isAuthDenied: boolean
}

export interface LoadFuxiAdminPageDirectoryOptions {
  hadManageAccess: boolean
  hasEditRole: boolean
  loadFailedMessage: string
  loadPublicDirectory: () => Promise<FuxiAdminDirectoryState>
  loadManageDirectory: () => Promise<Api.FuxiAdmin.ManageDirectoryResponse>
}

export interface LoadFuxiAdminPageDirectoryResult extends LoadFuxiAdminDirectoryStateResult {
  hasManageAccess: boolean
}

function isAuthDeniedHttpError(error: unknown): boolean {
  const code =
    typeof error === 'object' && error !== null && 'code' in error
      ? (error as { code?: unknown }).code
      : undefined

  return code === ApiStatus.unauthorized || code === ApiStatus.forbidden
}

export async function loadFuxiAdminDirectoryState(
  loadDirectory: () => Promise<FuxiAdminDirectoryState>,
  loadFailedMessage: string
): Promise<LoadFuxiAdminDirectoryStateResult> {
  try {
    return {
      directory: await loadDirectory(),
      loadErrorMessage: null,
      showErrorToast: false,
      isAuthDenied: false
    }
  } catch (error) {
    const isAuthDenied = isAuthDeniedHttpError(error)
    return {
      directory: null,
      loadErrorMessage: loadFailedMessage,
      showErrorToast: !isAuthDenied,
      isAuthDenied
    }
  }
}

export function resolveManageAccess({
  hadAccess,
  hasRole,
  gotDirectory,
  isAuthDenied
}: ResolveManageAccessOptions): boolean {
  if (!hasRole) {
    return false
  }
  if (gotDirectory) {
    return true
  }
  if (isAuthDenied) {
    return false
  }
  return hadAccess
}

export async function loadFuxiAdminPageDirectory({
  hadManageAccess,
  hasEditRole,
  loadFailedMessage,
  loadPublicDirectory,
  loadManageDirectory
}: LoadFuxiAdminPageDirectoryOptions): Promise<LoadFuxiAdminPageDirectoryResult> {
  const initialResult = await loadFuxiAdminDirectoryState(
    hasEditRole ? loadManageDirectory : loadPublicDirectory,
    loadFailedMessage
  )

  if (hasEditRole && !initialResult.directory && initialResult.isAuthDenied) {
    const fallbackResult = await loadFuxiAdminDirectoryState(loadPublicDirectory, loadFailedMessage)
    return {
      ...fallbackResult,
      hasManageAccess: false
    }
  }

  return {
    ...initialResult,
    hasManageAccess: resolveManageAccess({
      hadAccess: hadManageAccess,
      hasRole: hasEditRole,
      gotDirectory: initialResult.directory !== null,
      isAuthDenied: initialResult.isAuthDenied
    })
  }
}
