package repository

import (
	"errors"
	"testing"
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
