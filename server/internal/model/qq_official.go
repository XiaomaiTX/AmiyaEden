package model

import "time"

const (
	QQQueued         = "queued"
	QQExecuting      = "executing"
	QQRetryWait      = "retry_wait"
	QQAwaitingReview = "awaiting_review"
	QQSucceeded      = "succeeded"
	QQFailed         = "failed"
	QQUnknown        = "unknown"
	QQExpired        = "expired"
	QQCancelled      = "cancelled"
)

// QQOfficialApp serializes admission, configuration changes, and gateway ownership.
// The HTTP lease covers one upstream request; the renewable gateway lease is separate.
type QQOfficialApp struct {
	AppID             string `gorm:"primaryKey;size:64"`
	Enabled           bool
	Paused            bool
	Certified         bool
	Generation        int64
	LeaseID           string `gorm:"size:36"`
	LeaseTaskID       string `gorm:"size:36"`
	LeaseUntil        *time.Time
	TokenExpiresAt    *time.Time
	TokenState        string `gorm:"size:32"`
	TokenGeneration   int64
	GlobalHold        string `gorm:"size:32"`
	LastEventAt       *time.Time
	GatewayLeaseID    string `gorm:"size:36"`
	GatewayLeaseUntil *time.Time
	GatewayRevision   int64
	GatewayState      string `gorm:"size:16"`
	NextAdmissionAt   *time.Time
	LastUrgentGroup   string `gorm:"size:36"`
	LastRegularGroup  string `gorm:"size:36"`
	UrgentStreak      int
}

func (QQOfficialApp) TableName() string { return "qq_official_app" }

// QQOfficialInstallation stores the singleton binding. Keeping it outside the
// generic system_config catalogue prevents accidental credential enumeration.
type QQOfficialInstallation struct {
	ID                    uint   `gorm:"primaryKey;autoIncrement:false" json:"-"`
	AppID                 string `gorm:"size:32;not null"`
	Revision              int64  `gorm:"not null"`
	SecretCiphertext      []byte `json:"-"`
	ApprovalWritesEnabled bool   `gorm:"not null"`
}

func (QQOfficialInstallation) TableName() string { return "qq_official_installation" }

type QQOfficialGroup struct {
	ID                   string `gorm:"primaryKey;size:36"`
	AppID                string `gorm:"size:64;not null;uniqueIndex:idx_qq_official_platform_group"`
	OpenID               string `gorm:"size:128;not null;uniqueIndex:idx_qq_official_platform_group"`
	Alias                string `gorm:"size:128"`
	Present              bool
	Authorized           bool
	NotificationEnabled  bool
	TestVerified         bool
	ApprovalEnabled      bool
	PushState            string `gorm:"size:16"`
	ApprovalCapability   string `gorm:"size:16"`
	MembershipAt         *time.Time
	PushAt               *time.Time
	Generation           int64
	NotifyGeneration     int64
	ApprovalGeneration   int64
	FleetCorporation     int64 `gorm:"not null;default:0;index"`
	StructureCorporation int64 `gorm:"not null;default:0;index"`
	CreatedAt            time.Time
}

func (QQOfficialGroup) TableName() string { return "qq_official_group" }

type QQOfficialIntent struct {
	ID           string `gorm:"primaryKey;size:36"`
	AppID        string `gorm:"size:64;not null;uniqueIndex:idx_qq_official_intent_key"`
	Key          string `gorm:"column:stable_key;size:64;not null;uniqueIndex:idx_qq_official_intent_key"`
	Digest       string `gorm:"size:64;not null"`
	MergedTaskID string `gorm:"size:36;index"`
	CreatedAt    time.Time
}

func (QQOfficialIntent) TableName() string { return "qq_official_intent" }

type QQOfficialTask struct {
	ID                string `gorm:"primaryKey;size:36"`
	AppID             string `gorm:"size:64;not null;index:idx_qq_official_ready,priority:1;index:idx_qq_official_task_group,priority:1;index:idx_qq_official_task_source,priority:1"`
	IntentID          string `gorm:"size:36;not null;uniqueIndex:idx_qq_official_task_target"`
	GroupID           string `gorm:"size:36;not null;uniqueIndex:idx_qq_official_task_target;index:idx_qq_official_task_group,priority:2"`
	AppGeneration     int64
	GroupGeneration   int64
	ModeGeneration    int64
	Kind              string `gorm:"size:24;not null"`
	Source            string `gorm:"size:24;not null;index:idx_qq_official_task_source,priority:2"`
	Reference         string `gorm:"size:128;index:idx_qq_official_task_source,priority:3"`
	Urgent            bool
	OperatorID        uint
	ApplicationID     string   `gorm:"size:36;index"`
	Decision          string   `gorm:"size:8"`
	Text              string   `gorm:"type:text"`
	Cursor            string   `gorm:"type:text"`
	VisitedCursors    []string `gorm:"serializer:json;type:text"`
	PageCount         int
	RetryCount        int
	CredentialRetried bool
	Status            string `gorm:"size:24;not null;index:idx_qq_official_ready,priority:2;index:idx_qq_official_task_group,priority:3"`
	Reason            string `gorm:"size:48"`
	TraceID           string `gorm:"size:128"`
	UpstreamCode      int
	ReceiptDigest     string     `gorm:"size:64"`
	NextRunAt         time.Time  `gorm:"not null;index:idx_qq_official_ready,priority:3"`
	ExpiresAt         time.Time  `gorm:"not null"`
	CreatedAt         time.Time  `gorm:"index"`
	FinishedAt        *time.Time `gorm:"index"`
}

