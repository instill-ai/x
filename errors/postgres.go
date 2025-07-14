package errors

import (
	"errors"

	"github.com/jackc/pgconn"
)

// PostgreSQL error code 23505 indicates a "duplicate key value violates unique
// constraint," which occurs when you try to insert a record with a key that
// already exists in the database.
const pgDuplicateKeyCode = "23505"

func isDuplicateKeyErr(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgDuplicateKeyCode
}
