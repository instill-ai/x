package interceptor

import (
	"context"
	"io"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/mock"
	"go.opentelemetry.io/otel/log"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/emptypb"
)

// ============================================================================
// Mock Proto Message
// ============================================================================

// MockProtoMessage is a mock proto message for testing
type MockProtoMessage struct{}

func (m *MockProtoMessage) Reset()                             {}
func (m *MockProtoMessage) String() string                     { return "mock" }
func (m *MockProtoMessage) ProtoMessage()                      {}
func (m *MockProtoMessage) ProtoReflect() protoreflect.Message { return nil }

// ============================================================================
// Mock Logger
// ============================================================================

// MockLogger is a mock logger for testing
type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Info(msg string, fields ...zap.Field) {
	m.Called(msg, fields)
}

func (m *MockLogger) Warn(msg string, fields ...zap.Field) {
	m.Called(msg, fields)
}

func (m *MockLogger) Error(msg string, fields ...zap.Field) {
	m.Called(msg, fields)
}

func (m *MockLogger) Debug(msg string, fields ...zap.Field) {
	m.Called(msg, fields)
}

// ============================================================================
// Mock OTEL Logger
// ============================================================================

// MockOTELLogger is a mock OTEL logger for testing
type MockOTELLogger struct {
	mock.Mock
}

func (m *MockOTELLogger) Emit(ctx context.Context, record log.Record) {
	m.Called(ctx, record)
}

// ============================================================================
// Mock Server Stream
// ============================================================================

// MockServerStream is a mock server stream for testing
type MockServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (m *MockServerStream) Context() context.Context {
	return m.ctx
}

func (m *MockServerStream) RecvMsg(msg any) error {
	return nil
}

func (m *MockServerStream) SendMsg(msg any) error {
	return nil
}

func (m *MockServerStream) SendHeader(md metadata.MD) error {
	return nil
}

func (m *MockServerStream) SetHeader(md metadata.MD) error {
	return nil
}

func (m *MockServerStream) SetTrailer(md metadata.MD) {
}

// ============================================================================
// Mock Marshaler
// ============================================================================

// MockMarshaler is a mock marshaler for testing
type MockMarshaler struct {
	contentType string
	marshalErr  error
}

func (m *MockMarshaler) ContentType(any) string {
	return m.contentType
}

func (m *MockMarshaler) Marshal(any) ([]byte, error) {
	if m.marshalErr != nil {
		return nil, m.marshalErr
	}
	return []byte(`{"test": "data"}`), nil
}

func (m *MockMarshaler) Unmarshal([]byte, any) error {
	return nil
}

func (m *MockMarshaler) NewDecoder(r io.Reader) runtime.Decoder {
	return &MockDecoder{}
}

func (m *MockMarshaler) NewEncoder(w io.Writer) runtime.Encoder {
	return &MockEncoder{}
}

// ============================================================================
// Mock Decoder
// ============================================================================

// MockDecoder is a mock decoder for testing
type MockDecoder struct{}

func (d *MockDecoder) Decode(v any) error {
	return nil
}

// ============================================================================
// Mock Encoder
// ============================================================================

// MockEncoder is a mock encoder for testing
type MockEncoder struct{}

func (e *MockEncoder) Encode(v any) error {
	return nil
}

// ============================================================================
// Helper Functions
// ============================================================================

// StringPtr returns a pointer to the given string
func StringPtr(s string) *string {
	return &s
}

// ============================================================================
// Common Test Data
// ============================================================================

// Common test data that can be reused across tests
var (
	// TestProtoMessage is a common proto message for testing
	TestProtoMessage = &emptypb.Empty{}

	// TestContext is a common context for testing
	TestContext = context.Background()

	// TestFullMethod is a common full method name for testing
	TestFullMethod = "test.Service/Method"

	// TestServiceName is a common service name for testing
	TestServiceName = "test-service"

	// TestServiceVersion is a common service version for testing
	TestServiceVersion = "v1.0.0"
)

// ============================================================================
// Test Utilities
// ============================================================================

// CreateTestContext creates a context with optional shouldLog flag
func CreateTestContext(shouldLog bool) context.Context {
	ctx := context.Background()
	if !shouldLog {
		ctx = context.WithValue(ctx, shouldLogKey, false)
	}
	return ctx
}

// CreateTestServerStream creates a mock server stream with the given context
func CreateTestServerStream(ctx context.Context) *MockServerStream {
	return &MockServerStream{ctx: ctx}
}

// CreateTestUnaryServerInfo creates a mock unary server info
func CreateTestUnaryServerInfo(fullMethod string) *grpc.UnaryServerInfo {
	return &grpc.UnaryServerInfo{
		FullMethod: fullMethod,
	}
}

// CreateTestStreamServerInfo creates a mock stream server info
func CreateTestStreamServerInfo(fullMethod string) *grpc.StreamServerInfo {
	return &grpc.StreamServerInfo{
		FullMethod: fullMethod,
	}
}

// CreateTestMarshaler creates a mock marshaler with the given configuration
func CreateTestMarshaler(contentType string, marshalErr error) *MockMarshaler {
	return &MockMarshaler{
		contentType: contentType,
		marshalErr:  marshalErr,
	}
}
