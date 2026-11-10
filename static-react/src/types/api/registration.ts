export interface RegistrationInfo {
  character_id: number
  character_name: string
}

export interface RegistrationInput {
  nickname: string
  qq: string
  discord_id: string
}

export interface ContactConflict {
  contact: 'qq' | 'discord_id'
  primary_character_id: number
  primary_character_name: string
}
