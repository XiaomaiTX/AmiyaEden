package handler

import (
	"amiya-eden/internal/service"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestContactConflictResponse(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	conflict := &service.ContactConflictError{Contact: "qq", PrimaryCharacterID: 9001, PrimaryCharacterName: "Primary Owner"}
	if !contactConflictResponse(c, fmt.Errorf("wrapped: %w", conflict)) {
		t.Fatal("typed conflict not handled")
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
