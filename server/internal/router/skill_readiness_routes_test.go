package router

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/pkg/jwt"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSkillReadinessRegisteredRoutesAndPermissions(t *testing.T) {
	router, userToken, adminToken, guestToken, superToken := newFuxiHallManageRouteTestRouter(t)
	routes := []struct{ method, path, registered string }{
		{"GET", "/options", "/options"},
		{"POST", "/query", "/query"},
		{"POST", "/users/1/characters", "/users/:user_id/characters"},
		{"POST", "/characters/1/skills", "/characters/:character_id/skills"},
	}
	prefix := "/api/v1/dashboard/skill-readiness"
	for _, route := range routes {
		if !routeExists(router.Routes(), route.method, prefix+route.registered) {
			t.Fatalf("missing route %s", route.registered)
		}
		for _, token := range []string{userToken, guestToken, adminToken} {
			// The admin fixture has no primary corporation and therefore no dashboard capability.
			assertRouteStatus(t, router, route.method, prefix+route.path+"?token="+token, http.StatusForbidden)
		}
		assertRouteStatus(t, router, route.method, prefix+route.path, http.StatusUnauthorized)
	}
	if err := global.DB.AutoMigrate(&model.User{}, &model.EveCharacter{}, &model.SystemConfig{}, &model.SkillPlan{}); err != nil {
		t.Fatal(err)
	}
	users := []model.User{{BaseModel: model.BaseModel{ID: 1}, PrimaryCharacterID: 1001}, {BaseModel: model.BaseModel{ID: 2}, PrimaryCharacterID: 1002}, {BaseModel: model.BaseModel{ID: 5}, PrimaryCharacterID: 1005}}
	chars := []model.EveCharacter{{CharacterID: 1001, UserID: 1, CorporationID: 10}, {CharacterID: 1002, UserID: 2, CorporationID: 10}, {CharacterID: 1005, UserID: 5, CorporationID: 10}}
	for _, row := range []interface{}{&users, &chars, &model.UserRole{UserID: 5, RoleCode: model.RoleSeniorFC}} {
		if err := global.DB.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	seniorToken, err := jwt.GenerateToken(5, 1005, model.RoleSeniorFC, 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, route := range routes {
		assertRouteStatus(t, router, route.method, prefix+route.path+"?token="+userToken, http.StatusForbidden)
		assertRouteStatus(t, router, route.method, prefix+route.path+"?token="+seniorToken, http.StatusForbidden)
		if route.method == "GET" {
			continue
		}
		for _, token := range []string{adminToken, superToken} {
			req := httptest.NewRequest(route.method, prefix+route.path+"?token="+token, bytes.NewBufferString(`{"plan_id":0}`))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			var body struct{ Code int }
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Code != 400 || rec.Code != 200 {
				t.Fatalf("expected handler validation after authorized request: %s", rec.Body.String())
			}
		}
	}
	// Empty options succeed without ESI name resolution after removing the characters.
	if err := global.DB.Where("1 = 1").Delete(&model.EveCharacter{}).Error; err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest("GET", prefix+"/options?token="+superToken, nil))
	var body struct{ Code int }
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.Code != 200 {
		t.Fatalf("superadmin options: %s %v", rec.Body.String(), err)
	}
}
