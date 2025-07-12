package errors

import (
	"errors"
	"fmt"
)

var (
	// ErrOwnerTypeNotMatch is used when the owner type not match
	ErrOwnerTypeNotMatch = errors.New("owner type not match")

	// ErrNoDataDeleted is used when no data deleted occurs
	ErrNoDataDeleted = errors.New("no data deleted")

	// ErrNoDataUpdated is used when no data updated occurs
	ErrNoDataUpdated = errors.New("no data updated")
)

// NewPageTokenErr is used to create a new page token error
func NewPageTokenErr(err error) error {
	return fmt.Errorf("%w: invalid page token: %w", ErrInvalidArgument, err)
}
