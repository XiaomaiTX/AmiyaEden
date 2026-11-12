package service

import (
	"amiya-eden/internal/model"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
)

func pendingBlacklistApplication(t *testing.T, db *gorm.DB, userID uint, qq, discord string, characters ...int64) model.AdmissionApplication {
	t.Helper()
	app := model.AdmissionApplication{UserID: userID, Kind: model.AdmissionKindNewMember, TargetCorporationID: model.SystemCorporationID, Status: model.AdmissionStatusPending, Version: 1, ApplicantName: "Applicant", QQ: qq, DiscordID: discord, SubmittedAt: time.Now()}
	if err := db.Create(&app).Error; err != nil {
		t.Fatal(err)
	}
	for _, id := range characters {
		if err := db.Create(&model.AdmissionCharacter{ApplicationID: app.ID, CharacterID: id, CharacterName: "Pilot"}).Error; err != nil {
			t.Fatal(err)
		}
	}
	return app
}

func setAdmissionBlacklist(t *testing.T, db *gorm.DB, entries ...model.BlacklistEntry) {
	t.Helper()
	if err := db.Where("1 = 1").Delete(&model.BlacklistEntry{}).Error; err != nil {
		t.Fatal(err)
	}
	if len(entries) > 0 {
		if err := db.Create(&entries).Error; err != nil {
			t.Fatal(err)
		}
	}
}

func admissionBlacklistEntry(kind, value string) model.BlacklistEntry {
	return model.BlacklistEntry{Kind: kind, Value: value, Display: value, Reason: "reason " + value}
}

func admissionHitValues(hits []BlacklistHit) string {
	values := make([]string, 0, len(hits))
	for _, hit := range hits {
		values = append(values, hit.Kind+":"+hit.Value)
	}
	return "[" + strings.Join(values, " ") + "]"
}

func TestAdmissionBlacklistScreeningCoversAccountAndApplication(t *testing.T) {
	db := admissionApplicationDB(t, false)
	seedAdmissionUser(t, db, 1, 1001, model.RoleGuest)
	seedAdmissionUser(t, db, 2, model.SystemCorporationID, model.RoleAdmin)
	// The account binds 9001 and 9991 and now has QQ 22222222; the application
	// recorded 9001 and the since-unbound 9555, QQ 11111111 and Discord @tester.
	if err := db.Create(&model.EveCharacter{CharacterID: 9991, CharacterName: "Unselected alternate", UserID: 1, CorporationID: 1001}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.User{}).Where("id = ?", 1).Update("qq", "22222222").Error; err != nil {
		t.Fatal(err)
	}
	app := pendingBlacklistApplication(t, db, 1, "11111111", "@tester", 9001, 9555)
	svc := NewAdmissionService()
	entry := admissionBlacklistEntry
	for _, tc := range []struct {
		name    string
		entries []model.BlacklistEntry
		want    string
	}{
		{"empty blacklist", nil, "[]"},
		{"bound character not selected", []model.BlacklistEntry{entry(model.BlacklistKindCharacter, "9991")}, "[character:9991]"},
		{"submitted character later unbound", []model.BlacklistEntry{entry(model.BlacklistKindCharacter, "9555")}, "[character:9555]"},
		{"application QQ", []model.BlacklistEntry{entry(model.BlacklistKindQQ, "11111111")}, "[qq:11111111]"},
		{"current account QQ", []model.BlacklistEntry{entry(model.BlacklistKindQQ, "22222222")}, "[qq:22222222]"},
		{"current account Discord", []model.BlacklistEntry{entry(model.BlacklistKindDiscord, "discord-only")}, "[discord:discord-only]"},
		{"Discord differs by case and at-sign", []model.BlacklistEntry{{Kind: model.BlacklistKindDiscord, Value: "tester", Display: "Tester", Reason: "spy"}}, "[discord:tester]"},
		{"QQ prefixes", []model.BlacklistEntry{entry(model.BlacklistKindQQ, "111111111"), entry(model.BlacklistKindQQ, "1111111")}, "[]"},
		{"other people", []model.BlacklistEntry{entry(model.BlacklistKindCharacter, "9002"), entry(model.BlacklistKindDiscord, "test")}, "[]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setAdmissionBlacklist(t, db, tc.entries...)
			detail, err := svc.Detail(2, app.ID)
			if err != nil {
				t.Fatal(err)
			}
			if got := admissionHitValues(detail.BlacklistHits); got != tc.want {
				t.Fatalf("hits = %s, want %s", got, tc.want)
			}
		})
	}

	// A deleted account keeps its character rows, but only the snapshot counts.
	setAdmissionBlacklist(t, db,
		entry(model.BlacklistKindCharacter, "9991"), entry(model.BlacklistKindQQ, "22222222"),
		entry(model.BlacklistKindCharacter, "9555"), entry(model.BlacklistKindQQ, "11111111"))
	if err := db.Delete(&model.User{}, 1).Error; err != nil {
		t.Fatal(err)
	}
	detail, err := svc.Detail(2, app.ID)
	if err != nil || admissionHitValues(detail.BlacklistHits) != "[character:9555 qq:11111111]" {
		t.Fatalf("deleted applicant hits = %+v, %v", detail, err)
	}
}

