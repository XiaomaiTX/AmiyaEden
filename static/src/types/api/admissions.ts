export type AdmissionKind = 'new_member' | 'alt'
export type AdmissionStatus = 'pending' | 'invited' | 'rejected'
export type EvidenceKind = 'launcher' | 'seat_profile'
export interface AdmissionCharacter {
  character_id: number
  character_name: string
}
export interface AdmissionContext {
  is_member: boolean
  can_create_new_member: boolean
  can_create_alt: boolean
  qq: string
  discord_id: string
  targets: { corporation_id: number; name: string }[]
  characters: (AdmissionCharacter & { corporation_id: number })[]
  reviewable_corporation_ids: number[]
}
export interface AdmissionApplication {
  id: number
  user_id: number
  kind: AdmissionKind
  target_corporation_id: number
  status: AdmissionStatus
  version: number
  applicant_name: string
  primary_character_id: number
  qq: string
  discord_id: string
  submitted_at: string
  handled_by: number | null
  handler_name: string
  closed_at: string | null
  rejection_reason: string
  evidence_expires_at: string | null
  characters: AdmissionCharacter[]
}
export interface AdmissionWriteInput {
  kind: AdmissionKind
  target_corporation_id: number
  character_ids: number[]
  expected_version: number
}
export interface AdmissionEvidence {
  application_id: number
  kind: EvidenceKind
  width: number
  height: number
  byte_size: number
  updated_at: string
}
export interface AdmissionEvent {
  id: number
  application_id: number
  version: number
  actor_id: number
  actor_name: string
  action: string
  from_status: string
  to_status: AdmissionStatus
  reason: string
  snapshot: string
  created_at: string
}
export interface AdmissionSalary {
  application_id: number
  user_id: number
  amount: number
  wallet_transaction_id: number
  created_at: string
}
export interface AdmissionDetail {
  application: AdmissionApplication
  evidence: AdmissionEvidence[]
  events: AdmissionEvent[]
  salary: AdmissionSalary | null
  can_review: boolean
  is_owner: boolean
  /** Only for a pending application the reader can review; empty otherwise. */
  blacklist_hits: AdmissionBlacklistHit[]
}
export interface AdmissionBlacklistHit {
  kind: Api.Blacklist.Kind
  value: string
  display: string
  reason: string
}
/** `blacklisted` is set only on pending rows of the review list. */
export interface AdmissionListItem extends AdmissionApplication {
  blacklisted: boolean
}
export interface AdmissionList {
  list: AdmissionListItem[]
  total: number
  page: number
  pageSize: number
}
