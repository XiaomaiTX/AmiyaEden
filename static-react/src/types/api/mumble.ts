export interface MumblePublicNode {
  name: string
  description: string
  address: string
  port: number
}

export interface MumbleCredentialStatus {
  created: boolean
  enabled: boolean
  servers?: MumblePublicNode[]
  credential_version?: number
  password_rotated_at?: string | null
}

export interface MumbleCredentialResult {
  credential: MumbleCredentialStatus
  password: string
}
