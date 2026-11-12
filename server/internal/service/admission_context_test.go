package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"amiya-eden/internal/utils"
	"slices"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func admissionServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := newServiceTestDB(t, "admission", &model.User{}, &model.EveCharacter{}, &model.UserRole{}, &model.EveCharacterCorpRole{}, &model.SystemConfig{}, &model.EsiRoleMapping{}, &model.EsiTitleMapping{}, &model.EveCharacterTitle{}, &model.AuditEvent{})
	oldDB, oldRedis, oldLogger := global.DB, global.Redis, global.Logger
	global.DB = db
	global.Redis = redis.NewClient(&redis.Options{Addr: miniredis.RunT(t).Addr()})
	global.SetLogger(zap.NewNop())
	t.Cleanup(func() {
		_ = global.Redis.Close()
		global.DB, global.Redis = oldDB, oldRedis
		global.SetLogger(oldLogger)
		utils.InvalidateAllowCorporationsCache()
	})
	utils.InvalidateAllowCorporationsCache()
	if err := repository.NewSysConfigRepository().SetInt64Slice(model.SysConfigAllowCorporations, []int64{model.SystemCorporationID, model.AdmissionFMACorporationID}, "test"); err != nil {
		t.Fatal(err)
	}
	return db
}

func seedAdmissionUser(t *testing.T, db *gorm.DB, id uint, corporationID int64, role string) model.EveCharacter {
	t.Helper()
	character := model.EveCharacter{CharacterID: 9000 + int64(id), CharacterName: "Pilot", UserID: id, CorporationID: corporationID}
	for _, item := range []interface{}{
		&model.User{BaseModel: model.BaseModel{ID: id}, Nickname: "Applicant", DiscordID: "discord-only", PrimaryCharacterID: character.CharacterID, Role: role},
		&character, &model.UserRole{UserID: id, RoleCode: role},
	} {
		if err := db.Create(item).Error; err != nil {
			t.Fatal(err)
		}
	}
	return character
}

func TestAdmissionContextUsesAnyBoundCharacterAndDiscord(t *testing.T) {
	db := admissionServiceTestDB(t)
	seedAdmissionUser(t, db, 1, 1001, model.RoleGuest)
	svc := NewAdmissionService()
	context, err := svc.Context(1)
	if err != nil || !context.CanCreateNewMember || context.CanCreateAlt || context.QQ != "" || context.DiscordID == "" {
		t.Fatalf("Discord-only guest cannot apply: %+v %v", context, err)
	}
	if err := db.Create(&model.EveCharacter{CharacterID: 9100, CharacterName: "Member alternate", UserID: 1, CorporationID: model.AdmissionFMACorporationID}).Error; err != nil {
		t.Fatal(err)
	}
	context, err = svc.Context(1)
	if err != nil || !context.IsMember || context.CanCreateNewMember || !context.CanCreateAlt {
		t.Fatalf("bound alternate did not establish membership: %+v %v", context, err)
	}
}

func TestAdmissionReviewerScopeRequiresMatchingProvenance(t *testing.T) {
	db := admissionServiceTestDB(t)
	character := seedAdmissionUser(t, db, 1, model.AdmissionFMACorporationID, model.RoleRecruiter)
	svc := NewAdmissionService()
	for _, provenance := range []int64{0, model.SystemCorporationID, model.AdmissionFMACorporationID} {
		if err := db.Where("character_id = ?", character.CharacterID).Delete(&model.EveCharacterCorpRole{}).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&model.EveCharacterCorpRole{CharacterID: character.CharacterID, CorporationID: provenance, CorpRole: "Personnel_Manager"}).Error; err != nil {
			t.Fatal(err)
		}
		scope, err := svc.reviewableCorporations(db, 1)
		if err != nil {
			t.Fatal(err)
		}
		if provenance == model.AdmissionFMACorporationID {
			if !slices.Equal(scope, []int64{model.AdmissionFMACorporationID}) {
				t.Fatalf("wrong scope: %v", scope)
			}
		} else if len(scope) != 0 {
			t.Fatalf("unknown or mismatched provenance grants scope: %v", scope)
		}
	}
	for index, role := range []string{model.RoleAdmin, model.RoleSuperAdmin} {
		id := uint(index + 2)
		seedAdmissionUser(t, db, id, 1001, role)
		scope, err := svc.reviewableCorporations(db, id)
		if err != nil || !slices.Equal(scope, []int64{model.SystemCorporationID, model.AdmissionFMACorporationID}) {
			t.Fatalf("administrator scope: %v %v", scope, err)
		}
	}
}
