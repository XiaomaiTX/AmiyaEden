package router

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/utils"
	"amiya-eden/pkg/jwt"
	"bytes"
	"encoding/json"
	"image"
	"image/jpeg"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestAdmissionRealRoutesAllowGuestAndScopeRecruiter(t *testing.T) {
	r, _, guestToken, _ := newFuxiHallRouteTestRouter(t)
	_ = global.Redis.Close()
	global.Redis = redis.NewClient(&redis.Options{Addr: miniredis.RunT(t).Addr()})
	db := global.DB
	if err := db.AutoMigrate(&model.EveCharacter{}, &model.EveCharacterCorpRole{}, &model.SystemConfig{}, &model.AdmissionApplication{}, &model.AdmissionCharacter{}, &model.AdmissionCharacterClaim{}, &model.AdmissionEvidence{}, &model.AdmissionProcessingEvent{}, &model.AdmissionSalary{}, &model.AuditEvent{}, &model.BlacklistEntry{}); err != nil {
		t.Fatal(err)
	}
	utils.InvalidateAllowCorporationsCache()
	t.Cleanup(utils.InvalidateAllowCorporationsCache)
	db.Model(&model.User{}).Where("id = ?", 2).Updates(map[string]interface{}{"nickname": "Guest applicant", "primary_character_id": 1002})
	for _, item := range []interface{}{&model.EveCharacter{CharacterID: 1002, CharacterName: "Applicant", UserID: 2, CorporationID: 1001}, &model.EveCharacter{CharacterID: 1001, CharacterName: "Recruiter", UserID: 1, CorporationID: model.SystemCorporationID}, &model.UserRole{UserID: 1, RoleCode: model.RoleRecruiter}, &model.EveCharacterCorpRole{CharacterID: 1001, CorporationID: model.SystemCorporationID, CorpRole: "Director"}} {
		if err := db.Create(item).Error; err != nil {
			t.Fatal(err)
		}
	}
	request := func(method, path, token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}
	if got := request(http.MethodGet, "/api/v1/admissions/context", "").Code; got != http.StatusUnauthorized {
		t.Fatalf("anonymous=%d", got)
	}
	if rec := request(http.MethodGet, "/api/v1/admissions/context", guestToken); rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte(`"can_create_new_member":true`)) {
		t.Fatalf("guest context %s", rec.Body.String())
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("payload", `{"kind":"new_member","target_corporation_id":98185110,"character_ids":[1002]}`); err != nil {
		t.Fatal(err)
	}
	var picture bytes.Buffer
	if err := jpeg.Encode(&picture, image.NewRGBA(image.Rect(0, 0, 20, 20)), nil); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"launcher", "seat_profile"} {
		part, err := writer.CreateFormFile(kind, kind+".jpg")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = part.Write(picture.Bytes()); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admissions/applications", &body)
	req.Header.Set("Authorization", "Bearer "+guestToken)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	var created struct {
		Code int                        `json:"code"`
		Data model.AdmissionApplication `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil || created.Code != 200 {
		t.Fatalf("guest create: %s %v", rec.Body.String(), err)
	}
	recruiterToken, err := jwt.GenerateToken(1, 1001, model.RoleRecruiter, 1)
	if err != nil {
		t.Fatal(err)
	}
	// Recruiter-only accounts (e.g. FMA directors outside the allowlist) review
	// without gaining Login membership.
	if err = db.Where("user_id = ? AND role_code = ?", 1, model.RoleUser).Delete(&model.UserRole{}).Error; err != nil {
		t.Fatal(err)
	}
	if rec = request(http.MethodGet, "/api/v1/system/admissions/applications", recruiterToken); rec.Code != 200 || !bytes.Contains(rec.Body.Bytes(), []byte(`"total":1`)) {
		t.Fatalf("non-admin recruiter blocked: %s", rec.Body.String())
	}
	if rec = request(http.MethodGet, "/api/v1/badge-counts", recruiterToken); rec.Code != http.StatusForbidden {
		t.Fatalf("recruiter-only account passed the Login boundary: %d %s", rec.Code, rec.Body.String())
	}
	fma := model.AdmissionApplication{UserID: 4, Kind: model.AdmissionKindAlt, Status: model.AdmissionStatusPending, Version: 1, TargetCorporationID: model.AdmissionFMACorporationID, ApplicantName: "Other corporation", SubmittedAt: time.Now()}
	if err = db.Create(&fma).Error; err != nil {
		t.Fatal(err)
	}
	if rec = request(http.MethodGet, "/api/v1/admissions/applications/2", recruiterToken); rec.Code != 403 {
		t.Fatalf("cross corporation detail leaked: %s", rec.Body.String())
	}
	if rec = request(http.MethodGet, "/api/v1/system/admissions/applications?target_corporation_id=98824507", recruiterToken); !bytes.Contains(rec.Body.Bytes(), []byte(`"total":0`)) {
		t.Fatalf("cross corporation count leaked: %s", rec.Body.String())
	}
	if rec = request(http.MethodGet, "/api/v1/admissions/applications/1/evidence/launcher", guestToken); rec.Header().Get("Content-Type") != "image/jpeg" || rec.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatal("owner evidence unavailable or cacheable")
	}
	otherToken, err := jwt.GenerateToken(4, 1004, model.RoleGuest, 1)
	if err != nil {
		t.Fatal(err)
	}
	if rec = request(http.MethodGet, "/api/v1/admissions/applications/1/evidence/launcher", otherToken); rec.Code != 403 {
		t.Fatal("other applicant read evidence")
	}
	db.Model(&model.EveCharacterCorpRole{}).Where("character_id = ?", 1001).Update("corporation_id", 0)
	if rec = request(http.MethodGet, "/api/v1/system/admissions/applications", recruiterToken); rec.Code != 403 {
		t.Fatalf("stale recruiter role bypassed scope: %s", rec.Body.String())
	}
	db.Model(&model.EveCharacter{}).Where("character_id = ?", 1002).Update("corporation_id", model.SystemCorporationID)
	if rec = request(http.MethodGet, "/api/v1/admissions/applications/1", guestToken); rec.Code != 200 || !bytes.Contains(rec.Body.Bytes(), []byte(`"is_owner":true`)) {
		t.Fatal("owner history lost after joining")
	}
}
