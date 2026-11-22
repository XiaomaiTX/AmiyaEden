package service

import (
	"amiya-eden/internal/model"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestAdmissionAltNoteLifecycle(t *testing.T) {
	db := admissionApplicationDB(t, false)
	seedAdmissionUser(t, db, 1, model.SystemCorporationID, model.RoleUser)
	seedAdmissionUser(t, db, 2, model.SystemCorporationID, model.RoleAdmin)
	if err := db.Create(&model.EveCharacter{CharacterID: 9991, CharacterName: "Submitting alt", UserID: 1, CorporationID: 1001}).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewAdmissionService()
	in := AdmissionWriteInput{Kind: model.AdmissionKindAlt, TargetCorporationID: model.SystemCorporationID, CharacterIDs: []int64{9991}, Note: "  其他小号：甲、乙\n均已绑定联盟 SeAT。  "}
	app, err := svc.Create(1, in, nil)
	if err != nil {
		t.Fatal(err)
	}
	checkNote := func(want string) {
		t.Helper()
		for _, reader := range []uint{1, 2} {
			detail, err := svc.Detail(reader, app.ID)
			if err != nil || detail.Application.Note != want {
				t.Fatalf("reader %d note: %+v %v", reader, detail, err)
			}
			var snapshot model.AdmissionApplication
			if err := json.Unmarshal([]byte(detail.Events[len(detail.Events)-1].Snapshot), &snapshot); err != nil || snapshot.Note != want {
				t.Fatalf("note snapshot: %+v %v", snapshot, err)
			}
		}
		rows, _, err := svc.List(2, true, "", "", 0, 1, 20)
		if err != nil || len(rows) != 1 || rows[0].Note != want {
			t.Fatalf("review list note: %+v %v", rows, err)
		}
	}
	checkNote(strings.TrimSpace(in.Note))
	// A limit of 2000 characters also allows multi-byte Chinese text.
	in.Note = strings.Repeat("号", 2000)
	in.ExpectedVersion = app.Version
	app, err = svc.Edit(1, app.ID, in, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	checkNote(in.Note)
	in.Note += "号"
	in.ExpectedVersion = app.Version
	if _, err = svc.Edit(1, app.ID, in, nil, false); !errors.Is(err, ErrAdmissionInvalid) {
		t.Fatalf("oversized note accepted: %v", err)
	}
	checkNote(strings.Repeat("号", 2000))
	app, err = svc.Process(2, app.ID, app.Version, false, "Clarify other alts")
	if err != nil {
		t.Fatal(err)
	}
	in.Note = "Other alt: Pilot C"
	in.ExpectedVersion = app.Version
	app, err = svc.Edit(1, app.ID, in, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	checkNote(in.Note)
	in.Note = "  \n "
	in.ExpectedVersion = app.Version
	app, err = svc.Edit(1, app.ID, in, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	checkNote("")
	app, err = svc.Process(2, app.ID, app.Version, true, "")
	if err != nil || app.Status != model.AdmissionStatusInvited {
		t.Fatalf("invite: %+v %v", app, err)
	}
}

func TestAdmissionNoteValidation(t *testing.T) {
	db := admissionApplicationDB(t, false)
	svc := NewAdmissionService()
	for _, in := range []AdmissionWriteInput{
		{Kind: model.AdmissionKindAlt, Note: strings.Repeat("a", 2001)},
		{Kind: model.AdmissionKindNewMember, Note: "Alt-only note"},
	} {
		if _, err := svc.Create(1, in, nil); !errors.Is(err, ErrAdmissionInvalid) {
			t.Fatalf("invalid note accepted: %v", err)
		}
	}
	var count int64
	db.Model(&model.AdmissionApplication{}).Count(&count)
	if count != 0 {
		t.Fatal("invalid note created an application")
	}
}
