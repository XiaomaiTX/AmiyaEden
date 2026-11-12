package model

import "time"

const (
	AdmissionKindNewMember             = "new_member"
	AdmissionKindAlt                   = "alt"
	AdmissionStatusPending             = "pending"
	AdmissionStatusInvited             = "invited"
	AdmissionStatusRejected            = "rejected"
	AdmissionEvidenceLauncher          = "launcher"
	AdmissionEvidenceSeatProfile       = "seat_profile"
	AdmissionFMACorporationID    int64 = 98824507
	AdmissionInvitationSalary          = 10.0
)

// AdmissionApplication is one user's submission for one corporation, regardless
// of how many bound characters it contains. Evidence bytes live separately.
type AdmissionApplication struct {
	BaseModel
	UserID              uint                 `gorm:"not null;index;uniqueIndex:idx_admission_pending_new_user,where:kind = 'new_member' AND status = 'pending' AND deleted_at IS NULL" json:"user_id"`
	Kind                string               `gorm:"size:20;not null" json:"kind"`
	TargetCorporationID int64                `gorm:"not null;index:idx_admission_target_status,priority:1" json:"target_corporation_id"`
	Status              string               `gorm:"size:20;not null;index:idx_admission_target_status,priority:2" json:"status"`
	Version             uint                 `gorm:"not null;default:1" json:"version"`
	ApplicantName       string               `gorm:"size:128;not null" json:"applicant_name"`
	PrimaryCharacterID  int64                `json:"primary_character_id"`
	QQ                  string               `gorm:"size:20" json:"qq"`
	DiscordID           string               `gorm:"size:20" json:"discord_id"`
	SubmittedAt         time.Time            `gorm:"not null;index" json:"submitted_at"`
	HandledBy           *uint                `json:"handled_by"`
	HandlerName         string               `gorm:"size:128" json:"handler_name"`
	ClosedAt            *time.Time           `json:"closed_at"`
	RejectionReason     string               `gorm:"type:text" json:"rejection_reason"`
	EvidenceExpiresAt   *time.Time           `gorm:"index" json:"evidence_expires_at"`
	Characters          []AdmissionCharacter `gorm:"foreignKey:ApplicationID" json:"characters"`
}

func (AdmissionApplication) TableName() string { return "admission_application" }

type AdmissionCharacter struct {
	ApplicationID uint   `gorm:"primaryKey;autoIncrement:false" json:"application_id"`
	CharacterID   int64  `gorm:"primaryKey;autoIncrement:false" json:"character_id"`
	CharacterName string `gorm:"size:128;not null" json:"character_name"`
}

func (AdmissionCharacter) TableName() string { return "admission_character" }

// AdmissionCharacterClaim exists only while an application is pending. A real
// unique key, rather than a preflight lookup, serializes competing submissions.
type AdmissionCharacterClaim struct {
	CharacterID   int64 `gorm:"primaryKey;autoIncrement:false"`
	ApplicationID uint  `gorm:"not null;index"`
}

func (AdmissionCharacterClaim) TableName() string { return "admission_character_claim" }

type AdmissionEvidence struct {
	ApplicationID uint      `gorm:"primaryKey;autoIncrement:false" json:"application_id"`
	Kind          string    `gorm:"primaryKey;size:20" json:"kind"`
	Data          []byte    `gorm:"not null" json:"-"`
	Width         int       `gorm:"not null" json:"width"`
	Height        int       `gorm:"not null" json:"height"`
	ByteSize      int       `gorm:"not null" json:"byte_size"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (AdmissionEvidence) TableName() string { return "admission_evidence" }

type AdmissionProcessingEvent struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	ApplicationID uint      `gorm:"not null;index" json:"application_id"`
	Version       uint      `gorm:"not null" json:"version"`
	ActorID       uint      `gorm:"not null" json:"actor_id"`
	ActorName     string    `gorm:"size:128" json:"actor_name"`
	Action        string    `gorm:"size:20;not null" json:"action"`
	FromStatus    string    `gorm:"size:20" json:"from_status"`
	ToStatus      string    `gorm:"size:20;not null" json:"to_status"`
	Reason        string    `gorm:"type:text" json:"reason"`
	Snapshot      string    `gorm:"type:text;not null" json:"snapshot"`
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (AdmissionProcessingEvent) TableName() string { return "admission_processing_event" }

type AdmissionSalary struct {
	ApplicationID       uint      `gorm:"primaryKey;autoIncrement:false" json:"application_id"`
	UserID              uint      `gorm:"not null;index" json:"user_id"`
	Amount              float64   `gorm:"not null" json:"amount"`
	WalletTransactionID uint      `gorm:"not null;uniqueIndex" json:"wallet_transaction_id"`
	CreatedAt           time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (AdmissionSalary) TableName() string { return "admission_salary" }
