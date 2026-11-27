package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/service/internal/qqapi"
	"context"
	"errors"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// storeGatewayEvent advances the gateway's in-memory cursor only after this
// minimal projection commits under the current credentials and receiver lease.
func (s *QQOfficialService) storeGatewayEvent(ctx context.Context, owner string, event qqapi.Event) error {
	err := s.locked(ctx, nil, func(tx *gorm.DB, app *model.QQOfficialApp, now time.Time) error {
		if !qqOwnsGateway(app, owner, now) {
			return ErrQQOfficialUnavailable
		}
		row := model.QQOfficialEvent{AppID: s.appID, EventKey: qqHash(event.ID), Kind: event.Type, GroupOpenID: event.GroupOpenID, EventAt: event.At, CreatedAt: now}
		if req := event.JoinRequest; req != nil {
			row.RequestID, row.MemberOpenID, row.Username, row.AutoApproved = req.ID, req.MemberOpenID, qqDisplayName(req.Username), req.AutoApproved
			row.EventAt = req.AppliedAt
		}
		inserted, err := s.repo.InsertEvent(ctx, tx, &row)
		if err != nil {
			return err
		}
		if !inserted {
			return nil
		}
		count, err := s.repo.EventBacklog(ctx, tx, s.appID)
		if err != nil {
			return err
		}
		if count > qqEventMaxPending {
			return ErrQQOfficialCapacity
		}
		app.LastEventAt = &now
		return s.repo.SaveApp(ctx, tx, app)
	})
	if err == nil {
		s.signal()
	}
	return err
}

func qqDisplayName(s string) string {
	runes := []rune(s)
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, string(runes[:min(len(runes), 80)]))
}

func (s *QQOfficialService) TestGroup(ctx context.Context, actor uint, groupID, idempotencyKey string) (model.QQOfficialTask, error) {
	key, err := qqOperationKey(idempotencyKey)
	if err != nil {
		return model.QQOfficialTask{}, err
	}
	var result model.QQOfficialTask
	err = s.locked(ctx, nil, func(tx *gorm.DB, app *model.QQOfficialApp, now time.Time) error {
		input := qqIntentInput{key: qqSourceTest + ":" + key, digest: qqHash(groupID), kind: qqapi.SendText, source: qqSourceTest, text: qqTestText, operator: actor, expires: now.Add(5 * time.Minute)}
		rows, err := s.previous(ctx, tx, input)
		if err != nil {
			return err
		}
		if len(rows) != 0 {
			result = rows[0]
			return nil
		}
		group, err := s.repo.Group(ctx, tx, s.appID, groupID)
		if err != nil {
			return qqOfficialReadError(err)
		}
		if !group.Authorized || !group.Present || group.PushState == qqBlockedPush {
			return ErrQQOfficialNotAuthorized
		}
		rows, err = s.insert(ctx, tx, app, now, input, []model.QQOfficialGroup{group})
		if err == nil {
			result = rows[0]
			err = s.record(ctx, tx, actor, "qq_official_test_message", "qq_official_task", result.ID, nil)
		}
		return err
	})
	if err == nil {
		s.signal()
	}
	return result, err
}

func (s *QQOfficialService) coalesceRead(ctx context.Context, tx *gorm.DB, app *model.QQOfficialApp, now time.Time, in qqIntentInput, group model.QQOfficialGroup) (model.QQOfficialTask, error) {
	if !app.Enabled {
		return model.QQOfficialTask{}, ErrQQOfficialDisabled
	}
	if app.GlobalHold != "" {
		return model.QQOfficialTask{}, ErrQQOfficialUnavailable
	}
	row, err := s.repo.PendingRead(ctx, tx, s.appID, group.ID, in.kind, now)
	if err == nil && row.AppGeneration == app.Generation && row.GroupGeneration == group.Generation && (in.kind != qqapi.JoinList || row.ModeGeneration == group.ApprovalGeneration) {
		count, err := s.repo.MergedIntentCount(ctx, tx, s.appID, row.ID)
		if err != nil {
			return row, err
		}
		if count >= 32 {
			return row, ErrQQOfficialCapacity
		}
		alias := model.QQOfficialIntent{ID: uuid.NewString(), AppID: s.appID, Key: qqHash(in.key), Digest: in.digest, MergedTaskID: row.ID, CreatedAt: now}
		return row, s.repo.InsertIntent(ctx, tx, &alias, nil)
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return row, err
	}
	rows, err := s.insert(ctx, tx, app, now, in, []model.QQOfficialGroup{group})
	if err != nil {
		return row, err
	}
	return rows[0], nil
}

