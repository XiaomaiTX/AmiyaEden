package model

import "time"

const (
	QQGovernanceReviewMatched          = "matched"
	QQGovernanceReviewUnmatched        = "unmatched"
	QQGovernanceReviewWait             = "review_wait"
	QQGovernanceActionPending          = "pending"
	QQGovernanceActionRunning          = "running"
	QQGovernanceActionRetryWait        = "retry_wait"
	QQGovernanceActionSucceeded        = "succeeded"
	QQGovernanceActionCancelled        = "cancelled"
	QQGovernanceActionDead             = "dead"
	QQGovernanceActionApprove          = "approve"
	QQGovernanceActionReject           = "reject"
	QQGovernanceActionSetCard          = "set_card"
	QQGovernanceActionSnapshot         = "snapshot"
	QQGovernanceActionRefreshGroupInfo = "refresh_group_info"
	QQGovernanceActionComputeBatch     = "compute_batch"
	QQGovernanceActionRecheck          = "recheck"
	QQGovernanceActionKick             = "kick"
	QQGovernanceActionNotify           = "notify"
	QQGovernanceViolationReview        = "review_only"
	QQGovernanceViolationAutoKick      = "auto_kick_after_confirmed_mismatch"
	QQGovernanceMemberJoinPending      = "join_pending"
	QQGovernanceMemberValid            = "in_group_valid"
	QQGovernanceMemberReview           = "in_group_review"
	QQGovernanceMemberInvalidCand      = "in_group_invalid_candidate"
	QQGovernanceMemberInvalidConf      = "in_group_invalid_confirmed"
	QQGovernanceMemberLeft             = "left"
	QQGovernanceRunPending             = "pending"
	QQGovernanceRunRunning             = "running"
	QQGovernanceRunCompleted           = "completed"
	QQGovernanceRunFailed              = "failed"
	QQGovernanceRunMemberPending       = "pending"
	QQGovernanceRunMemberDone          = "done"
	QQGovernanceAlertOpen              = "open"
	QQGovernanceAlertAcknowledged      = "acknowledged"
	QQGovernanceAlertResolved          = "resolved"
	QQGovernanceRiskNormal             = 0
	QQGovernanceRiskLevelOne           = 1
	QQGovernanceRiskLevelTwo           = 2
	QQGovernanceRiskLevelThree         = 3
)

// QQGroupGovernancePolicy 保存单个 QQ 群的准入与名片规则。首期仅使用
// Enabled、军团/职权白名单、拒绝策略与名片模板；其余字段由后续巡检阶段增加。
// CardSyncEnabled 控制巡检是否在每次完整快照后对匹配成员调用 set_group_card；
// 它默认关闭，关闭时不影响资格判断和目标名片计算，但禁止所有新的名片写入。
type QQGroupGovernancePolicy struct {
	BaseModel
	GroupID                   int64  `gorm:"not null;uniqueIndex" json:"group_id"`
	Enabled                   bool   `gorm:"not null;default:false;index" json:"enabled"`
	AllowedCorporationIDsJSON string `gorm:"type:text;not null;default:'[]'" json:"allowed_corporation_ids_json"`
	AllowedRoleCodesJSON      string `gorm:"type:text;not null;default:'[]'" json:"allowed_role_codes_json"`
	AutoRejectUnmatched       bool   `gorm:"not null;default:false" json:"auto_reject_unmatched"`
	MemberViolationPolicy     string `gorm:"size:64;not null;default:'review_only'" json:"member_violation_policy"`
	CardTemplate              string `gorm:"size:256;not null;default:''" json:"card_template"`
	CardSyncEnabled           bool   `gorm:"not null;default:false" json:"card_sync_enabled"`
	UpdatedBy                 uint   `gorm:"not null;default:0" json:"updated_by"`
}

func (QQGroupGovernancePolicy) TableName() string { return "qq_group_governance_policy" }

// QQGovernanceEvent 保存完成去重和审查所需的最小事件摘要，绝不保存原始 OneBot payload。
type QQGovernanceEvent struct {
	BaseModel
	EventKey    string `gorm:"size:256;not null;uniqueIndex" json:"event_key"`
	EventType   string `gorm:"size:64;not null;index" json:"event_type"`
	GroupID     int64  `gorm:"not null;index" json:"group_id"`
	QQ          int64  `gorm:"not null;index" json:"qq"`
	RequestFlag string `gorm:"size:256;not null;default:''" json:"-"`
}

func (QQGovernanceEvent) TableName() string { return "qq_governance_event" }

