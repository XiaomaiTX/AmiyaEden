package service

import (
	"amiya-eden/internal/model"
	"errors"
	"gorm.io/gorm"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// A target change must not replace private materials between authorization and
// reading the response. Exercise the production PostgreSQL row-lock semantics.
func TestAdmissionPrivateReadTargetChangePostgres(t *testing.T) {
	for _, kind := range []string{"list", "detail", "evidence"} {
		t.Run(kind, func(t *testing.T) {
			db := admissionApplicationDB(t, true)
			owner := seedAdmissionUser(t, db, 1, 1001, model.RoleGuest)
			reviewer := seedAdmissionUser(t, db, 2, model.SystemCorporationID, model.RoleRecruiter)
			if err := db.Create(&model.EveCharacterCorpRole{CharacterID: reviewer.CharacterID, CorporationID: model.SystemCorporationID, CorpRole: "Personnel_Manager"}).Error; err != nil {
				t.Fatal(err)
			}
			svc := NewAdmissionService()
			input := newAdmissionInput(owner.CharacterID)
			app, err := svc.Create(1, input, admissionImages(t))
			if err != nil {
				t.Fatal(err)
			}
			locked, release := make(chan struct{}), make(chan struct{})
			var releaseOnce sync.Once
			unlock := func() { releaseOnce.Do(func() { close(release) }) }
			t.Cleanup(unlock)
			var paused atomic.Bool
			if err := db.Callback().Query().Before("gorm:preload").Register("pause_private_read", func(tx *gorm.DB) {
				_, locking := tx.Statement.Clauses["FOR"]
				if tx.Statement.Table == "admission_application" && locking && tx.RowsAffected > 0 && paused.CompareAndSwap(false, true) {
					close(locked)
					<-release
				}
			}); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = db.Callback().Query().Remove("pause_private_read") })
			readDone := make(chan error, 1)
			go func() {
				switch kind {
				case "list":
					rows, _, err := svc.List(2, true, "pending", "", 0, 1, 20)
					if err == nil && (len(rows) != 1 || rows[0].TargetCorporationID != model.SystemCorporationID || len(rows[0].Characters) != 1) {
						err = errors.New("private list crossed application versions")
					}
					readDone <- err
				case "detail":
					detail, err := svc.Detail(2, app.ID)
					if err == nil && (detail.Application.TargetCorporationID != model.SystemCorporationID || detail.Application.Version != 1) {
						err = errors.New("private detail crossed application versions")
					}
					readDone <- err
				default:
					_, err := svc.Evidence(2, app.ID, model.AdmissionEvidenceSeatProfile)
					readDone <- err
				}
			}()
			select {
			case <-locked:
			case <-time.After(5 * time.Second):
				t.Fatal("private read did not reach application lock")
			}
			input.ExpectedVersion = 1
			input.TargetCorporationID = model.AdmissionFMACorporationID
			images := admissionImages(t)
			editDone := make(chan error, 1)
			go func() { _, err := svc.Edit(1, app.ID, input, images, false); editDone <- err }()
			// Observe a real PostgreSQL lock wait rather than relying on goroutine timing.
			deadline := time.Now().Add(5 * time.Second)
			for {
				var waiting int64
				if err := db.Raw("SELECT count(*) FROM pg_stat_activity WHERE pid <> pg_backend_pid() AND wait_event_type = 'Lock' AND query LIKE '%admission_application%'").Scan(&waiting).Error; err != nil {
					t.Fatal(err)
				}
				if waiting > 0 {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("target edit was not blocked by the private read")
				}
				time.Sleep(10 * time.Millisecond)
			}
			unlock()
			for _, done := range []chan error{readDone, editDone} {
				select {
				case err := <-done:
					if err != nil {
						t.Fatal(err)
					}
				case <-time.After(5 * time.Second):
					t.Fatal("concurrent request did not finish")
				}
			}
			if _, err := svc.Detail(2, app.ID); !errors.Is(err, ErrAdmissionDenied) {
				t.Fatalf("old target reviewer retained detail access: %v", err)
			}
			if _, err := svc.Evidence(2, app.ID, model.AdmissionEvidenceSeatProfile); !errors.Is(err, ErrAdmissionDenied) {
				t.Fatalf("old target reviewer retained evidence access: %v", err)
			}
		})
	}
}

