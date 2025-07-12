package errors

import "errors"

var (
	// ErrCheckUpdateImmutableFields is used when the update immutable fields error
	ErrCheckUpdateImmutableFields = errors.New("update immutable fields error")

	// ErrCheckOutputOnlyFields is used when the output only fields error
	ErrCheckOutputOnlyFields = errors.New("can not contain output only fields")

	// ErrCheckRequiredFields is used when the required fields missing
	ErrCheckRequiredFields = errors.New("required fields missing")

	// ErrFieldMask is used when the field mask error
	ErrFieldMask = errors.New("field mask error")

	// ErrSematicVersion is used when the sematic version error
	ErrSematicVersion = errors.New("not a legal version, should be the format vX.Y.Z or vX.Y.Z-identifiers")

	// ErrUpdateMask is used when the update mask error
	ErrUpdateMask = errors.New("update mask error")
)
