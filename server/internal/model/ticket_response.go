package model

import "time"

type TicketListItem struct {
	ID                     uint       `json:"id"`
	UserID                 uint       `json:"user_id"`
	CategoryID             uint       `json:"category_id"`
	Title                  string     `json:"title"`
	Description            string     `json:"description"`
	Status                 string     `json:"status"`
	HandledBy              *uint      `json:"handled_by,omitempty"`
	HandledAt              *time.Time `json:"handled_at,omitempty"`
	ClosedAt               *time.Time `json:"closed_at,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
	RequesterName          string     `json:"requester_name"`
	RequesterCharacterName string     `json:"requester_character_name,omitempty"`
	CategoryName           string     `json:"category_name"`
	CategoryNameEN         string     `json:"category_name_en"`
}

type TicketStatusHistoryItem struct {
	ID                     uint      `json:"id"`
	TicketID               uint      `json:"ticket_id"`
	FromStatus             string    `json:"from_status"`
	ToStatus               string    `json:"to_status"`
	ChangedBy              uint      `json:"changed_by"`
	ChangedByName          string    `json:"changed_by_name"`
	ChangedByCharacterName string    `json:"changed_by_character_name,omitempty"`
	ChangedAt              time.Time `json:"changed_at"`
}
