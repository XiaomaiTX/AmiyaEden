package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func readinessDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:readiness_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent), NamingStrategy: schema.NamingStrategy{SingularTable: true}})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.EveCharacter{}, &model.SkillPlan{}, &model.SkillPlanSkill{}, &model.EveCharacterSkill{}, &model.EveCharacterSkills{}); err != nil {
		t.Fatal(err)
	}
	old := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = old })
	return db
}

func seedReadinessDB(t *testing.T, db *gorm.DB) {
	t.Helper()
	rows := []interface{}{
		&[]model.User{{BaseModel: model.BaseModel{ID: 1}, Nickname: "A", Status: 0}, {BaseModel: model.BaseModel{ID: 2}, Nickname: "B"}, {BaseModel: model.BaseModel{ID: 3}, Nickname: "Deleted"}},
		&[]model.EveCharacter{
			{CharacterID: 1, UserID: 1, CorporationID: 10, CharacterName: "A-X"},
			{CharacterID: 2, UserID: 1, CorporationID: 20, CharacterName: "A-Y"},
			{CharacterID: 3, UserID: 2, CorporationID: 10, CharacterName: "B-X"},
			{CharacterID: 4, UserID: 3, CorporationID: 30, CharacterName: "Deleted user"},
			{CharacterID: 5, UserID: 99, CorporationID: 40, CharacterName: "Orphan"},
			{CharacterID: 6, UserID: 2, CorporationID: 0, CharacterName: "Unknown corp"},
			{CharacterID: 7, UserID: 2, CorporationID: 50, CharacterName: "Deleted character"},
		},
		&[]model.SkillPlan{{ID: 1, Title: "Corp", PlanScope: model.SkillPlanScopeCorp}, {ID: 2, Title: "Personal", PlanScope: model.SkillPlanScopePersonal}},
		&model.SkillPlanSkill{SkillPlanID: 1, SkillTypeID: 100, RequiredLevel: 4},
		&[]model.EveCharacterSkill{{CharacterID: 1, UpdatedTime: 1000}, {CharacterID: 2, UpdatedTime: 2000}},
		&[]model.EveCharacterSkills{{CharacterID: 1, SkillID: 100, TrainedLevel: 4}, {CharacterID: 1, SkillID: 101, TrainedLevel: 5}, {CharacterID: 2, SkillID: 100, TrainedLevel: 2}, {CharacterID: 99, SkillID: 100, TrainedLevel: 5}},
	}
	for _, row := range rows {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Delete(&model.User{}, 3).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("character_id = ?", 7).Delete(&model.EveCharacter{}).Error; err != nil {
		t.Fatal(err)
	}
}

func TestSkillReadinessRepositoryScopeAndOptions(t *testing.T) {
	db := readinessDB(t)
	seedReadinessDB(t, db)
	repo := NewSkillReadinessRepository()
	cases := []struct {
		name               string
		filter             SkillReadinessFilter
		characters, skills int
	}{
		{"all", SkillReadinessFilter{PlanID: 1}, 4, 2},
		{"single corp", SkillReadinessFilter{PlanID: 1, CorporationIDs: []int64{10}}, 2, 1},
		{"multiple corps", SkillReadinessFilter{PlanID: 1, CorporationIDs: []int64{10, 20}}, 3, 2},
		{"user within corp", SkillReadinessFilter{PlanID: 1, UserID: 1, CorporationIDs: []int64{10}}, 1, 1},
		{"excluded user", SkillReadinessFilter{PlanID: 1, UserID: 3}, 0, 0},
		{"wrong corp character", SkillReadinessFilter{PlanID: 1, CharacterID: 2, CorporationIDs: []int64{10}}, 0, 0},
		{"full skills", SkillReadinessFilter{PlanID: 1, CharacterID: 1, AllSkills: true}, 1, 2},
		{"empty", SkillReadinessFilter{PlanID: 1, CorporationIDs: []int64{999}}, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			snapshot, err := repo.Load(context.Background(), tc.filter)
			if err != nil {
				t.Fatal(err)
			}
			if len(snapshot.Characters) != tc.characters || len(snapshot.Skills) != tc.skills {
				t.Fatalf("characters=%d skills=%d", len(snapshot.Characters), len(snapshot.Skills))
			}
		})
	}
	ids, plans, err := repo.Options(context.Background())
	if err != nil || len(ids) != 2 || ids[0] != 10 || ids[1] != 20 || len(plans) != 1 || plans[0].ID != 1 {
		t.Fatalf("options: %v %+v %v", ids, plans, err)
	}
	if err := db.Unscoped().Where("character_id = ?", 1).Delete(&model.EveCharacter{}).Error; err != nil {
		t.Fatal(err)
	}
	snapshot, err := repo.Load(context.Background(), SkillReadinessFilter{PlanID: 1, CharacterID: 1, AllSkills: true})
	if err != nil || len(snapshot.Skills) != 0 || len(snapshot.Snapshots) != 0 {
		t.Fatalf("unbound snapshot = %+v %v", snapshot, err)
	}
}

func TestSkillReadinessRepositoryBatchQueryCount(t *testing.T) {
	db := readinessDB(t)
	if err := db.Create(&model.User{BaseModel: model.BaseModel{ID: 1}}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.SkillPlan{ID: 1, Title: "Corp", PlanScope: model.SkillPlanScopeCorp}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.SkillPlanSkill{SkillPlanID: 1, SkillTypeID: 100, RequiredLevel: 1}).Error; err != nil {
		t.Fatal(err)
	}
	characters := make([]model.EveCharacter, readinessBatchSize+1)
	for i := range characters {
		characters[i] = model.EveCharacter{CharacterID: int64(i + 1), UserID: 1, CorporationID: 10}
	}
	if err := db.CreateInBatches(characters, 100).Error; err != nil {
		t.Fatal(err)
	}
	queries := 0
	if err := db.Callback().Query().Before("gorm:query").Register("readiness_count", func(*gorm.DB) { queries++ }); err != nil {
		t.Fatal(err)
	}
	snapshot, err := NewSkillReadinessRepository().Load(context.Background(), SkillReadinessFilter{PlanID: 1})
	if err != nil || len(snapshot.Characters) != len(characters) || queries != 6 {
		t.Fatalf("batch query count=%d snapshot characters=%d err=%v", queries, len(snapshot.Characters), err)
	}
	// Character Scan is a Row callback; six Query calls are plan + requirements + two batches of summary/skills.
}

func TestSkillReadinessRepositoryErrors(t *testing.T) {
	db := readinessDB(t)
	if _, err := NewSkillReadinessRepository().Load(context.Background(), SkillReadinessFilter{PlanID: 99}); err != gorm.ErrRecordNotFound {
		t.Fatalf("missing plan = %v", err)
	}
	if err := db.Migrator().DropTable(&model.User{}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := NewSkillReadinessRepository().Options(context.Background()); err == nil {
		t.Fatal("query failure swallowed")
	}
}
