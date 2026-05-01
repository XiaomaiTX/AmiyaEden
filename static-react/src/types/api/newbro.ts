export interface DirectReferralStatus {
  show_card: boolean
  needs_profile_qq: boolean
}

export interface DirectReferrerCandidate {
  user_id: number
  nickname: string
  primary_character_id: number
  primary_character_name: string
}

export interface CheckDirectReferrerParams {
  qq: string
}

export interface ConfirmDirectReferrerParams {
  referrer_user_id: number
}

export interface SupportSettings {
  max_character_sp: number
  multi_character_sp: number
  multi_character_threshold: number
  refresh_interval_days: number
  bonus_rate: number
}

export interface UpdateSupportSettingsParams {
  max_character_sp: number
  multi_character_sp: number
  multi_character_threshold: number
  refresh_interval_days: number
  bonus_rate: number
}

export interface RecruitSettings {
  recruit_qq_url: string
  recruit_reward_amount: number
  recruit_cooldown_days: number
}

export interface UpdateRecruitSettingsParams {
  recruit_qq_url: string
  recruit_reward_amount: number
  recruit_cooldown_days: number
}

export interface MentorCandidate {
  mentor_user_id: number
  mentor_character_id: number
  mentor_character_name: string
  mentor_nickname: string
  qq: string
  discord_id: string
  active_mentee_count: number
  last_online_at: string | null
}

export type MentorRelationshipStatus = 'pending' | 'active' | 'rejected' | 'revoked' | 'graduated'

export interface RelationshipView {
  id: number
  mentee_user_id: number
  mentor_user_id: number
  status: MentorRelationshipStatus
  applied_at: string
  responded_at: string | null
  revoked_at: string | null
  graduated_at: string | null
  mentor_character_id: number
  mentor_character_name: string
  mentor_nickname: string
  mentor_qq: string
  mentor_discord_id: string
  mentee_character_id: number
  mentee_character_name: string
  mentee_nickname: string
}

export interface MyStatusResponse {
  is_eligible: boolean
  disqualified_reason: string
  current_relationship: RelationshipView | null
}

export interface MyAffiliationResponse {
  is_currently_newbro: boolean
  current_affiliation: CaptainEligiblePlayerCurrentAffiliation | null
}

export interface ApplyParams {
  mentor_user_id: number
}

export interface ApplyResponse {
  id: number
  mentee_user_id: number
  mentee_primary_character_id_at_start: number
  mentor_user_id: number
  status: MentorRelationshipStatus
  applied_at: string
  responded_at: string | null
  revoked_at: string | null
  revoked_by: number | null
  graduated_at: string | null
  created_at?: string
  updated_at?: string
}

export interface MenteeListItem {
  relationship_id: number
  mentee_user_id: number
  mentee_character_id: number
  mentee_character_name: string
  mentee_nickname: string
  mentee_qq: string
  mentee_discord_id: string
  mentee_total_sp: number
  mentee_total_pap: number
  mentee_days_active: number
  status: MentorRelationshipStatus
  applied_at: string
  responded_at: string | null
  graduated_at: string | null
  distributed_stages: number[]
  distributed_reward_amount: number
}

export type MenteeListResponse = import('@/types/api/common').PaginatedResponse<MenteeListItem>

export interface AffiliationSummary {
  affiliation_id: number
  captain_user_id: number
  captain_character_id: number
  captain_character_name: string
  started_at: string
  ended_at: string | null
}

export type MyAffiliationHistoryResponse = import('@/types/api/common').PaginatedResponse<AffiliationSummary>

export interface RelationshipActionParams {
  relationship_id: number
}

export type EmptyResponse = Record<string, never>
export type EndAffiliationResponse = EmptyResponse

export type CaptainPlayerStatus = 'all' | 'active' | 'historical'

export type MenteeStatusFilter = 'active' | 'pending' | 'rejected' | 'revoked' | 'graduated' | 'all'

export type MentorMenteesParams = Partial<{
  current: number
  size: number
  status: MenteeStatusFilter
}>

export type AdminRelationshipsParams = Partial<{
  current: number
  size: number
  status: MenteeStatusFilter
  keyword: string
}>

export type AdminRelationshipsResponse = import('@/types/api/common').PaginatedResponse<RelationshipView>

export interface RewardDistributionView {
  id: number
  relationship_id: number
  stage_id: number
  stage_order: number
  mentor_user_id: number
  mentor_character_name: string
  mentor_nickname: string
  mentee_user_id: number
  mentee_character_name: string
  mentee_nickname: string
  reward_amount: number
  distributed_at: string
  wallet_ref_id: string
}

export type AdminRewardDistributionsParams = Partial<{
  current: number
  size: number
  keyword: string
}>

export type AdminRewardDistributionsResponse = import('@/types/api/common').PaginatedResponse<RewardDistributionView>

export interface MentorSettings {
  max_character_sp: number
  max_account_age_days: number
}

export interface UpdateMentorSettingsParams {
  max_character_sp: number
  max_account_age_days: number
}

export type RewardConditionType = 'skill_points' | 'pap_count' | 'days_active'

export interface RewardStage {
  id: number
  stage_order: number
  name: string
  condition_type: RewardConditionType
  threshold: number
  reward_amount: number
  created_at?: string
  updated_at?: string
}

export interface RewardStageInput {
  stage_order: number
  name: string
  condition_type: RewardConditionType
  threshold: number
  reward_amount: number
}

