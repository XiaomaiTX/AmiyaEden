import { $t } from '@/locales'
import { isHttpError } from '@/utils/http/error'

type ErrorPayload = {
  msg?: unknown
}

type ErrorWithResponse = {
  response?: {
    data?: ErrorPayload
  }
}

function readMsgField(data: unknown): string {
  if (!data || typeof data !== 'object') {
    return ''
  }
  const payload = data as ErrorPayload
  return typeof payload.msg === 'string' ? payload.msg.trim() : ''
}

export function resolveTaskLoadErrorMessage(error: unknown): string {
  if (isHttpError(error) && typeof error.message === 'string' && error.message.trim()) {
    const HttpError = error
    return HttpError.message.trim()
  }

  const axiosMsg = readMsgField((error as ErrorWithResponse | undefined)?.response?.data)
  if (axiosMsg) {
    return axiosMsg
  }

  if (error instanceof Error && typeof error.message === 'string' && error.message.trim()) {
    return error.message.trim()
  }

  return $t('httpMsg.networkError')
}
