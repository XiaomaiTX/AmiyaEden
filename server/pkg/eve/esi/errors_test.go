package esi

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestHTTPErrorMessageFormatMatchesLegacy(t *testing.T) {
	cases := []struct {
		err  *HTTPError
		want string
	}{
		{newHTTPError(http.StatusForbidden, "", "/corporations/1/assets/", "denied"),
			"ESI error 403 on /corporations/1/assets/: denied"},
		{newHTTPError(http.StatusForbidden, "POST ", "/characters/affiliation/", "denied"),
			"ESI error 403 on POST /characters/affiliation/: denied"},
		{newHTTPError(http.StatusBadRequest, "PUT ", "/fleets/1/", "bad"),
			"ESI error 400 on PUT /fleets/1/: bad"},
		{newHTTPError(http.StatusInternalServerError, "DELETE ", "/fleets/1/members/2/", "boom"),
			"ESI error 500 on DELETE /fleets/1/members/2/: boom"},
		{newPaginatedHTTPError(http.StatusForbidden, 2, "/corporations/1/assets/", "denied"),
			"ESI error 403 on page 2 of /corporations/1/assets/: denied"},
	}
	for _, tc := range cases {
		if got := tc.err.Error(); got != tc.want {
			t.Fatalf("Error() = %q, want %q", got, tc.want)
		}
	}
}

func TestIsHTTPStatus(t *testing.T) {
	forbidden := newHTTPError(http.StatusForbidden, "", "/p", "denied")

	if !IsHTTPStatus(forbidden, http.StatusForbidden) {
		t.Fatal("expected direct 403 to match")
	}
	if !IsHTTPStatus(fmt.Errorf("fetch assets: %w", forbidden), http.StatusForbidden) {
		t.Fatal("expected wrapped 403 to match")
	}
	if IsHTTPStatus(fmt.Errorf("fetch assets: %w", forbidden), http.StatusUnauthorized) {
		t.Fatal("expected 403 not to match 401")
	}
	if IsHTTPStatus(errors.New("ESI error 403 on /p: denied"), http.StatusForbidden) {
		t.Fatal("plain error without HTTPError type must not match")
	}
	if IsHTTPStatus(nil, http.StatusForbidden) {
		t.Fatal("nil error must not match")
	}
	if IsHTTPStatus(errors.New("dial tcp: connection refused"), http.StatusForbidden) {
		t.Fatal("network error must not match")
	}
}