func TestAdmissionBlacklistHitsAreShownToReviewersOnly(t *testing.T) {
	db := admissionApplicationDB(t, false)
	seedAdmissionUser(t, db, 1, 1001, model.RoleGuest)
	seedAdmissionUser(t, db, 2, model.SystemCorporationID, model.RoleAdmin)
	seedAdmissionUser(t, db, 3, 1001, model.RoleGuest)
	if err := db.Model(&model.User{}).Where("id = ?", 1).Update("qq", "22222222").Error; err != nil {
		t.Fatal(err)
	}
	hit := pendingBlacklistApplication(t, db, 1, "", "", 9001)
	clean := pendingBlacklistApplication(t, db, 3, "", "", 9003)
	own := pendingBlacklistApplication(t, db, 2, "", "", 9002)
	setAdmissionBlacklist(t, db,
		admissionBlacklistEntry(model.BlacklistKindCharacter, "9001"),
		admissionBlacklistEntry(model.BlacklistKindQQ, "22222222"),
		admissionBlacklistEntry(model.BlacklistKindCharacter, "9002"))
	svc := NewAdmissionService()

	detail, err := svc.Detail(2, hit.ID)
	if err != nil || fmt.Sprint(detail.BlacklistHits) != "[{character 9001 9001 reason 9001} {qq 22222222 22222222 reason 22222222}]" {
		t.Fatalf("reviewer hits = %+v, %v", detail, err)
	}
	for _, reader := range []struct {
		name   string
		userID uint
		id     uint
	}{{"applicant", 1, hit.ID}, {"reviewer reading own application", 2, own.ID}} {
		detail, err := svc.Detail(reader.userID, reader.id)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := json.Marshal(detail)
		if !strings.Contains(string(body), `"blacklist_hits":[]`) {
			t.Fatalf("%s received hits: %s", reader.name, body)
		}
	}
	history, _, err := svc.List(1, false, "", "", 0, 1, 20)
	if err != nil || len(history) != 1 || history[0].Blacklisted {
		t.Fatalf("own history flagged: %+v, %v", history, err)
	}

	flagged := func(status string) map[uint]bool {
		t.Helper()
		rows, _, err := svc.List(2, true, status, "", 0, 1, 20)
		if err != nil {
			t.Fatal(err)
		}
		flags := map[uint]bool{}
		for _, row := range rows {
			flags[row.ID] = row.Blacklisted
		}
		return flags
	}
	if got := flagged("pending"); len(got) != 3 || !got[hit.ID] || got[clean.ID] || got[own.ID] {
		t.Fatalf("review list flags = %v", got)
	}
	rows, _, err := svc.List(2, true, "pending", "", 0, 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(rows)
	if !strings.Contains(string(body), fmt.Sprintf(`"id":%d,`, hit.ID)) || !strings.Contains(string(body), `"blacklisted":true`) {
		t.Fatalf("list row is not flat: %s", body)
	}

	if _, err = svc.Process(2, hit.ID, 1, false, "On the alliance blacklist"); err != nil {
		t.Fatal(err)
	}
	if detail, err = svc.Detail(2, hit.ID); err != nil || len(detail.BlacklistHits) != 0 || detail.BlacklistHits == nil {
		t.Fatalf("closed application screened: %+v, %v", detail, err)
	}
	if got := flagged(""); got[hit.ID] {
		t.Fatalf("closed row flagged: %v", got)
	}
}

func TestAdmissionInvitedRefusedForBlacklistedApplicant(t *testing.T) {
	db := admissionApplicationDB(t, false)
	seedAdmissionUser(t, db, 1, 1001, model.RoleGuest)
	seedAdmissionUser(t, db, 2, model.SystemCorporationID, model.RoleAdmin)
	app := pendingBlacklistApplication(t, db, 1, "11111111", "", 9001)
	svc := NewAdmissionService()
	detail, err := svc.Detail(2, app.ID)
	if err != nil || len(detail.BlacklistHits) != 0 {
		t.Fatalf("hits before refresh = %+v, %v", detail, err)
	}
	// The blacklist is refreshed after the reviewer loaded the page.
	setAdmissionBlacklist(t, db, admissionBlacklistEntry(model.BlacklistKindQQ, "11111111"))
	if _, err = svc.Process(2, app.ID, 1, true, ""); !errors.Is(err, ErrAdmissionBlacklisted) {
		t.Fatalf("invite blacklisted applicant: %v", err)
	}
	var kept model.AdmissionApplication
	if err = db.First(&kept, app.ID).Error; err != nil || kept.Status != model.AdmissionStatusPending || kept.Version != 1 || kept.HandledBy != nil {
		t.Fatalf("refused invite changed the application: %+v, %v", kept, err)
	}
	for _, table := range []any{&model.AdmissionProcessingEvent{}, &model.AdmissionSalary{}, &model.WalletTransaction{}} {
		var count int64
		if err = db.Model(table).Count(&count).Error; err != nil || count != 0 {
			t.Fatalf("refused invite wrote %T rows: %d, %v", table, count, err)
		}
	}
	rejected, err := svc.Process(2, app.ID, 1, false, "On the alliance blacklist")
	if err != nil || rejected.Status != model.AdmissionStatusRejected {
		t.Fatalf("reject blacklisted applicant: %+v, %v", rejected, err)
	}
}
