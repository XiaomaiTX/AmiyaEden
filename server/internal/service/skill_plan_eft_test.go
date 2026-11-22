package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"testing"
)

func TestSkillPlanEFTImport(t *testing.T) {
	db := newServiceErrorMessageTestDB(t, &model.SkillPlan{}, &model.SkillPlanSkill{})
	useServiceErrorMessageTestDB(t, db)
	for _, sql := range []string{
		"CREATE TABLE invtypes (typeid INTEGER, typename TEXT)",
		"CREATE TABLE dgmtypeattributes (typeid INTEGER, attributeid INTEGER, valueint INTEGER)",
		"INSERT INTO invtypes VALUES (1, 'Flycatcher'), (2, 'Launcher'), (3, 'Probe')",
		"INSERT INTO dgmtypeattributes VALUES (1,182,10),(1,277,1),(2,182,20),(2,277,2),(3,182,20),(3,277,3),(20,182,30),(20,277,4)",
	} {
		if err := db.Exec(sql).Error; err != nil {
			t.Fatal(err)
		}
	}
	svc := NewSkillPlanService()
	skills, err := svc.normalizeSkillPlanInputs(nil, "[Flycatcher, Test]\r\n\r\nLauncher, Probe\r\n\r\nProbe x16", "en")
	if err != nil {
		t.Fatal(err)
	}
	levels := map[int]int{}
	for _, skill := range skills {
		levels[skill.SkillTypeID] = skill.RequiredLevel
	}
	if len(levels) != 3 || levels[10] != 1 || levels[20] != 3 || levels[30] != 4 {
		t.Fatalf("requirements = %v", levels)
	}
	if _, err := svc.parseSkillPlanEFT("[Flycatcher, Test]\nUnknown x1"); err == nil {
		t.Fatal("unknown item accepted")
	}
	if _, err := parseSkillPlanEFTNames("[invalid]"); err == nil {
		t.Fatal("invalid header accepted")
	}
	entries, err := parseSkillPlanEFTNames("[Flycatcher, Test]\n[Empty High slot]\nLauncher /OFFLINE\nLauncher, Probe /offline\nProbe x16")
	if err != nil || len(entries) != 5 || entries[1].name != "Launcher" || entries[2].name != "Launcher" || entries[3].name != "Probe" {
		t.Fatalf("empty slot and offline parsing %v %v", entries, err)
	}
}

func TestDisabledSkillPlanVisibility(t *testing.T) {
	db := newServiceErrorMessageTestDB(t, &model.SkillPlan{}, &model.SkillPlanSkill{})
	useServiceErrorMessageTestDB(t, db)
	plans := []model.SkillPlan{
		{Title: "Active", PlanScope: model.SkillPlanScopeCorp, CreatedBy: 1},
		{Title: "Hidden", PlanScope: model.SkillPlanScopeCorp, CreatedBy: 1, Disabled: true},
		{Title: "Personal", PlanScope: model.SkillPlanScopePersonal, CreatedBy: 2},
		{Title: "Other", PlanScope: model.SkillPlanScopePersonal, CreatedBy: 3},
	}
	if err := db.Create(&plans).Error; err != nil {
		t.Fatal(err)
	}
	repo := repository.NewSkillPlanRepository()
	for _, manager := range []bool{false, true} {
		rows, total, err := repo.ListByScope(1, 10, "", model.SkillPlanScopeCorp, nil, manager)
		expected := int64(1)
		if manager {
			expected = 2
		}
		if err != nil || total != expected || int64(len(rows)) != expected {
			t.Fatalf("list manager=%v rows=%v total=%v err=%v", manager, rows, total, err)
		}
	}
	visible, err := repo.ListVisibleForUser(2)
	if err != nil || len(visible) != 2 {
		t.Fatalf("visible = %v %v", visible, err)
	}
	svc := NewSkillPlanService()
	if _, err := svc.GetSkillPlan(plans[1].ID, "en", nil); err == nil {
		t.Fatal("member can read disabled plan")
	}
	if !canReadDisabledSkillPlans([]string{model.RoleAdmin}) || canReadDisabledSkillPlans([]string{model.RoleUser}) {
		t.Fatal("role visibility mismatch")
	}
	if err := db.Model(&plans[1]).Update("disabled", false).Error; err != nil {
		t.Fatal(err)
	}
	visible, err = repo.ListVisibleForUser(2)
	if err != nil || len(visible) != 3 {
		t.Fatalf("restored = %v %v", visible, err)
	}
}
