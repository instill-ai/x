package errors

// package errors contains domain errors that different layers can use to add
// meaning to an error and that middleware can transform to a status code or
// retry policy. This is implemented as a separate package in order to avoid
// cycle import errors.

import "errors"

var (
	// ErrInvalidArgument is used when the provided argument is incorrect (e.g.
	// format, reserved).
	ErrInvalidArgument = errors.New("invalid")

	// ErrNotFound is used when a resource doesn't exist.
	ErrNotFound = errors.New("not found")

	// ErrUnauthorized is used when a request can't be performed due to
	// insufficient permissions.
	ErrUnauthorized = errors.New("unauthorized")

	// ErrAlreadyExists is used when a resource can't be created because it
	// already exists.
	ErrAlreadyExists = AddMessage(errors.New("resource already exists"), "Resource already exists.")
)
