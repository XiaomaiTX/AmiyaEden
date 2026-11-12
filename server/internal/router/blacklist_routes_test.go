package router

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/jwt"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// blacklistRouteTransport stands in for the Google Sheet export and ESI so the
// real refresh route runs without network access.
type blacklistRouteTransport struct{}

func (blacklistRouteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var body string
	switch {
	case req.URL.Host == "docs.google.com":
		body = "Main,Date,By,Reason,Verification,known alt1,known alt2,known alt3,Discord Tag,QQ (for CN),Link\n" +
			"Route Pilot,2024-01-01,tester,Scam,,,,,,10000001,\n"
	case strings.HasSuffix(req.URL.Path, "/universe/ids/"):
		body = `{"characters":[{"id":90000001,"name":"Route Pilot"}]}`
	default:
		return nil, fmt.Errorf("unexpected outbound request %s", req.URL)
	}
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
}

func TestBlacklistRoutesRequireAdminWithSystemManage(t *testing.T) {
	r, userToken, guestToken, superToken := newFuxiHallRouteTestRouter(t)
	_ = global.Redis.Close()
	global.Redis = redis.NewClient(&redis.Options{Addr: miniredis.RunT(t).Addr()})
	oldTransport := http.DefaultTransport
	http.DefaultTransport = blacklistRouteTransport{}
	t.Cleanup(func() { http.DefaultTransport = oldTransport })
	if err := global.DB.AutoMigrate(&model.SystemConfig{}, &model.EveCharacter{}, &model.AuditEvent{}, &model.BlacklistEntry{}, &model.BlacklistUnresolvedName{}, &model.BlacklistRefreshRun{}); err != nil {
		t.Fatal(err)
	}
	const restrictedCorporationID = 1001
	for _, row := range []any{
		&model.User{BaseModel: model.BaseModel{ID: 400}, Nickname: "Admin", Status: 1, PrimaryCharacterID: 1004},
		&model.UserRole{UserID: 400, RoleCode: model.RoleAdmin},
		&model.EveCharacter{CharacterID: 1004, UserID: 400, CorporationID: model.SystemCorporationID},
		&model.User{BaseModel: model.BaseModel{ID: 401}, Nickname: "Restricted admin", Status: 1, PrimaryCharacterID: 1005},
		&model.UserRole{UserID: 401, RoleCode: model.RoleAdmin},
		&model.EveCharacter{CharacterID: 1005, UserID: 401, CorporationID: restrictedCorporationID},
	} {
		if err := global.DB.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := service.NewCorporationPolicyService().UpdatePolicies(service.CorporationPolicyConfig{
		Version:     1,
		DefaultMode: "allow",
		Policies:    []service.CorporationPolicy{{CorporationID: restrictedCorporationID, Capabilities: []string{}}},
	}); err != nil {
		t.Fatal(err)
	}
	adminToken, err := jwt.GenerateToken(400, 1004, model.RoleAdmin, 1)
	if err != nil {
		t.Fatal(err)
	}
	restrictedToken, err := jwt.GenerateToken(401, 1005, model.RoleAdmin, 1)
	if err != nil {
		t.Fatal(err)
	}
	send := func(method, path, token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "/api/v1/system/blacklist"+path, nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}
	ok := func(rec *httptest.ResponseRecorder) bool {
		return rec.Code == http.StatusOK && bytes.Contains(rec.Body.Bytes(), []byte(`"code":200`))
	}

	routes := []struct{ method, path string }{
		{http.MethodGet, ""},
		{http.MethodGet, "/refresh"},
		{http.MethodPost, "/refresh"},
	}
	for _, route := range routes {
		for name, token := range map[string]string{"anonymous": "", "guest": guestToken, "user": userToken} {
			if rec := send(route.method, route.path, token); ok(rec) {
				t.Fatalf("%s reached %s %s: %s", name, route.method, route.path, rec.Body.String())
			}
		}
		if rec := send(route.method, route.path, restrictedToken); rec.Code != http.StatusForbidden {
			t.Fatalf("admin without system.manage: %s %s = %d %s", route.method, route.path, rec.Code, rec.Body.String())
		}
	}

	if rec := send(http.MethodGet, "/refresh", adminToken); !ok(rec) || !bytes.Contains(rec.Body.Bytes(), []byte(`"data":null`)) {
		t.Fatalf("summary before first refresh: %s", rec.Body.String())
	}
	if rec := send(http.MethodPost, "/refresh", adminToken); !ok(rec) || !bytes.Contains(rec.Body.Bytes(), []byte(`"characters":1`)) || !bytes.Contains(rec.Body.Bytes(), []byte(`"operator_name":"Admin"`)) {
		t.Fatalf("admin refresh: %s", rec.Body.String())
	}
	if rec := send(http.MethodPost, "/refresh", superToken); ok(rec) || !bytes.Contains(rec.Body.Bytes(), []byte(`"msg":"blacklist.errors.refreshCooldown"`)) {
		t.Fatalf("super admin bypassed cooldown: %s", rec.Body.String())
	}
	if rec := send(http.MethodGet, "?kind=character", superToken); !ok(rec) || !bytes.Contains(rec.Body.Bytes(), []byte(`"total":1`)) || !bytes.Contains(rec.Body.Bytes(), []byte(`"display":"Route Pilot"`)) {
		t.Fatalf("super admin list: %s", rec.Body.String())
	}
	if rec := send(http.MethodGet, "?keyword=10000001", adminToken); !ok(rec) || !bytes.Contains(rec.Body.Bytes(), []byte(`"kind":"qq"`)) {
		t.Fatalf("admin QQ search: %s", rec.Body.String())
	}
	if rec := send(http.MethodGet, "?kind=alliance", adminToken); ok(rec) || !bytes.Contains(rec.Body.Bytes(), []byte(`"msg":"blacklist.errors.invalidQuery"`)) {
		t.Fatalf("invalid kind accepted: %s", rec.Body.String())
	}
}
