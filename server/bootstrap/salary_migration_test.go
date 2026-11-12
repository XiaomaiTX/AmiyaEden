package bootstrap

import (
	"amiya-eden/internal/model"
	"fmt"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"testing"
	"time"
)

func TestSalaryMigrationPreservesCustomSettings(t *testing.T) {
	for _, legacy := range []string{"0", "18"} {
		t.Run(legacy, func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:salary_migration_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
			if err != nil {
				t.Fatal(err)
			}
			if err = db.AutoMigrate(&model.SystemConfig{}); err != nil {
				t.Fatal(err)
			}
			rows := []model.SystemConfig{{Key: model.SysConfigPAPAdminAward, Value: legacy}, {Key: model.SysConfigPAPFCSalary, Value: "789"}, {Key: model.SysConfigPAPFCSalaryLimit, Value: "2"}, {Key: model.SysConfigSalaryShop, Value: "6.5"}}
			if err = db.Create(&rows).Error; err != nil {
				t.Fatal(err)
			}
			for range 2 {
				if err = migrateSalarySettings(db); err != nil {
					t.Fatal(err)
				}
			}
			for key, want := range map[string]string{model.SysConfigSalaryRecruitment: "10", model.SysConfigSalaryWelfare: legacy, model.SysConfigSalaryShop: "6.5", model.SysConfigSalarySRP: "10", model.SysConfigPAPFCSalary: "789", model.SysConfigPAPFCSalaryLimit: "2"} {
				var row model.SystemConfig
				if err = db.First(&row, "key = ?", key).Error; err != nil || row.Value != want {
					t.Fatalf("%s = %s want %s: %v", key, row.Value, want, err)
				}
			}
		})
	}
}
