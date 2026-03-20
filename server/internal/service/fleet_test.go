package service

import (
	"amiya-eden/internal/model"
	"testing"
	"time"
)

func TestNormalizeAutoSrpMode(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "submit only", in: model.FleetAutoSrpSubmitOnly, want: model.FleetAutoSrpSubmitOnly},
		{name: "auto approve", in: model.FleetAutoSrpAutoApprove, want: model.FleetAutoSrpAutoApprove},
		{name: "empty defaults disabled", in: "", want: model.FleetAutoSrpDisabled},
		{name: "unknown defaults disabled", in: "surprise", want: model.FleetAutoSrpDisabled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeAutoSrpMode(tt.in); got != tt.want {
				t.Fatalf("normalizeAutoSrpMode(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestFleetServiceCanManageFleet(t *testing.T) {
	svc := &FleetService{}
	fleet := &model.Fleet{FCUserID: 42}

	tests := []struct {
		name     string
		userID   uint
		userRole string
		want     bool
	}{
		{name: "owner", userID: 42, userRole: model.RoleUser, want: true},
		{name: "admin", userID: 7, userRole: model.RoleAdmin, want: true},
		{name: "super admin", userID: 8, userRole: model.RoleSuperAdmin, want: true},
		{name: "other user", userID: 9, userRole: model.RoleUser, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := svc.canManageFleet(fleet, tt.userID, tt.userRole); got != tt.want {
				t.Fatalf("canManageFleet() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFleetServiceBuildCorporationPapFilter(t *testing.T) {
	svc := &FleetService{}
	location := time.FixedZone("UTC+8", 8*60*60)
	now := time.Date(2026, time.March, 21, 10, 30, 0, 0, location)

	t.Run("default last month", func(t *testing.T) {
		filter, period, year, err := svc.buildCorporationPapFilter("", 0, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if period != CorporationPapPeriodLastMonth {
			t.Fatalf("period = %q, want %q", period, CorporationPapPeriodLastMonth)
		}
		if year != nil {
			t.Fatalf("year = %v, want nil", *year)
		}
		if filter.StartAt == nil || filter.StartAt.Format(time.DateOnly) != "2026-02-01" {
			t.Fatalf("start = %v, want 2026-02-01", filter.StartAt)
		}
		if filter.EndAt == nil || filter.EndAt.Format(time.DateOnly) != "2026-03-01" {
			t.Fatalf("end = %v, want 2026-03-01", filter.EndAt)
		}
	})

	t.Run("at year normalizes current year when missing", func(t *testing.T) {
		filter, period, year, err := svc.buildCorporationPapFilter(CorporationPapPeriodAtYear, 0, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if period != CorporationPapPeriodAtYear {
			t.Fatalf("period = %q, want %q", period, CorporationPapPeriodAtYear)
		}
		if year == nil || *year != 2026 {
			t.Fatalf("year = %v, want 2026", year)
		}
		if filter.StartAt == nil || filter.StartAt.Format(time.DateOnly) != "2026-01-01" {
			t.Fatalf("start = %v, want 2026-01-01", filter.StartAt)
		}
		if filter.EndAt == nil || filter.EndAt.Format(time.DateOnly) != "2027-01-01" {
			t.Fatalf("end = %v, want 2027-01-01", filter.EndAt)
		}
	})

	t.Run("last year alias", func(t *testing.T) {
		_, period, year, err := svc.buildCorporationPapFilter("last_year", 0, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if period != CorporationPapPeriodAtYear {
			t.Fatalf("period = %q, want %q", period, CorporationPapPeriodAtYear)
		}
		if year == nil || *year != 2025 {
			t.Fatalf("year = %v, want 2025", year)
		}
	})

	t.Run("invalid period", func(t *testing.T) {
		if _, _, _, err := svc.buildCorporationPapFilter("bad_period", 0, now); err == nil {
			t.Fatal("expected invalid period error")
		}
	})
}
