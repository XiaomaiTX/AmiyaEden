package handler

import (
	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSplitCSV(t *testing.T) {
	input := " esi-location.read_location.v1,esi-ui.open_window.v1; esi-wallet.read_character_wallet.v1  publicData "
	want := []string{
		"esi-location.read_location.v1",
		"esi-ui.open_window.v1",
		"esi-wallet.read_character_wallet.v1",
		"publicData",
	}

	got := splitCSV(input)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("splitCSV(%q) = %v, want %v", input, got, want)
	}
}

func TestSplitAny(t *testing.T) {
	tests := []struct {
		name  string
		input string
		seps  string
		want  []string
	}{
		{
			name:  "repeated separators are ignored",
			input: ",,alpha; beta  gamma;;",
			seps:  ",; ",
			want:  []string{"alpha", "beta", "gamma"},
		},
		{
			name:  "leading and trailing separators are ignored",
			input: "  one two ",
			seps:  " ",
			want:  []string{"one", "two"},
		},
		{
			name:  "string without separators stays whole",
			input: "publicData",
			seps:  ",; ",
			want:  []string{"publicData"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitAny(tt.input, tt.seps)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("splitAny(%q, %q) = %v, want %v", tt.input, tt.seps, got, tt.want)
			}
		})
	}
}

type eveSSOTestFallback struct {
	apicontract.StrictServerInterface
}

func newEveSSOTestWrapper() *apicontract.ServerInterfaceWrapper {
	adapter := apicontract.NewStrictHandler(struct {
		eveSSOTestFallback
		*EveSSOHandler
	}{EveSSOHandler: &EveSSOHandler{}})
	return &apicontract.ServerInterfaceWrapper{Handler: adapter}
}

func TestLoginRejectsAdminOnlyExtraScopesForPublicRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(
		http.MethodGet,
		"/api/v1/sso/eve/login?scopes=esi-killmails.read_corporation_killmails.v1",
		nil,
	)

	newEveSSOTestWrapper().GetEveSsoLogin(ctx)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for public corp killmail scope request, got %d", rec.Code)
	}
	var resp response.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != response.CodeForbidden {
		t.Fatalf("expected response code %d, got %d", response.CodeForbidden, resp.Code)
	}
}

func TestBindLoginRejectsAdminOnlyExtraScopesForNonAdminUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Set("userID", uint(42))
	ctx.Set("roles", []string{"user"})
	ctx.Request = httptest.NewRequest(
		http.MethodGet,
		"/api/v1/sso/eve/bind?scopes=esi-killmails.read_corporation_killmails.v1",
		nil,
	)

	newEveSSOTestWrapper().GetEveSsoBind(ctx)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for non-admin corp killmail scope request, got %d", rec.Code)
	}
	var resp response.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != response.CodeForbidden {
		t.Fatalf("expected response code %d, got %d", response.CodeForbidden, resp.Code)
	}
}

func TestEveSsoStrictRedirectPreservesHeadersAndRelativeLocation(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/sso/eve/callback", nil)
	out := eveSsoRedirectResponse{
		request: request,
		headers: apicontract.EveSsoCallback302ResponseHeaders{
			Location: "../auth/继续?x=1&y=2", CacheControl: "no-store", ReferrerPolicy: "no-referrer",
		},
	}
	w := httptest.NewRecorder()
	if err := out.VisitEveSsoCallbackResponse(w); err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusFound || w.Header().Get("Location") != "/api/v1/sso/auth/%e7%bb%a7%e7%bb%ad?x=1&y=2" {
		t.Fatalf("unexpected redirect: status=%d location=%q", w.Code, w.Header().Get("Location"))
	}
	if w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Fatalf("unexpected redirect cache/referrer headers: %v", w.Header())
	}
	if !strings.Contains(w.Body.String(), "?x=1&amp;y=2") {
		t.Fatalf("redirect body did not HTML-escape the location: %s", w.Body.String())
	}
}

func TestEveSsoStrictRegistrationErrorKeepsNoStoreAndEnvelope(t *testing.T) {
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/sso/eve/registration", nil)
	h := &EveSSOHandler{svc: &service.EveSSOService{}}
	out, err := h.GetEveRegistration(ctx, apicontract.GetEveRegistrationRequestObject{})
	if err != nil {
		t.Fatal(err)
	}
	if err := out.VisitGetEveRegistrationResponse(w); err != nil {
		t.Fatal(err)
	}
	var envelope apicontract.BusinessError
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "no-store" || envelope.Code != 400 || envelope.Data != nil {
		t.Fatalf("unexpected registration response: status=%d headers=%v body=%s", w.Code, w.Header(), w.Body.String())
	}
}
