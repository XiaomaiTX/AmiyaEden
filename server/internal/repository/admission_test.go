package repository

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/testutil"
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestAdmissionRecruiterProvenancePostgres(t *testing.T) {
	db := admissionRepositoryDB(t)
	if err := db.AutoMigrate(&model.EveCharacter{}, &model.EveCharacterCorpRole{}); err != nil {
		t.Fatal(err)
	}
	oldDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = oldDB })
	character := model.EveCharacter{CharacterID: 9001, CharacterName: "Recruiter", UserID: 1, CorporationID: model.AdmissionFMACorporationID}
	if err := db.Create(&character).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.EveCharacterCorpRole{CharacterID: character.CharacterID, CorpRole: "Director"}).Error; err != nil {
		t.Fatal(err)
	}
	repo := NewAdmissionRepository()
	scope := func() []int64 {
		ids, err := repo.UserCorporationsWithRoles(db, 1, []int64{model.SystemCorporationID, model.AdmissionFMACorporationID}, []string{"Director", "Personnel_Manager"})
		if err != nil {
			t.Fatal(err)
		}
		return ids
	}
	if len(scope()) != 0 {
		t.Fatal("unknown provenance granted recruitment authority")
	}
	roles := NewAutoRoleRepository()
	if err := roles.SyncCharacterCorpRoles(character.CharacterID, character.CorporationID, []string{"Director"}); err != nil {
		t.Fatal(err)
	}
	if ids := scope(); len(ids) != 1 || ids[0] != character.CorporationID {
		t.Fatalf("valid provenance denied: %v", ids)
	}
	if err := db.Model(&character).Update("corporation_id", model.SystemCorporationID).Error; err != nil {
		t.Fatal(err)
	}
	if len(scope()) != 0 {
		t.Fatal("old corporation roles granted authority after transfer")
	}
	if err := roles.SyncCharacterCorpRoles(character.CharacterID, model.AdmissionFMACorporationID, []string{"Personnel_Manager"}); err == nil {
		t.Fatal("affiliation change during role request was not detected")
	}
	var snapshot model.EveCharacterCorpRole
	if err := db.First(&snapshot).Error; err != nil {
		t.Fatal(err)
	}
	if snapshot.CorporationID != model.AdmissionFMACorporationID || snapshot.CorpRole != "Director" {
		t.Fatalf("failed sync overwrote previous snapshot: %+v", snapshot)
	}
}

func admissionRepositoryDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := testutil.Postgres(t)
	if err := db.AutoMigrate(&model.AdmissionApplication{}, &model.AdmissionCharacter{}, &model.AdmissionCharacterClaim{}, &model.AdmissionEvidence{}, &model.AdmissionProcessingEvent{}, &model.AdmissionSalary{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func repositoryAdmission(userID uint, target int64) model.AdmissionApplication {
	return model.AdmissionApplication{UserID: userID, Kind: model.AdmissionKindNewMember, TargetCorporationID: target, Status: model.AdmissionStatusPending, ApplicantName: "Applicant", SubmittedAt: time.Now()}
}

func TestAdmissionRepositoryClaimsAndRollbackPostgres(t *testing.T) {
	db, repo := admissionRepositoryDB(t), NewAdmissionRepository()
	first, second := repositoryAdmission(1, model.SystemCorporationID), repositoryAdmission(2, model.AdmissionFMACorporationID)
	for _, application := range []*model.AdmissionApplication{&first, &second} {
		if err := repo.CreateTx(db, application); err != nil {
			t.Fatal(err)
		}
	}
	characters := []model.AdmissionCharacter{{ApplicationID: first.ID, CharacterID: 9001, CharacterName: "Bound alternate"}}
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := repo.ReplaceCharactersTx(tx, first.ID, characters); err != nil {
			return err
		}
		return repo.ClaimCharactersTx(tx, first.ID, characters)
	}); err != nil {
		t.Fatal(err)
	}
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&second).Update("applicant_name", "must roll back").Error; err != nil {
			return err
		}
		return repo.ClaimCharactersTx(tx, second.ID, characters)
	})
	if err == nil {
		t.Fatal("character claimed by two pending applications")
	}
	kept, err := repo.GetByID(db, second.ID, false)
	if err != nil || kept.ApplicantName != "Applicant" {
		t.Fatalf("failed claim left a partial update: %+v, %v", kept, err)
	}
	if err := repo.ReleaseClaimsTx(db, first.ID); err != nil {
		t.Fatal(err)
	}
	if err := repo.ClaimCharactersTx(db, second.ID, characters); err != nil {
		t.Fatal("released character cannot be claimed:", err)
	}
}

func TestAdmissionRepositoryScopeVersionAndEvidencePostgres(t *testing.T) {
	db, repo := admissionRepositoryDB(t), NewAdmissionRepository()
	first, second := repositoryAdmission(1, model.SystemCorporationID), repositoryAdmission(2, model.AdmissionFMACorporationID)
	for _, application := range []*model.AdmissionApplication{&first, &second} {
		if err := repo.CreateTx(db, application); err != nil {
			t.Fatal(err)
		}
		characters := []model.AdmissionCharacter{{ApplicationID: application.ID, CharacterID: int64(application.ID) + 9000, CharacterName: "Character"}}
		if err := repo.ReplaceCharactersTx(db, application.ID, characters); err != nil {
			t.Fatal(err)
		}
	}
	items, count, err := repo.List(db, AdmissionListFilter{CorporationIDs: []int64{model.SystemCorporationID}}, 1, 20)
	if err != nil || count != 1 || len(items) != 1 || items[0].ID != first.ID || len(items[0].Characters) != 1 {
		t.Fatalf("scope or batched character query failed: %+v %d %v", items, count, err)
	}
	_, count, err = repo.List(db, AdmissionListFilter{}, 1, 20)
	if err != nil || count != 0 {
		t.Fatalf("empty reviewer scope is not closed: %d %v", count, err)
	}
	first.Version++
	first.QQ = ""
	if err := repo.UpdateTx(db, &first, 1); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateTx(db, &first, 1); !errors.Is(err, ErrAdmissionVersionConflict) {
		t.Fatalf("stale version accepted: %v", err)
	}
	evidence := model.AdmissionEvidence{ApplicationID: first.ID, Kind: model.AdmissionEvidenceLauncher, Data: []byte("private"), Width: 100, Height: 100, ByteSize: 7}
	if err := repo.PutEvidenceTx(db, &evidence); err != nil {
		t.Fatal(err)
	}
	metadata, err := repo.EvidenceMetadata(db, first.ID)
	if err != nil || len(metadata) != 1 || len(metadata[0].Data) != 0 {
		t.Fatalf("metadata loaded private bytes: %+v %v", metadata, err)
	}
	salary := model.AdmissionSalary{ApplicationID: first.ID, UserID: 3, Amount: 10, WalletTransactionID: 100}
	if err := repo.AddSalaryTx(db, &salary); err != nil {
		t.Fatal(err)
	}
	if err := repo.AddSalaryTx(db, &salary); err == nil {
		t.Fatal("duplicate salary accepted")
	}
}
