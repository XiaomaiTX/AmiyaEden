import { HttpError } from '@/utils/http/error'
import { ApiStatus } from '@/utils/http/status'

export interface LoadFuxiAdminDirectoryStateResult {
  directory: Api.FuxiAdmin.DirectoryResponse | null
  loadErrorMessage: string | null
  showErrorToast: boolean
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
      showErrorToast: !(error instanceof HttpError && error.code === ApiStatus.unauthorized)
    }
  }
}
