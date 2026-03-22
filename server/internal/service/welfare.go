package service

import (
	"amiya-eden/internal/model"
	"amiya-eden/internal/repository"
	"errors"
)

// WelfareService 福利业务逻辑层
type WelfareService struct {
	repo *repository.WelfareRepository
}

func NewWelfareService() *WelfareService {
	return &WelfareService{
		repo: repository.NewWelfareRepository(),
	}
}

// ─────────────────────────────────────────────
//  管理员端 - 福利定义 CRUD
// ─────────────────────────────────────────────

// AdminCreateWelfare 创建福利
func (s *WelfareService) AdminCreateWelfare(w *model.Welfare) error {
	if w.Name == "" {
		return errors.New("福利名称不能为空")
	}
	if w.DistMode != model.WelfareDistModePerUser && w.DistMode != model.WelfareDistModePerCharacter {
		return errors.New("无效的发放模式")
	}
	if w.RequireSkillPlan && (w.SkillPlanID == nil || *w.SkillPlanID == 0) {
		return errors.New("需要技能计划时必须选择技能计划")
	}
	if !w.RequireSkillPlan {
		w.SkillPlanID = nil
	}
	return s.repo.CreateWelfare(w)
}

// AdminUpdateWelfareRequest 更新福利请求
type AdminUpdateWelfareRequest struct {
	Name             string `json:"name"`
	Description      string `json:"description"`
	DistMode         string `json:"dist_mode"`
	RequireSkillPlan bool   `json:"require_skill_plan"`
	SkillPlanID      *uint  `json:"skill_plan_id"`
	Status           int8   `json:"status"`
}

// AdminUpdateWelfare 更新福利
func (s *WelfareService) AdminUpdateWelfare(id uint, req *AdminUpdateWelfareRequest) (*model.Welfare, error) {
	w, err := s.repo.GetWelfareByID(id)
	if err != nil {
		return nil, errors.New("福利不存在")
	}

	if req.Name == "" {
		return nil, errors.New("福利名称不能为空")
	}
	if req.DistMode != model.WelfareDistModePerUser && req.DistMode != model.WelfareDistModePerCharacter {
		return nil, errors.New("无效的发放模式")
	}
	if req.RequireSkillPlan && (req.SkillPlanID == nil || *req.SkillPlanID == 0) {
		return nil, errors.New("需要技能计划时必须选择技能计划")
	}

	w.Name = req.Name
	w.Description = req.Description
	w.DistMode = req.DistMode
	w.RequireSkillPlan = req.RequireSkillPlan
	if req.RequireSkillPlan {
		w.SkillPlanID = req.SkillPlanID
	} else {
		w.SkillPlanID = nil
	}
	w.Status = req.Status

	if err := s.repo.UpdateWelfare(w); err != nil {
		return nil, err
	}
	return w, nil
}

// AdminDeleteWelfare 删除福利（仅当无发放记录时允许）
func (s *WelfareService) AdminDeleteWelfare(id uint) error {
	if _, err := s.repo.GetWelfareByID(id); err != nil {
		return errors.New("福利不存在")
	}

	count, err := s.repo.CountDistributionsByWelfareID(id)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("该福利已有发放记录，无法删除")
	}

	return s.repo.DeleteWelfare(id)
}

// AdminListWelfares 查询福利列表
func (s *WelfareService) AdminListWelfares(page, pageSize int, filter repository.WelfareFilter) ([]model.Welfare, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.ListWelfares(page, pageSize, filter)
}
