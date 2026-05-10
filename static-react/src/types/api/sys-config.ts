export interface BasicConfig {
  corp_id: number
  site_title: string
}

export interface AllowCorporationsConfig {
  allow_corporations: number[]
}

export interface UpdateAllowCorporationsParams {
  allow_corporations: number[]
}

export interface CharacterESIRestrictionConfig {
  enforce_character_esi_restriction: boolean
}

export interface UpdateCharacterESIRestrictionParams {
  enforce_character_esi_restriction: boolean
}

export interface SDEConfig {
  api_key: string
  proxy: string
  download_url: string
}

export interface UpdateSDEConfigParams {
  api_key?: string
  proxy?: string
  download_url?: string
}
