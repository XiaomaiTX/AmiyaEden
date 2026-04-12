import { ApiStatus } from '@/utils/http/status'

export interface LoadFuxiAdminDirectoryStateResult {
  directory: Api.FuxiAdmin.DirectoryResponse | null
  loadErrorMessage: string | null
  showErrorToast: boolean
}

function isUnauthorizedHttpError(error: unknown): boolean {
  return (
    typeof error === 'object' &&
    error !== null &&
    'code' in error &&
    (error as { code?: unknown }).code === ApiStatus.unauthorized
  )
}

export async function loadFuxiAdminDirectoryState(
  loadDirectory: () => Promise<Api.FuxiAdmin.DirectoryResponse>,
  loadFailedMessage: string
): Promise<LoadFuxiAdminDirectoryStateResult> {
  try {
    return {
      directory: await loadDirectory(),
      loadErrorMessage: null,
      showErrorToast: false
    }
  } catch (error) {
    return {
      directory: null,
      loadErrorMessage: loadFailedMessage,
      showErrorToast: !isUnauthorizedHttpError(error)
    }
  }
}
