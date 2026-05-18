package model

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMapSolarSystemBooleanFlagsSupportNullTrueFalse(t *testing.T) {
	dsn := fmt.Sprintf("file:map_solar_system_bool_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := db.AutoMigrate(&MapSolarSystem{}); err != nil {
		t.Fatalf("auto migrate mapSolarSystems: %v", err)
	}

	seed := MapSolarSystem{
		SolarSystemID:   30000142,
		SolarSystemName: "Jita",
		Border:          sql.NullBool{Bool: true, Valid: true},
		Fringe:          sql.NullBool{Bool: false, Valid: true},
		Corridor:        sql.NullBool{Bool: true, Valid: true},
		Hub:             sql.NullBool{Bool: true, Valid: true},
		International:   sql.NullBool{Bool: false, Valid: true},
		Regional:        sql.NullBool{},
		Constellation:   sql.NullBool{},
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("create mapSolarSystems row: %v", err)
	}

	var got MapSolarSystem
	if err := db.Where(`"solarSystemID" = ?`, seed.SolarSystemID).First(&got).Error; err != nil {
		t.Fatalf("load mapSolarSystems row: %v", err)
	}

	if !got.Border.Valid || !got.Border.Bool {
		t.Fatalf("unexpected border value: %#v", got.Border)
	}
	if !got.Fringe.Valid || got.Fringe.Bool {
		t.Fatalf("unexpected fringe value: %#v", got.Fringe)
	}
	if got.Regional.Valid {
		t.Fatalf("regional should remain NULL, got %#v", got.Regional)
	}
	if got.Constellation.Valid {
		t.Fatalf("constellation should remain NULL, got %#v", got.Constellation)
	}

	data, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal mapSolarSystem json: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal mapSolarSystem json: %v", err)
	}

	border, ok := payload["border"].(map[string]any)
	if !ok {
		t.Fatalf("json border type = %T, want object", payload["border"])
	}
	if border["Bool"] != true || border["Valid"] != true {
		t.Fatalf("json border payload = %#v, want Bool=true Valid=true", border)
	}
}
