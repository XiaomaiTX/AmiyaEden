package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/service/internal/qqapi"
	"context"
	"testing"
	"time"
)

func TestQQOfficialInvalidTokenFencesOtherWorkerCaches(t *testing.T) {
	s, db, now := qqFixture(t)
	qqEnable(t, s)
	task := qqSubmitTest(t, s, qqGroupFixture(t, s, 1))
	stale := qqToken(*now)
	s.setToken(stale, 0)
	other := NewQQOfficialService(db, s.appID, "sample-secret", true)
	other.now = s.now
	other.setToken(stale, 0)
	s.client = qqFakeClient{execute: func(context.Context, string, qqapi.Request) (qqapi.Result, error) {
		return qqapi.Result{}, &qqapi.OperationError{Kind: qqapi.InvalidToken, HTTPStatus: 401}
	}}
	other.client = qqFakeClient{
		refresh: func(context.Context) (qqapi.Token, error) {
			return qqapi.Token{AccessToken: "new", ExpiresAt: now.Add(time.Hour)}, nil
		},
		execute: func(_ context.Context, token string, _ qqapi.Request) (qqapi.Result, error) {
			if token != "new" {
				t.Fatalf("revoked token used again: %q", token)
			}
			return qqapi.Result{MessageID: "confirmed"}, nil
		},
	}
	if got, err := s.dispatchOnce(t.Context()); !got || err != nil {
		t.Fatalf("dispatch %t %v", got, err)
	}
	*now = now.Add(time.Second)
	if got, err := other.dispatchOnce(t.Context()); !got || err != nil {
		t.Fatalf("refresh %t %v", got, err)
	}
	token, epoch := other.currentToken()
	if token.AccessToken != "new" || epoch != 1 {
		t.Fatalf("cache token updated=%t epoch=%d", token.AccessToken == "new", epoch)
	}
	*now = now.Add(time.Second)
	if got, err := other.dispatchOnce(t.Context()); !got || err != nil {
		t.Fatalf("retry %t %v", got, err)
	}
	row, attempts, err := s.Task(t.Context(), task.ID)
	if err != nil || row.Status != model.QQSucceeded || len(attempts) != 2 || !row.CredentialRetried {
		t.Fatalf("retry status=%s attempts=%d error=%v", row.Status, len(attempts), err)
	}
}
