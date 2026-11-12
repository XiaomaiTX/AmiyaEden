package service

import (
	"amiya-eden/internal/model"
	"testing"
	"time"
)

func TestAdmissionBadgeCountsOnlyReviewablePendingApplications(t *testing.T) {
	db := admissionApplicationDB(t, false)
	fma := seedAdmissionUser(t, db, 1, model.AdmissionFMACorporationID, model.RoleRecruiter)
	fuxi := seedAdmissionUser(t, db, 2, model.SystemCorporationID, model.RoleUser)
	seedAdmissionUser(t, db, 3, 1001, model.RoleRecruiter) // A stale label grants nothing.
	seedAdmissionUser(t, db, 4, 1001, model.RoleAdmin)
	seedAdmissionUser(t, db, 5, 1001, model.RoleSuperAdmin)
	for _, c := range []model.EveCharacter{fma, fuxi} {
		if err := db.Create(&model.EveCharacterCorpRole{CharacterID: c.CharacterID, CorporationID: c.CorporationID, CorpRole: "Personnel_Manager"}).Error; err != nil {
			t.Fatal(err)
		}
	}
	for i, corp := range []int64{model.SystemCorporationID, model.SystemCorporationID, model.AdmissionFMACorporationID, model.AdmissionFMACorporationID, model.AdmissionFMACorporationID} {
		status := model.AdmissionStatusPending
		if i == 3 {
			status = model.AdmissionStatusInvited
		}
		if i == 4 {
			status = model.AdmissionStatusRejected
		}
		if err := db.Create(&model.AdmissionApplication{UserID: uint(100 + i), Kind: model.AdmissionKindAlt, TargetCorporationID: corp, Status: status, SubmittedAt: time.Now()}).Error; err != nil {
			t.Fatal(err)
		}
	}
	svc := NewAdmissionService()
	for id, want := range map[uint]int64{1: 1, 2: 2, 3: 0, 4: 3, 5: 3} {
		count, err := svc.PendingReviewCount(id)
		if err != nil || count != want {
			t.Fatalf("user %d: count %d want %d, err %v", id, count, want, err)
		}
	}
	if err := db.Model(&model.AdmissionApplication{}).Where("target_corporation_id = ?", model.AdmissionFMACorporationID).Update("status", model.AdmissionStatusInvited).Error; err != nil {
		t.Fatal(err)
	}
	if count, err := svc.PendingReviewCount(1); err != nil || count != 0 {
		t.Fatalf("FMA reviewer saw Fuxi-only queue: %d %v", count, err)
	}
	// Authority is re-evaluated even if the user's role label has not changed.
	if err := db.Where("character_id = ?", fuxi.CharacterID).Delete(&model.EveCharacterCorpRole{}).Error; err != nil {
		t.Fatal(err)
	}
	if count, err := svc.PendingReviewCount(2); err != nil || count != 0 {
		t.Fatalf("revoked authority: %d %v", count, err)
	}
}
