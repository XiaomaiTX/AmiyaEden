package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"testing"
)

func TestQQGovernanceServiceSavePolicy(t *testing.T) {
	db := newServiceTestDB(t, "qq_governance_policy", &model.QQGroupGovernancePolicy{})
	svc := NewQQGovernanceServiceWithRepository(repository.NewQQGovernanceRepositoryWithDB(db))
	policy, err := svc.SavePolicy(QQGovernancePolicyInput{GroupID: 100, Enabled: true, AllowedCorporationIDs: []int64{200}, AllowedRoleCodes: []string{model.RoleAdmin}, MemberViolationPolicy: model.QQGovernanceViolationAutoKick, CardTemplate: "{nickname}"}, 1)
	if err != nil {
		t.Fatalf("save valid policy: %v", err)
	}
	if policy.GroupID != 100 || policy.MemberViolationPolicy != model.QQGovernanceViolationAutoKick {
		t.Fatalf("policy = %#v", policy)
	}
}
