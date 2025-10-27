package errors

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// PostgreSQL error code 23505 indicates a "duplicate key value violates unique
// constraint," which occurs when you try to insert a record with a key that
// already exists in the database.
const pgDuplicateKeyCode = "23505"

func isDuplicateKeyErr(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgDuplicateKeyCode
}

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

// RepositoryError transforms different database errors to domain ones.
func RepositoryErr(err error) error {
	if err == nil {
		return nil
	}

	if isDuplicateKeyErr(err) || errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrAlreadyExists
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}

	return err
}
