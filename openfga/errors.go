package openfga

import "errors"

// OpenFGA specific errors
var (
	ErrStoreNotFound    = errors.New("openfga store not found")
	ErrModelNotFound    = errors.New("openfga authorization model not found")
	ErrInvalidUserType  = errors.New("invalid user type")
	ErrClientNotSet     = errors.New("openfga client not initialized")
	ErrStoreIDNotSet    = errors.New("store ID not set")
	ErrModelIDNotSet    = errors.New("authorization model ID not set")
	ErrInvalidRequest   = errors.New("invalid request parameters")
)
