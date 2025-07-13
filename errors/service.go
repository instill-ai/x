package errors

import (
	"errors"
	"fmt"
)

var (
	// ErrUnauthenticated is used when the authentication fails
	ErrUnauthenticated = errors.New("unauthenticated")

	// ErrRateLimiting is used when the rate limiting occurs
	ErrRateLimiting = errors.New("rate limiting")

	// ErrExceedMaxBatchSize is used when the batch size exceeds the maximum limit
	ErrExceedMaxBatchSize = errors.New("the batch size can not exceed 32")

	// ErrTriggerFail is used when the trigger fails
	ErrTriggerFail = errors.New("failed to trigger the pipeline")

	// ErrCanNotUsePlaintextSecret is used when the plaintext value in credential field is detected
	ErrCanNotUsePlaintextSecret = AddMessage(
		fmt.Errorf("%w: plaintext value in credential field", ErrInvalidArgument),
		"Plaintext values are forbidden in credential fields. You can create a secret and reference it with the syntax ${secret.my-secret}.",
	)

	// ErrInvalidTokenTTL is used when the token TTL is invalid
	ErrInvalidTokenTTL = errors.New("invalid token ttl")

	// ErrInvalidRole is used when the role is invalid
	ErrInvalidRole = errors.New("invalid role")

	// ErrInvalidOwnerNamespace is used when the owner namespace format is invalid
	ErrInvalidOwnerNamespace = errors.New("invalid owner namespace format")

	// ErrStateCanOnlyBeActive is used when the state can only be active
	ErrStateCanOnlyBeActive = errors.New("state can only be active")

	// ErrCanNotRemoveOwnerFromOrganization is used when trying to remove owner from organization
	ErrCanNotRemoveOwnerFromOrganization = errors.New("can not remove owner from organization")

	// ErrCanNotSetAnotherOwner is used when trying to set another user as owner
	ErrCanNotSetAnotherOwner = errors.New("can not set another user as owner")

	// ErrPasswordNotMatch is used when passwords do not match
	ErrPasswordNotMatch = errors.New("password not match")
)
