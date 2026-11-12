import request from '@/utils/http'
import type {
  AdmissionApplication,
  AdmissionContext,
  AdmissionDetail,
  AdmissionList,
  AdmissionWriteInput,
  EvidenceKind
} from '@/types/api/admissions'

const base = '/api/v1/admissions'
const options = { showErrorMessage: false }
export function getAdmissionContext() {
  return request.get<AdmissionContext>({ url: `${base}/context`, ...options })
}
export function listAdmissions(reviewer: boolean, page = 1, status = '', target = '', kind = '') {
  return request.get<AdmissionList>({
    url: reviewer ? '/api/v1/system/admissions/applications' : `${base}/applications/me`,
    params: {
      current: page,
      size: 20,
      status,
      kind,
      ...(target ? { target_corporation_id: target } : {})
    },
    ...options
  })
}
export function getAdmission(id: number) {
  return request.get<AdmissionDetail>({ url: `${base}/applications/${id}`, ...options })
}
export function getAdmissionEvidence(id: number, kind: EvidenceKind) {
  return request.get<Blob>({
    url: `${base}/applications/${id}/evidence/${kind}`,
    responseType: 'blob',
    ...options
  })
}
export function saveAdmission(
  input: AdmissionWriteInput,
  files: Partial<Record<EvidenceKind, Blob>>,
  id?: number,
  resubmit = false
) {
  const data = new FormData()
  data.append('payload', JSON.stringify(input))
  for (const [kind, blob] of Object.entries(files)) data.append(kind, blob, `${kind}.jpg`)
  return request.request<AdmissionApplication>({
    url: `${base}/applications${id ? `/${id}${resubmit ? '/resubmit' : ''}` : ''}`,
    method: id && !resubmit ? 'PUT' : 'POST',
    data,
    ...options
  })
}
export function processAdmission(
  id: number,
  expected_version: number,
  action: 'invited' | 'reject',
  reason = ''
) {
  return request.post<AdmissionApplication>({
    url: `/api/v1/system/admissions/applications/${id}/${action}`,
    data: { expected_version, reason },
    ...options
  })
}
