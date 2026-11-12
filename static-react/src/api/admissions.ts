import { requestJson, requestBlob } from '@/api/http-client'
import { assertSuccess, type ApiResponse } from '@/api/response'
import type {
  AdmissionApplication,
  AdmissionContext,
  AdmissionDetail,
  AdmissionList,
  AdmissionWriteInput,
  EvidenceKind,
} from '@/types/api/admissions'

const base = '/api/v1/admissions'
export async function getAdmissionContext() {
  const context = assertSuccess(
    await requestJson<ApiResponse<AdmissionContext>>(`${base}/context`),
    'admissions.errors.internal'
  )!
  if (
    !context ||
    !Array.isArray(context.characters) ||
    !Array.isArray(context.targets) ||
    !Array.isArray(context.reviewable_corporation_ids)
  )
    throw new Error('admissions.errors.internal')
  return context
}
export async function listAdmissions(
  reviewer: boolean,
  page = 1,
  status = '',
  target = '',
  kind = ''
) {
  const params = new URLSearchParams({
    current: String(page),
    size: '20',
    status,
    kind,
    ...(target ? { target_corporation_id: target } : {}),
  })
  const path = reviewer ? '/api/v1/system/admissions/applications' : `${base}/applications/me`
  return assertSuccess(
    await requestJson<ApiResponse<AdmissionList>>(`${path}?${params}`),
    'admissions.errors.internal'
  )!
}
export async function getAdmission(id: number) {
  return assertSuccess(
    await requestJson<ApiResponse<AdmissionDetail>>(`${base}/applications/${id}`),
    'admissions.errors.internal'
  )!
}
export function getAdmissionEvidence(id: number, kind: EvidenceKind) {
  return requestBlob(`${base}/applications/${id}/evidence/${kind}`)
}
export async function saveAdmission(
  input: AdmissionWriteInput,
  files: Partial<Record<EvidenceKind, Blob>>,
  id?: number,
  resubmit = false
) {
  const data = new FormData()
  data.append('payload', JSON.stringify(input))
  for (const [kind, blob] of Object.entries(files)) data.append(kind, blob, `${kind}.jpg`)
  const path = `${base}/applications${id ? `/${id}${resubmit ? '/resubmit' : ''}` : ''}`
  return assertSuccess(
    await requestJson<ApiResponse<AdmissionApplication>>(path, {
      method: id && !resubmit ? 'PUT' : 'POST',
      body: data,
    }),
    'admissions.errors.internal'
  )!
}
export async function processAdmission(
  id: number,
  expected_version: number,
  action: 'invited' | 'reject',
  reason = ''
) {
  return assertSuccess(
    await requestJson<ApiResponse<AdmissionApplication>>(
      `/api/v1/system/admissions/applications/${id}/${action}`,
      { method: 'POST', body: JSON.stringify({ expected_version, reason }) }
    ),
    'admissions.errors.internal'
  )!
}
