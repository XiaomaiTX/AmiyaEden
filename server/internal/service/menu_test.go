package service

import (
	"amiya-eden/internal/model"
	"testing"
)

func TestFilterMenusBySystemRoleRestrictions(t *testing.T) {
	menus := []model.Menu{
		{Name: "Operation"},
		{Name: "Fleets"},
		{Name: "FleetDetail"},
		{Name: "FleetConfigs"},
		{Name: "CorporationPap"},
	}

	t.Run("user loses fleet management menus but keeps fleet configs", func(t *testing.T) {
		filtered := filterMenusBySystemRoleRestrictions(menus, []string{model.RoleUser}, nil)
		got := make(map[string]struct{}, len(filtered))
		for _, menu := range filtered {
			got[menu.Name] = struct{}{}
		}

		if _, ok := got["Fleets"]; ok {
			t.Fatal("expected Fleets to be filtered for user role")
		}
		if _, ok := got["FleetDetail"]; ok {
			t.Fatal("expected FleetDetail to be filtered for user role")
		}
		if _, ok := got["FleetConfigs"]; !ok {
			t.Fatal("expected FleetConfigs to remain visible for user role")
		}
		if _, ok := got["CorporationPap"]; !ok {
			t.Fatal("expected CorporationPap to remain visible for user role")
		}
	})

	t.Run("fc keeps restricted fleet menus", func(t *testing.T) {
		filtered := filterMenusBySystemRoleRestrictions(menus, []string{model.RoleFC}, nil)
		got := make(map[string]struct{}, len(filtered))
		for _, menu := range filtered {
			got[menu.Name] = struct{}{}
		}

		for _, name := range []string{"Fleets", "FleetDetail", "FleetConfigs"} {
			if _, ok := got[name]; !ok {
				t.Fatalf("expected %s to remain visible for fc role", name)
			}
		}
	})

	t.Run("non newbro loses selection page and empty root", func(t *testing.T) {
		isCurrentlyNewbro := false
		filtered := filterMenusBySystemRoleRestrictions(
			[]model.Menu{
				{Name: "NewbroRoot"},
				{Name: "NewbroSelectCaptain"},
			},
			[]string{model.RoleUser},
			&isCurrentlyNewbro,
		)
		if len(filtered) != 0 {
			t.Fatalf("expected newbro menus to be removed for non-newbro user, got %d entries", len(filtered))
		}
	})

	t.Run("admin loses captain page without captain role but keeps manage page", func(t *testing.T) {
		filtered := filterMenusBySystemRoleRestrictions(
			[]model.Menu{
				{Name: "NewbroRoot"},
				{Name: "NewbroCaptainDashboard"},
				{Name: "NewbroManage"},
			},
			[]string{model.RoleAdmin},
			nil,
		)
		got := make(map[string]struct{}, len(filtered))
		for _, menu := range filtered {
			got[menu.Name] = struct{}{}
		}
		if _, ok := got["NewbroCaptainDashboard"]; ok {
			t.Fatal("expected captain dashboard to be filtered for admin without captain role")
		}
		if _, ok := got["NewbroManage"]; !ok {
			t.Fatal("expected manage page to remain visible for admin")
		}
		if _, ok := got["NewbroRoot"]; !ok {
			t.Fatal("expected root menu to remain when manage page is still visible")
		}
	})
}
