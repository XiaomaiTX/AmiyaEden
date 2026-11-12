package bootstrap

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/testutil"
	"testing"
	"time"
)

func TestAdmissionMigrationPostgres(t *testing.T) {
	db := testutil.Postgres(t)
	if err := migrateAdmissions(db); err != nil {
		t.Fatal(err)
	}
	application := model.AdmissionApplication{
		UserID: 1, Kind: model.AdmissionKindNewMember, TargetCorporationID: model.SystemCorporationID,
		Status: model.AdmissionStatusPending, ApplicantName: "Existing applicant", SubmittedAt: time.Now(),
	}
	if err := db.Create(&application).Error; err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := migrateAdmissions(db); err != nil {
			t.Fatal(err)
		}
	}
	var kept model.AdmissionApplication
	if err := db.First(&kept, application.ID).Error; err != nil {
		t.Fatal(err)
	}
	if kept.ApplicantName != application.ApplicantName || kept.Version != 1 {
		t.Fatalf("migration altered existing record: %+v", kept)
	}
	duplicate := application
	duplicate.ID = 0
	if err := db.Create(&duplicate).Error; err == nil {
		t.Fatal("duplicate pending new-member application accepted")
	}
	duplicate.Kind = model.AdmissionKindAlt
	if err := db.Create(&duplicate).Error; err != nil {
		t.Fatal("new-member uniqueness incorrectly blocks alternate application:", err)
	}
	for _, table := range []string{"admission_application", "admission_character", "admission_character_claim", "admission_evidence", "admission_processing_event", "admission_salary"} {
		if !db.Migrator().HasTable(table) {
			t.Fatalf("missing table %s", table)
		}
	}
}
