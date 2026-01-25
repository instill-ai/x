package resource

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/gofrs/uuid"
	"golang.org/x/text/unicode/norm"
	"google.golang.org/grpc/metadata"

	"github.com/instill-ai/x/constant"
)

// GetRequestSingleHeader gets a request header, assuming the value is a
// single-value string HTTP header.
func GetRequestSingleHeader(ctx context.Context, key string) string {
	metaHeader := metadata.ValueFromIncomingContext(ctx, strings.ToLower(key))
	if len(metaHeader) != 1 {
		return ""
	}
	return metaHeader[0]
}

// GetRequesterUIDAndUserUID extracts the requester and user UIDs from the
// request header. Handles both raw UUIDs and permalink format (e.g., "users/{uid}").
func GetRequesterUIDAndUserUID(ctx context.Context) (uuid.UUID, uuid.UUID) {
	requesterUID := extractUIDFromPermalink(GetRequestSingleHeader(ctx, constant.HeaderRequesterUIDKey))
	userUID := extractUIDFromPermalink(GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey))
	if strings.TrimSpace(requesterUID) == "" {
		requesterUID = userUID
	}
	return uuid.FromStringOrNil(requesterUID), uuid.FromStringOrNil(userUID)
}

// extractUIDFromPermalink extracts the UID from a permalink or returns the input as-is.
// Handles formats like "users/{uid}" (permalink), "organizations/{uid}", or just "{uid}".
// Note: This extracts from permalink format (users/{uid}), not AIP resource name (users/{user_id}).
func extractUIDFromPermalink(permalink string) string {
	return ExtractResourceID(permalink)
}

// ExtractResourceID extracts the last segment (resource ID) from an AIP-compliant resource name.
// Works with both permalinks (users/{uid}) and resource names (namespaces/{ns}/projects/{project}).
// Examples:
//   - "namespaces/ns/projects/proj-123" -> "proj-123"
//   - "users/user-uid-456" -> "user-uid-456"
//   - "proj-123" -> "proj-123" (returns as-is if no slash)
func ExtractResourceID(resourceName string) string {
	resourceName = strings.TrimSpace(resourceName)
	if resourceName == "" {
		return ""
	}
	if idx := strings.LastIndex(resourceName, "/"); idx != -1 {
		return resourceName[idx+1:]
	}
	return resourceName
}

// base62Chars is the character set for base62 encoding (alphanumeric, URL-safe).
const base62Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// alphabet for simple random ID generation.
var alphabet = "abcdefghijklmnopqrstuvwxyz"

// GenerateShortID generates an 8-character random lowercase alphabetic ID.
// Useful for human-friendly identifiers where collision risk is acceptable.
func GenerateShortID() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	id := make([]byte, 8)
	for i := range id {
		id[i] = alphabet[r.Intn(len(alphabet))]
	}
	return string(id)
}

// GeneratePrefixedID creates an immutable canonical resource ID following AIP standard.
// The format is: {prefix}-{base62(sha256(uid)[:10])}
// Example: "col-8f3A2k9E7c1"
//
// This produces ~80 bits of entropy from the UUID, encoded in base62 for URL safety.
// The prefix indicates the resource type (e.g., "col" for collection, "grp" for group).
func GeneratePrefixedID(prefix string, uid uuid.UUID) string {
	hash := sha256.Sum256([]byte(uid.String()))
	// Take first 10 bytes (80 bits) and convert to base62
	encoded := encodeBase62(hash[:10])
	return fmt.Sprintf("%s-%s", prefix, encoded)
}

// GenerateOwnerID creates an owner (user/organization) ID from a display name and UUID.
// The format is: {slug-from-name}-{base62(sha256(uid)[:10])}
// Example: "John Doe" + UUID -> "john-doe-8f3A2k9E7c1xYz"
// Example: "Acme Corp" + UUID -> "acme-corp-8f3A2k9E7c1xYz"
//
// This combines human-readable slug with UUID-based hash for uniqueness.
// The slug portion is truncated to 20 characters to keep total length reasonable.
// Uses GeneratePrefixedID internally for consistent hash generation (80 bits entropy).
// Falls back to fallbackPrefix if name is empty or produces no valid slug.
func GenerateOwnerID(displayName string, uid uuid.UUID, fallbackPrefix string) string {
	// Generate slug from display name (max 20 chars)
	slug := GenerateSlug(displayName, 20)

	// If slug is empty (e.g., name is all special chars), use fallback
	if slug == "" {
		slug = fallbackPrefix
	}

	// Use GeneratePrefixedID for consistent hash generation
	return GeneratePrefixedID(slug, uid)
}

// encodeBase62 encodes bytes to a base62 string.
func encodeBase62(data []byte) string {
	// Convert bytes to base62
	var result strings.Builder
	// Process in chunks to avoid overflow
	// Each byte pair gives us ~2 base62 chars
	for i := 0; i < len(data); i += 2 {
		var num uint16
		if i+1 < len(data) {
			num = uint16(data[i])<<8 | uint16(data[i+1])
		} else {
			num = uint16(data[i]) << 8
		}
		// Convert to base62 (2 characters per 16-bit number)
		for j := 0; j < 2; j++ {
			result.WriteByte(base62Chars[num%62])
			num /= 62
		}
	}
	return result.String()
}

// GenerateSlug converts a display name to a URL-safe slug.
// Example: "My Research Collection" -> "my-research-collection"
// Rules:
//   - Convert to lowercase
//   - Replace spaces and underscores with dashes
//   - Remove non-alphanumeric characters (except dashes)
//   - Collapse multiple dashes into one
//   - Trim leading/trailing dashes
//   - Optionally truncate to maxLen (0 means no limit)
func GenerateSlug(displayName string, maxLen int) string {
	// Normalize unicode characters (e.g., accented characters)
	slug := norm.NFKD.String(displayName)

	// Convert to lowercase
	slug = strings.ToLower(slug)

	// Replace spaces and underscores with dashes
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")

	// Remove non-alphanumeric characters (except dashes)
	var builder strings.Builder
	for _, r := range slug {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' {
			// Only keep ASCII letters and digits
			if r < 128 {
				builder.WriteRune(r)
			}
		}
	}
	slug = builder.String()

	// Collapse multiple dashes into one
	multiDashRegex := regexp.MustCompile(`-+`)
	slug = multiDashRegex.ReplaceAllString(slug, "-")

	// Trim leading/trailing dashes
	slug = strings.Trim(slug, "-")

	// Truncate if needed
	if maxLen > 0 && len(slug) > maxLen {
		slug = slug[:maxLen]
		// Try to cut at a dash boundary if possible
		if lastDash := strings.LastIndex(slug, "-"); lastDash > maxLen/2 {
			slug = slug[:lastDash]
		}
		slug = strings.Trim(slug, "-")
	}

	return slug
}
