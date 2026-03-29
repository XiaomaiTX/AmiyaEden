package handler

import (
	"amiya-eden/pkg/response"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNormalizeLedgerPaginationRejectsOversizedPageSize(t *testing.T) {
	page, pageSize := normalizeLedgerPagination(0, 5000)

	if page != 1 {
		t.Fatalf("page = %d, want 1", page)
	}
	if pageSize != 1000 {
		t.Fatalf("pageSize = %d, want 1000", pageSize)
	}
}

func TestNormalizeFleetMembersPaginationUsesFixedPageSize(t *testing.T) {
	page, pageSize := normalizePagination(0, 500, 260, 260)

	if page != 1 {
		t.Fatalf("page = %d, want 1", page)
	}
	if pageSize != 260 {
		t.Fatalf("pageSize = %d, want 260", pageSize)
	}
}

func TestNormalizeStandardPaginationRejectsOversizedPageSize(t *testing.T) {
	page, pageSize := normalizePagination(-3, 101, 20, 100)

	if page != 1 {
		t.Fatalf("page = %d, want 1", page)
	}
	if pageSize != 20 {
		t.Fatalf("pageSize = %d, want 20", pageSize)
	}
}

func TestParsePaginationQueryUsesDefaultsForMissingAndInvalidValues(t *testing.T) {
	t.Run("missing values", func(t *testing.T) {
		ctx := newPaginationQueryTestContext("")
		page, pageSize, err := parsePaginationQuery(ctx, 20, 100)
		if err != nil {
			t.Fatalf("parsePaginationQuery() error = %v, want nil", err)
		}

		if page != 1 {
			t.Fatalf("page = %d, want 1", page)
		}
		if pageSize != 20 {
			t.Fatalf("pageSize = %d, want 20", pageSize)
		}
	})

	t.Run("invalid current returns error", func(t *testing.T) {
		ctx := newPaginationQueryTestContext("?current=bad")
		_, _, err := parsePaginationQuery(ctx, 20, 100)
		if err == nil {
			t.Fatal("expected error for invalid current query")
		}
	})

	t.Run("invalid size returns error", func(t *testing.T) {
		ctx := newPaginationQueryTestContext("?current=bad&size=oops")
		_, _, err := parsePaginationQuery(ctx, 20, 100)
		if err == nil {
			t.Fatal("expected error for invalid query")
		}
	})

	t.Run("out of range values", func(t *testing.T) {
		ctx := newPaginationQueryTestContext("?current=0&size=101")
		page, pageSize, err := parsePaginationQuery(ctx, 20, 100)
		if err != nil {
			t.Fatalf("parsePaginationQuery() error = %v, want nil", err)
		}

		if page != 1 {
			t.Fatalf("page = %d, want 1", page)
		}
		if pageSize != 20 {
			t.Fatalf("pageSize = %d, want 20", pageSize)
		}
	})
}

func TestParsePaginationQuerySupportsUnboundedSize(t *testing.T) {
	ctx := newPaginationQueryTestContext("?current=2&size=250")
	page, pageSize, err := parseUnboundedPaginationQuery(ctx, 20)
	if err != nil {
		t.Fatalf("parseUnboundedPaginationQuery() error = %v, want nil", err)
	}

	if page != 2 {
		t.Fatalf("page = %d, want 2", page)
	}
	if pageSize != 250 {
		t.Fatalf("pageSize = %d, want 250", pageSize)
	}
}

func TestParseUnboundedPaginationQueryCapsOversizedPageSize(t *testing.T) {
	ctx := newPaginationQueryTestContext("?current=1&size=5000")
	page, pageSize, err := parseUnboundedPaginationQuery(ctx, 20)
	if err != nil {
		t.Fatalf("parseUnboundedPaginationQuery() error = %v, want nil", err)
	}
	if page != 1 {
		t.Fatalf("page = %d, want 1", page)
	}
	if pageSize != 1000 {
		t.Fatalf("pageSize = %d, want 1000 (capped)", pageSize)
	}
}

func TestRequireFleetID(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		ctx := newPaginationQueryTestContext("")
		ctx.Params = gin.Params{{Key: "id", Value: "abc-123"}}
		id := requireFleetID(ctx)
		if id != "abc-123" {
			t.Fatalf("requireFleetID() = %q, want %q", id, "abc-123")
		}
	})

	t.Run("missing", func(t *testing.T) {
		ctx := newPaginationQueryTestContext("")
		id := requireFleetID(ctx)
		if id != "" {
			t.Fatalf("requireFleetID() = %q, want empty", id)
		}
	})
}

func TestRequireUintID(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		ctx := newPaginationQueryTestContext("")
		ctx.Params = gin.Params{{Key: "id", Value: "42"}}
		id := requireUintID(ctx, "id")
		if id != 42 {
			t.Fatalf("requireUintID() = %d, want 42", id)
		}
	})

	t.Run("zero", func(t *testing.T) {
		ctx := newPaginationQueryTestContext("")
		ctx.Params = gin.Params{{Key: "id", Value: "0"}}
		id := requireUintID(ctx, "id")
		if id != 0 {
			t.Fatalf("requireUintID() = %d, want 0 (invalid)", id)
		}
	})

	t.Run("non-numeric", func(t *testing.T) {
		ctx := newPaginationQueryTestContext("")
		ctx.Params = gin.Params{{Key: "id", Value: "abc"}}
		id := requireUintID(ctx, "id")
		if id != 0 {
			t.Fatalf("requireUintID() = %d, want 0 (invalid)", id)
		}
	})

	t.Run("uses custom display name in error response", func(t *testing.T) {
		ctx, recorder := newPaginationQueryTestContextWithRecorder("")
		ctx.Params = gin.Params{{Key: "id", Value: "abc"}}
		id := requireUintID(ctx, "id", "用户ID")
		if id != 0 {
			t.Fatalf("requireUintID() = %d, want 0 (invalid)", id)
		}

		var resp response.Response
		if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}
		if resp.Code != response.CodeParamError {
			t.Fatalf("response code = %d, want %d", resp.Code, response.CodeParamError)
		}
		if resp.Msg != "无效的用户ID" {
			t.Fatalf("response msg = %q, want %q", resp.Msg, "无效的用户ID")
		}
	})
}

func TestParseLedgerPaginationQueryUsesDefaultAndClampsOversizedValues(t *testing.T) {
	t.Run("invalid size returns error", func(t *testing.T) {
		ctx := newPaginationQueryTestContext("?size=oops")
		_, _, err := parseLedgerPaginationQuery(ctx, 20)
		if err == nil {
			t.Fatal("expected error for invalid size query")
		}
	})

	t.Run("oversized value clamps", func(t *testing.T) {
		ctx := newPaginationQueryTestContext("?current=0&size=5000")
		page, pageSize, err := parseLedgerPaginationQuery(ctx, 20)
		if err != nil {
			t.Fatalf("parseLedgerPaginationQuery() error = %v, want nil", err)
		}

		if page != 1 {
			t.Fatalf("page = %d, want 1", page)
		}
		if pageSize != 1000 {
			t.Fatalf("pageSize = %d, want 1000", pageSize)
		}
	})
}

func newPaginationQueryTestContext(rawQuery string) *gin.Context {
	ctx, _ := newPaginationQueryTestContextWithRecorder(rawQuery)
	return ctx
}

func newPaginationQueryTestContextWithRecorder(rawQuery string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("GET", "/test"+rawQuery, nil)
	return ctx, recorder
}
