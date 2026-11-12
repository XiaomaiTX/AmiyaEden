package router

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/pkg/jwt"
	"bytes"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSalaryRoutesRequireAdminAndAcceptZero(t *testing.T) {
	r, userToken, guestToken, superToken := newFuxiHallRouteTestRouter(t)
	_ = global.Redis.Close()
	global.Redis = redis.NewClient(&redis.Options{Addr: miniredis.RunT(t).Addr()})
	if err := global.DB.AutoMigrate(&model.SystemConfig{}, &model.EveCharacter{}, &model.AuditEvent{}); err != nil {
		t.Fatal(err)
	}
	for _, row := range []any{&model.User{BaseModel: model.BaseModel{ID: 400}, Nickname: "Admin", QQ: "12345", Status: 1, PrimaryCharacterID: 1004}, &model.UserRole{UserID: 400, RoleCode: model.RoleAdmin}, &model.EveCharacter{CharacterID: 1004, UserID: 400, CorporationID: model.SystemCorporationID}} {
		if err := global.DB.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	adminToken, err := jwt.GenerateToken(400, 1004, model.RoleAdmin, 1)
	if err != nil {
		t.Fatal(err)
	}
	send := func(method, token, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "/api/v1/system/salary-settings", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}
	for _, token := range []string{"", guestToken, userToken} {
		rec := send(http.MethodGet, token, "")
		if rec.Code == http.StatusOK && bytes.Contains(rec.Body.Bytes(), []byte(`"code":200`)) {
			t.Fatal("unauthorized salary access")
		}
	}
	const body = `{"recruitment":0,"welfare_delivery":0,"shop_delivery":0,"srp_payout":0,"fc_salary":0,"fc_salary_monthly_limit":0}`
	for _, token := range []string{adminToken, superToken} {
		rec := send(http.MethodPut, token, body)
		if !bytes.Contains(rec.Body.Bytes(), []byte(`"code":200`)) || !bytes.Contains(rec.Body.Bytes(), []byte(`"recruitment":0`)) {
			t.Fatalf("zero salary update: %s", rec.Body.String())
		}
	}
	if rec := send(http.MethodPut, superToken, `{"recruitment":0}`); bytes.Contains(rec.Body.Bytes(), []byte(`"code":200`)) {
		t.Fatal("incomplete configuration accepted")
	}
	for _, token := range []string{"", userToken, guestToken} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/operation-salary-rates", nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		ok := bytes.Contains(rec.Body.Bytes(), []byte(`"code":200`))
		if token == "" || token == guestToken {
			if ok {
				t.Fatal("anonymous access to operation rates")
			}
			continue
		}
		if !ok || !bytes.Contains(rec.Body.Bytes(), []byte(`"recruitment":0`)) || bytes.Contains(rec.Body.Bytes(), []byte(`fc_salary`)) {
			t.Fatalf("operator rates must contain operation amounts only: %s", rec.Body.String())
		}
	}
}