// QQGroupMemberState 是动作版本校验的权威运行态。
type QQGroupMemberState struct {
	BaseModel
	GroupID           int64      `gorm:"not null;uniqueIndex:idx_qq_governance_member" json:"group_id"`
	QQ                int64      `gorm:"not null;uniqueIndex:idx_qq_governance_member" json:"qq"`
	UserID            uint       `gorm:"not null;default:0;index" json:"user_id"`
	Status            string     `gorm:"size:64;not null;index" json:"status"`
	TargetCard        string     `gorm:"size:128;not null;default:''" json:"target_card"`
	Version           uint64     `gorm:"not null;default:1" json:"version"`
	MismatchCount     int        `gorm:"not null;default:0" json:"mismatch_count"`
	FirstMismatchAt   *time.Time `json:"first_mismatch_at"`
	UnknownCount      int        `gorm:"not null;default:0" json:"unknown_count"`
	LastCardUpdatedAt *time.Time `json:"last_card_updated_at"`
	LastCheckedAt     time.Time  `gorm:"not null" json:"last_checked_at"`
}

func (QQGroupMemberState) TableName() string { return "qq_group_member_state" }

// QQGovernanceReconcileRun is one immutable full-group membership snapshot.
// ActiveKey is populated only while the run is non-terminal, so a group can
// never have two concurrent full scans. The invariant is enforced by the
// partial unique index defined in bootstrap/db.go (qqGovernanceIndexStatements);
// the column itself is a plain non-unique string so completed runs can share
// the empty terminal value.
type QQGovernanceReconcileRun struct {
	BaseModel
	GroupID        int64      `gorm:"not null;index" json:"group_id"`
	ActiveKey      string     `gorm:"size:64;not null;default:''" json:"-"`
	Status         string     `gorm:"size:32;not null;index" json:"status"`
	ExpectedCount  int        `gorm:"not null;default:0" json:"expected_count"`
	ProcessedCount int        `gorm:"not null;default:0" json:"processed_count"`
	FailedCount    int        `gorm:"not null;default:0" json:"failed_count"`
	StartedAt      *time.Time `json:"started_at"`
	CompletedAt    *time.Time `json:"completed_at"`
	LastError      string     `gorm:"type:text;not null;default:''" json:"last_error"`
}

func (QQGovernanceReconcileRun) TableName() string { return "qq_governance_reconcile_run" }

// QQGovernanceReconcileMember freezes the ordered membership set for a run.
// It deliberately avoids using OneBot response order as a cursor.
// Card 保存该巡检快照中读取到的当前群名片，供批处理比较时无需再次请求 OneBot。
type QQGovernanceReconcileMember struct {
	BaseModel
	RunID     uint   `gorm:"not null;uniqueIndex:idx_qq_governance_run_member;index" json:"run_id"`
	GroupID   int64  `gorm:"not null;index" json:"group_id"`
	QQ        int64  `gorm:"not null;uniqueIndex:idx_qq_governance_run_member" json:"qq"`
	Card      string `gorm:"size:128;not null;default:''" json:"card"`
	Status    string `gorm:"size:32;not null;index" json:"status"`
	LastError string `gorm:"type:text;not null;default:''" json:"last_error"`
}

func (QQGovernanceReconcileMember) TableName() string { return "qq_governance_reconcile_member" }

// QQGroupRuntimeSnapshot 保存最近一次成功读取到的群状态，供管理页在断连时展示。
type QQGroupRuntimeSnapshot struct {
	BaseModel
	GroupID           int64      `gorm:"not null;uniqueIndex" json:"group_id"`
	GroupName         string     `gorm:"size:256;not null;default:''" json:"group_name"`
	MemberCount       int        `gorm:"not null;default:0" json:"member_count"`
	MaxMemberCount    int        `gorm:"not null;default:0" json:"max_member_count"`
	BotIsAdmin        *bool      `json:"bot_is_admin"`
	LastSyncAttemptAt time.Time  `gorm:"not null" json:"last_sync_attempt_at"`
	LastSyncedAt      *time.Time `json:"last_synced_at"`
	LastSyncError     string     `gorm:"type:text;not null;default:''" json:"last_sync_error"`
}

func (QQGroupRuntimeSnapshot) TableName() string { return "qq_group_runtime_snapshot" }

// QQGovernanceReview 记录每次资格判断和最小必要身份快照。
type QQGovernanceReview struct {
	BaseModel
	EventID                uint   `gorm:"not null;index" json:"event_id"`
	PolicyID               uint   `gorm:"not null;index" json:"policy_id"`
	GroupID                int64  `gorm:"not null;index" json:"group_id"`
	QQ                     int64  `gorm:"not null;index" json:"qq"`
	Source                 string `gorm:"size:32;not null" json:"source"`
	Decision               string `gorm:"size:32;not null;index" json:"decision"`
	Reason                 string `gorm:"size:256;not null" json:"reason"`
	UserID                 uint   `gorm:"not null;default:0" json:"user_id"`
	Nickname               string `gorm:"size:128;not null;default:''" json:"nickname"`
	PrimaryCharacterName   string `gorm:"size:128;not null;default:''" json:"primary_character_name"`
	PrimaryCorporationName string `gorm:"size:128;not null;default:''" json:"primary_corporation_name"`
	RoleCodesJSON          string `gorm:"type:text;not null;default:'[]'" json:"role_codes_json"`
}

