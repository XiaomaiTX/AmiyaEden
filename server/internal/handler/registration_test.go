package handler

import (
	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/service"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestContactConflictResponse(t *testing.T) {
	w := httptest.NewRecorder()
	conflict := &service.ContactConflictError{Contact: "qq", PrimaryCharacterID: 9001, PrimaryCharacterName: "Primary Owner"}
	payload, ok := contactConflictPayload(fmt.Errorf("wrapped: %w", conflict))
	if !ok {
		t.Fatal("typed conflict not handled")
	}
	out := apicontract.CompleteEveRegistration200JSONResponse{
		Headers: apicontract.CompleteEveRegistration200ResponseHeaders{CacheControl: "no-store"},
	}
	if err := out.FromContactConflict(payload); err != nil {
		t.Fatal(err)
	}
	if err := out.VisitCompleteEveRegistrationResponse(w); err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("status = %d, headers = %v", w.Code, w.Header())
	}
	var body struct {
		Code int                          `json:"code"`
		Data service.ContactConflictError `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Code != 409 || body.Data.PrimaryCharacterName != "Primary Owner" {
		t.Fatalf("response: %s", w.Body.String())
	}
}