func (s *QQOfficialService) VerifyGroup(ctx context.Context, actor uint, openID, idempotencyKey string) (model.QQOfficialTask, error) {
	key, err := qqOperationKey(idempotencyKey)
	if err != nil || !qqValidOpenID(openID) {
		return model.QQOfficialTask{}, ErrQQOfficialInvalid
	}
	var result model.QQOfficialTask
	err = s.locked(ctx, nil, func(tx *gorm.DB, app *model.QQOfficialApp, now time.Time) error {
		in := qqIntentInput{key: qqSourceVerify + ":" + key, digest: qqHash(openID), kind: qqapi.GroupState, source: qqSourceVerify, operator: actor, expires: now.Add(5 * time.Minute)}
		rows, err := s.previous(ctx, tx, in)
		if err != nil {
			return err
		}
		if len(rows) != 0 {
			result = rows[0]
			return nil
		}
		group, err := s.repo.GroupByOpenID(ctx, tx, s.appID, openID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			count, countErr := s.repo.GroupCount(ctx, tx, s.appID)
			if countErr != nil {
				return countErr
			}
			if count >= qqMaxGroups {
				return ErrQQOfficialCapacity
			}
			group = qqNewGroup(s.appID, openID, now)
			err = s.repo.SaveGroup(ctx, tx, &group)
		}
		if err != nil {
			return err
		}
		result, err = s.coalesceRead(ctx, tx, app, now, in, group)
		if err != nil {
			return err
		}
		return s.record(ctx, tx, actor, "qq_official_verify_group", "qq_official_task", result.ID, nil)
	})
	if err == nil {
		s.signal()
	}
	return result, err
}

func (s *QQOfficialService) RefreshJoinRequests(ctx context.Context, actor uint, groupID, idempotencyKey string) (model.QQOfficialTask, error) {
	key, err := qqOperationKey(idempotencyKey)
	if err != nil {
		return model.QQOfficialTask{}, err
	}
	var result model.QQOfficialTask
	err = s.locked(ctx, nil, func(tx *gorm.DB, app *model.QQOfficialApp, now time.Time) error {
		in := qqIntentInput{key: qqSourceRefresh + ":" + key, digest: qqHash(groupID), kind: qqapi.JoinList, source: qqSourceRefresh, operator: actor, expires: now.Add(15 * time.Minute)}
		rows, err := s.previous(ctx, tx, in)
		if err != nil {
			return err
		}
		if len(rows) != 0 {
			result = rows[0]
			return nil
		}
		group, err := s.repo.Group(ctx, tx, s.appID, groupID)
		if err != nil {
			return qqOfficialReadError(err)
		}
		if !group.Authorized || !group.Present {
			return ErrQQOfficialNotAuthorized
		}
		result, err = s.coalesceRead(ctx, tx, app, now, in, group)
		if err != nil {
			return err
		}
		return s.record(ctx, tx, actor, "qq_official_refresh_join", "qq_official_task", result.ID, nil)
	})
	if err == nil {
		s.signal()
	}
	return result, err
}

func (s *QQOfficialService) JoinRequests(ctx context.Context, group, state string, offset, limit int) ([]model.QQOfficialJoinRequest, int64, error) {
	if s == nil {
		return []model.QQOfficialJoinRequest{}, 0, nil
	}
	bound, err := s.repo.Group(ctx, nil, s.appID, group)
	if err != nil {
		return nil, 0, qqOfficialReadError(err)
	}
	if !bound.Authorized {
		return nil, 0, ErrQQOfficialNotAuthorized
	}
	now := s.now()
	rows, total, err := s.repo.JoinRequests(ctx, s.appID, group, state, now, offset, limit)
	for i := range rows {
		if rows[i].Status == "pending" && !rows[i].ExpiresAt.After(now) {
			rows[i].Status = "expired"
		}
	}
	return rows, total, err
}

