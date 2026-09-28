package store

import (
	"errors"

	"gorm.io/gorm"
)

var (
	ErrNotFound = errors.New("store: not found")
	ErrConflict = errors.New("store: conflict")
)

// NormalizeError maps persistence-specific errors to stable Macro errors.
func NormalizeError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, gorm.ErrRecordNotFound):
		return errors.Join(ErrNotFound, err)
	case errors.Is(err, gorm.ErrDuplicatedKey):
		return errors.Join(ErrConflict, err)
	default:
		return err
	}
}
