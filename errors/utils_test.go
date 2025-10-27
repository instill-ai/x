package errors

import (
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	qt "github.com/frankban/quicktest"
)

func TestConvertToGRPCError(t *testing.T) {
	c := qt.New(t)

	c.Run("nil error", func(c *qt.C) {
		c.Assert(ConvertToGRPCError(nil), qt.IsNil)
	})

	c.Run("already a gRPC status", func(c *qt.C) {
		originalErr := status.Error(codes.FailedPrecondition, "pipeline recipe error")
		result := ConvertToGRPCError(originalErr)

		c.Assert(result, qt.IsNotNil)
		st, ok := status.FromError(result)
		c.Assert(ok, qt.IsTrue)
		c.Assert(st.Code(), qt.Equals, codes.FailedPrecondition)
		c.Assert(st.Message(), qt.Equals, "pipeline recipe error")
	})

	c.Run("gRPC status with end-user message", func(c *qt.C) {
		originalErr := AddMessage(
			status.Error(codes.FailedPrecondition, "pipeline recipe error"),
			"Invalid recipe in pipeline",
		)
		result := ConvertToGRPCError(originalErr)

		c.Assert(result, qt.IsNotNil)
		st, ok := status.FromError(result)
		c.Assert(ok, qt.IsTrue)
		c.Assert(st.Code(), qt.Equals, codes.FailedPrecondition)
		c.Assert(st.Message(), qt.Equals, "Invalid recipe in pipeline")
	})

	c.Run("domain error with message", func(c *qt.C) {
		originalErr := AddMessage(ErrNotFound, "User not found")
		result := ConvertToGRPCError(originalErr)

		c.Assert(result, qt.IsNotNil)
		st, ok := status.FromError(result)
		c.Assert(ok, qt.IsTrue)
		c.Assert(st.Code(), qt.Equals, codes.NotFound)
		c.Assert(st.Message(), qt.Equals, "User not found")
	})

	c.Run("domain error without message", func(c *qt.C) {
		result := ConvertToGRPCError(ErrNotFound)

		c.Assert(result, qt.IsNotNil)
		st, ok := status.FromError(result)
		c.Assert(ok, qt.IsTrue)
		c.Assert(st.Code(), qt.Equals, codes.NotFound)
		c.Assert(st.Message(), qt.Equals, "not found")
	})
}

func TestConvertGRPCCode(t *testing.T) {
	c := qt.New(t)

	c.Run("nil error", func(c *qt.C) {
		c.Assert(ConvertGRPCCode(nil), qt.Equals, codes.OK)
	})

	c.Run("already a gRPC status", func(c *qt.C) {
		originalErr := status.Error(codes.FailedPrecondition, "test error")
		c.Assert(ConvertGRPCCode(originalErr), qt.Equals, codes.FailedPrecondition)
	})

	testcases := []struct {
		name     string
		in       error
		wantCode codes.Code
	}{
		{
			name:     "ErrAlreadyExists",
			in:       ErrAlreadyExists,
			wantCode: codes.AlreadyExists,
		},
		{
			name:     "gorm.ErrDuplicatedKey",
			in:       gorm.ErrDuplicatedKey,
			wantCode: codes.AlreadyExists,
		},
		{
			name:     "ErrNotFound",
			in:       ErrNotFound,
			wantCode: codes.NotFound,
		},
		{
			name:     "gorm.ErrRecordNotFound",
			in:       gorm.ErrRecordNotFound,
			wantCode: codes.NotFound,
		},
		{
			name:     "ErrNoDataDeleted",
			in:       ErrNoDataDeleted,
			wantCode: codes.NotFound,
		},
		{
			name:     "ErrNoDataUpdated",
			in:       ErrNoDataUpdated,
			wantCode: codes.NotFound,
		},
		{
			name:     "ErrMembershipNotFound",
			in:       ErrMembershipNotFound,
			wantCode: codes.NotFound,
		},
		{
			name:     "duplicate key error",
			in:       &pgconn.PgError{Code: "23505"},
			wantCode: codes.AlreadyExists,
		},
		{
			name:     "ErrInvalidArgument",
			in:       ErrInvalidArgument,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "ErrOwnerTypeNotMatch",
			in:       ErrOwnerTypeNotMatch,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "bcrypt.ErrMismatchedHashAndPassword",
			in:       bcrypt.ErrMismatchedHashAndPassword,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "ErrCheckUpdateImmutableFields",
			in:       ErrCheckUpdateImmutableFields,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "ErrCheckOutputOnlyFields",
			in:       ErrCheckOutputOnlyFields,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "ErrCheckRequiredFields",
			in:       ErrCheckRequiredFields,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "ErrExceedMaxBatchSize",
			in:       ErrExceedMaxBatchSize,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "ErrTriggerFail",
			in:       ErrTriggerFail,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "ErrFieldMask",
			in:       ErrFieldMask,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "ErrSematicVersion",
			in:       ErrSematicVersion,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "ErrUpdateMask",
			in:       ErrUpdateMask,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "ErrResourceID",
			in:       ErrResourceID,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "ErrCanNotRemoveOwnerFromOrganization",
			in:       ErrCanNotRemoveOwnerFromOrganization,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "ErrCanNotSetAnotherOwner",
			in:       ErrCanNotSetAnotherOwner,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "ErrInvalidRole",
			in:       ErrInvalidRole,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "ErrInvalidTokenTTL",
			in:       ErrInvalidTokenTTL,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "ErrStateCanOnlyBeActive",
			in:       ErrStateCanOnlyBeActive,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "ErrPasswordNotMatch",
			in:       ErrPasswordNotMatch,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "ErrInvalidOwnerNamespace",
			in:       ErrInvalidOwnerNamespace,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "ErrCanNotUsePlaintextSecret",
			in:       ErrCanNotUsePlaintextSecret,
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "ErrUnauthorized",
			in:       ErrUnauthorized,
			wantCode: codes.PermissionDenied,
		},
		{
			name:     "ErrUnauthenticated",
			in:       ErrUnauthenticated,
			wantCode: codes.Unauthenticated,
		},
		{
			name:     "ErrRateLimiting",
			in:       ErrRateLimiting,
			wantCode: codes.ResourceExhausted,
		},
		{
			name:     "unknown error",
			in:       fmt.Errorf("some unknown error"),
			wantCode: codes.Unknown,
		},
		{
			name:     "wrapped ErrNotFound",
			in:       fmt.Errorf("finding item: %w", ErrNotFound),
			wantCode: codes.NotFound,
		},
	}

	for _, tc := range testcases {
		c.Run(tc.name, func(c *qt.C) {
			got := ConvertGRPCCode(tc.in)
			c.Assert(got, qt.Equals, tc.wantCode, qt.Commentf("%s", tc.name))
		})
	}
}
