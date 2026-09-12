export interface MumbleCredentialStatus {
  created: boolean
  enabled: boolean
  server_address?: string
  server_port?: number
  credential_version?: number
  password_rotated_at?: string | null
}

export interface MumbleCredentialResult {
  credential: MumbleCredentialStatus
  password: string
}
