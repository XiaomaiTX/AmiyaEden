export interface MumbleCredentialStatus {
  created: boolean
  enabled: boolean
  stable_user_id?: number
  credential_version?: number
  password_rotated_at?: string | null
}

export interface MumbleCredentialResult {
  credential: MumbleCredentialStatus
  password: string
}
