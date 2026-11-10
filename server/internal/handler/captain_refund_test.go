package handler

import (
	"bytes"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"testing"
)

func TestRefundHandlerRejectsInvalidTimeAndID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, path := range []string{"/preview", "/bad"} {
		router := gin.New()
		h := NewCaptainRefundHandler()
		router.POST("/preview", h.Preview)
		router.GET("/:id", h.DetailMine)
		method := "GET"
		if path == "/preview" {
			method = "POST"
		}
		request := httptest.NewRequest(method, path, bytes.NewBufferString(`{"start":"not-a-time"}`))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		var payload struct {
			Code int `json:"code"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		if payload.Code != 400 {
			t.Fatalf("invalid request returned %s", recorder.Body.String())
		}
	}
}
