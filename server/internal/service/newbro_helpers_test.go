package service

import (
	"reflect"
	"testing"
)

func TestUniqueNonZeroUserIDsPreservesFirstSeenOrder(t *testing.T) {
	got := uniqueNonZeroUserIDs([]uint{0, 42, 7, 42, 0, 9, 7})
	want := []uint{42, 7, 9}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("uniqueNonZeroUserIDs() = %v, want %v", got, want)
	}
}
