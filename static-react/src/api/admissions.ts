import * as api from './generated/client'
import type {
  AdmissionWriteInput,
  AdmissionEvidenceKind as EvidenceKind,
} from '@/api/generated/client'
import { unwrapContract } from './response'

function unwrap<T>(request: Parameters<typeof unwrapContract<T>>[0]) {
  return unwrapContract(request, 'admissions.errors.internal')
}

export async function getAdmissionEvidence(id: number, kind: EvidenceKind): Promise<Blob> {
  const result = await api.getAdmissionEvidence(id, kind)
  if ('code' in result) throw new Error(result.msg || 'admissions.errors.evidence')
  if (result.type !== 'image/jpeg') throw new Error('admissions.errors.evidence')
  return result
}

export function saveAdmission(
  input: AdmissionWriteInput,
  files: Partial<Record<EvidenceKind, Blob>>,
  id?: number,
  resubmit = false
) {
  const data: api.AdmissionMultipartInput = { payload: JSON.stringify(input) }
  if (files.launcher) {
    data.launcher = new File([files.launcher], 'launcher.jpg', { type: files.launcher.type })
  }
  if (files.seat_profile) {
    data.seat_profile = new File([files.seat_profile], 'seat_profile.jpg', {
      type: files.seat_profile.type,
    })
  }
  return unwrap(
    id
      ? resubmit
        ? api.resubmitAdmission(id, data)
        : api.editAdmission(id, data)
      : api.createAdmission(data)
  )
}
