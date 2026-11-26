package handler

import (
	"amiya-eden/internal/apicontract"
	"amiya-eden/pkg/response"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRefundHandlerRejectsInvalidTimeAndID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adapter := apicontract.NewStrictHandler(strictNewbroTestOperations{
		CaptainRefundHandler: NewCaptainRefundHandler(),
	})
	wrapper := apicontract.ServerInterfaceWrapper{Handler: adapter}
	router := gin.New()
	const base = "/api/v1/newbro/captain/refunds"
	router.POST(base+"/preview", wrapper.PreviewCaptainRefund)
	router.GET(base+"/:id", wrapper.GetMyCaptainRefund)

	for _, tc := range []struct{ name, method, path, body string }{
		{"invalid time", http.MethodPost, base + "/preview", `{"start":"not-a-time"}`},
		{"invalid ID", http.MethodGet, base + "/bad", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			request.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			var payload struct {
				Code int `json:"code"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			if recorder.Code != http.StatusOK || payload.Code != response.CodeParamError {
				t.Fatalf("invalid request returned HTTP %d: %s", recorder.Code, recorder.Body)
			}
		})
	}
}
