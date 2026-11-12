import type { AdmissionContext, AdmissionKind } from '@/types/api/admissions'

export function canAccessAdmission(
  context: AdmissionContext | null,
  kind: AdmissionKind | 'review' | 'history'
) {
  if (kind === 'history') return true
  if (!context) return false
  if (kind === 'review')
    return (
      Array.isArray(context.reviewable_corporation_ids) &&
      context.reviewable_corporation_ids.length > 0
    )
  return (kind === 'new_member' ? context.can_create_new_member : context.can_create_alt) === true
}

export async function compressAdmissionImage(file: File): Promise<Blob> {
  if (!['image/jpeg', 'image/png'].includes(file.type) || file.size > 20 * 1024 * 1024)
    throw new Error('admissions.errors.evidence')
  const bitmap = await createImageBitmap(file)
  try {
    if (bitmap.width * bitmap.height > 24000000) throw new Error('admissions.errors.evidence')
    const scale = Math.min(1, 1280 / Math.max(bitmap.width, bitmap.height))
    const canvas = document.createElement('canvas')
    canvas.width = Math.max(1, Math.round(bitmap.width * scale))
    canvas.height = Math.max(1, Math.round(bitmap.height * scale))
    const ctx = canvas.getContext('2d')
    if (!ctx) throw new Error('admissions.errors.evidence')
    ctx.fillStyle = '#fff'
    ctx.fillRect(0, 0, canvas.width, canvas.height)
    ctx.drawImage(bitmap, 0, 0, canvas.width, canvas.height)
    const blob = await new Promise<Blob>((resolve, reject) =>
      canvas.toBlob(
        (value) => (value ? resolve(value) : reject(new Error('admissions.errors.evidence'))),
        'image/jpeg',
        0.7
      )
    )
    if (blob.size > 2 * 1024 * 1024) throw new Error('admissions.errors.evidence')
    return blob
  } finally {
    bitmap.close()
  }
}
