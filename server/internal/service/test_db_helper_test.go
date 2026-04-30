package service

import (
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var serviceTestDBSeq uint64

func newServiceTestDB(t *testing.T, prefix string, models ...interface{}) *gorm.DB {
	t.Helper()

	name := strings.NewReplacer("/", "_", "\\", "_", " ", "_", ":", "_").Replace(t.Name())
	dsn := fmt.Sprintf(
		"file:%s_%s_%d_%d?mode=memory&cache=shared",
		prefix,
		name,
		time.Now().UnixNano(),
		atomic.AddUint64(&serviceTestDBSeq, 1),
	)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if len(models) > 0 {
		if err := db.AutoMigrate(models...); err != nil {
			t.Fatalf("auto migrate: %v", err)
		}
	}
	return db
}
