package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"amiya-eden/internal/utils"
	"slices"
	"testing"
)

func assertRecruiterRoles(t *testing.T, userID uint, expected ...string) {
	t.Helper()
	roles, err := repository.NewRoleRepository().GetUserRoleCodes(userID)
	if err != nil {
		t.Fatal(err)
	}
	for _, role := range expected {
		if !slices.Contains(roles, role) {
			t.Fatalf("roles %v missing %s", roles, role)
		}
	}
	if !slices.Contains(expected, model.RoleRecruiter) && slices.Contains(roles, model.RoleRecruiter) {
		t.Fatalf("stale recruiter retained: %v", roles)
	}
}

func TestRecruiterRoleIsGrantedAndRevokedWithoutChangingOtherRoles(t *testing.T) {
	db := admissionServiceTestDB(t)
	character := seedAdmissionUser(t, db, 1, model.AdmissionFMACorporationID, model.RoleFC)
	repo, svc := repository.NewAutoRoleRepository(), NewAutoRoleService()
	if err := repo.SyncCharacterCorpRoles(character.CharacterID, character.CorporationID, []string{"Director"}); err != nil {
		t.Fatal(err)
	}
	if err := svc.SyncUserAutoRoles(t.Context(), 1); err != nil {
		t.Fatal(err)
	}
	assertRecruiterRoles(t, 1, model.RoleFC, model.RoleRecruiter)
	if err := repo.SyncCharacterCorpRoles(character.CharacterID, character.CorporationID, nil); err != nil {
		t.Fatal(err)
	}
	if err := svc.SyncUserAutoRoles(t.Context(), 1); err != nil {
		t.Fatal(err)
	}
	assertRecruiterRoles(t, 1, model.RoleFC)
}

func TestNewAllowedDirectorIsPromotedToUserAlongsideRecruiter(t *testing.T) {
	db := admissionServiceTestDB(t)
	character := seedAdmissionUser(t, db, 1, model.SystemCorporationID, model.RoleGuest)
	if err := repository.NewAutoRoleRepository().SyncCharacterCorpRoles(character.CharacterID, character.CorporationID, []string{"Director"}); err != nil {
		t.Fatal(err)
	}
	if err := NewAutoRoleService().SyncUserAutoRoles(t.Context(), 1); err != nil {
		t.Fatal(err)
	}
	assertRecruiterRoles(t, 1, model.RoleUser, model.RoleRecruiter)
}

func TestNonMemberFMARecruiterDoesNotGainMembership(t *testing.T) {
	db := admissionServiceTestDB(t)
	if err := repository.NewSysConfigRepository().SetInt64Slice(model.SysConfigAllowCorporations, []int64{model.SystemCorporationID}, "test"); err != nil {
		t.Fatal(err)
	}
	utils.InvalidateAllowCorporationsCache()
	character := seedAdmissionUser(t, db, 1, model.AdmissionFMACorporationID, model.RoleGuest)
	if err := repository.NewAutoRoleRepository().SyncCharacterCorpRoles(character.CharacterID, character.CorporationID, []string{"Director"}); err != nil {
		t.Fatal(err)
	}
	if err := NewAutoRoleService().SyncUserAutoRoles(t.Context(), 1); err != nil {
		t.Fatal(err)
	}
	roles, err := repository.NewRoleRepository().GetUserRoleCodes(1)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(roles, []string{model.RoleRecruiter}) || model.HasNonGuestRole(roles) {
		t.Fatalf("non-member FMA recruiter gained membership: %v", roles)
	}
}

func TestRecruiterRoleIgnoresManualAssignmentAndMappings(t *testing.T) {
	db := admissionServiceTestDB(t)
	character := seedAdmissionUser(t, db, 1, model.AdmissionFMACorporationID, model.RoleUser)
	svc := NewRoleService()
	if err := svc.SetUserRoles(t.Context(), 99, []string{model.RoleSuperAdmin}, 1, []string{model.RoleUser, model.RoleRecruiter}); err != nil {
		t.Fatal(err)
	}
	assertRecruiterRoles(t, 1, model.RoleUser)
	if _, err := NewAutoRoleService().CreateEsiRoleMapping("Accountant", model.RoleRecruiter); err == nil {
		t.Fatal("manual role mapping accepted")
	}
	if _, err := NewAutoRoleService().CreateEsiTitleMapping(character.CorporationID, 1, "Title", model.RoleRecruiter); err == nil {
		t.Fatal("manual title mapping accepted")
	}
	if err := repository.NewAutoRoleRepository().SyncCharacterCorpRoles(character.CharacterID, character.CorporationID, []string{"Personnel_Manager"}); err != nil {
		t.Fatal(err)
	}
	if err := svc.SetUserRoles(t.Context(), 99, []string{model.RoleSuperAdmin}, 1, []string{model.RoleUser}); err != nil {
		t.Fatal(err)
	}
	assertRecruiterRoles(t, 1, model.RoleUser, model.RoleRecruiter)
	if err := NewSysConfigService().UpdateAllowCorporations([]int64{model.SystemCorporationID}); err != nil {
		t.Fatal(err)
	}
	assertRecruiterRoles(t, 1, model.RoleUser, model.RoleRecruiter)
	context, err := NewAdmissionService().Context(1)
	if err != nil || context.IsMember || !slices.Equal(context.ReviewableCorporationIDs, []int64{model.AdmissionFMACorporationID}) {
		t.Fatalf("review scope should be independent of membership allowlist: %+v %v", context, err)
	}
}

func TestRecruiterRoleRevokedAfterCharacterUnbind(t *testing.T) {
	db := admissionServiceTestDB(t)
	seedAdmissionUser(t, db, 1, model.SystemCorporationID, model.RoleUser)
	character := model.EveCharacter{CharacterID: 9200, CharacterName: "Recruiter alt", UserID: 1, CorporationID: model.AdmissionFMACorporationID}
	if err := db.Create(&character).Error; err != nil {
		t.Fatal(err)
	}
	if err := repository.NewAutoRoleRepository().SyncCharacterCorpRoles(character.CharacterID, character.CorporationID, []string{"Director"}); err != nil {
		t.Fatal(err)
	}
	if err := NewAutoRoleService().SyncRecruiterRole(t.Context(), 1); err != nil {
		t.Fatal(err)
	}
	assertRecruiterRoles(t, 1, model.RoleUser, model.RoleRecruiter)
	if err := NewEveSSOService().UnbindCharacter(1, character.CharacterID); err != nil {
		t.Fatal(err)
	}
	assertRecruiterRoles(t, 1, model.RoleUser)
}
