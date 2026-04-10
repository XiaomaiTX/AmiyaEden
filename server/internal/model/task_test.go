package model

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestTaskExecutionAutoMigrateCreatesHistoryIndexes(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:task_model_indexes?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := db.AutoMigrate(&TaskSchedule{}, &TaskExecution{}); err != nil {
		t.Fatalf("auto migrate task models: %v", err)
	}

	migrator := db.Migrator()
	if !migrator.HasTable(&TaskSchedule{}) {
		t.Fatal("expected task_schedules table to exist")
	}
	if !migrator.HasTable(&TaskExecution{}) {
		t.Fatal("expected task_executions table to exist")
	}
	if !migrator.HasIndex(&TaskExecution{}, "idx_task_exec_name_started") {
		t.Fatal("expected composite task execution index to exist")
	}
	if !migrator.HasIndex(&TaskExecution{}, "idx_task_exec_started_at") {
		t.Fatal("expected standalone started_at task execution index to exist")
	}
}