func (s *QQOfficialService) DecideJoinRequest(ctx context.Context, actor uint, id, decision, idempotencyKey string) (model.QQOfficialTask, error) {
	key, err := qqOperationKey(idempotencyKey)
	if err != nil || (decision != "approve" && decision != "decline") {
		return model.QQOfficialTask{}, ErrQQOfficialInvalid
	}
	var result model.QQOfficialTask
	err = s.locked(ctx, nil, func(tx *gorm.DB, app *model.QQOfficialApp, now time.Time) error {
		in := qqIntentInput{key: qqSourceDecision + ":" + key, digest: qqHash(id + ":" + decision), kind: qqapi.JoinDecision,
			source: qqSourceDecision, application: id, decision: decision, operator: actor, urgent: true, expires: now.Add(5 * time.Minute)}
		rows, err := s.previous(ctx, tx, in)
		if err != nil {
			return err
		}
		if len(rows) != 0 {
			result = rows[0]
			return nil
		}
		if !s.approvalWrites {
			return ErrQQOfficialNotAuthorized
		}
		req, err := s.repo.JoinRequest(ctx, tx, s.appID, id)
		if err != nil {
			return qqOfficialReadError(err)
		}
		if req.Status != "pending" || req.TaskID != "" {
			return ErrQQOfficialConflict
		}
		if !req.ExpiresAt.After(now) {
			return ErrQQOfficialExpired
		}
		if req.ExpiresAt.Before(in.expires) {
			in.expires = req.ExpiresAt
		}
		group, err := s.repo.Group(ctx, tx, s.appID, req.GroupID)
		if err != nil {
			return qqOfficialReadError(err)
		}
		if !group.Authorized || !group.Present || !group.ApprovalEnabled || group.ApprovalCapability != qqAvailable {
			return ErrQQOfficialNotAuthorized
		}
		rows, err = s.insert(ctx, tx, app, now, in, []model.QQOfficialGroup{group})
		if err != nil {
			return err
		}
		result = rows[0]
		req.Status, req.TaskID = "queued", result.ID
		if err = s.repo.SaveJoinRequest(ctx, tx, &req); err != nil {
			return err
		}
		return s.record(ctx, tx, actor, "qq_official_join_decision", "qq_official_join_request", id, map[string]any{"decision": decision, "task_id": result.ID})
	})
	if err == nil {
		s.signal()
	}
	return result, err
}

func (s *QQOfficialService) projectDecision(ctx context.Context, tx *gorm.DB, task model.QQOfficialTask) error {
	if task.ApplicationID == "" {
		return nil
	}
	req, err := s.repo.JoinRequest(ctx, tx, s.appID, task.ApplicationID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if req.TaskID != task.ID || req.Status != "queued" {
		return nil
	}
	switch task.Status {
	case model.QQSucceeded:
		if task.Decision == "approve" {
			req.Status = "approved"
		} else {
			req.Status = "declined"
		}
	case model.QQFailed, model.QQCancelled:
		req.Status = "failed"
	case model.QQUnknown:
		req.Status = "unknown"
	case model.QQExpired:
		req.Status = "expired"
	default:
		return nil
	}
	return s.repo.SaveJoinRequest(ctx, tx, &req)
}

func (s *QQOfficialService) mergeJoinRequest(ctx context.Context, tx *gorm.DB, group model.QQOfficialGroup, req qqapi.JoinRequest, refresh string, now time.Time) error {
	if req.ID == "" || req.MemberOpenID == "" || req.AppliedAt.IsZero() || req.AppliedAt.After(now.Add(5*time.Minute)) || !req.AppliedAt.Add(24*time.Hour).After(now) {
		return nil
	}
	key := qqHash(req.ID)
	row, err := s.repo.JoinRequestByKey(ctx, tx, s.appID, group.ID, key)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		row = model.QQOfficialJoinRequest{ID: uuid.NewString(), AppID: s.appID, GroupID: group.ID, RequestKey: key, RequestID: req.ID, MemberOpenID: req.MemberOpenID,
			Username: qqDisplayName(req.Username), AppliedAt: req.AppliedAt, ExpiresAt: req.AppliedAt.Add(24 * time.Hour), Status: "pending", CreatedAt: now}
	} else if err != nil {
		return err
	}
	if req.AutoApproved && (row.Status == "pending" || row.Status == "queued") {
		row.Status = "closed_elsewhere"
	}
	if refresh != "" {
		row.RefreshID = refresh
	}
	return s.repo.SaveJoinRequest(ctx, tx, &row)
}

