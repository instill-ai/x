package blobstorage

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
)

func TestUploadFile_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !bytes.Equal(body, []byte("testdata")) {
			t.Errorf("unexpected body: %s", string(body))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := zaptest.NewLogger(t)
	err := UploadFile(context.Background(), logger, server.URL, []byte("testdata"), "text/plain")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestUploadFile_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("fail"))
	}))
	defer server.Close()

	logger := zaptest.NewLogger(t)
	err := UploadFile(context.Background(), logger, server.URL, []byte("testdata"), "text/plain")
	if err == nil {
		t.Errorf("expected error, got nil")
	}
}

func TestUploadFile_RequestCreationError(t *testing.T) {
	logger := zaptest.NewLogger(t)
	// Invalid URL to force error
	err := UploadFile(context.Background(), logger, "http://[::1]:namedport", []byte("testdata"), "text/plain")
	if err == nil {
		t.Errorf("expected error, got nil")
	}
}

func TestUploadFile_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	logger := zaptest.NewLogger(t)
	err := UploadFile(ctx, logger, "http://example.com", []byte("testdata"), "text/plain")
	if err == nil {
		t.Errorf("expected error due to context cancellation, got nil")
	}
}

func TestUploadFile_LogsBodyAsString(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("this is a text error"))
	}))
	defer server.Close()

	// Use a zap logger with a buffer to capture logs
	var buf bytes.Buffer
	logger := zap.New(zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
		zapcore.AddSync(&buf),
		zap.DebugLevel,
	))

	err := UploadFile(context.Background(), logger, server.URL, []byte("testdata"), "text/plain")
	if err == nil {
		t.Errorf("expected error, got nil")
	}

	logged := buf.String()
	if !strings.Contains(logged, "this is a text error") {
		t.Errorf("expected log to contain response body as string, got: %s", logged)
	}
}

func TestUploadFile_LogsBinaryBodyAsString(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte{0x00, 0x01, 0x02, 0x03})
	}))
	defer server.Close()

	var buf bytes.Buffer
	logger := zap.New(zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
		zapcore.AddSync(&buf),
		zap.DebugLevel,
	))

	err := UploadFile(context.Background(), logger, server.URL, []byte("testdata"), "text/plain")
	if err == nil {
		t.Errorf("expected error, got nil")
	}

	logged := buf.String()
	// Check for the presence of non-printable characters or their string representation
	if !strings.Contains(logged, `"\u0000\u0001\u0002\u0003"`) {
		t.Errorf("expected log to contain binary data as unicode-escaped string, got: %s", logged)
	}
}
