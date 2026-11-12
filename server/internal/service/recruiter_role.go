package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"context"
	"slices"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

func derivedRecruiterRoles(current []string, qualified bool) []string {
	roles := make([]string, 0, len(current)+1)
	for _, code := range current {
		if code == model.RoleRecruiter || (qualified && code == model.RoleGuest) {
			continue
		}
		roles = append(roles, code)
	}
	if qualified {
		roles = append(roles, model.RoleRecruiter)
	}
	if len(roles) == 0 {
		roles = append(roles, model.RoleGuest)
	}
	return roles
}

func (s *AutoRoleService) SyncAllRecruiterRoles(ctx context.Context) {
	ids, err := s.userRepo.ListAllIDs()
	if err != nil {
		global.Logger.Warn("[Recruiter] 查询用户失败", zap.Error(err))
		return
	}
	for _, userID := range ids {
		if err := s.SyncRecruiterRole(ctx, userID); err != nil {
			global.Logger.Warn("[Recruiter] 同步职权失败", zap.Uint("user_id", userID), zap.Error(err))
		}
	}
}

// SyncRecruiterRole is intentionally independent of other add-only mappings.
// Sharing a user lock with manual assignment prevents that path from erasing a
// newly derived result or restoring a stale recruiter role after revocation.
func (s *AutoRoleService) SyncRecruiterRole(ctx context.Context, userID uint) error {
	changed := false
	err := s.roleRepo.InUserRoleTx(userID, func(tx *gorm.DB) error {
		current, err := s.roleRepo.GetUserRoleCodesTx(tx, userID)
		if err != nil {
			return err
		}
		scope, err := NewAdmissionService().qualifyingRecruiterCorporations(tx, userID)
		if err != nil {
			return err
		}
		qualified := len(scope) > 0
		if slices.Contains(current, model.RoleRecruiter) == qualified {
			return nil
		}
		if err := s.roleRepo.SetUserRolesTx(tx, userID, derivedRecruiterRoles(current, qualified)); err != nil {
			return err
		}
		changed = true
		return nil
	})
	if err != nil {
		return err
	}
	if changed {
		s.roleSvc.InvalidateUserCache(ctx, userID)
		s.roleSvc.SyncUserPrimaryRole(userID)
		NotifyMumbleIdentityChanged(ctx, userID)
	}
	return nil
}
