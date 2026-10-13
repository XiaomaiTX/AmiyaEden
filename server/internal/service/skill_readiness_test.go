package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

type readinessStoreStub struct {
	snapshot *repository.SkillReadinessSnapshot
	filter   repository.SkillReadinessFilter
	err      error
}

func (r *readinessStoreStub) Load(_ context.Context, filter repository.SkillReadinessFilter) (*repository.SkillReadinessSnapshot, error) {
	r.filter = filter
	return r.snapshot, r.err
}
func (r *readinessStoreStub) Options(context.Context) ([]int64, []model.SkillPlan, error) {
	return []int64{10}, []model.SkillPlan{{ID: 1, Title: "Corp plan"}}, r.err
}
func readinessFixture() *repository.SkillReadinessSnapshot {
	return &repository.SkillReadinessSnapshot{
		Plan:         model.SkillPlan{ID: 1, PlanScope: model.SkillPlanScopeCorp},
		Requirements: []model.SkillPlanSkill{{SkillTypeID: 100, RequiredLevel: 4}, {SkillTypeID: 101, RequiredLevel: 3}},
		Characters: []repository.SkillReadinessCharacter{
			{CharacterID: 1, UserID: 1, CorporationID: 10, Nickname: "A", TokenInvalid: true},
			{CharacterID: 2, UserID: 1, CorporationID: 20, Nickname: "A"},
			{CharacterID: 3, UserID: 2, CorporationID: 10, Nickname: "B"},
		},
		Snapshots: []model.EveCharacterSkill{{CharacterID: 1, UpdatedTime: 1000}, {CharacterID: 2, UpdatedTime: 2000}},
		Skills: []model.EveCharacterSkills{
			{CharacterID: 1, SkillID: 100, ActiveLevel: 2, TrainedLevel: 4},
			{CharacterID: 1, SkillID: 101, ActiveLevel: 3, TrainedLevel: 2},
			{CharacterID: 2, SkillID: 100, ActiveLevel: 1, TrainedLevel: 2},
		},
	}
}
func readinessTestService(store *readinessStoreStub) *SkillReadinessService {
	return &SkillReadinessService{
		repo:         store,
		resolveNames: func(context.Context, []int64) map[int64]string { return map[int64]string{10: "Corp X", 20: "Corp Y"} },
		loadTypes: func(ids []int, _ string) (map[int]repository.TypeInfo, error) {
			result := make(map[int]repository.TypeInfo)
			for _, id := range ids {
				result[id] = repository.TypeInfo{TypeID: id, TypeName: "Skill", GroupName: "Group"}
			}
			return result, nil
		},
	}
}

func TestSkillReadinessQueryAggregationAndPagination(t *testing.T) {
	store := &readinessStoreStub{snapshot: readinessFixture()}
	svc := readinessTestService(store)
	for _, page := range []int{1, 2, 3, int(^uint(0) >> 1)} {
		result, err := svc.Query(context.Background(), SkillReadinessRequest{PlanID: 1, Page: page, PageSize: 1, CorporationIDs: []int64{20, 10, 10}})
		if err != nil {
			t.Fatal(err)
		}
		want := SkillReadinessSummary{CharacterCount: 3, UserCount: 2, SatisfiedUserCount: 1, SatisfiedCharacterCount: 1, UnknownCharacterCount: 1}
		if result.Summary != want || result.Total != 2 {
			t.Fatalf("summary = %+v", result)
		}
		if page == 1 && (len(result.List) != 1 || result.List[0].UserID != 1 || result.List[0].CharacterCount != 2 || result.List[0].SatisfiedCharacterCount != 1) {
			t.Fatalf("first page = %+v", result.List)
		}
		if page == 2 && (len(result.List) != 1 || result.List[0].UserID != 2 || result.List[0].UnknownCharacterCount != 1) {
			t.Fatalf("second page = %+v", result.List)
		}
		if page > 2 && len(result.List) != 0 {
			t.Fatal("out-of-range page must be empty")
		}
		if !reflect.DeepEqual(store.filter.CorporationIDs, []int64{10, 20}) || store.filter.AllSkills {
			t.Fatalf("filter = %+v", store.filter)
		}
	}
}