func (s *QQOfficialService) processEvents(ctx context.Context, tx *gorm.DB, now time.Time, limit int) (int, error) {
	rows, err := s.repo.Events(ctx, tx, s.appID, limit)
	if err != nil {
		return 0, err
	}
	for i := range rows {
		row := &rows[i]
		if err = s.applyEvent(ctx, tx, *row, now); err != nil {
			return 0, err
		}
		row.ProcessedAt = &now
		// Do not retain applicant details in two tables once consumed.
		row.RequestID, row.MemberOpenID, row.Username = "", "", ""
		if err = s.repo.SaveEvent(ctx, tx, row); err != nil {
			return 0, err
		}
	}
	return len(rows), nil
}

func (s *QQOfficialService) applyEvent(ctx context.Context, tx *gorm.DB, event model.QQOfficialEvent, now time.Time) error {
	group, err := s.repo.GroupByOpenID(ctx, tx, s.appID, event.GroupOpenID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if event.Kind == "GROUP_JOIN_REQUEST" {
			return nil
		}
		count, err := s.repo.GroupCount(ctx, tx, s.appID)
		if err != nil || count >= qqMaxGroups {
			return err
		}
		group = qqNewGroup(s.appID, event.GroupOpenID, now)
	} else if err != nil {
		return err
	}
	at := event.EventAt
	switch event.Kind {
	case "GROUP_ADD_ROBOT":
		if group.MembershipAt != nil && !at.After(*group.MembershipAt) {
			return nil
		}
		if group.MembershipAt != nil {
			// A new membership is a distinct authorization even if the intervening
			// exit event was lost. Never carry queued work across it.
			group.Authorized, group.NotificationEnabled, group.TestVerified, group.ApprovalEnabled = false, false, false, false
			group.ApprovalCapability = qqUnknownCapability
			group.Generation++
			if err = s.repo.InvalidateQueued(ctx, tx, s.appID, group.ID, "", "group_changed", now); err != nil {
				return err
			}
		}
		group.Present, group.MembershipAt = true, &at
		if group.PushAt != nil && group.PushAt.Before(at) {
			group.PushState, group.PushAt = qqUnknownCapability, nil
		}
	case "GROUP_DEL_ROBOT":
		if group.MembershipAt != nil && at.Before(*group.MembershipAt) {
			return nil
		}
		group.Present, group.MembershipAt = false, &at
		group.Authorized, group.NotificationEnabled, group.TestVerified, group.ApprovalEnabled = false, false, false, false
		group.PushState, group.ApprovalCapability = qqUnknownCapability, qqUnknownCapability
		group.Generation++
		if err = s.repo.InvalidateQueued(ctx, tx, s.appID, group.ID, "", "bot_not_in_group", now); err != nil {
			return err
		}
	case "GROUP_MSG_RECEIVE":
		if group.MembershipAt != nil && at.Before(*group.MembershipAt) || group.PushAt != nil && !at.After(*group.PushAt) {
			return nil
		}
		group.PushAt, group.PushState = &at, qqAllowedPush
	case "GROUP_MSG_REJECT":
		if group.MembershipAt != nil && at.Before(*group.MembershipAt) || group.PushAt != nil && at.Before(*group.PushAt) {
			return nil
		}
		group.PushAt, group.PushState = &at, qqBlockedPush
		group.NotificationEnabled, group.TestVerified = false, false
		group.NotifyGeneration++
		if err = s.repo.InvalidateQueued(ctx, tx, s.appID, group.ID, "notification", "push_rejected", now); err != nil {
			return err
		}
	case "GROUP_JOIN_REQUEST":
		if !group.Present || !group.Authorized || group.MembershipAt != nil && at.Before(*group.MembershipAt) {
			return nil
		}
		return s.mergeJoinRequest(ctx, tx, group, qqapi.JoinRequest{ID: event.RequestID, MemberOpenID: event.MemberOpenID, Username: event.Username, AppliedAt: at, AutoApproved: event.AutoApproved}, "", now)
	default:
		return nil
	}
	return s.repo.SaveGroup(ctx, tx, &group)
}
