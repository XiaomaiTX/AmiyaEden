package esi

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/utils"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestCorpRolesSnapshotProvenanceAndFailedSync(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:corp_roles_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.EveCharacter{}, &model.EveCharacterCorpRole{}, &model.SystemConfig{}); err != nil {
		t.Fatal(err)
	}
	oldDB, oldLogger, oldHook := global.DB, global.Logger, OnCorporationAuthorityChanged
	global.DB = db
	global.SetLogger(zap.NewNop())
	utils.InvalidateAllowCorporationsCache()
	t.Cleanup(func() {
		global.DB = oldDB
		global.SetLogger(oldLogger)
		OnCorporationAuthorityChanged = oldHook
		utils.InvalidateAllowCorporationsCache()
	})
	character := model.EveCharacter{CharacterID: 9001, CharacterName: "Recruiter", UserID: 1, CorporationID: model.SystemCorporationID}
	if err := db.Create(&character).Error; err != nil {
		t.Fatal(err)
	}
	status, changed := http.StatusOK, 0
	OnCorporationAuthorityChanged = func(_ context.Context, ids []int64) { changed += len(ids) }
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"roles":["Personnel_Manager"],"roles_at_hq":["Director"]}`))
	}))
	defer server.Close()
	task := &CorpRolesTask{}
	ctx := &TaskContext{CharacterID: character.CharacterID, Client: NewClientWithConfig(server.URL, "")}
	if err := task.Execute(ctx); err != nil {
		t.Fatal(err)
	}
	status = http.StatusForbidden
	if err := task.Execute(ctx); err == nil {
		t.Fatal("failed ESI request reported success")
	}
	var roles []model.EveCharacterCorpRole
	if err := db.Find(&roles).Error; err != nil {
		t.Fatal(err)
	}
	if changed != 1 || len(roles) != 2 {
		t.Fatalf("failed sync changed authority: hooks=%d roles=%+v", changed, roles)
	}
	for _, role := range roles {
		if role.CorporationID != model.SystemCorporationID {
			t.Fatalf("wrong source: %+v", role)
		}
	}
	interval := task.Interval()
	if interval.Active != 24*time.Hour || interval.Inactive != 24*time.Hour {
		t.Fatal("role sync cadence changed")
	}
	// FMA remains a recruitment target even if it is not in the member allowlist.
	if err := db.Model(&character).Update("corporation_id", model.AdmissionFMACorporationID).Error; err != nil {
		t.Fatal(err)
	}
	status = http.StatusOK
	if err := task.Execute(ctx); err != nil {
		t.Fatal(err)
	}
	if err := db.Find(&roles).Error; err != nil {
		t.Fatal(err)
	}
	if len(roles) != 2 || roles[0].CorporationID != model.AdmissionFMACorporationID || roles[1].CorporationID != model.AdmissionFMACorporationID {
		t.Fatalf("FMA authority snapshot missing: %+v", roles)
	}
}
