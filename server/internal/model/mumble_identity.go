package model

import "time"

// MumbleIdentity is the Seat-owned projection of a User into the Mumble
// protocol identity namespace. Its own primary key is the stable Mumble user
// ID; it is deliberately unrelated to User.ID and 0 is never allocated.
type MumbleIdentity struct {
	BaseModel
	SeatUserID        uint       `gorm:"not null;uniqueIndex" json:"seat_user_id"`
	CredentialHash    string     `gorm:"type:text;not null;default:''" json:"-"`
	CredentialEnabled bool       `gorm:"not null;default:true" json:"credential_enabled"`
	CredentialVersion uint       `gorm:"not null;default:0" json:"credential_version"`
	IdentityVersion   uint       `gorm:"not null;default:1" json:"identity_version"`
	PasswordRotatedAt *time.Time `json:"password_rotated_at,omitempty"`
	LastUsedAt        *time.Time `json:"last_used_at,omitempty"`
}

func (MumbleIdentity) TableName() string { return "mumble_identity" }

// StableMumbleUserID returns the stable protocol identity. BaseModel.ID is a
// sequence owned by mumble_identity, not the Seat user table.
func (m MumbleIdentity) StableMumbleUserID() uint32 { return uint32(m.ID) }
