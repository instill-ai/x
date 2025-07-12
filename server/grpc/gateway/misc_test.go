package gateway

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/instill-ai/x/constant"

	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Mock marshaler for testing
type mockMarshaler struct {
	contentType string
	marshalErr  error
}

func (m *mockMarshaler) ContentType(any) string {
	return m.contentType
}

func (m *mockMarshaler) Marshal(any) ([]byte, error) {
	if m.marshalErr != nil {
		return nil, m.marshalErr
	}
	return []byte(`{"test": "data"}`), nil
}

func (m *mockMarshaler) Unmarshal([]byte, any) error {
	return nil
}

func (m *mockMarshaler) NewDecoder(r io.Reader) runtime.Decoder {
	return &mockDecoder{}
}

func (m *mockMarshaler) NewEncoder(w io.Writer) runtime.Encoder {
	return &mockEncoder{}
}

type mockDecoder struct{}

func (d *mockDecoder) Decode(v any) error {
	return nil
}

type mockEncoder struct{}

func (e *mockEncoder) Encode(v any) error {
	return nil
}

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
		t.Run(tt.name, func(t *testing.T) {
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
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
				assert.Equal(t, tt.expectedCode, w.Code, tt.description)
			}
		})
	}
}

func TestErrorHandler(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		contentType    string
		marshalErr     error
		teHeader       string
		headerMD       metadata.MD
		trailerMD      metadata.MD
		expectedStatus int
		description    string
	}{
		{
			name:           "successful error handling",
			err:            status.Error(codes.InvalidArgument, "invalid argument"),
			contentType:    "application/json",
			marshalErr:     nil,
			teHeader:       "",
			expectedStatus: 200, // default
			description:    "should handle error successfully",
		},
		{
			name:           "unauthenticated error",
			err:            status.Error(codes.Unauthenticated, "unauthorized"),
			contentType:    "application/json",
			marshalErr:     nil,
			teHeader:       "",
			expectedStatus: 200, // default
			description:    "should set WWW-Authenticate header for unauthenticated error",
		},
		{
			name:           "marshal error",
			err:            status.Error(codes.Internal, "internal error"),
			contentType:    "application/json",
			marshalErr:     assert.AnError,
			teHeader:       "",
			expectedStatus: 500,
			description:    "should return 500 when marshal fails",
		},
		{
			name:           "with trailers",
			err:            status.Error(codes.OK, "success"),
			contentType:    "application/json",
			marshalErr:     nil,
			teHeader:       "trailers",
			headerMD:       metadata.New(map[string]string{"test-header": "value"}),
			trailerMD:      metadata.New(map[string]string{"test-trailer": "value"}),
			expectedStatus: 200, // default
			description:    "should handle trailers when TE header includes trailers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create context with server metadata
			ctx := context.Background()
			serverMD := runtime.ServerMetadata{}
			if tt.headerMD != nil {
				serverMD.HeaderMD = tt.headerMD
			}
			if tt.trailerMD != nil {
				serverMD.TrailerMD = tt.trailerMD
			}
			ctx = runtime.NewServerMetadataContext(ctx, serverMD)

			// Create request with TE header
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.teHeader != "" {
				req.Header.Set("TE", tt.teHeader)
			}

			// Create response writer
			w := httptest.NewRecorder()

			// Create mock marshaler
			marshaler := &mockMarshaler{
				contentType: tt.contentType,
				marshalErr:  tt.marshalErr,
			}

			// Create mock mux
			mux := &runtime.ServeMux{}

			// Call ErrorHandler
			ErrorHandler(ctx, mux, marshaler, w, req, tt.err)

			// Verify results
			if tt.err != nil {
				s := status.Convert(tt.err)
				if s.Code() == codes.Unauthenticated {
					assert.Equal(t, s.Message(), w.Header().Get("WWW-Authenticate"), tt.description)
				}

				if tt.contentType == "application/json" {
					assert.Equal(t, "application/problem+json", w.Header().Get("Content-Type"), tt.description)
				} else {
					assert.Equal(t, tt.contentType, w.Header().Get("Content-Type"), tt.description)
				}

				if tt.marshalErr != nil {
					assert.Equal(t, http.StatusInternalServerError, w.Code, tt.description)
				}

				if tt.teHeader != "" && strings.Contains(strings.ToLower(tt.teHeader), "trailers") {
					assert.Equal(t, "chunked", w.Header().Get("Transfer-Encoding"), tt.description)
				}
			}
		})
	}
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
		t.Run(tt.name, func(t *testing.T) {
			key, match := CustomHeaderMatcher(tt.key)
			assert.Equal(t, tt.expectedKey, key, tt.description)
			assert.Equal(t, tt.expectedMatch, match, tt.description)
		})
	}
}

