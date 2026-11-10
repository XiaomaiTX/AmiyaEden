package service

import (
	"amiya-eden/internal/model"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

func (s *QQGroupBusinessService) Inspect(ctx context.Context, now time.Time) error {
	rules, e := s.repo.Rules(ctx)
	if e != nil {
		return e
	}
	for _, rule := range rules {
		if !rule.InspectionEnabled {
			continue
		}
		if e = s.startInspection(ctx, rule.GroupID, now); e != nil {
			return e
		}
	}
	items, e := s.repo.PendingItems(ctx)
	if e != nil {
		return e
	}
	for _, item := range items {
		if e = s.judgeMember(ctx, item.RunID, item.QQ, now); e != nil {
			return e
		}
	}
	return nil
}
func (s *QQGroupBusinessService) startInspection(ctx context.Context, group int64, now time.Time) error {
	return s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		rule, e := s.repo.Rule(tx, group)
		if e != nil {
			return e
		}
		if !rule.InspectionEnabled || now.Sub(rule.LastInspectionAt) < 15*time.Minute {
			return nil
		}
		n, e := s.repo.OpenInspections(tx, group)
		if e != nil {
			return e
		}
		if n > 0 {
			return nil
		}
		run := model.QQGroupInspection{ID: uuid.NewString(), GroupID: group, RuleVersion: rule.Version, Status: "snapshot_pending", SnapshotAt: now}
		op, e := s.dispatch.Submit(ctx, tx, "inspection:"+run.ID+":snapshot", 2, time.Now().Add(30*time.Minute), model.QQBotRequest{Kind: "member_list", GroupID: group})
		if e != nil {
			return e
		}
		run.OperationID = op.ID
		if e = s.repo.CreateInspection(tx, &run); e != nil {
			return e
		}
		rule.LastInspectionAt = now
		return s.repo.SaveRule(tx, &rule)
	})
}
func (s *QQGroupBusinessService) consumeSnapshot(tx *gorm.DB, op model.QQBotOperation) error {
	run, e := s.repo.InspectionForOperation(tx, op.ID)
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return nil
	}
	if e != nil {
		return e
	}
	if run.Status != "snapshot_pending" {
		return nil
	}
	if op.Status != "succeeded" {
		run.Status = op.Status
		return s.repo.SaveInspection(tx, &run)
	}
	rule, e := s.repo.Rule(tx, run.GroupID)
	if e != nil {
		return e
	}
	if !rule.InspectionEnabled || rule.Version != run.RuleVersion {
		run.Status = "cancelled"
		return s.repo.SaveInspection(tx, &run)
	}
	var snapshot []struct {
		QQ int64 `json:"user_id"`
	}
	if e = json.Unmarshal([]byte(op.Result), &snapshot); e != nil {
		return e
	}
	frozen := map[int64]bool{}
	for _, m := range snapshot {
		if m.QQ <= 0 {
			return errors.New("invalid snapshot member")
		}
		frozen[m.QQ] = true
	}
	known, e := s.repo.GroupMembers(tx, run.GroupID)
	if e != nil {
		return e
	}
	for _, m := range known {
		if _, ok := frozen[m.QQ]; !ok {
			frozen[m.QQ] = false
		}
	}
	items := make([]model.QQGroupInspectionItem, 0, len(frozen))
	for qq, present := range frozen {
		items = append(items, model.QQGroupInspectionItem{RunID: run.ID, QQ: qq, Present: present})
	}
	if e = s.repo.FreezeItems(tx, items); e != nil {
		return e
	}
	run.Status = "judging"
	if len(items) == 0 {
		run.Status = "completed"
	}
	return s.repo.SaveInspection(tx, &run)
}
func (s *QQGroupBusinessService) resetChain(tx *gorm.DB, m *model.QQGroupMember) error {
	if e := s.repo.CancelOperation(tx, m.KickOperationID); e != nil {
		return e
	}
	m.Confirmations = 0
	m.MismatchSince = nil
	m.ChainID = ""
	m.LastMismatchRunID = ""
	m.KickOperationID = ""
	m.KickStatus = ""
	return nil
}
func (s *QQGroupBusinessService) judgeMember(ctx context.Context, runID string, qq int64, now time.Time) error {
	return s.repo.Transaction(ctx, func(tx *gorm.DB) error {
		run, e := s.repo.Inspection(tx, runID)
		if e != nil {
			return e
		}
		item, e := s.repo.InspectionItem(tx, runID, qq)
		if e != nil || item.Done {
			return e
		}
		rule, e := s.repo.Rule(tx, run.GroupID)
		if e != nil {
			return e
		}
		if !rule.InspectionEnabled || rule.Version != run.RuleVersion {
			item.Decision = "rule_changed"
		} else {
			m, e := s.repo.Member(tx, run.GroupID, qq)
			if e != nil {
				return e
			}
			if m.MembershipReceivedAt.After(run.SnapshotAt) {
				item.Decision = "newer_membership_event"
			} else {
				if !item.Present {
					m.Present = false
					if e = s.resetChain(tx, &m); e != nil {
						return e
					}
					if e = s.repo.CancelOperation(tx, m.CardOperationID); e != nil {
						return e
					}
					item.Decision = "absent"
				} else {
					if !m.Present {
						m.Present = true
						m.Epoch++
						m.CardTarget = ""
						if e = s.resetChain(tx, &m); e != nil {
							return e
						}
					}
					q, evidence, e := s.qualify(tx, rule, qq)
					if e != nil {
						return e
					}
					m.Evidence = evidence
					item.Decision = q
					item.Evidence = evidence
					if q != "mismatch" {
						if e = s.resetChain(tx, &m); e != nil {
							return e
						}
					} else {
						if m.ConfirmRuleVersion != rule.Version {
							if e = s.resetChain(tx, &m); e != nil {
								return e
							}
							m.ConfirmRuleVersion = rule.Version
						}
						if m.LastMismatchRunID != run.ID {
							if m.MismatchSince == nil {
								since := now
								m.MismatchSince = &since
								m.ChainID = run.ID
							}
							m.Confirmations++
							m.LastMismatchRunID = run.ID
						}
						if rule.AutoKick && m.Confirmations >= rule.MismatchConfirmations && m.MismatchSince != nil && now.Sub(*m.MismatchSince) >= time.Duration(rule.ObservationHours)*time.Hour && m.KickOperationID == "" {
							op, e := s.dispatch.Submit(ctx, tx, fmt.Sprintf("kick:%d:%d:%d:%d:%s", m.GroupID, m.QQ, m.Epoch, rule.Version, m.ChainID), 2, time.Now().Add(15*time.Minute), model.QQBotRequest{Kind: "group_kick", GroupID: m.GroupID, UserID: m.QQ})
							if e != nil {
								return e
							}
							m.KickOperationID = op.ID
							m.KickStatus = "pending"
							item.OperationID = op.ID
						}
					}
					if e = s.planCard(ctx, tx, rule, &m); e != nil {
						return e
					}
				}
				if e = s.repo.SaveMember(tx, &m); e != nil {
					return e
				}
			}
		}
		item.Done = true
		if e = s.repo.SaveItem(tx, &item); e != nil {
			return e
		}
		remaining, e := s.repo.RemainingItems(tx, run.ID)
		if e != nil {
			return e
		}
		if remaining == 0 {
			run.Status = "completed"
			return s.repo.SaveInspection(tx, &run)
		}
		return nil
	})
}
func (s *QQGroupBusinessService) validKick(tx *gorm.DB, op model.QQBotOperation) (bool, error) {
	var request model.QQBotRequest
	if e := json.Unmarshal([]byte(op.Request), &request); e != nil {
		return false, e
	}
	rule, e := s.repo.Rule(tx, request.GroupID)
	if e != nil {
		return false, e
	}
	m, e := s.repo.KickMember(tx, op.ID)
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	if !rule.InspectionEnabled || !rule.AutoKick || !m.Present || m.ConfirmRuleVersion != rule.Version || m.Confirmations < rule.MismatchConfirmations || m.MismatchSince == nil || time.Since(*m.MismatchSince) < time.Duration(rule.ObservationHours)*time.Hour {
		return false, nil
	}
	run, e := s.repo.InspectionEvidence(tx, m.LastMismatchRunID)
	if e != nil {
		return false, e
	}
	if run.RuleVersion != rule.Version {
		return false, nil
	}
	item, e := s.repo.JudgementEvidence(tx, run.ID, m.QQ)
	if e != nil {
		return false, e
	}
	if !item.Done || item.Decision != "mismatch" {
		return false, nil
	}
	q, _, e := s.qualify(tx, rule, m.QQ)
	if e != nil {
		return false, e
	}
	if q != "mismatch" {
		if e = s.resetChain(tx, &m); e != nil {
			return false, e
		}
		return false, s.repo.SaveMember(tx, &m)
	}
	return true, nil
}
