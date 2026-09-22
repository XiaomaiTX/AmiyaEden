package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"reflect"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupMumbleConfigTest(t *testing.T) *SysConfigService {
	t.Helper()
	oldDB := global.DB
	t.Cleanup(func() { global.DB = oldDB })
	db, err := gorm.Open(sqlite.Open("file:sys_config_mumble?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.SystemConfig{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Exec("DELETE FROM system_config").Error; err != nil {
		t.Fatalf("clean system_config: %v", err)
	}
	global.DB = db
	return NewSysConfigService()
}

func TestSysConfigServiceMumbleConfig(t *testing.T) {
	svc := setupMumbleConfigTest(t)

	want := MumbleRuntimeConfig{
		ServiceToken: "mumble-to-seat-token", ServerURL: "https://mumble.internal.example/",
		RevalidateToken: "seat-to-mumble-token", RevalidateTimeoutMS: 750,
		PublicNodes: []MumblePublicNode{
			{Name: " 香港-1 ", Description: " 香港节点 ", Address: " hk.mumble.example.com ", Port: 64738},
			{Address: "fra.mumble.example.com", Port: 64739},
		},
		DisplayNameTemplate: " {nickname} ({character_name}) ",
	}
	if err := svc.UpdateMumbleConfig(want); err != nil {
		t.Fatalf("update: %v", err)
	}
	got := svc.GetMumbleConfig()
	if got.ServerURL != "https://mumble.internal.example" || got.ServiceToken != want.ServiceToken || got.RevalidateToken != want.RevalidateToken || got.RevalidateTimeoutMS != want.RevalidateTimeoutMS {
		t.Fatalf("config = %+v", got)
	}
	wantNodes := []MumblePublicNode{
		{Name: "香港-1", Description: "香港节点", Address: "hk.mumble.example.com", Port: 64738},
		{Address: "fra.mumble.example.com", Port: 64739},
	}
	if !reflect.DeepEqual(got.PublicNodes, wantNodes) {
		t.Fatalf("public nodes = %+v", got.PublicNodes)
	}
	if got.DisplayNameTemplate != "{nickname} ({character_name})" {
		t.Fatalf("display name template = %q", got.DisplayNameTemplate)
	}
	if err := svc.UpdateMumbleConfig(MumbleRuntimeConfig{RevalidateTimeoutMS: 1000}); err != nil {
		t.Fatalf("zero-value update: %v", err)
	}
	got = svc.GetMumbleConfig()
	if len(got.PublicNodes) != 0 {
		t.Fatalf("unset public nodes = %+v", got.PublicNodes)
	}
	if got.DisplayNameTemplate != defaultMumbleDisplayNameTemplate {
		t.Fatalf("empty template must use default, got %q", got.DisplayNameTemplate)
	}
}

func TestSysConfigServiceMumbleConfigValidation(t *testing.T) {
	svc := setupMumbleConfigTest(t)
	base := MumbleRuntimeConfig{
		ServiceToken: "mumble-to-seat-token", ServerURL: "https://mumble.internal.example",
		RevalidateToken: "seat-to-mumble-token", RevalidateTimeoutMS: 1000,
	}
	invalidURL := base
	invalidURL.ServerURL = "javascript:alert(1)"
	if err := svc.UpdateMumbleConfig(invalidURL); err == nil {
		t.Fatal("invalid URL must fail")
	}
	sameTokens := base
	sameTokens.RevalidateToken = sameTokens.ServiceToken
	if err := svc.UpdateMumbleConfig(sameTokens); err == nil {
		t.Fatal("same directional tokens must fail")
	}
	invalidTimeout := base
	invalidTimeout.RevalidateTimeoutMS = 0
	if err := svc.UpdateMumbleConfig(invalidTimeout); err == nil {
		t.Fatal("zero timeout must fail")
	}
	invalidTemplate := base
	invalidTemplate.DisplayNameTemplate = "{unknown}"
	if err := svc.UpdateMumbleConfig(invalidTemplate); err == nil {
		t.Fatal("unknown display name placeholder must fail")
	}

	assertNodesRejected := func(mutate func(nodes []MumblePublicNode) []MumblePublicNode) {
		t.Helper()
		cfg := base
		cfg.PublicNodes = mutate([]MumblePublicNode{{Name: "node", Address: "mumble.example.com", Port: 64738}})
		if err := svc.UpdateMumbleConfig(cfg); err == nil {
			t.Fatal("invalid public nodes must fail")
		}
	}
	assertNodesRejected(func(nodes []MumblePublicNode) []MumblePublicNode {
		nodes[0].Address = "mumble.example.com/trailing"
		return nodes
	})
	assertNodesRejected(func(nodes []MumblePublicNode) []MumblePublicNode {
		nodes[0].Address = "mumble example.com"
		return nodes
	})
	assertNodesRejected(func(nodes []MumblePublicNode) []MumblePublicNode {
		nodes[0].Address = strings.Repeat("a", 254)
		return nodes
	})
	assertNodesRejected(func(nodes []MumblePublicNode) []MumblePublicNode {
		nodes[0].Address = "   "
		return nodes
	})
	assertNodesRejected(func(nodes []MumblePublicNode) []MumblePublicNode {
		nodes[0].Port = -1
		return nodes
	})
	assertNodesRejected(func(nodes []MumblePublicNode) []MumblePublicNode {
		nodes[0].Port = 65536
		return nodes
	})
	assertNodesRejected(func(nodes []MumblePublicNode) []MumblePublicNode {
		return append(nodes, MumblePublicNode{Name: "dup", Address: "mumble.example.com", Port: 64738})
	})
	assertNodesRejected(func(nodes []MumblePublicNode) []MumblePublicNode {
		nodes[0].Name = strings.Repeat("名", 22)
		return nodes
	})
	assertNodesRejected(func(nodes []MumblePublicNode) []MumblePublicNode {
		nodes[0].Description = strings.Repeat("述", 86)
		return nodes
	})
	assertNodesRejected(func(nodes []MumblePublicNode) []MumblePublicNode {
		for i := 0; i < maxMumblePublicNodes; i++ {
			nodes = append(nodes, MumblePublicNode{Address: "mumble.example.com", Port: 10000 + i})
		}
		return nodes
	})
}

func TestSysConfigServiceMigrateLegacyMumblePublicNodes(t *testing.T) {
	t.Run("folds legacy single node and deletes legacy keys", func(t *testing.T) {
		svc := setupMumbleConfigTest(t)
		if err := global.DB.Create(&[]model.SystemConfig{
			{Key: "mumble.public_address", Value: " mumble.example.com "},
			{Key: "mumble.public_port", Value: "64738"},
		}).Error; err != nil {
			t.Fatal(err)
		}
		svc.MigrateLegacyMumblePublicNodes()
		got := svc.GetMumbleConfig()
		want := []MumblePublicNode{{Address: "mumble.example.com", Port: 64738}}
		if !reflect.DeepEqual(got.PublicNodes, want) {
			t.Fatalf("migrated nodes = %+v", got.PublicNodes)
		}
		for _, key := range legacyMumblePublicKeys {
			if _, exists := svc.repo.GetIfExists(key); exists {
				t.Fatalf("legacy key %s must be deleted", key)
			}
		}
	})
	t.Run("idempotent on second run", func(t *testing.T) {
		svc := setupMumbleConfigTest(t)
		if err := global.DB.Create(&model.SystemConfig{Key: "mumble.public_address", Value: "mumble.example.com"}).Error; err != nil {
			t.Fatal(err)
		}
		svc.MigrateLegacyMumblePublicNodes()
		svc.MigrateLegacyMumblePublicNodes()
		got := svc.GetMumbleConfig()
		if len(got.PublicNodes) != 1 || got.PublicNodes[0].Address != "mumble.example.com" {
			t.Fatalf("second run must not duplicate nodes: %+v", got.PublicNodes)
		}
	})
	t.Run("keeps existing public nodes", func(t *testing.T) {
		svc := setupMumbleConfigTest(t)
		want := []MumblePublicNode{{Name: "node", Address: "hk.mumble.example.com", Port: 64738}}
		if err := svc.UpdateMumbleConfig(MumbleRuntimeConfig{RevalidateTimeoutMS: 1000, PublicNodes: want}); err != nil {
			t.Fatal(err)
		}
		if err := global.DB.Create(&model.SystemConfig{Key: "mumble.public_address", Value: "legacy.example.com"}).Error; err != nil {
			t.Fatal(err)
		}
		svc.MigrateLegacyMumblePublicNodes()
		got := svc.GetMumbleConfig()
		if !reflect.DeepEqual(got.PublicNodes, want) {
			t.Fatalf("existing nodes must stay untouched: %+v", got.PublicNodes)
		}
		if _, exists := svc.repo.GetIfExists("mumble.public_address"); exists {
			t.Fatal("legacy key must still be deleted")
		}
	})
}
