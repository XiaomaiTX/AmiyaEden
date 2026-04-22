package service

import (
	"amiya-eden/global"
	"amiya-eden/internal/model"
	"amiya-eden/pkg/background"
	"context"
	"errors"
	"runtime/debug"
	"time"

	"go.uber.org/zap"
)

const asyncMailDispatchTimeout = 30 * time.Second

func dispatchMailAttemptAsync(
	send func(ctx context.Context) (MailAttemptSummary, error),
	onError func(summary MailAttemptSummary, err error),
	panicMessage string,
	panicFields ...zap.Field,
) {
	if send == nil {
		return
	}

	runDispatch := func(ctx context.Context) error {
		defer func() {
			recovered := recover()
			if recovered == nil || global.Logger == nil {
				return
			}

			fields := append([]zap.Field{}, panicFields...)
			fields = append(fields,
				zap.Any("panic", recovered),
				zap.ByteString("stack", debug.Stack()),
			)
			global.Logger.Error(panicMessage, fields...)
		}()

		ctx, cancel := context.WithTimeout(ctx, asyncMailDispatchTimeout)
		defer cancel()

		summary, err := send(ctx)
		if err != nil && !errors.Is(err, context.Canceled) && onError != nil {
			onError(summary.withError(err), err)
		}
		return err
	}

	_ = background.RunOrSchedule(context.Background(), global.EnsureBackgroundTaskManager(), "mail_dispatch", runDispatch)
}

func cloneShopOrderForMail(order *model.ShopOrder) *model.ShopOrder {
	if order == nil {
		return nil
	}

	clone := *order
	clone.TransactionID = cloneUintPointer(order.TransactionID)
	clone.ReviewedBy = cloneUintPointer(order.ReviewedBy)
	clone.ReviewedAt = cloneTimePointer(order.ReviewedAt)
	return &clone
}

func cloneWelfareForMail(welfare *model.Welfare) *model.Welfare {
	if welfare == nil {
		return nil
	}

	clone := *welfare
	clone.PayByFuxiCoin = cloneIntPointer(welfare.PayByFuxiCoin)
	clone.MaxCharAgeMonths = cloneIntPointer(welfare.MaxCharAgeMonths)
	clone.MinimumPap = cloneIntPointer(welfare.MinimumPap)
	clone.MinimumFuxiLegionYears = cloneIntPointer(welfare.MinimumFuxiLegionYears)
	clone.SkillPlanIDs = append([]uint(nil), welfare.SkillPlanIDs...)
	clone.SkillPlanNames = append([]string(nil), welfare.SkillPlanNames...)
	return &clone
}

func cloneWelfareApplicationForMail(app *model.WelfareApplication) *model.WelfareApplication {
	if app == nil {
		return nil
	}

	clone := *app
	clone.UserID = cloneUintPointer(app.UserID)
	clone.ReviewedAt = cloneTimePointer(app.ReviewedAt)
	return &clone
}

func cloneUintPointer(value *uint) *uint {
	if value == nil {
		return nil
	}

	clone := *value
	return &clone
}

func cloneIntPointer(value *int) *int {
	if value == nil {
		return nil
	}

	clone := *value
	return &clone
}

func cloneTimePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}

	clone := *value
	return &clone
}
