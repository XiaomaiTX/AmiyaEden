/**
 * HTTP 请求封装模块
 * 基于 Axios 封装的 HTTP 请求工具，提供统一的请求/响应处理
 *
 * ## 主要功能
 *
 * - 请求/响应拦截器（自动添加 Token、统一错误处理）
 * - 401 未授权自动登出（带防抖机制）
 * - 统一的错误消息提示，保留契约响应信封或二进制响应
 *
 * @module utils/http
 * @author Art Design Pro Team
 */

import axios, { AxiosRequestConfig, AxiosResponse, InternalAxiosRequestConfig } from 'axios'
import { useUserStore } from '@/store/modules/user'
import { ApiStatus } from './status'
import { HttpError, handleError, showError } from './error'
import { decodeBlobResponse } from './response-body'
import { $t } from '@/locales'
import { BaseResponse } from '@/types'

/** 请求配置常量 */
const REQUEST_TIMEOUT = 15000
const LOGOUT_DELAY = 500
const UNAUTHORIZED_DEBOUNCE_TIME = 3000

/** 401防抖状态 */
let isUnauthorizedErrorShown = false
let unauthorizedTimer: NodeJS.Timeout | null = null

/** 扩展 AxiosRequestConfig */
interface ContractAxiosRequestConfig extends AxiosRequestConfig {
  showErrorMessage?: boolean
  sensitive?: boolean
}

const { VITE_API_URL, VITE_WITH_CREDENTIALS } = import.meta.env

/** Axios实例 */
const axiosInstance = axios.create({
  timeout: REQUEST_TIMEOUT,
  baseURL: VITE_API_URL,
  withCredentials: VITE_WITH_CREDENTIALS === 'true',
  validateStatus: (status) => status >= 200 && status < 300,
  transformResponse: [
    (data, headers) => {
      const contentType = headers['content-type']
      if (typeof contentType === 'string' && contentType.includes('application/json')) {
        try {
          return JSON.parse(data)
        } catch {
          return data
        }
      }
      return data
    }
  ]
})

/** 请求拦截器 */
axiosInstance.interceptors.request.use(
  (request: InternalAxiosRequestConfig) => {
    const { accessToken } = useUserStore()
    if (accessToken) request.headers.set('Authorization', `Bearer ${accessToken}`)

    if (request.data && !(request.data instanceof FormData) && !request.headers['Content-Type']) {
      request.headers.set('Content-Type', 'application/json')
      request.data = JSON.stringify(request.data)
    }

    return request
  },
  (error) => {
    showError(createHttpError($t('httpMsg.requestConfigError'), ApiStatus.error))
    return Promise.reject(error)
  }
)

/** 响应拦截器 */
axiosInstance.interceptors.response.use(
  async (response: AxiosResponse<BaseResponse>) => {
    if (response.config.responseType === 'blob' && response.data instanceof Blob) {
      const body = decodeBlobResponse(response.data)
      if (body instanceof Blob) return response
      response.data = (await body) as BaseResponse
    }
    const { code, msg, data } = response.data ?? {}
    const config: ContractAxiosRequestConfig = response.config
    if (code === ApiStatus.unauthorized) handleUnauthorizedError(config.sensitive ? undefined : msg)
    if (typeof code === 'number' && code !== ApiStatus.success) {
      // Sensitive requests must not surface raw error details, even if an upstream reflects input.
      if (config.sensitive) throw createHttpError($t('httpMsg.requestFailed'), code)
      showError(
        new HttpError(msg || $t('httpMsg.requestFailed'), code, { data }),
        config.showErrorMessage !== false
      )
    }
    return response
  },
  (error) => {
    const status = error.response?.status
    if (status === ApiStatus.unauthorized) handleUnauthorizedError()
    if (error.config?.sensitive) {
      throw createHttpError($t('httpMsg.requestFailed'), status || ApiStatus.error)
    }
    return Promise.reject(handleError(error))
  }
)

/** 统一创建HttpError */
function createHttpError(message: string, code: number) {
  return new HttpError(message, code)
}

/** 处理401错误（带防抖） */
function handleUnauthorizedError(message?: string): never {
  const error = createHttpError(message || $t('httpMsg.unauthorized'), ApiStatus.unauthorized)

  if (!isUnauthorizedErrorShown) {
    isUnauthorizedErrorShown = true
    logOut()

    unauthorizedTimer = setTimeout(resetUnauthorizedError, UNAUTHORIZED_DEBOUNCE_TIME)

    showError(error, true)
    throw error
  }

  throw error
}

/** 重置401防抖状态 */
function resetUnauthorizedError() {
  isUnauthorizedErrorShown = false
  if (unauthorizedTimer) clearTimeout(unauthorizedTimer)
  unauthorizedTimer = null
}

/** 退出登录函数 */
function logOut() {
  setTimeout(() => {
    useUserStore().logOut()
  }, LOGOUT_DELAY)
}

/** 返回生成客户端声明的契约信封或二进制响应。 */
export async function request<T>(config: ContractAxiosRequestConfig): Promise<T> {
  try {
    const response = await axiosInstance.request<T>(config)
    return response.data
  } catch (error) {
    if (!config.sensitive && error instanceof HttpError && error.code !== ApiStatus.unauthorized) {
      showError(error, config.showErrorMessage !== false)
    }
    throw error
  }
}
