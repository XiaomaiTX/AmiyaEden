// Package testutil supplies an isolated PostgreSQL schema for integration tests.
package testutil

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Postgres never touches application tables: every test gets a fresh schema.
// AMIYA_ADMISSION_TEST_DSN must point to a disposable local test database.
func Postgres(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("AMIYA_ADMISSION_TEST_DSN"))
	if dsn == "" {
		t.Skip("set AMIYA_ADMISSION_TEST_DSN for isolated PostgreSQL integration tests")
	}
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	adminSQL, err := admin.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = adminSQL.Close() })
	name := fmt.Sprintf("admission_test_%d", time.Now().UnixNano())
	if err := admin.Exec("CREATE SCHEMA " + name).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := admin.Exec("DROP SCHEMA " + name + " CASCADE").Error; err != nil {
			t.Error(err)
		}
	})
	cfg.RuntimeParams["search_path"] = name
	conn := stdlib.OpenDB(*cfg)
	t.Cleanup(func() { _ = conn.Close() })
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: conn}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	return db
}
