package repository

import (
	"amiya-eden/global"
	"errors"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestShouldRetrySDELayoutFallback(t *testing.T) {
	t.Run("relation missing should retry", func(t *testing.T) {
		err := errors.New(`ERROR: relation "invtypes" does not exist (SQLSTATE 42P01)`)
		if !shouldRetrySDELayoutFallback(err) {
			t.Fatalf("expected layout fallback retry for relation-not-found error")
		}
	})

	t.Run("column missing should retry", func(t *testing.T) {
		err := errors.New(`ERROR: column t.published does not exist (SQLSTATE 42703)`)
		if !shouldRetrySDELayoutFallback(err) {
			t.Fatalf("expected layout fallback retry for column-not-found error")
		}
	})

	t.Run("type mismatch should not retry", func(t *testing.T) {
		err := errors.New(`failed to encode args[7]: unable to encode 1 into binary format for bool (OID 16): cannot find encode plan`)
		if shouldRetrySDELayoutFallback(err) {
			t.Fatalf("did not expect layout fallback retry for type mismatch error")
		}
	})
}

func TestSDELayoutFallbackWithSQLite(t *testing.T) {
	db := newSDERepoTestDB(t)
	for _, stmt := range []string{
		"SELECT * FROM missing_sde_table",
		"SELECT missing_sde_column FROM (SELECT 1 AS id)",
	} {
		if err := db.Exec(stmt).Error; err == nil || !shouldRetrySDELayoutFallback(err) {
			t.Errorf("%q: expected SQLite schema error to trigger fallback, got %v", stmt, err)
		}
	}
	if err := db.Exec("SELECT FROM").Error; err == nil || shouldRetrySDELayoutFallback(err) {
		t.Errorf("expected SQLite syntax error not to trigger fallback, got %v", err)
	}
}

func TestQuerySDEWithLayoutOnlyRetriesSchemaMismatch(t *testing.T) {
	oldDB, oldRedis := global.DB, global.Redis
	global.DB, global.Redis = newSDERepoTestDB(t), nil
	resetSDEQueryErrorStateForTest()
	t.Cleanup(func() {
		global.DB, global.Redis = oldDB, oldRedis
		resetSDEQueryErrorStateForTest()
	})

	layoutErr := &pgconn.PgError{Code: "42703", Message: "column does not exist"}
	connectionErr := errors.New("connection lost")
	for _, tt := range []struct {
		name        string
		primaryErr  error
		wantLayouts []bool
		wantErr     error
	}{
		{"healthy query", nil, []bool{true}, nil},
		{"layout mismatch falls back", layoutErr, []bool{true, false}, nil},
		{"connection error is reported without another query", connectionErr, []bool{true}, connectionErr},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var layouts []bool
			got, err := querySDEWithLayout("GetTypes", func(camelCase bool) ([]int, error) {
				layouts = append(layouts, camelCase)
				if camelCase && tt.primaryErr != nil {
					return nil, tt.primaryErr
				}
				return []int{123}, nil
			})
			if !slices.Equal(layouts, tt.wantLayouts) || !errors.Is(err, tt.wantErr) {
				t.Fatalf("layouts = %v, error = %v; want %v, %v", layouts, err, tt.wantLayouts, tt.wantErr)
			}
			if tt.wantErr == nil && !slices.Equal(got, []int{123}) {
				t.Fatalf("query result = %v; want [123]", got)
			}
			if tt.wantErr != nil && fetchSDEStatusFromDB(t, global.DB)["last_query_error"] != tt.wantErr.Error() {
				t.Fatal("query failure was not recorded")
			}
		})
	}
}

func TestWrapSDEFallbackError(t *testing.T) {
	primary := errors.New(`failed to encode args[7]: unable to encode 1 into binary format for bool (OID 16): cannot find encode plan`)
	fallback := errors.New(`ERROR: relation "invtypes" does not exist (SQLSTATE 42P01)`)

	wrapped := wrapSDEFallbackError(primary, fallback)
	if wrapped == nil {
		t.Fatalf("expected wrapped error")
	}
	if wrapped.Error() != primary.Error() {
		t.Fatalf("expected primary error to be returned for non-layout failures, got: %s", wrapped.Error())
	}
}
