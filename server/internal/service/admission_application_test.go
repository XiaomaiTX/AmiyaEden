package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/testutil"
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm"
)

func admissionApplicationDB(t *testing.T, postgres bool) *gorm.DB {
	t.Helper()
	db := admissionServiceTestDB(t)
	if postgres {
		db = testutil.Postgres(t)
		global.DB = db
		if err := db.AutoMigrate(&model.User{}, &model.EveCharacter{}, &model.UserRole{}, &model.EveCharacterCorpRole{}, &model.SystemConfig{}); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.AutoMigrate(&model.AdmissionApplication{}, &model.AdmissionCharacter{}, &model.AdmissionCharacterClaim{}, &model.AdmissionEvidence{}, &model.AdmissionProcessingEvent{}, &model.AdmissionSalary{}, &model.SystemWallet{}, &model.WalletTransaction{}, &model.AuditEvent{}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(clearCorpPolicyCache)
	return db
}

func admissionImages(t *testing.T) map[string][]byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 1600, 900)), nil); err != nil {
		t.Fatal(err)
	}
	return map[string][]byte{model.AdmissionEvidenceLauncher: buf.Bytes(), model.AdmissionEvidenceSeatProfile: buf.Bytes()}
}

func newAdmissionInput(chars ...int64) AdmissionWriteInput {
	return AdmissionWriteInput{Kind: model.AdmissionKindNewMember, TargetCorporationID: model.SystemCorporationID, CharacterIDs: chars}
}