func (QQOfficialTask) TableName() string { return "qq_official_task" }

// An admitted attempt is counted even if the outcome is lost or no bytes were sent.
type QQOfficialAttempt struct {
	ID           string    `gorm:"primaryKey;size:36"`
	AppID        string    `gorm:"size:64;not null;index:idx_qq_official_attempt_app,priority:1;index:idx_qq_official_attempt_kind,priority:1;index:idx_qq_official_attempt_group,priority:1"`
	TaskID       string    `gorm:"size:36;index"`
	GroupID      string    `gorm:"size:36;index:idx_qq_official_attempt_group,priority:2"`
	Kind         string    `gorm:"size:24;not null;index:idx_qq_official_attempt_kind,priority:2;index:idx_qq_official_attempt_group,priority:3"`
	AdmittedAt   time.Time `gorm:"index:idx_qq_official_attempt_app,priority:2;index:idx_qq_official_attempt_kind,priority:3;index:idx_qq_official_attempt_group,priority:4"`
	FinishedAt   *time.Time
	Outcome      string `gorm:"size:48"`
	HTTPStatus   int
	PlatformCode int
	TraceID      string `gorm:"size:128"`
}

func (QQOfficialAttempt) TableName() string { return "qq_official_attempt" }

type QQOfficialCooldown struct {
	AppID       string `gorm:"primaryKey;size:64"`
	Scope       string `gorm:"primaryKey;size:96"`
	Until       *time.Time
	Failures    int
	ThrottleRun int
	UpdatedAt   time.Time
}

func (QQOfficialCooldown) TableName() string { return "qq_official_cooldown" }

// QQOfficialEvent contains only the fields needed to project recognized events.
type QQOfficialEvent struct {
	ID           int64  `gorm:"primaryKey"`
	AppID        string `gorm:"size:64;not null;uniqueIndex:idx_qq_official_event_key;index:idx_qq_official_event_pending,priority:1"`
	EventKey     string `gorm:"size:64;not null;uniqueIndex:idx_qq_official_event_key"`
	Kind         string `gorm:"size:32;not null"`
	GroupOpenID  string `gorm:"size:128;not null"`
	EventAt      time.Time
	RequestID    string `gorm:"size:384"`
	MemberOpenID string `gorm:"size:128"`
	Username     string `gorm:"size:128"`
	AutoApproved bool
	CreatedAt    time.Time  `gorm:"index"`
	ProcessedAt  *time.Time `gorm:"index:idx_qq_official_event_pending,priority:2"`
}

func (QQOfficialEvent) TableName() string { return "qq_official_event" }

type QQOfficialJoinRequest struct {
	ID           string    `gorm:"primaryKey;size:36"`
	AppID        string    `gorm:"size:64;not null;uniqueIndex:idx_qq_official_join_key;index:idx_qq_official_join_group,priority:1"`
	GroupID      string    `gorm:"size:36;not null;uniqueIndex:idx_qq_official_join_key;index:idx_qq_official_join_group,priority:2"`
	RequestKey   string    `gorm:"size:64;not null;uniqueIndex:idx_qq_official_join_key"`
	RequestID    string    `gorm:"size:384"`
	MemberOpenID string    `gorm:"size:128"`
	Username     string    `gorm:"size:128"`
	AppliedAt    time.Time `gorm:"index:idx_qq_official_join_group,priority:3"`
	ExpiresAt    time.Time `gorm:"index"`
	Status       string    `gorm:"size:24"`
	TaskID       string    `gorm:"size:36"`
	RefreshID    string    `gorm:"size:36"`
	CreatedAt    time.Time
}

func (QQOfficialJoinRequest) TableName() string { return "qq_official_join_request" }

func QQOfficialModels() []interface{} {
	return []interface{}{&QQOfficialInstallation{}, &QQOfficialApp{}, &QQOfficialGroup{}, &QQOfficialIntent{}, &QQOfficialTask{},
		&QQOfficialAttempt{}, &QQOfficialCooldown{}, &QQOfficialEvent{}, &QQOfficialJoinRequest{}, &QQOfficialStructureAlert{}}
}
