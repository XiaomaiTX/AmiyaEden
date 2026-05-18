package repository

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

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
	return strings.Contains(msg, "relation") && strings.Contains(msg, "does not exist") ||
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