func TestAdmissionApplicationLifecycle(t *testing.T) {
	db := admissionApplicationDB(t, false)
	char := seedAdmissionUser(t, db, 1, 1001, model.RoleGuest)
	seedAdmissionUser(t, db, 2, model.SystemCorporationID, model.RoleAdmin)
	if err := db.Create(&model.EveCharacter{CharacterID: 9991, CharacterName: "Alternate", UserID: 1, CorporationID: 1001}).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewAdmissionService()
	input := newAdmissionInput(char.CharacterID, 9991)
	if _, err := svc.Create(1, input, nil); !errors.Is(err, ErrAdmissionEvidence) {
		t.Fatalf("missing evidence: %v", err)
	}
	var count int64
	db.Model(&model.AdmissionApplication{}).Count(&count)
	if count != 0 {
		t.Fatal("partial create")
	}
	app, err := svc.Create(1, input, admissionImages(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(app.Characters) != 2 || app.QQ != "" || app.DiscordID == "" {
		t.Fatalf("whole Discord-only application: %+v", app)
	}
	if _, err = svc.Create(1, input, admissionImages(t)); !errors.Is(err, ErrAdmissionConflict) {
		t.Fatalf("duplicate: %v", err)
	}
	if _, err = svc.Detail(999, app.ID); !errors.Is(err, ErrAdmissionDenied) {
		t.Fatalf("cross user detail: %v", err)
	}
	if _, err = svc.Evidence(999, app.ID, "launcher"); !errors.Is(err, ErrAdmissionDenied) {
		t.Fatal("cross user evidence")
	}
	if _, err = svc.Process(2, app.ID, 1, false, " "); !errors.Is(err, ErrAdmissionInvalid) {
		t.Fatal("empty rejection")
	}
	app, err = svc.Process(2, app.ID, 1, false, "Bind every character")
	if err != nil {
		t.Fatal(err)
	}
	deadline := *app.EvidenceExpiresAt
	input.ExpectedVersion = app.Version
	app, err = svc.Edit(1, app.ID, input, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if !app.EvidenceExpiresAt.Equal(deadline) {
		t.Fatal("edit extended retention")
	}
	if _, err = svc.Edit(1, app.ID, input, nil, true); !errors.Is(err, ErrAdmissionConflict) {
		t.Fatal("stale version accepted")
	}
	input.ExpectedVersion = app.Version
	app, err = svc.Edit(1, app.ID, input, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if app.ID != 1 || app.Status != model.AdmissionStatusPending || app.EvidenceExpiresAt != nil {
		t.Fatal("resubmit did not preserve id/reset deadline")
	}
	// Becoming a member in-game must not prevent registering the original order.
	db.Model(&model.EveCharacter{}).Where("user_id = ?", 1).Update("corporation_id", model.SystemCorporationID)
	app, err = svc.Process(2, app.ID, app.Version, true, "")
	if err != nil {
		t.Fatal(err)
	}
	if app.Status != model.AdmissionStatusInvited {
		t.Fatal("not closed")
	}
	if _, err = svc.Process(2, app.ID, app.Version, true, ""); !errors.Is(err, ErrAdmissionConflict) {
		t.Fatal("repeat paid")
	}
	input.ExpectedVersion = app.Version
	if _, err = svc.Edit(1, app.ID, input, nil, false); !errors.Is(err, ErrAdmissionInvalid) {
		t.Fatal("closed editable")
	}
	detail, err := svc.Detail(1, app.ID)
	if err != nil || detail.Salary == nil || detail.Salary.Amount != 10 || len(detail.Events) != 5 {
		t.Fatalf("history/salary: %+v %v", detail, err)
	}
	if _, err = svc.CleanupEvidence(deadline.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
}

func TestAdmissionWalletPolicyAndRequiredAuditRollback(t *testing.T) {
	for _, role := range []string{model.RoleRecruiter, model.RoleAdmin, model.RoleSuperAdmin} {
		t.Run(role, func(t *testing.T) {
			db := admissionApplicationDB(t, false)
			char := seedAdmissionUser(t, db, 1, 1001, model.RoleGuest)
			actor := seedAdmissionUser(t, db, 2, 2002, role)
			if role == model.RoleRecruiter {
				db.Model(&model.EveCharacter{}).Where("character_id = ?", actor.CharacterID).Update("corporation_id", model.SystemCorporationID)
				actor.CorporationID = model.SystemCorporationID
				db.Create(&model.EveCharacterCorpRole{CharacterID: actor.CharacterID, CorporationID: actor.CorporationID, CorpRole: "Director"})
			}
			setWalletCapabilityPolicy(t, db, actor.CorporationID, false)
			svc := NewAdmissionService()
			app, err := svc.Create(1, newAdmissionInput(char.CharacterID), admissionImages(t))
			if err != nil {
				t.Fatal(err)
			}
			if _, err = svc.Process(2, app.ID, 1, true, ""); !errors.Is(err, ErrAdmissionWalletDisabled) {
				t.Fatalf("wallet policy bypass %s: %v", role, err)
			}
			var count int64
			db.Model(&model.AdmissionSalary{}).Count(&count)
			if count != 0 {
				t.Fatal("salary committed")
			}
			db.First(app, app.ID)
			if app.Status != model.AdmissionStatusPending || app.Version != 1 {
				t.Fatal("disabled closure committed")
			}
			setWalletCapabilityPolicy(t, db, actor.CorporationID, true)
			if err = db.Migrator().DropTable(&model.AuditEvent{}); err != nil {
				t.Fatal(err)
			}
			if _, err = svc.Process(2, app.ID, 1, true, ""); !errors.Is(err, ErrAdmissionInternal) {
				t.Fatalf("missing audit allowed closure: %v", err)
			}
			db.Model(&model.WalletTransaction{}).Count(&count)
			if count != 0 {
				t.Fatal("wallet not rolled back")
			}
			if err = db.AutoMigrate(&model.AuditEvent{}); err != nil {
				t.Fatal(err)
			}
			if _, err = svc.Process(2, app.ID, 1, true, ""); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAdmissionReviewerCannotProcessOwnApplication(t *testing.T) {
	db := admissionApplicationDB(t, false)
	char := seedAdmissionUser(t, db, 2, model.SystemCorporationID, model.RoleUser)
	if err := db.Create(&model.EveCharacterCorpRole{CharacterID: char.CharacterID, CorporationID: char.CorporationID, CorpRole: "Personnel_Manager"}).Error; err != nil {
		t.Fatal(err)
	}
	app := model.AdmissionApplication{UserID: 2, Kind: model.AdmissionKindAlt, TargetCorporationID: model.SystemCorporationID, Status: model.AdmissionStatusPending, Version: 1, SubmittedAt: time.Now()}
	if err := db.Create(&app).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewAdmissionService()
	if _, err := svc.Process(2, app.ID, 1, true, ""); !errors.Is(err, ErrAdmissionDenied) {
		t.Fatalf("self invite: %v", err)
	}
	if _, err := svc.Process(2, app.ID, 1, false, "Self reject"); !errors.Is(err, ErrAdmissionDenied) {
		t.Fatalf("self reject: %v", err)
	}
	detail, err := svc.Detail(2, app.ID)
	if err != nil || detail.CanReview || !detail.IsOwner {
		t.Fatalf("owner detail must not offer review actions: %+v %v", detail, err)
	}
	if count, err := svc.PendingReviewCount(2); err != nil || count != 0 {
		t.Fatalf("own application counted as reviewable: %d %v", count, err)
	}
	var salaries int64
	db.Model(&model.AdmissionSalary{}).Count(&salaries)
	if salaries != 0 {
		t.Fatal("self review paid salary")
	}
}

func TestAdmissionEvidenceExpiryAndDeadline(t *testing.T) {
	for _, pair := range [][2]string{{"2024-11-30T10:11:12Z", "2025-02-28T10:11:12Z"}, {"2023-11-30T10:11:12Z", "2024-02-29T10:11:12Z"}, {"2026-01-31T10:11:12Z", "2026-04-30T10:11:12Z"}} {
		start, _ := time.Parse(time.RFC3339, pair[0])
		if got := admissionEvidenceDeadline(start).Format(time.RFC3339); got != pair[1] {
			t.Fatalf("deadline %s", got)
		}
	}
	if _, err := NormalizeAdmissionEvidence("qq", []byte("no")); err == nil {
		t.Fatal("third evidence category accepted")
	}
	if _, err := NormalizeAdmissionEvidence("launcher", make([]byte, AdmissionEvidenceMaxBytes+1)); err == nil {
		t.Fatal("oversize accepted")
	}
	db := admissionApplicationDB(t, false)
	char := seedAdmissionUser(t, db, 1, 1001, model.RoleGuest)
	seedAdmissionUser(t, db, 2, 1001, model.RoleAdmin)
	svc := NewAdmissionService()
	input := newAdmissionInput(char.CharacterID)
	app, err := svc.Create(1, input, admissionImages(t))
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := svc.Evidence(1, app.ID, "launcher")
	if err != nil || evidence.Width != 1280 || evidence.Height != 720 {
		t.Fatalf("normalize %+v %v", evidence, err)
	}
	app, err = svc.Process(2, app.ID, 1, false, "Correct evidence")
	if err != nil {
		t.Fatal(err)
	}
	past := time.Now().UTC().Add(-time.Second)
	db.Model(app).Update("evidence_expires_at", past)
	if _, err = svc.Evidence(1, app.ID, "launcher"); !errors.Is(err, ErrAdmissionEvidence) {
		t.Fatal("expired bytes readable")
	}
	input.ExpectedVersion = app.Version
	if _, err = svc.Edit(1, app.ID, input, nil, true); !errors.Is(err, ErrAdmissionEvidence) {
		t.Fatal("expired evidence reused")
	}
	if n, err := svc.CleanupEvidence(time.Now()); err != nil || n != 1 {
		t.Fatalf("cleanup %d %v", n, err)
	}
	if _, err = svc.Edit(1, app.ID, input, admissionImages(t), true); err != nil {
		t.Fatal(err)
	}
	if n, err := svc.CleanupEvidence(time.Now().AddDate(1, 0, 0)); err != nil || n != 0 {
		t.Fatal("pending evidence deleted")
	}
}

func TestAdmissionInvitedConcurrencyPostgres(t *testing.T) {
	for _, kind := range []string{model.AdmissionKindNewMember, model.AdmissionKindAlt} {
		t.Run(kind, func(t *testing.T) {
			db := admissionApplicationDB(t, true)
			char := seedAdmissionUser(t, db, 1, 1001, model.RoleGuest)
			if kind == model.AdmissionKindAlt {
				if err := db.Create(&model.EveCharacter{CharacterID: 9991, CharacterName: "Member", UserID: 1, CorporationID: model.SystemCorporationID}).Error; err != nil {
					t.Fatal(err)
				}
			}
			seedAdmissionUser(t, db, 2, model.SystemCorporationID, model.RoleAdmin)
			seedAdmissionUser(t, db, 3, model.SystemCorporationID, model.RoleSuperAdmin)
			svc := NewAdmissionService()
			input := newAdmissionInput(char.CharacterID)
			input.Kind = kind
			var images map[string][]byte
			if kind == model.AdmissionKindNewMember {
				images = admissionImages(t)
			}
			app, err := svc.Create(1, input, images)
			if err != nil {
				t.Fatal(err)
			}
			var wg sync.WaitGroup
			results := make(chan error, 2)
			for _, actor := range []uint{2, 3} {
				wg.Add(1)
				go func(actor uint) { defer wg.Done(); _, err := svc.Process(actor, app.ID, 1, true, ""); results <- err }(actor)
			}
			wg.Wait()
			close(results)
			success := 0
			for err := range results {
				if err == nil {
					success++
				} else if !errors.Is(err, ErrAdmissionConflict) {
					t.Fatal(err)
				}
			}
			if success != 1 {
				t.Fatalf("success=%d", success)
			}
			var salaries []model.AdmissionSalary
			db.Find(&salaries)
			var transactions []model.WalletTransaction
			db.Find(&transactions)
			if kind == model.AdmissionKindAlt {
				if len(salaries) != 0 || len(transactions) != 0 {
					t.Fatalf("alt invitation paid salary: %v %v", salaries, transactions)
				}
			} else if len(salaries) != 1 || len(transactions) != 1 || transactions[0].UserID != salaries[0].UserID || transactions[0].Amount != 10 {
				t.Fatalf("duplicate/incorrect salary: %v %v", salaries, transactions)
			}
			if _, err := svc.Process(2, app.ID, 1, true, ""); !errors.Is(err, ErrAdmissionConflict) {
				t.Fatalf("repeat processing: %v", err)
			}
		})
	}
}
