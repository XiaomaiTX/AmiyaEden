package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"amiya-eden/internal/apicontract"
	"amiya-eden/pkg/response"

	"github.com/gin-gonic/gin"
)

func TestNormalizePagination(t *testing.T) {
	for _, tc := range []struct {
		name                                                 string
		page, size, defaultSize, maxSize, wantPage, wantSize int
	}{
		{"standard defaults", -3, 101, 20, 100, 1, 20},
		{"fleet member fixed size", 0, 500, 260, 260, 1, 260},
		{"refund defaults", 0, 0, 200, 200, 1, 200},
		{"explicit values", 2, 250, 20, 1000, 2, 250},
	} {
		t.Run(tc.name, func(t *testing.T) {
			page, size := normalizePagination(tc.page, tc.size, tc.defaultSize, tc.maxSize)
			if page != tc.wantPage || size != tc.wantSize {
				t.Fatalf("pagination = (%d, %d), want (%d, %d)", page, size, tc.wantPage, tc.wantSize)
			}
		})
	}
}

func TestNormalizeLedgerPaginationCapsOversizedPageSize(t *testing.T) {
	page, size := normalizeLedgerPagination(0, 5000)
	if page != 1 || size != 1000 {
		t.Fatalf("pagination = (%d, %d), want (1, 1000)", page, size)
	}
}

type paginationBindingProbe struct {
	apicontract.StrictServerInterface
	tickets *apicontract.AdminListTicketsRequestObject
	fleet   *apicontract.GetFleetRequestObject
	users   *apicontract.ListSystemUsersRequestObject
}

func (p *paginationBindingProbe) AdminListTickets(_ context.Context, req apicontract.AdminListTicketsRequestObject) (apicontract.AdminListTicketsResponseObject, error) {
	p.tickets = &req
	return nil, nil
}

func (p *paginationBindingProbe) GetFleet(_ context.Context, req apicontract.GetFleetRequestObject) (apicontract.GetFleetResponseObject, error) {
	p.fleet = &req
	return nil, nil
}

func (p *paginationBindingProbe) ListSystemUsers(_ context.Context, req apicontract.ListSystemUsersRequestObject) (apicontract.ListSystemUsersResponseObject, error) {
	p.users = &req
	return nil, nil
}

func TestGeneratedPaginationAndFleetPathBinding(t *testing.T) {
	p := &paginationBindingProbe{}
	r := gin.New()
	apicontract.RegisterHandlersWithOptions(r, apicontract.NewStrictHandler(p), apicontract.GinServerOptions{
		BaseURL: "/api/v1",
		MiddlewaresForOperation: func(_ string, policy []string) []gin.HandlerFunc {
			// This test isolates parsing from authentication.
			chain := make([]gin.HandlerFunc, len(policy))
			for i := range chain {
				chain[i] = func(c *gin.Context) { c.Next() }
			}
			return chain
		},
	})
	get := func(path string) *httptest.ResponseRecorder {
		t.Helper()
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1"+path, nil))
		return rec
	}
	const tickets = "/system/ticket/tickets"
	if code := get(tickets).Code; code != http.StatusOK || p.tickets == nil || p.tickets.Params.Current != nil || p.tickets.Params.Size != nil {
		t.Fatalf("omitted pagination: HTTP %d, request %#v", code, p.tickets)
	}
	if code := get(tickets + "?current=2&size=40&user_id=42").Code; code != http.StatusOK || p.tickets == nil {
		t.Fatalf("explicit pagination: HTTP %d, request %#v", code, p.tickets)
	}
	params := p.tickets.Params
	if params.Current == nil || *params.Current != 2 || params.Size == nil || *params.Size != 40 || params.UserId == nil || *params.UserId != 42 {
		t.Fatalf("unexpected parsed parameters: %#v", params)
	}
	for _, query := range []string{"?current=bad", "?size=oops", "?user_id=18446744073709551616"} {
		p.tickets = nil
		rec := get(tickets + query)
		var body struct {
			Code int `json:"code"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || rec.Code != http.StatusOK || body.Code != 400 || p.tickets != nil {
			t.Errorf("query %s: HTTP %d, body %s, request %#v, decode error %v", query, rec.Code, rec.Body, p.tickets, err)
		}
	}
	if code := get("/operation/fleets/abc-123").Code; code != http.StatusOK || p.fleet == nil || p.fleet.Id != "abc-123" {
		t.Fatalf("fleet ID: HTTP %d, request %#v", code, p.fleet)
	}
	const users = "/system/user"
	for _, query := range []string{"", "?status=", "?status=&status=2"} {
		p.users = nil
		if code := get(users + query).Code; code != http.StatusOK || p.users == nil || p.users.Params.Status != nil {
			t.Fatalf("empty status %q: HTTP %d, request %#v", query, code, p.users)
		}
	}
	for query, want := range map[string]int{"?status=abc": 0, "?status=2&status=1": 2, "?status=0": 0} {
		p.users = nil
		if code := get(users + query).Code; code != http.StatusOK || p.users == nil || p.users.Params.Status == nil || *p.users.Params.Status != want {
			t.Fatalf("status %q: HTTP %d, request %#v, want status %d", query, code, p.users, want)
		}
	}

}

// Embed the fallback indirectly so the concrete operation methods take precedence.
type migratedOperationFallback struct {
	apicontract.StrictServerInterface
}

type migratedOperations struct {
	migratedOperationFallback
	*MentorMentorHandler
	*MeHandler
	*SkillReadinessHandler
	*SysConfigHandler
	*TaskHandler
	*ESIRefreshHandler
	*ToolBookmarkHandler
	*WebhookHandler
	*UploadHandler
}

func newMigratedOperationAdapter(operations migratedOperations) apicontract.ServerInterface {
	return apicontract.NewStrictHandler(operations)
}

func registerMigratedOperationRoutes(r *gin.Engine, adapter apicontract.ServerInterface, onError apicontract.GinOperationErrorHandler) {
	apicontract.RegisterHandlersWithOptions(r, adapter, apicontract.GinServerOptions{
		BaseURL:                  "/api/v1",
		ErrorHandlerForOperation: onError,
		// Authentication is outside these handler tests; each test supplies caller state as needed.
		MiddlewaresForOperation: func(_ string, chain []string) []gin.HandlerFunc {
			if len(chain) == 0 {
				return nil
			}
			return []gin.HandlerFunc{func(c *gin.Context) { c.Next() }}
		},
	})
}

func migratedOperationParamError(c *gin.Context, _ string, _ error, _ int) {
	c.Abort()
	response.Fail(c, response.CodeParamError, "请求参数错误")
}
