package sterr

import (
	"encoding/gob"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CreateErrorBadRequest creates a BadRequest detailed error status
func CreateErrorBadRequest(msg string, fieldViolations []*errdetails.BadRequest_FieldViolation) (*status.Status, error) {
	st := status.New(codes.InvalidArgument, msg)
	st, err := st.WithDetails(
		&errdetails.BadRequest{
			FieldViolations: fieldViolations,
		},
	)

	if err != nil {
		return nil, err
	}

	return st, nil
}

// CreateErrorPreconditionFailure creates a PreconditionFailure detailed error status
func CreateErrorPreconditionFailure(msg string, violations []*errdetails.PreconditionFailure_Violation) (*status.Status, error) {
	st := status.New(codes.FailedPrecondition, msg)
	st, err := st.WithDetails(
		&errdetails.PreconditionFailure{
			Violations: violations,
		},
	)

	if err != nil {
		return nil, err
	}

	return st, nil
}

// CreateErrorResourceInfo creates a ResourceInfo detailed error status
func CreateErrorResourceInfo(code codes.Code, msg string, rscType string, rscName string, owner string, desc string) (*status.Status, error) {
	st := status.New(code, msg)
	st, err := st.WithDetails(
		&errdetails.ResourceInfo{
			ResourceType: rscType,
			ResourceName: rscName,
			Owner:        owner,
			Description:  desc,
		},
	)

	if err != nil {
		return nil, err
	}

	return st, nil
}

// RegisterMessageTypes registers common and custom types in gob
func RegisterMessageTypes() {
	gob.Register(&errdetails.ErrorInfo{})
	gob.Register(&errdetails.RetryInfo{})
	gob.Register(&errdetails.DebugInfo{})
	gob.Register(&errdetails.QuotaFailure{})
	gob.Register(&errdetails.PreconditionFailure{})
	gob.Register(&errdetails.BadRequest{})
	gob.Register(&errdetails.RequestInfo{})
	gob.Register(&errdetails.ResourceInfo{})
	gob.Register(&errdetails.Help{})
	gob.Register(&errdetails.LocalizedMessage{})
	gob.Register(&errdetails.QuotaFailure_Violation{})
	gob.Register(&errdetails.PreconditionFailure_Violation{})
	gob.Register(&errdetails.BadRequest_FieldViolation{})
	gob.Register(&errdetails.Help_Link{})
}
