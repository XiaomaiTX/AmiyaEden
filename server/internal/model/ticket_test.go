package model

import "testing"

func TestTicketModelTableNames(t *testing.T) {
	if (Ticket{}).TableName() != "ticket" {
		t.Fatalf("Ticket table name = %q, want %q", (Ticket{}).TableName(), "ticket")
	}
	if (TicketCategory{}).TableName() != "ticket_category" {
		t.Fatalf("TicketCategory table name = %q, want %q", (TicketCategory{}).TableName(), "ticket_category")
	}
	if (TicketReply{}).TableName() != "ticket_reply" {
		t.Fatalf("TicketReply table name = %q, want %q", (TicketReply{}).TableName(), "ticket_reply")
	}
	if (TicketStatusHistory{}).TableName() != "ticket_status_history" {
		t.Fatalf("TicketStatusHistory table name = %q, want %q", (TicketStatusHistory{}).TableName(), "ticket_status_history")
	}
}