export interface UpdateRewardStagesParams {
  stages: RewardStageInput[]
}

export interface RewardProcessResult {
  processed_relationships: number
  rewards_distributed: number
  total_coin_awarded: number
  graduated_count: number
}

export interface CaptainCandidate {
  captain_user_id: number
  captain_character_id: number
  captain_character_name: string
  captain_nickname: string
  active_newbro_count: number
  last_online_at: string | null
}

export interface CaptainOverview {
  captain_user_id: number
  captain_character_id: number
  captain_character_name: string
  captain_nickname: string
  active_player_count: number
  historical_player_count: number
  attributed_bounty_total: number
  attribution_record_count: number
}

export interface CaptainPlayerListItem {
  player_user_id: number
  player_character_id: number
  player_character_name: string
  player_nickname: string
  started_at: string
  ended_at: string | null
  attributed_bounty_total: number
}

export type CaptainPlayersResponse = import('@/types/api/common').PaginatedResponse<CaptainPlayerListItem>

export interface CaptainEligiblePlayerCurrentAffiliation {
  affiliation_id: number
  captain_user_id: number
  captain_character_id: number
  captain_character_name: string
  captain_nickname: string
  started_at: string
}

export interface CaptainEligiblePlayerListItem {
  player_user_id: number
  player_character_id: number
  player_character_name: string
  player_nickname: string
  current_affiliation: CaptainEligiblePlayerCurrentAffiliation | null
}

export type CaptainEligiblePlayersResponse =
  import('@/types/api/common').PaginatedResponse<CaptainEligiblePlayerListItem>

export interface CaptainAttributionItem {
  id: number
  player_user_id: number
  player_character_id: number
  player_character_name: string
  captain_character_id: number
  captain_character_name: string
  captain_wallet_journal_id: number
  wallet_journal_id: number
  ref_type: string
  system_id: number
  journal_at: string
  amount: number
  processed_at: string | null
}

export interface CaptainAttributionSummary {
  attributed_bounty_total: number
  record_count: number
}

export interface CaptainAttributionsResponse {
  summary: CaptainAttributionSummary
  list: CaptainAttributionItem[]
  total: number
  page: number
  page_size: number
}

export interface CaptainRewardSettlementItem {
  id: number
  captain_user_id: number
  captain_character_id: number
  captain_character_name: string
  captain_nickname: string
  attribution_count: number
  attributed_isk_total: number
  bonus_rate: number
  credited_value: number
  processed_at: string
}

export interface CaptainRewardSummary {
  settlement_count: number
  total_credited_value: number
  last_processed_at: string | null
}

export interface CaptainRewardSettlementsResponse {
  summary: CaptainRewardSummary
  list: CaptainRewardSettlementItem[]
  total: number
  page: number
  page_size: number
}

export type CaptainPlayersParams = Partial<{
  current: number
  size: number
  status: CaptainPlayerStatus
}>

export type CaptainEligiblePlayersParams = Partial<{
  current: number
  size: number
  keyword: string
}>

export interface CaptainEnrollPlayerParams {
  player_user_id: number
}

export interface CaptainEndAffiliationParams {
  player_user_id: number
}

export type CaptainAttributionsParams = Partial<{
  current: number
  size: number
  player_user_id: number
  ref_type: string
  start_date: string
  end_date: string
}>

export type CaptainRewardSettlementsParams = Partial<{
  current: number
  size: number
  keyword: string
}>

export type AdminCaptainsParams = Partial<{
  current: number
  size: number
  keyword: string
}>

export type AdminCaptainsResponse = import('@/types/api/common').PaginatedResponse<CaptainOverview>

export interface AdminCaptainDetail {
  overview: CaptainOverview
  players: CaptainPlayerListItem[]
  players_total: number
  attributions: CaptainAttributionItem[]
  attributions_total: number
  attribution_summary: CaptainAttributionSummary
}

export type AdminAffiliationHistoryParams = Partial<{
  current: number
  size: number
  captain_search: string
  player_search: string
  change_start_date: string
  change_end_date: string
}>

export interface AdminAffiliationHistoryItem {
  affiliation_id: number
  player_user_id: number
  player_character_id: number
  player_character_name: string
  player_nickname: string
  captain_user_id: number
  captain_character_id: number
  captain_character_name: string
  captain_nickname: string
  changed_by_character_name: string
  started_at: string
  ended_at: string | null
  created_at: string
}

export type AdminAffiliationHistoryResponse =
  import('@/types/api/common').PaginatedResponse<AdminAffiliationHistoryItem>

export interface RecruitEntry {
  id: number
  qq: string
  entered_at: string
  source: 'link' | 'direct_referral'
  status: 'ongoing' | 'valid' | 'stalled'
  matched_user_id: number
  rewarded_at: string | null
}

export interface RecruitLink {
  id: number
  code: string
  source: 'link' | 'direct_referral'
  generated_at: string
  entries: RecruitEntry[]
}

export interface AdminRecruitLink extends RecruitLink {
  user_id: number
}

export type AdminRecruitLinksParams = Partial<{
  current: number
  size: number
}>

export type AdminRecruitLinksResponse = import('@/types/api/common').PaginatedResponse<AdminRecruitLink>

export interface GenerateLinkResponse {
  id: number
  code: string
  generated_at: string
  is_new: boolean
}

export interface SubmitQQRequest {
  qq: string
}

export interface SubmitQQResponse {
  qq_url: string
}
