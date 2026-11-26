package handler

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"amiya-eden/internal/apicontract"
	"amiya-eden/internal/service"
	"amiya-eden/pkg/response"
)

func TestSaveFittingResponseKeepsUnhydratedSlotsNull(t *testing.T) {
	var result apicontract.SaveFitting200JSONResponse
	if err := result.FromInfoFittingSuccess(apicontract.InfoFittingSuccess{
		Code: response.CodeOK, Msg: "success", Data: infoFittingDTO(service.FittingResponse{FittingID: 42}),
	}); err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	if err := result.VisitSaveFittingResponse(recorder); err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Data struct {
			FittingID int64           `json:"fitting_id"`
			Slots     json.RawMessage `json:"slots"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Data.FittingID != 42 || string(decoded.Data.Slots) != "null" {
		t.Fatalf("expected fitting 42 with an explicit null slots field, got %s", recorder.Body.String())
	}
}
