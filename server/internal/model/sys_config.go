package model

import "time"

// SystemConfig 系统配置（key/value 键值对）
type SystemConfig struct {
	Key       string    `gorm:"primarykey;size:128"    json:"key"`
	Value     string    `gorm:"type:text;not null"     json:"value"`
	Desc      string    `gorm:"size:256"               json:"desc"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"         json:"updated_at"`
}

func (SystemConfig) TableName() string { return "system_config" }

// ─── 已知配置 Key ───

const (
	SysConfigPAPWalletPerPAP    = "pap.wallet_per_pap"   // 每 1 PAP 兑换多少伏羲币（float）
	SysConfigPAPExchangeEnabled = "pap.exchange_enabled" // PAP 兑换是否开启（bool）
	SysConfigPAPFCSalary        = "pap.fc_salary"        // FC 工资（float）
	SysConfigPAPFCSalaryLimit   = "pap.fc_salary_limit"  // FC 工资每月上限次数（int）
	SysConfigPAPAdminAward      = "pap.admin_award"      // 管理发放奖励（int）

	SysConfigWebhookURL           = "webhook.url"            // Webhook URL
	SysConfigWebhookEnabled       = "webhook.enabled"        // 是否启用（bool）
	SysConfigWebhookType          = "webhook.type"           // discord | feishu | dingtalk | onebot
	SysConfigWebhookFleetTemplate = "webhook.fleet_template" // 舰队行动通知模板

	SysConfigWebhookOBTargetType = "webhook.ob_target_type" // OneBot 目标类型 group | private
	SysConfigWebhookOBTargetID   = "webhook.ob_target_id"   // 目标群号或用户 QQ
	SysConfigWebhookOBToken      = "webhook.ob_token"       // access token（可空）

	SysConfigWebhookQQGroupIDs = "webhook.qq_group_ids" // QQ 群治理通知目标群号数组 (JSON)

	SysConfigSDEAPIKey          = "sde.api_key"      // SDE 查询 API Key
	SysConfigSDEProxy           = "sde.proxy"        // SDE 下载代理
	SysConfigSDEDownloadURL     = "sde.download_url" // SDE 下载地址
	SysConfigSDEStatus          = "sde.status"       // SDE 状态快照（JSON）
	SysConfigAlliancePAPBaseURL = "alliance_pap.base_url"
	SysConfigAlliancePAPAPIKey  = "alliance_pap.api_key"
	SysConfigOneBotEnabled      = "onebot.enabled"
	SysConfigOneBotAccessToken  = "onebot.access_token"
	SysConfigOneBotBotQQ        = "onebot.bot_qq"
	SysConfigOneBotAllowedCIDRs = "onebot.allowed_cidrs"
	SysConfigQQGovernanceScanIntervalMinutes      = "qq_governance.scan_interval_minutes"
	SysConfigQQGovernanceMismatchConfirmations    = "qq_governance.mismatch_confirmations"
	SysConfigQQGovernanceMismatchObservationHours = "qq_governance.mismatch_observation_hours"

	SysConfigAllowCorporations                               = "app.allow_corporations"                 // 允许访问的公司 ID 列表 (JSON 数组)
	SysConfigCorporationAccessPolicies                       = "app.corporation_access_policies"        // 军团能力策略配置 (JSON)
	SysConfigEnforceCharacterESIRestriction                  = "auth.enforce_character_esi_restriction" // 是否强制限制失效人物 ESI 停留在人物页面
	SysConfigDashboardCorpStructuresAuth                     = "dashboard.corporation_structures_authorizations"
	SysConfigDashboardCorpStructuresFuelNoticeThresholdDays  = "dashboard.corporation_structures_fuel_notice_threshold_days"
	SysConfigDashboardCorpStructuresTimerNoticeThresholdDays = "dashboard.corporation_structures_timer_notice_threshold_days"

	SysConfigNewbroMaxCharacterSP          = "newbro.max_character_sp"
	SysConfigNewbroMultiCharacterSP        = "newbro.multi_character_sp"
	SysConfigNewbroMultiCharacterThreshold = "newbro.multi_character_threshold"
	SysConfigNewbroRefreshIntervalDays     = "newbro.refresh_interval_days"
	SysConfigNewbroBonusRate               = "newbro.bonus_rate"
	SysConfigNewbroRecruitQQURL            = "newbro.recruit_qq_url"
	SysConfigNewbroRecruitRewardAmount     = "newbro.recruit_reward_amount"
	SysConfigNewbroRecruitCooldownDays     = "newbro.recruit_cooldown_days"

	SysConfigMenteeMaxCharacterSP    = "mentor.mentee_max_character_sp"
	SysConfigMenteeMaxAccountAgeDays = "mentor.mentee_max_account_age_days"

	SysConfigWelfareAutoApproveFuxiCoinThreshold = "welfare.auto_approve_fuxi_coin_threshold"
	SysConfigMulticharFullRewardCount            = "multichar.full_reward_count"    // 获得 100% 奖励的人物数（int）
	SysConfigMulticharReducedRewardCount         = "multichar.reduced_reward_count" // 获得折扣奖励的人物数（int）
	SysConfigMulticharReducedRewardPct           = "multichar.reduced_reward_pct"   // 折扣奖励百分比（int, 0-100）

	SysConfigSRPAmountLimit = "srp.amount_limit" // SRP 职权单笔审批/发放上限（ISK）

	SysConfigESITaskIntervals = "esi.task_intervals" // ESI 子任务刷新间隔覆盖（JSON）

	SysConfigDefaultSDEAPIKey                = "change_me_sde_api_key"
	SysConfigDefaultSDEProxy                 = ""
	SysConfigDefaultSDEDownloadURL           = "https://api.github.com/repos/garveen/eve-sde-converter/releases/latest"
	SysConfigDefaultAlliancePAPBaseURL       = "http://jp.newdoublex.space:25220"
	SysConfigDefaultAlliancePAPAPIKey        = ""
	SysConfigDefaultOneBotEnabled            = false
	SysConfigDefaultOneBotAccessToken        = "change_me_onebot_reverse_ws_token"
	SysConfigDefaultOneBotBotQQ        int64 = 0
	SysConfigDefaultQQGovernanceScanIntervalMinutes      = 15
	SysConfigDefaultQQGovernanceMismatchConfirmations    = 2
	SysConfigDefaultQQGovernanceMismatchObservationHours = 2

	SysConfigDefaultNewbroMaxCharacterSP                            int64   = 20_000_000
	SysConfigDefaultNewbroMultiCharacterSP                          int64   = 10_000_000
	SysConfigDefaultNewbroMultiCharacterThreshold                           = 3
	SysConfigDefaultNewbroRefreshIntervalDays                               = 7
	SysConfigDefaultNewbroBonusRate                                 float64 = 20
	SysConfigDefaultNewbroRecruitRewardAmount                       float64 = 50
	SysConfigDefaultNewbroRecruitCooldownDays                               = 90
	SysConfigDefaultMenteeMaxCharacterSP                            int64   = 4_000_000
	SysConfigDefaultMenteeMaxAccountAgeDays                                 = 7
	SysConfigDefaultWelfareAutoApproveFuxiCoinThreshold                     = 500
	SysConfigDefaultPAPFCSalary                                     float64 = 400
	SysConfigDefaultPAPFCSalaryLimit                                int     = 5
	SysConfigDefaultPAPAdminAward                                   int     = 10
	SysConfigDefaultEnforceCharacterESIRestriction                          = true
	SysConfigDefaultDashboardCorpStructuresFuelNoticeThresholdDays          = 7
	SysConfigDefaultDashboardCorpStructuresTimerNoticeThresholdDays         = 7

	SysConfigDefaultMulticharFullRewardCount    = 3
	SysConfigDefaultMulticharReducedRewardCount = 3
	SysConfigDefaultMulticharReducedRewardPct   = 50

	SysConfigDefaultSRPAmountLimit float64 = 600_000_000 // 6 亿 ISK
)
