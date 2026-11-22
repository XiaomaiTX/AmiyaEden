package repository

import (
	"amiya-eden/internal/model"
	"context"
	"gorm.io/gorm"
	"time"
)

func (r *FuxiHallRepository) WithContext(ctx context.Context) *FuxiHallRepository {
	return &FuxiHallRepository{db: r.db.WithContext(ctx)}
}

// A fixed number of grouped queries serves every card; public reads never scan history.
func (r *FuxiHallRepository) ContributionCounts(ids []uint, now time.Time) (map[uint]model.FuxiHallContributions, error) {
	totals := make(map[uint]model.FuxiHallContributions, len(ids))
	if len(ids) == 0 {
		return totals, nil
	}
	type aggregate struct {
		UserID uint
		Count  int64
	}
	queries := []struct {
		query *gorm.DB
		field string
	}{
		{r.db.Model(&model.Fleet{}).Where("deleted_at IS NULL AND end_at <= ?", now).Select("fc_user_id AS user_id, COUNT(*) AS count").Where("fc_user_id IN ?", ids).Group("fc_user_id"), "fleets"},
		{r.db.Model(&model.WelfareApplication{}).Select("reviewed_by AS user_id, COUNT(*) AS count").Where("status = ? AND reviewed_by IN ?", model.WelfareAppStatusDelivered, ids).Group("reviewed_by"), "welfare"},
		{r.db.Model(&model.ShopOrder{}).Select("reviewed_by AS user_id, COUNT(*) AS count").Where("status = ? AND reviewed_by IN ?", model.OrderStatusDelivered, ids).Group("reviewed_by"), "welfare"},
		{r.db.Model(&model.SrpApplication{}).Select("paid_by AS user_id, COUNT(*) AS count").Where("payout_status = ? AND paid_by IN ?", model.SrpPayoutPaid, ids).Group("paid_by"), "srp"},
		{r.db.Model(&model.AdmissionApplication{}).Select("handled_by AS user_id, COUNT(*) AS count").Where("status = ? AND kind IN ? AND handled_by IN ?", model.AdmissionStatusInvited, []string{model.AdmissionKindAlt, model.AdmissionKindNewMember}, ids).Group("handled_by"), "recruitment"},
		{r.db.Table("newbro_recruitment_entry AS entry").Select("link.user_id AS user_id, COUNT(DISTINCT entry.matched_user_id) AS count").Joins("JOIN newbro_recruitment AS link ON link.id = entry.recruitment_id AND link.deleted_at IS NULL").Where("entry.deleted_at IS NULL AND entry.status = ? AND entry.matched_user_id > 0 AND link.user_id IN ?", model.RecruitEntryStatusValid, ids).Group("link.user_id"), "recruitment"},
		{r.db.Model(&model.MentorMenteeRelationship{}).Select("mentor_user_id AS user_id, COUNT(DISTINCT mentee_user_id) AS count").Where("status IN ? AND mentor_user_id IN ?", []string{model.MentorRelationStatusActive, model.MentorRelationStatusGraduated, model.MentorRelationStatusRevoked}, ids).Where("responded_at IS NOT NULL").Group("mentor_user_id"), "mentoring"},
	}
	for _, item := range queries {
		var rows []aggregate
		if err := item.query.Scan(&rows).Error; err != nil {
			return nil, err
		}
		for _, row := range rows {
			v := totals[row.UserID]
			switch item.field {
			case "fleets":
				v.Fleets += row.Count
			case "welfare":
				v.Welfare += row.Count
			case "srp":
				v.SRP += row.Count
			case "recruitment":
				v.Recruitment += row.Count
			case "mentoring":
				v.Mentoring += row.Count
			}
			totals[row.UserID] = v
		}
	}
	// Match the corporation PAP total: issued amounts across all fleet categories and characters.
	var papRows []struct {
		UserID uint
		Total  float64
	}
	if err := r.db.Model(&model.FleetPapLog{}).Select("user_id, COALESCE(SUM(pap_count), 0) AS total").Where("user_id IN ?", ids).Group("user_id").Scan(&papRows).Error; err != nil {
		return nil, err
	}
	for _, row := range papRows {
		v := totals[row.UserID]
		v.CorpPAP = row.Total
		totals[row.UserID] = v
	}
	return totals, nil
}

func (r *FuxiHallRepository) SaveContributionSnapshot(card *model.FuxiHallCard) error {
	return r.db.Model(card).Select("contribution_counts", "contributions_updated_at").Updates(card).Error
}
