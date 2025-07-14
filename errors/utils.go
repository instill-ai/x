package errors

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ConvertToGRPCError converts an error to a gRPC status error
func ConvertToGRPCError(err error) error {
	if err == nil {
		return nil
	}

	// If it's already a status, respect the code but update message if needed
	if st, ok := status.FromError(err); ok {
		if msg := Message(err); msg != "" {
			// This conversion is used to preserve the status details.
			p := st.Proto()
			p.Message = msg
			st = status.FromProto(p)
		}
		return st.Err()
	}

	// Use the unified code mapping
	code := ConvertGRPCCode(err)
	return status.Error(code, MessageOrErr(err))
}

// ConvertGRPCCode extracts the gRPC status code from an error
func ConvertGRPCCode(err error) codes.Code {
	if err == nil {
		return codes.OK
	}

	// If it's already a status, return its code
	if st, ok := status.FromError(err); ok {
		return st.Code()
	}

	// Map domain errors to gRPC codes
	switch {
	case errors.Is(err, ErrAlreadyExists):
		return codes.AlreadyExists
	case
		isDuplicateKeyErr(err),
		errors.Is(err, ErrNotFound),
		errors.Is(err, ErrNoDataDeleted),
		errors.Is(err, ErrNoDataUpdated),
		errors.Is(err, ErrMembershipNotFound):
		return codes.NotFound
	case
		errors.Is(err, ErrInvalidArgument),
		errors.Is(err, ErrOwnerTypeNotMatch),
		errors.Is(err, bcrypt.ErrMismatchedHashAndPassword),
		errors.Is(err, ErrCheckUpdateImmutableFields),
		errors.Is(err, ErrCheckOutputOnlyFields),
		errors.Is(err, ErrCheckRequiredFields),
		errors.Is(err, ErrExceedMaxBatchSize),
		errors.Is(err, ErrTriggerFail),
		errors.Is(err, ErrFieldMask),
		errors.Is(err, ErrSematicVersion),
		errors.Is(err, ErrUpdateMask),
		errors.Is(err, ErrResourceID),
		errors.Is(err, ErrCanNotRemoveOwnerFromOrganization),
		errors.Is(err, ErrCanNotSetAnotherOwner),
		errors.Is(err, ErrInvalidRole),
		errors.Is(err, ErrInvalidTokenTTL),
		errors.Is(err, ErrStateCanOnlyBeActive),
		errors.Is(err, ErrPasswordNotMatch),
		errors.Is(err, ErrInvalidOwnerNamespace),
		errors.Is(err, ErrCanNotUsePlaintextSecret):
		return codes.InvalidArgument
	case errors.Is(err, ErrUnauthorized):
		return codes.PermissionDenied
	case errors.Is(err, ErrUnauthenticated):
		return codes.Unauthenticated
	case errors.Is(err, ErrRateLimiting):
		return codes.ResourceExhausted
	default:
		return codes.Unknown
	}
}
