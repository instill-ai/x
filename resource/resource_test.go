package resource_test

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"google.golang.org/grpc/metadata"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/x/constant"
	"github.com/instill-ai/x/resource"
)

func TestGetRequesterUIDAndUserUID(t *testing.T) {
	requesterUIDStr := uuid.Must(uuid.NewV4()).String()
	userUIDStr := uuid.Must(uuid.NewV4()).String()
	m := make(map[string]string)
	m[constant.HeaderRequesterUIDKey] = requesterUIDStr
	m[constant.HeaderUserUIDKey] = userUIDStr
	ctx := metadata.NewIncomingContext(context.Background(), metadata.New(m))

	c := qt.New(t)
	checkRequesterUID, checkUserUID := resource.GetRequesterUIDAndUserUID(ctx)
	requesterUID := uuid.FromStringOrNil(requesterUIDStr)
	userUID := uuid.FromStringOrNil(userUIDStr)

	c.Check(checkRequesterUID, qt.Equals, requesterUID)
	c.Check(checkUserUID, qt.Equals, userUID)
}

func TestGenerateShortID(t *testing.T) {
	c := qt.New(t)

	id1 := resource.GenerateShortID()
	id2 := resource.GenerateShortID()

	// Should be 8 characters
	c.Check(len(id1), qt.Equals, 8)
	c.Check(len(id2), qt.Equals, 8)

	// Should be all lowercase letters
	for _, char := range id1 {
		c.Check(char >= 'a' && char <= 'z', qt.IsTrue, qt.Commentf("char %c should be lowercase letter", char))
	}

	// Two IDs should be different (with high probability)
	c.Check(id1, qt.Not(qt.Equals), id2)
}

func TestGeneratePrefixedID(t *testing.T) {
	c := qt.New(t)

	uid := uuid.Must(uuid.FromString("550e8400-e29b-41d4-a716-446655440000"))

	// Test with different prefixes
	colID := resource.GeneratePrefixedID("col", uid)
	grpID := resource.GeneratePrefixedID("grp", uid)
	prjID := resource.GeneratePrefixedID("prj", uid)

	// Should have correct prefix format
	c.Check(colID[:4], qt.Equals, "col-")
	c.Check(grpID[:4], qt.Equals, "grp-")
	c.Check(prjID[:4], qt.Equals, "prj-")

	// Same UID should produce same ID (deterministic)
	colID2 := resource.GeneratePrefixedID("col", uid)
	c.Check(colID, qt.Equals, colID2)

	// Different UIDs should produce different IDs
	uid2 := uuid.Must(uuid.FromString("550e8400-e29b-41d4-a716-446655440001"))
	colID3 := resource.GeneratePrefixedID("col", uid2)
	c.Check(colID, qt.Not(qt.Equals), colID3)

	// Should be URL-safe (only alphanumeric and dash)
	for _, char := range colID {
		isValid := (char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-'
		c.Check(isValid, qt.IsTrue, qt.Commentf("char %c should be URL-safe", char))
	}
}

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		name        string
		displayName string
		maxLen      int
		expected    string
	}{
		{
			name:        "simple conversion",
			displayName: "My Research Collection",
			maxLen:      0,
			expected:    "my-research-collection",
		},
		{
			name:        "with special characters",
			displayName: "Hello, World! @2024",
			maxLen:      0,
			expected:    "hello-world-2024",
		},
		{
			name:        "with underscores",
			displayName: "hello_world_test",
			maxLen:      0,
			expected:    "hello-world-test",
		},
		{
			name:        "multiple spaces and dashes",
			displayName: "hello   --  world",
			maxLen:      0,
			expected:    "hello-world",
		},
		{
			name:        "leading and trailing spaces",
			displayName: "  hello world  ",
			maxLen:      0,
			expected:    "hello-world",
		},
		{
			name:        "with maxLen truncation",
			displayName: "this is a very long collection name",
			maxLen:      20,
			expected:    "this-is-a-very-long",
		},
		{
			name:        "unicode characters",
			displayName: "Café résumé",
			maxLen:      0,
			expected:    "cafe-resume",
		},
		{
			name:        "numbers only",
			displayName: "12345",
			maxLen:      0,
			expected:    "12345",
		},
		{
			name:        "empty string",
			displayName: "",
			maxLen:      0,
			expected:    "",
		},
		{
			name:        "only special characters",
			displayName: "!@#$%^&*()",
			maxLen:      0,
			expected:    "",
		},
	}

	c := qt.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resource.GenerateSlug(tt.displayName, tt.maxLen)
			c.Check(result, qt.Equals, tt.expected)
		})
	}
}
