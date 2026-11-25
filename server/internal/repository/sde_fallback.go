package repository

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

// querySDEWithLayout retries only when imported table or column naming may differ.
func querySDEWithLayout[T any](operation string, query func(camelCase bool) (T, error)) (T, error) {
	result, err := query(true)
	if err == nil {
		return result, nil
	}
	if shouldRetrySDELayoutFallback(err) {
		fallback, fallbackErr := query(false)
		if fallbackErr == nil {
			return fallback, nil
		}
		err = wrapSDEFallbackError(err, fallbackErr)
	}
	reportSDEQueryError(operation, err)
	var zero T
	return zero, err
}

func shouldRetrySDELayoutFallback(err error) bool {
	if err == nil {
		return false
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "42P01", "42703":
			return true
		default:
			return false
		}
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no such table:") || strings.Contains(msg, "no such column:") ||
		strings.Contains(msg, "relation") && strings.Contains(msg, "does not exist") ||
		strings.Contains(msg, "column") && strings.Contains(msg, "does not exist")
}

func wrapSDEFallbackError(primaryErr, fallbackErr error) error {
	if fallbackErr == nil {
		return nil
	}
	if shouldRetrySDELayoutFallback(primaryErr) {
		return fmt.Errorf("%w; fallback query failed: %v", primaryErr, fallbackErr)
	}
	return primaryErr
}
