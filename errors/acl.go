package errors

import "errors"

// ErrMembershipNotFound is used when the membership not found
var ErrMembershipNotFound = errors.New("membership not found")

// ErrPermissionDenied is used when the user doesn't have permission
var ErrPermissionDenied = errors.New("permission denied")