func TestAdmissionSubmissionBoundaries(t *testing.T) {
	db := admissionApplicationDB(t, false)
	guest := seedAdmissionUser(t, db, 1, 1001, model.RoleGuest)
	other := seedAdmissionUser(t, db, 2, 1001, model.RoleGuest)
	member := seedAdmissionUser(t, db, 3, model.SystemCorporationID, model.RoleUser)
	svc := NewAdmissionService()
	for _, in := range []AdmissionWriteInput{
		newAdmissionInput(), newAdmissionInput(other.CharacterID), newAdmissionInput(guest.CharacterID, guest.CharacterID),
		{Kind: "wrong", TargetCorporationID: model.SystemCorporationID, CharacterIDs: []int64{guest.CharacterID}},
		{Kind: model.AdmissionKindAlt, TargetCorporationID: model.SystemCorporationID, CharacterIDs: []int64{guest.CharacterID}},
		{Kind: model.AdmissionKindNewMember, TargetCorporationID: 42, CharacterIDs: []int64{guest.CharacterID}},
	} {
		if _, err := svc.Create(1, in, admissionImages(t)); err == nil {
			t.Fatalf("invalid submission accepted: %+v", in)
		}
	}
	var count int64
	db.Model(&model.AdmissionApplication{}).Count(&count)
	if count != 0 {
		t.Fatal("invalid submission partially committed")
	}
	input := AdmissionWriteInput{Kind: model.AdmissionKindAlt, TargetCorporationID: model.SystemCorporationID, CharacterIDs: []int64{member.CharacterID}}
	if _, err := svc.Create(3, input, nil); !errors.Is(err, ErrAdmissionInvalid) {
		t.Fatalf("already in target: %v", err)
	}
	alt := model.EveCharacter{CharacterID: 9998, CharacterName: "Alt", UserID: 3, CorporationID: 1001}
	db.Create(&alt)
	input.CharacterIDs = []int64{alt.CharacterID}
	if _, err := svc.Create(3, input, admissionImages(t)); !errors.Is(err, ErrAdmissionEvidence) {
		t.Fatalf("alt accepted images: %v", err)
	}
	app, err := svc.Create(3, input, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = svc.Create(3, input, nil); !errors.Is(err, ErrAdmissionConflict) {
		t.Fatalf("character already claimed: %v", err)
	}
	detail, err := svc.Detail(3, app.ID)
	if err != nil || len(detail.Evidence) != 0 {
		t.Fatalf("alt must need no evidence: %+v %v", detail, err)
	}
}

func TestAdmissionReviewScopeMatrix(t *testing.T) {
	db := admissionApplicationDB(t, false)
	svc := NewAdmissionService()
	apps := []*model.AdmissionApplication{}
	for i, corp := range []int64{model.SystemCorporationID, model.AdmissionFMACorporationID} {
		owner := uint(i + 1)
		char := seedAdmissionUser(t, db, owner, 1001, model.RoleGuest)
		in := newAdmissionInput(char.CharacterID)
		in.TargetCorporationID = corp
		app, err := svc.Create(owner, in, admissionImages(t))
		if err != nil {
			t.Fatal(err)
		}
		apps = append(apps, app)
	}
	for index, tc := range []struct {
		role  string
		scope []int64
	}{
		{model.RoleRecruiter, []int64{model.SystemCorporationID}},
		{model.RoleRecruiter, []int64{model.AdmissionFMACorporationID}},
		{model.RoleRecruiter, []int64{model.SystemCorporationID, model.AdmissionFMACorporationID}},
		{model.RoleAdmin, nil}, {model.RoleSuperAdmin, nil},
	} {
		id := uint(index + 10)
		seedAdmissionUser(t, db, id, 1001, tc.role)
		for j, corp := range tc.scope {
			char := model.EveCharacter{CharacterID: 100000 + int64(index*10+j), CharacterName: "Recruiter", UserID: id, CorporationID: corp}
			db.Create(&char)
			db.Create(&model.EveCharacterCorpRole{CharacterID: char.CharacterID, CorporationID: corp, CorpRole: "Personnel_Manager"})
		}
		want := len(tc.scope)
		if tc.role != model.RoleRecruiter {
			want = 2
		}
		list, total, err := svc.List(id, true, "pending", "", 0, 1, 20)
		if err != nil || int(total) != want || len(list) != want {
			t.Fatalf("scope list %s %v: %d %v", tc.role, tc.scope, total, err)
		}
		for _, app := range apps {
			allowed := tc.role != model.RoleRecruiter
			for _, corp := range tc.scope {
				allowed = allowed || corp == app.TargetCorporationID
			}
			_, err = svc.Detail(id, app.ID)
			if (err == nil) != allowed {
				t.Fatalf("detail scope leak: %v", err)
			}
			_, err = svc.Evidence(id, app.ID, "seat_profile")
			if (err == nil) != allowed {
				t.Fatalf("evidence scope leak: %v", err)
			}
			if !allowed {
				_, total, err = svc.List(id, true, "", "", app.TargetCorporationID, 1, 20)
				if err != nil || total != 0 {
					t.Fatalf("filter count leak %d %v", total, err)
				}
				if _, err = svc.Process(id, app.ID, app.Version, true, ""); !errors.Is(err, ErrAdmissionDenied) {
					t.Fatal("cross-corp action allowed")
				}
			}
		}
	}
}

func TestAdmissionInvitedInjectedFailureRollbackPostgres(t *testing.T) {
	for _, table := range []string{"wallet_transaction", "admission_salary", "admission_processing_event", "audit_event"} {
		t.Run(table, func(t *testing.T) {
			db := admissionApplicationDB(t, true)
			char := seedAdmissionUser(t, db, 1, 1001, model.RoleGuest)
			seedAdmissionUser(t, db, 2, model.SystemCorporationID, model.RoleAdmin)
			svc := NewAdmissionService()
			app, err := svc.Create(1, newAdmissionInput(char.CharacterID), admissionImages(t))
			if err != nil {
				t.Fatal(err)
			}
			if err = db.Callback().Create().Before("gorm:create").Register("admission_failure", func(tx *gorm.DB) {
				if tx.Statement.Table == table {
					_ = tx.AddError(errors.New("injected failure"))
				}
			}); err != nil {
				t.Fatal(err)
			}
			if _, err = svc.Process(2, app.ID, 1, true, ""); !errors.Is(err, ErrAdmissionInternal) {
				t.Fatalf("injected %s: %v", table, err)
			}
			if err := db.Callback().Create().Remove("admission_failure"); err != nil {
				t.Fatal(err)
			}
			var stored model.AdmissionApplication
			db.First(&stored, app.ID)
			if stored.Status != model.AdmissionStatusPending || stored.Version != 1 {
				t.Fatal("closure escaped rollback")
			}
			for _, m := range []interface{}{&model.WalletTransaction{}, &model.AdmissionSalary{}} {
				var count int64
				db.Model(m).Count(&count)
				if count != 0 {
					t.Fatalf("partial payment %s", table)
				}
			}
			var claims, events int64
			db.Model(&model.AdmissionCharacterClaim{}).Count(&claims)
			db.Model(&model.AdmissionProcessingEvent{}).Count(&events)
			if claims != 1 || events != 1 {
				t.Fatal("claims/history changed after failed process")
			}
			if _, err = svc.Process(2, app.ID, 1, true, ""); err != nil {
				t.Fatalf("retry: %v", err)
			}
		})
	}
}

func TestAdmissionCleanupResubmitConcurrencyPostgres(t *testing.T) {
	db := admissionApplicationDB(t, true)
	char := seedAdmissionUser(t, db, 1, 1001, model.RoleGuest)
	seedAdmissionUser(t, db, 2, 1001, model.RoleAdmin)
	svc := NewAdmissionService()
	in := newAdmissionInput(char.CharacterID)
	images := admissionImages(t)
	app, err := svc.Create(1, in, images)
	if err != nil {
		t.Fatal(err)
	}
	app, err = svc.Process(2, app.ID, 1, false, "Replace")
	if err != nil {
		t.Fatal(err)
	}
	db.Model(app).Update("evidence_expires_at", time.Now().UTC().Add(-time.Second))
	in.ExpectedVersion = app.Version
	var wg sync.WaitGroup
	results := make(chan error, 2)
	wg.Add(2)
	go func() { defer wg.Done(); _, err := svc.CleanupEvidence(time.Now()); results <- err }()
	go func() { defer wg.Done(); _, err := svc.Edit(1, app.ID, in, images, true); results <- err }()
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal(err)
		}
	}
	detail, err := svc.Detail(1, app.ID)
	if err != nil || detail.Application.Status != model.AdmissionStatusPending || len(detail.Evidence) != 2 {
		t.Fatalf("new materials deleted: %+v %v", detail, err)
	}
}
