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
)