func TestSkillReadinessNeverCombinesCharactersAndDistinguishesEmptySnapshot(t *testing.T) {
	snapshot := readinessFixture()
	snapshot.Skills = []model.EveCharacterSkills{
		{CharacterID: 1, SkillID: 100, TrainedLevel: 4},
		{CharacterID: 2, SkillID: 101, TrainedLevel: 3},
		// Orphan skill rows do not establish a successful snapshot.
		{CharacterID: 3, SkillID: 100, TrainedLevel: 5},
		{CharacterID: 3, SkillID: 101, TrainedLevel: 5},
	}
	rows := evaluateSkillReadiness(snapshot)
	if rows[0].Status != SkillReadinessUnsatisfied || rows[1].Status != SkillReadinessUnsatisfied || rows[2].Status != SkillReadinessUnknown {
		t.Fatalf("rows = %+v", rows)
	}
	result := aggregateSkillReadiness(snapshot, rows, SkillReadinessRequest{Page: 1, PageSize: 20})
	if result.Summary.SatisfiedUserCount != 0 {
		t.Fatal("skills from distinct characters were combined")
	}
	snapshot.Skills = nil
	rows = evaluateSkillReadiness(snapshot)
	if rows[0].Status != SkillReadinessUnsatisfied || rows[0].MatchedSkills != 0 || rows[2].Status != SkillReadinessUnknown {
		t.Fatalf("empty snapshots = %+v", rows)
	}
}

func TestSkillReadinessRequestAndPlanValidation(t *testing.T) {
	for _, req := range []SkillReadinessRequest{
		{}, {PlanID: 1, Page: -1}, {PlanID: 1, PageSize: -1}, {PlanID: 1, PageSize: 101}, {PlanID: 1, CorporationIDs: []int64{0}}, {PlanID: 1, CorporationIDs: []int64{-1}}, {PlanID: 1, Language: "xx"},
	} {
		if _, err := normalizeSkillReadinessRequest(req); !errors.Is(err, ErrSkillReadinessRequest) {
			t.Fatalf("accepted %+v", req)
		}
	}
	normalized, err := normalizeSkillReadinessRequest(SkillReadinessRequest{PlanID: 1})
	if err != nil || normalized.Page != 1 || normalized.PageSize != 20 || normalized.Language != "zh" {
		t.Fatalf("defaults = %+v, %v", normalized, err)
	}
	for _, scope := range []string{model.SkillPlanScopePersonal, model.SkillPlanScopeCorp} {
		snapshot := readinessFixture()
		snapshot.Plan.PlanScope = scope
		if scope == model.SkillPlanScopeCorp {
			snapshot.Requirements = nil
		}
		svc := readinessTestService(&readinessStoreStub{snapshot: snapshot})
		if _, err := svc.Query(context.Background(), SkillReadinessRequest{PlanID: 1}); !errors.Is(err, ErrSkillReadinessPlan) {
			t.Fatalf("accepted invalid plan: %v", err)
		}
	}
	store := &readinessStoreStub{err: gorm.ErrRecordNotFound}
	svc := readinessTestService(store)
	if _, err := svc.Query(context.Background(), SkillReadinessRequest{PlanID: 1}); !errors.Is(err, ErrSkillReadinessPlan) {
		t.Fatal(err)
	}
	store.err = errors.New("database unavailable")
	if _, err := svc.Query(context.Background(), SkillReadinessRequest{PlanID: 1}); !errors.Is(err, store.err) {
		t.Fatal("database failure swallowed")
	}
}

func TestSkillReadinessCharacterDetails(t *testing.T) {
	for _, known := range []bool{true, false} {
		snapshot := readinessFixture()
		snapshot.Characters = snapshot.Characters[:1]
		snapshot.Skills = snapshot.Skills[:1] // requirement 101 is missing
		if !known {
			snapshot.Snapshots = nil
		}
		store := &readinessStoreStub{snapshot: snapshot}
		svc := readinessTestService(store)
		result, err := svc.CharacterSkills(context.Background(), 1, SkillReadinessRequest{PlanID: 1, Language: "en"})
		if err != nil {
			t.Fatal(err)
		}
		if !store.filter.AllSkills || store.filter.CharacterID != 1 || result.Character.CorporationName != "Corp X" || !result.Character.TokenInvalid {
			t.Fatalf("detail = %+v", result)
		}
		if len(result.Requirements) != 2 {
			t.Fatal("missing requirement rows")
		}
		if known {
			if len(result.Skills) != 1 || *result.Requirements[0].CurrentLevel != 4 || !*result.Requirements[0].Satisfied || *result.Requirements[1].CurrentLevel != 0 || *result.Requirements[1].Satisfied {
				t.Fatalf("requirements = %+v", result.Requirements)
			}
		} else {
			if len(result.Skills) != 0 || result.Requirements[0].CurrentLevel != nil || result.Requirements[0].ActiveLevel != nil || result.Requirements[0].Satisfied != nil {
				t.Fatal("unknown skills fabricated as zero")
			}
		}
	}
	store := &readinessStoreStub{snapshot: readinessFixture()}
	store.snapshot.Characters = nil
	svc := readinessTestService(store)
	if _, err := svc.CharacterSkills(context.Background(), 5, SkillReadinessRequest{PlanID: 1}); !errors.Is(err, ErrSkillReadinessCharacter) {
		t.Fatal("out-of-scope character accepted")
	}
	rows, err := svc.UserCharacters(context.Background(), 1, SkillReadinessRequest{PlanID: 1, CorporationIDs: []int64{10}})
	if err != nil || len(rows) != 0 || store.filter.UserID != 1 || !reflect.DeepEqual(store.filter.CorporationIDs, []int64{10}) {
		t.Fatalf("user scope: %+v %v", rows, err)
	}
	result, err := svc.Query(context.Background(), SkillReadinessRequest{PlanID: 1})
	if err != nil || result.Total != 0 || result.List == nil || result.Summary != (SkillReadinessSummary{}) {
		t.Fatalf("empty scope = %+v %v", result, err)
	}
}

