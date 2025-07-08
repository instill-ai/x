package paginate

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

func TestEncodeTokenAndDecodeToken_Success(t *testing.T) {
	createdAt := time.Now().UTC().Truncate(time.Nanosecond)
	uuid := "123e4567-e89b-12d3-a456-426614174000"
	token := EncodeToken(createdAt, uuid)

	parsedTime, parsedUUID, err := DecodeToken(token)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !parsedTime.Equal(createdAt) {
		t.Errorf("expected time %v, got %v", createdAt, parsedTime)
	}
	if parsedUUID != uuid {
		t.Errorf("expected uuid %q, got %q", uuid, parsedUUID)
	}
}

func TestDecodeToken_InvalidBase64(t *testing.T) {
	_, _, err := DecodeToken("not_base64!!")
	if err == nil {
		t.Error("expected error for invalid base64, got nil")
	}
}

func TestDecodeToken_InvalidFormat(t *testing.T) {
	// base64 of just one part
	bad := base64.StdEncoding.EncodeToString([]byte("onlyonepart"))
	_, _, err := DecodeToken(bad)
	if err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Errorf("expected invalid token error, got %v", err)
	}
}

func TestDecodeToken_InvalidTime(t *testing.T) {
	// base64 of bad time,uuid
	bad := base64.StdEncoding.EncodeToString([]byte("notatime,123e4567-e89b-12d3-a456-426614174000"))
	_, _, err := DecodeToken(bad)
	if err == nil {
		t.Error("expected error for invalid time, got nil")
	}
}

func TestEncodeToken_ProducesValidBase64(t *testing.T) {
	createdAt := time.Now().UTC()
	uuid := "abc"
	token := EncodeToken(createdAt, uuid)
	_, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		t.Errorf("EncodeToken did not produce valid base64: %v", err)
	}
}
