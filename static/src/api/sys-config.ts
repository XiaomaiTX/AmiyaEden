import request from '@/utils/http'

export function fetchBasicConfig() {
  return request.get<Api.SysConfig.BasicConfig>({
    url: '/api/v1/system/basic-config'
  })
}

export function updateBasicConfig(data: Api.SysConfig.UpdateBasicConfigParams) {
  return request.put({
    url: '/api/v1/system/basic-config',
    data
  })
}