func TestSkillReadinessOptionsAndTypeFailure(t *testing.T) {
	store := &readinessStoreStub{snapshot: readinessFixture()}
	svc := readinessTestService(store)
	svc.resolveNames = func(context.Context, []int64) map[int64]string { return nil }
	options, err := svc.Options(context.Background())
	if err != nil || len(options.Plans) != 1 || options.Corporations[0].CorporationID != 10 || options.Corporations[0].CorporationName != "" {
		t.Fatalf("options = %+v %v", options, err)
	}
	store.snapshot.Characters = store.snapshot.Characters[:1]
	svc.loadTypes = func([]int, string) (map[int]repository.TypeInfo, error) { return nil, errors.New("SDE unavailable") }
	if _, err := svc.CharacterSkills(context.Background(), 1, SkillReadinessRequest{PlanID: 1}); err == nil {
		t.Fatal("type failure swallowed")
	}
}

func TestSkillReadinessDatabaseAcceptance(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:readiness_acceptance_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent), NamingStrategy: schema.NamingStrategy{SingularTable: true}})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.EveCharacter{}, &model.SkillPlan{}, &model.SkillPlanSkill{}, &model.EveCharacterSkill{}, &model.EveCharacterSkills{}); err != nil {
		t.Fatal(err)
	}
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })
	seeds := []interface{}{
		&[]model.User{{BaseModel: model.BaseModel{ID: 1}, Nickname: "A"}, {BaseModel: model.BaseModel{ID: 2}, Nickname: "B"}},
		&[]model.EveCharacter{{CharacterID: 1, UserID: 1, CorporationID: 10}, {CharacterID: 2, UserID: 1, CorporationID: 20}, {CharacterID: 3, UserID: 2, CorporationID: 10}},
		&model.SkillPlan{ID: 1, Title: "Plan", PlanScope: model.SkillPlanScopeCorp},
		&model.SkillPlanSkill{SkillPlanID: 1, SkillTypeID: 100, RequiredLevel: 4},
		&[]model.EveCharacterSkill{{CharacterID: 1, UpdatedTime: 1000}, {CharacterID: 2, UpdatedTime: 2000}},
		&[]model.EveCharacterSkills{{CharacterID: 1, SkillID: 100, TrainedLevel: 4}, {CharacterID: 2, SkillID: 100, TrainedLevel: 3}},
	}
	for _, seed := range seeds {
		if err := db.Create(seed).Error; err != nil {
			t.Fatal(err)
		}
	}
	svc := readinessTestService(&readinessStoreStub{})
	svc.repo = repository.NewSkillReadinessRepository()
	for _, ids := range [][]int64{{10}, {10, 20}, nil} {
		expectedCharacters := 3
		expectedACharacters := 2
		if len(ids) == 1 {
			expectedCharacters = 2
			expectedACharacters = 1
		}
		result, err := svc.Query(context.Background(), SkillReadinessRequest{PlanID: 1, CorporationIDs: ids, PageSize: 1})
		if err != nil {
			t.Fatal(err)
		}
		want := SkillReadinessSummary{CharacterCount: expectedCharacters, UserCount: 2, SatisfiedUserCount: 1, SatisfiedCharacterCount: 1, UnknownCharacterCount: 1}
		if result.Summary != want || result.List[0].CharacterCount != expectedACharacters || result.List[0].SatisfiedCharacterCount != 1 {
			t.Fatalf("corporations %v: %+v", ids, result)
		}
		rows, err := svc.UserCharacters(context.Background(), 1, SkillReadinessRequest{PlanID: 1, CorporationIDs: ids})
		if err != nil || len(rows) != expectedACharacters {
			t.Fatalf("user characters = %+v %v", rows, err)
		}
	}
	if _, err := svc.CharacterSkills(context.Background(), 2, SkillReadinessRequest{PlanID: 1, CorporationIDs: []int64{10}}); !errors.Is(err, ErrSkillReadinessCharacter) {
		t.Fatal("out-of-scope character accepted", err)
	}
}