func TestInjectOwnerToContext(t *testing.T) {
	tests := []struct {
		name        string
		owner       *mgmtpb.User
		expected    map[string]string
		description string
	}{
		{
			name: "valid owner",
			owner: &mgmtpb.User{
				Uid: stringPtr("test-user-123"),
			},
			expected: map[string]string{
				constant.HeaderAuthTypeKey: "user",
				constant.HeaderUserUIDKey:  "test-user-123",
			},
			description: "should inject owner metadata to context",
		},
		{
			name:  "nil owner",
			owner: nil,
			expected: map[string]string{
				constant.HeaderAuthTypeKey: "user",
				constant.HeaderUserUIDKey:  "",
			},
			description: "should handle nil owner gracefully",
		},
		{
			name: "empty uid",
			owner: &mgmtpb.User{
				Uid: stringPtr(""),
			},
			expected: map[string]string{
				constant.HeaderAuthTypeKey: "user",
				constant.HeaderUserUIDKey:  "",
			},
			description: "should handle empty uid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			resultCtx := InjectOwnerToContext(ctx, tt.owner)

			// Extract metadata from context
			md, ok := metadata.FromOutgoingContext(resultCtx)
			assert.True(t, ok, tt.description)

			// Verify expected metadata
			for key, expectedValue := range tt.expected {
				values := md.Get(key)
				if expectedValue == "" {
					// For empty expected values, check if the key exists but has empty value
					if len(values) > 0 {
						assert.Equal(t, "", values[0], tt.description)
					}
				} else {
					assert.Len(t, values, 1, tt.description)
					assert.Equal(t, expectedValue, values[0], tt.description)
				}
			}
		})
	}
}

// Test error handling edge cases
func TestErrorHandlerEdgeCases(t *testing.T) {
	// Test with nil error
	ctx := context.Background()
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	marshaler := &mockMarshaler{contentType: "application/json"}
	mux := &runtime.ServeMux{}

	ErrorHandler(ctx, mux, marshaler, w, req, nil)
	// Should not panic and should handle gracefully
	assert.NotNil(t, w)

	// Test with context without server metadata
	ctx = context.Background()
	req = httptest.NewRequest("GET", "/test", nil)
	w = httptest.NewRecorder()
	err := status.Error(codes.Internal, "test error")

	ErrorHandler(ctx, mux, marshaler, w, req, err)
	// Should handle gracefully even without server metadata
	assert.NotNil(t, w)
}

// Test header manipulation in HTTPResponseModifier
func TestHTTPResponseModifierHeaderManipulation(t *testing.T) {
	// Create context with server metadata containing x-http-code
	headerMD := metadata.New(map[string]string{
		"x-http-code": "404",
	})
	ctx := runtime.NewServerMetadataContext(context.Background(), runtime.ServerMetadata{
		HeaderMD: headerMD,
	})

	w := httptest.NewRecorder()

	// Call HTTPResponseModifier
	err := HTTPResponseModifier(ctx, w, &emptypb.Empty{})
	assert.NoError(t, err)

	// Verify that the header was deleted from the response
	assert.Empty(t, w.Header().Get("Grpc-Metadata-X-Http-Code"))
	assert.Equal(t, 404, w.Code)
}

// Test trailer handling in ErrorHandler
func TestErrorHandlerTrailerHandling(t *testing.T) {
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
	marshaler := &mockMarshaler{contentType: "application/json"}
	mux := &runtime.ServeMux{}
	err := status.Error(codes.OK, "success")

	ErrorHandler(ctx, mux, marshaler, w, req, err)

	// Verify trailer headers were set
	assert.Equal(t, "chunked", w.Header().Get("Transfer-Encoding"))
	assert.Contains(t, w.Header().Get("Trailer"), "Grpc-Trailer-Test-Trailer")
}

// Test content type handling in ErrorHandler
func TestErrorHandlerContentTypeHandling(t *testing.T) {
	tests := []struct {
		name         string
		contentType  string
		expectedType string
		description  string
	}{
		{
			name:         "json content type",
			contentType:  "application/json",
			expectedType: "application/problem+json",
			description:  "should convert application/json to application/problem+json",
		},
		{
			name:         "xml content type",
			contentType:  "application/xml",
			expectedType: "application/xml",
			description:  "should keep non-json content types unchanged",
		},
		{
			name:         "empty content type",
			contentType:  "",
			expectedType: "",
			description:  "should handle empty content type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			marshaler := &mockMarshaler{contentType: tt.contentType}
			mux := &runtime.ServeMux{}
			err := status.Error(codes.InvalidArgument, "test error")

			ErrorHandler(ctx, mux, marshaler, w, req, err)

			assert.Equal(t, tt.expectedType, w.Header().Get("Content-Type"), tt.description)
		})
	}
}

func stringPtr(s string) *string {
	return &s
}