func (QQGovernanceReview) TableName() string { return "qq_governance_review" }

// QQGovernanceActionTask 是可恢复的 QQ 写动作队列。payload 只保存动作必需参数，不能包含令牌或原始事件。
type QQGovernanceActionTask struct {
	BaseModel
	ActionType     string     `gorm:"size:32;not null;index" json:"action_type"`
	RunID          uint       `gorm:"not null;default:0;index" json:"run_id"`
	IdempotencyKey string     `gorm:"size:256;not null;uniqueIndex" json:"idempotency_key"`
	GroupID        int64      `gorm:"not null;index" json:"group_id"`
	QQ             int64      `gorm:"not null;index" json:"qq"`
	TargetVersion  uint64     `gorm:"not null;default:0" json:"target_version"`
	PayloadJSON    string     `gorm:"type:text;not null;default:'{}'" json:"-"`
	Status         string     `gorm:"size:32;not null;index" json:"status"`
	Priority       int        `gorm:"not null;index" json:"priority"`
	RetryCount     int        `gorm:"not null;default:0" json:"retry_count"`
	RunAfter       time.Time  `gorm:"not null;index" json:"run_after"`
	LeaseToken     string     `gorm:"size:64;not null;default:''" json:"-"`
	ClaimedAt      *time.Time `json:"claimed_at"`
	LeaseExpiresAt *time.Time `gorm:"index" json:"lease_expires_at"`
	Source         string     `gorm:"size:32;not null;default:'automatic'" json:"source"`
	LastError      string     `gorm:"type:text;not null;default:''" json:"last_error"`
	CompletedAt    *time.Time `json:"completed_at"`
}

func (QQGovernanceActionTask) TableName() string { return "qq_governance_action_task" }

// QQGovernanceActionLog 为每次向 OneBot 发送动作建立可审计、脱敏的执行记录。
type QQGovernanceActionLog struct {
	BaseModel
	TaskID         uint   `gorm:"not null;index" json:"task_id"`
	ActionType     string `gorm:"size:32;not null;index" json:"action_type"`
	RequestSummary string `gorm:"type:text;not null" json:"request_summary"`
	Result         string `gorm:"size:32;not null;index" json:"result"`
	ErrorMessage   string `gorm:"type:text;not null;default:''" json:"error_message"`
	Attempt        int    `gorm:"not null;default:1" json:"attempt"`
}

func (QQGovernanceActionLog) TableName() string { return "qq_governance_action_log" }

// QQGovernanceRiskControlState 记录单机器人最近动作失败窗口与熔断状态。
type QQGovernanceRiskControlState struct {
	BaseModel
	BotQQ        int64      `gorm:"not null;uniqueIndex" json:"bot_qq"`
	Level        int        `gorm:"not null;default:0" json:"level"`
	OpenedAt     *time.Time `json:"opened_at"`
	OpenUntil    *time.Time `json:"open_until"`
	HalfOpenLeft int        `gorm:"not null;default:0" json:"half_open_left"`
	UpdatedBy    uint       `gorm:"not null;default:0" json:"updated_by"`
}

func (QQGovernanceRiskControlState) TableName() string { return "qq_governance_risk_control_state" }

// QQGovernanceAlert 是仅在 QQ 群治理管理页展示的持久告警，不进入全局通知中心。
type QQGovernanceAlert struct {
	BaseModel
	AlertKey       string     `gorm:"size:256;not null;uniqueIndex" json:"alert_key"`
	Kind           string     `gorm:"size:64;not null;index" json:"kind"`
	GroupID        int64      `gorm:"not null;default:0;index" json:"group_id"`
	QQ             int64      `gorm:"not null;default:0;index" json:"qq"`
	TaskID         uint       `gorm:"not null;default:0;index" json:"task_id"`
	Status         string     `gorm:"size:32;not null;index" json:"status"`
	Message        string     `gorm:"type:text;not null" json:"message"`
	AcknowledgedBy uint       `gorm:"not null;default:0" json:"acknowledged_by"`
	AcknowledgedAt *time.Time `json:"acknowledged_at"`
	ResolvedAt     *time.Time `json:"resolved_at"`
}

func (QQGovernanceAlert) TableName() string { return "qq_governance_alert" }
