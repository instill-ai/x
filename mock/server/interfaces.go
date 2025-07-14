package server

import (
	"context"
	"io"

	"go.opentelemetry.io/otel/log"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Logger interface for logging operations
type Logger interface {
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Debug(msg string, fields ...zap.Field)
}

// OTELLogger interface for OpenTelemetry logging
type OTELLogger interface {
	Emit(ctx context.Context, record log.Record)
}

// ServerStream interface for gRPC server stream operations
type ServerStream interface {
	grpc.ServerStream
	Context() context.Context
	RecvMsg(msg any) error
	SendMsg(msg any) error
	SendHeader(md metadata.MD) error
	SetHeader(md metadata.MD) error
	SetTrailer(md metadata.MD)
}

// Marshaler interface for marshaling/unmarshaling operations
type Marshaler interface {
	ContentType(any) string
	Marshal(any) ([]byte, error)
	Unmarshal([]byte, any) error
	NewDecoder(r io.Reader) Decoder
	NewEncoder(w io.Writer) Encoder
}

// Decoder interface for decoding operations
type Decoder interface {
	Decode(v any) error
}

// Encoder interface for encoding operations
type Encoder interface {
	Encode(v any) error
}

// ProtoMessage interface for protobuf message operations
type ProtoMessage interface {
	Reset()
	String() string
	ProtoMessage()
	ProtoReflect() protoreflect.Message
}
