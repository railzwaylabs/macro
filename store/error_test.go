package store_test

import (
	"errors"
	"testing"

	"github.com/railzwaylabs/macro/store"
	"gorm.io/gorm"
)

func TestNormalizeError(t *testing.T) {
	t.Parallel()

	if err := store.NormalizeError(gorm.ErrRecordNotFound); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("NormalizeError() = %v, want ErrNotFound", err)
	}
	if err := store.NormalizeError(gorm.ErrDuplicatedKey); !errors.Is(err, store.ErrConflict) {
		t.Fatalf("NormalizeError() = %v, want ErrConflict", err)
	}
}
