package gateway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/frankban/quicktest"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/instill-ai/x/constant"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/protoadapt"
	"google.golang.org/protobuf/types/known/emptypb"

	"errors"
	"io"
)

func TestHTTPResponseModifier(t *testing.T) {
	tests := []struct {
		name          string
		headerMD      metadata.MD
		expectedCode  int
		expectedError bool
		description   string
	}{
		{
			name:          "no server metadata",
			headerMD:      nil,
			expectedCode:  200, // default
			expectedError: false,
			description:   "should return nil when no server metadata exists",
		},
		{
			name: "valid http code",
			headerMD: metadata.New(map[string]string{
				"x-http-code": "201",
			}),
			expectedCode:  201,
			expectedError: false,
			description:   "should set http status code from metadata",
		},
		{
			name: "invalid http code",
			headerMD: metadata.New(map[string]string{
				"x-http-code": "invalid",
			}),
			expectedCode:  200, // default
			expectedError: true,
			description:   "should return error for invalid http code",
		},
		{
			name: "no x-http-code header",
			headerMD: metadata.New(map[string]string{
				"other-header": "value",
			}),
			expectedCode:  200, // default
			expectedError: false,
			description:   "should not modify response when x-http-code is not present",
		},
	}

	for _, tt := range tests {
		qt := quicktest.New(t)
		qt.Run(tt.name, func(c *quicktest.C) {
			// Create context with or without server metadata
			ctx := context.Background()
			if tt.headerMD != nil {
				ctx = runtime.NewServerMetadataContext(ctx, runtime.ServerMetadata{
					HeaderMD: tt.headerMD,
				})
			}

			// Create response writer
			w := httptest.NewRecorder()

			// Call HTTPResponseModifier
			err := HTTPResponseModifier(ctx, w, &emptypb.Empty{})

			// Verify results
			if tt.expectedError {
				c.Check(err, quicktest.Not(quicktest.IsNil))
			} else {
				c.Check(err, quicktest.IsNil)
				c.Check(w.Code, quicktest.Equals, tt.expectedCode)
			}
		})
	}
}

func TestErrorHandler_WithMocks(t *testing.T) {
	qt := quicktest.New(t)

	// Use generated mock and cast to runtime.Marshaler
	mockMarshaler := &mockMarshaler{contentType: "application/json"}

	ctx := context.Background()
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	mux := &runtime.ServeMux{}
	err := status.Error(codes.InvalidArgument, "test error")

	// Call ErrorHandler
	ErrorHandler(ctx, mux, mockMarshaler, w, req, err)

	// Verify results
	qt.Check(w.Header().Get("Content-Type"), quicktest.Equals, "application/problem+json")
	qt.Check(w.Code, quicktest.Equals, http.StatusBadRequest)
}

func TestErrorHandler_MarshalError(t *testing.T) {
	qt := quicktest.New(t)

	// Use custom mock with marshal error
	mockMarshaler := &mockMarshaler{contentType: "application/json", marshalErr: errors.New("marshal error")}

	ctx := context.Background()
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	mux := &runtime.ServeMux{}
	err := status.Error(codes.Internal, "test error")

	// Call ErrorHandler
	ErrorHandler(ctx, mux, mockMarshaler, w, req, err)

	// Verify results
	qt.Check(w.Code, quicktest.Equals, http.StatusInternalServerError)
}

func TestErrorHandler_Unauthenticated(t *testing.T) {
	qt := quicktest.New(t)

	// Use custom mock
	mockMarshaler := &mockMarshaler{contentType: "application/json"}

	ctx := context.Background()
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	mux := &runtime.ServeMux{}
	err := status.Error(codes.Unauthenticated, "unauthorized")

	// Call ErrorHandler
	ErrorHandler(ctx, mux, mockMarshaler, w, req, err)

	// Verify results
	qt.Check(w.Header().Get("WWW-Authenticate"), quicktest.Equals, "unauthorized")
	qt.Check(w.Code, quicktest.Equals, http.StatusUnauthorized)
}

func TestErrorHandler_WithTrailers(t *testing.T) {
	qt := quicktest.New(t)

	// Use custom mock
	mockMarshaler := &mockMarshaler{contentType: "application/json"}

	// Create context with server metadata containing trailers
	headerMD := metadata.New(map[string]string{"test-header": "header-value"})
	trailerMD := metadata.New(map[string]string{"test-trailer": "trailer-value"})
	ctx := runtime.NewServerMetadataContext(context.Background(), runtime.ServerMetadata{
		HeaderMD:  headerMD,
		TrailerMD: trailerMD,
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("TE", "trailers")
	w := httptest.NewRecorder()
	mux := &runtime.ServeMux{}
	err := status.Error(codes.OK, "success")

	// Call ErrorHandler
	ErrorHandler(ctx, mux, mockMarshaler, w, req, err)

	// Verify trailer headers were set
	qt.Check(w.Header().Get("Transfer-Encoding"), quicktest.Equals, "chunked")
	qt.Check(w.Header().Get("Trailer"), quicktest.Contains, "Grpc-Trailer-Test-Trailer")
}

func TestCustomHeaderMatcher(t *testing.T) {
	tests := []struct {
		name          string
		key           string
		expectedKey   string
		expectedMatch bool
		description   string
	}{
		{
			name:          "jwt header",
			key:           "jwt-token",
			expectedKey:   "jwt-token",
			expectedMatch: true,
			description:   "should match jwt- prefixed headers",
		},
		{
			name:          "jwt header case insensitive",
			key:           "JWT-TOKEN",
			expectedKey:   "JWT-TOKEN",
			expectedMatch: true,
			description:   "should match jwt- prefixed headers case insensitive",
		},
		{
			name:          "instill header",
			key:           "instill-version",
			expectedKey:   "instill-version",
			expectedMatch: true,
			description:   "should match instill- prefixed headers",
		},
		{
			name:          "github header",
			key:           "x-github-event",
			expectedKey:   "x-github-event",
			expectedMatch: true,
			description:   "should match x-github prefixed headers",
		},
		{
			name:          "accept header",
			key:           "accept",
			expectedKey:   "accept",
			expectedMatch: true,
			description:   "should match accept header",
		},
		{
			name:          "request-id header",
			key:           "request-id",
			expectedKey:   "request-id",
			expectedMatch: true,
			description:   "should match request-id header",
		},
		{
			name:          "traceparent header",
			key:           "traceparent",
			expectedKey:   "traceparent",
			expectedMatch: true,
			description:   "should match traceparent header",
		},
		{
			name:          "tracestate header",
			key:           "tracestate",
			expectedKey:   "tracestate",
			expectedMatch: true,
			description:   "should match tracestate header",
		},
		{
			name:          "unknown header",
			key:           "unknown-header",
			expectedKey:   "",
			expectedMatch: false,
			description:   "should not match unknown headers",
		},
		{
			name:          "empty key",
			key:           "",
			expectedKey:   "",
			expectedMatch: false,
			description:   "should not match empty key",
		},
	}

	for _, tt := range tests {
		qt := quicktest.New(t)
		qt.Run(tt.name, func(c *quicktest.C) {
			key, match := CustomHeaderMatcher(tt.key)
			c.Check(key, quicktest.Equals, tt.expectedKey)
			c.Check(match, quicktest.Equals, tt.expectedMatch)
		})
	}
}

func TestInjectOwnerToContext(t *testing.T) {
	tests := []struct {
		name        string
		userUID     string
		expected    map[string]string
		description string
	}{
		{
			name:    "valid user uid",
			userUID: "test-user-123",
			expected: map[string]string{
				constant.HeaderAuthTypeKey: "user",
				constant.HeaderUserUIDKey:  "test-user-123",
			},
			description: "should inject user uid metadata to context",
		},
		{
			name:    "empty uid",
			userUID: "",
			expected: map[string]string{
				constant.HeaderAuthTypeKey: "user",
				constant.HeaderUserUIDKey:  "",
			},
			description: "should handle empty uid",
		},
	}

	for _, tt := range tests {
		qt := quicktest.New(t)
		qt.Run(tt.name, func(c *quicktest.C) {
			ctx := context.Background()
			resultCtx := InjectOwnerToContext(ctx, tt.userUID)

			// Extract metadata from context
			md, ok := metadata.FromOutgoingContext(resultCtx)
			c.Check(ok, quicktest.Equals, true)

			// Verify expected metadata
			for key, expectedValue := range tt.expected {
				values := md.Get(key)
				if expectedValue == "" {
					// For empty expected values, check if the key exists but has empty value
					if len(values) > 0 {
						c.Check(values[0], quicktest.Equals, "")
					}
				} else {
					c.Check(len(values), quicktest.Equals, 1)
					c.Check(values[0], quicktest.Equals, expectedValue)
				}
			}
		})
	}
}

func TestErrorHandler_GRPCToHTTPCodeConversion(t *testing.T) {
	tests := []struct {
		name           string
		grpcCode       codes.Code
		details        []any
		expectedStatus int
		description    string
	}{
		{
			name:           "OK to 200",
			grpcCode:       codes.OK,
			expectedStatus: http.StatusOK,
			description:    "should map codes.OK to HTTP 200",
		},
		{
			name:           "Canceled to 499",
			grpcCode:       codes.Canceled,
			expectedStatus: 499,
			description:    "should map codes.Canceled to HTTP 499",
		},
		{
			name:           "Unknown to 500",
			grpcCode:       codes.Unknown,
			expectedStatus: http.StatusInternalServerError,
			description:    "should map codes.Unknown to HTTP 500",
		},
		{
			name:           "InvalidArgument to 400",
			grpcCode:       codes.InvalidArgument,
			expectedStatus: http.StatusBadRequest,
			description:    "should map codes.InvalidArgument to HTTP 400",
		},
		{
			name:           "Unauthenticated to 401",
			grpcCode:       codes.Unauthenticated,
			expectedStatus: http.StatusUnauthorized,
			description:    "should map codes.Unauthenticated to HTTP 401",
		},
	}

	for _, tt := range tests {
		qt := quicktest.New(t)
		qt.Run(tt.name, func(c *quicktest.C) {
			// Create mock marshaler
			mockMarshaler := &mockMarshaler{contentType: "application/json"}

			// Create gRPC status with the specified code
			st := status.New(tt.grpcCode, "test error")
			if len(tt.details) > 0 {
				st, _ = st.WithDetails(tt.details[0].(protoadapt.MessageV1))
			}

			// Create context and request
			ctx := context.Background()
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			mux := &runtime.ServeMux{}

			// Call ErrorHandler
			ErrorHandler(ctx, mux, mockMarshaler, w, req, st.Err())

			// Verify HTTP status code
			c.Check(w.Code, quicktest.Equals, tt.expectedStatus)

			// Verify WWW-Authenticate header for Unauthenticated errors
			if tt.grpcCode == codes.Unauthenticated {
				c.Check(w.Header().Get("WWW-Authenticate"), quicktest.Equals, "test error")
			}
		})
	}
}

func TestErrorHandler_FailedPreconditionWithDetails(t *testing.T) {
	tests := []struct {
		name           string
		violationType  string
		expectedStatus int
		description    string
	}{
		{
			name:           "UPDATE violation to 422",
			violationType:  "UPDATE",
			expectedStatus: http.StatusUnprocessableEntity,
			description:    "should map UPDATE violation to HTTP 422",
		},
		{
			name:           "DELETE violation to 422",
			violationType:  "DELETE",
			expectedStatus: http.StatusUnprocessableEntity,
			description:    "should map DELETE violation to HTTP 422",
		},
		{
			name:           "STATE violation to 422",
			violationType:  "STATE",
			expectedStatus: http.StatusUnprocessableEntity,
			description:    "should map STATE violation to HTTP 422",
		},
		{
			name:           "unknown violation to 400",
			violationType:  "UNKNOWN",
			expectedStatus: http.StatusBadRequest,
			description:    "should map unknown violation to HTTP 400",
		},
	}

	for _, tt := range tests {
		qt := quicktest.New(t)
		qt.Run(tt.name, func(c *quicktest.C) {

			// Create mock marshaler
			mockMarshaler := &mockMarshaler{contentType: "application/json"}

			// Create PreconditionFailure detail
			preconditionFailure := &errdetails.PreconditionFailure{
				Violations: []*errdetails.PreconditionFailure_Violation{
					{
						Type: tt.violationType,
					},
				},
			}

			// Create gRPC status with FailedPrecondition and details
			st := status.New(codes.FailedPrecondition, "precondition failed")
			st, _ = st.WithDetails(preconditionFailure)

			// Create context and request
			ctx := context.Background()
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			mux := &runtime.ServeMux{}

			// Call ErrorHandler
			ErrorHandler(ctx, mux, mockMarshaler, w, req, st.Err())

			// Verify HTTP status code
			c.Check(w.Code, quicktest.Equals, tt.expectedStatus)
		})
	}
}

func TestErrorHandler_FailedPreconditionMultipleViolations(t *testing.T) {
	// Create PreconditionFailure with multiple violations
	preconditionFailure := &errdetails.PreconditionFailure{
		Violations: []*errdetails.PreconditionFailure_Violation{
			{
				Type: "UNKNOWN", // First violation - should be used
			},
			{
				Type: "UPDATE", // Second violation - should NOT be used
			},
		},
	}

	// Create gRPC status with FailedPrecondition and details
	st := status.New(codes.FailedPrecondition, "precondition failed")
	st, _ = st.WithDetails(preconditionFailure)

	// Create context and request
	ctx := context.Background()
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	marshaler := &mockMarshaler{contentType: "application/json"}
	mux := &runtime.ServeMux{}

	// Call ErrorHandler
	ErrorHandler(ctx, mux, marshaler, w, req, st.Err())

	// Should use the first violation type (UNKNOWN) which maps to 400
	qt := quicktest.New(t)
	qt.Check(w.Code, quicktest.Equals, http.StatusBadRequest)
}

func TestErrorHandler_FailedPreconditionEmptyViolations(t *testing.T) {
	// Create PreconditionFailure with empty violations
	preconditionFailure := &errdetails.PreconditionFailure{
		Violations: []*errdetails.PreconditionFailure_Violation{},
	}

	// Create gRPC status with FailedPrecondition and details
	st := status.New(codes.FailedPrecondition, "precondition failed")
	st, _ = st.WithDetails(preconditionFailure)

	// Create context and request
	ctx := context.Background()
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	marshaler := &mockMarshaler{contentType: "application/json"}
	mux := &runtime.ServeMux{}

	// Call ErrorHandler
	ErrorHandler(ctx, mux, marshaler, w, req, st.Err())

	// Should default to 400 when no violations
	qt := quicktest.New(t)
	qt.Check(w.Code, quicktest.Equals, http.StatusBadRequest)
}

type mockMarshaler struct {
	contentType string
	marshalErr  error
}

func (m *mockMarshaler) ContentType(v interface{}) string {
	return m.contentType
}

func (m *mockMarshaler) Marshal(v interface{}) ([]byte, error) {
	if m.marshalErr != nil {
		return nil, m.marshalErr
	}
	return []byte(`{"error":"test"}`), nil
}

func (m *mockMarshaler) Unmarshal(data []byte, v interface{}) error {
	return nil
}

func (m *mockMarshaler) NewDecoder(r io.Reader) runtime.Decoder {
	return &mockDecoder{}
}

func (m *mockMarshaler) NewEncoder(w io.Writer) runtime.Encoder {
	return &mockEncoder{}
}

type mockDecoder struct{}

func (d *mockDecoder) Decode(v interface{}) error {
	return nil
}

type mockEncoder struct{}

func (e *mockEncoder) Encode(v interface{}) error {
	return nil
}
