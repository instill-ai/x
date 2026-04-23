package acl

import "time"

// Config holds the configuration for the ACL client.
type Config struct {
	// Host is the OpenFGA server hostname.
	Host string
	// Port is the OpenFGA server port.
	Port int
	// Replica holds replica configuration for read/write separation.
	Replica ReplicaConfig
	// Cache holds permission caching configuration.
	Cache CacheConfig
	// ListObjects holds the truncation-guard configuration for the
	// StreamedListObjects code path. Zero values are filled in from
	// DefaultListObjectsConfig at client construction time.
	ListObjects ListObjectsConfig
}

// ReplicaConfig holds configuration for read replica.
type ReplicaConfig struct {
	// Host is the replica server hostname.
	Host string
	// Port is the replica server port.
	Port int
	// ReplicationTimeFrame is the time in seconds to direct reads to primary after a write.
	ReplicationTimeFrame int
}

// CacheConfig holds configuration for permission caching.
type CacheConfig struct {
	// Enabled indicates whether CheckPermission Redis caching is enabled.
	// When OpenFGA's own in-memory cache is already active (OPENFGA_CHECK_QUERY_CACHE_ENABLED),
	// enabling this adds a Redis hop for negligible benefit — keep it disabled in that case.
	Enabled bool
	// ListPermissionsEnabled indicates whether ListPermissions / ListPublicPermissions
	// Redis caching is enabled. StreamedListObjects is expensive (seconds per call),
	// so caching the result in Redis provides a significant latency reduction even
	// when OpenFGA's own cache is active (which only partially covers ListObjects).
	ListPermissionsEnabled bool
	// TTL is the cache time-to-live in seconds (shared by both cache layers).
	TTL int
}

// DefaultCacheConfig returns the default cache configuration.
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		Enabled:                false, // CheckPermission cache off by default (OpenFGA server cache suffices)
		ListPermissionsEnabled: true,  // ListPermissions cache on by default (StreamedListObjects is slow)
		TTL:                    60,
	}
}

// ListObjectsConfig holds the truncation-guard configuration for the
// StreamedListObjects code path. Both fields must mirror the matching
// OpenFGA server flags (OPENFGA_LIST_OBJECTS_DEADLINE,
// OPENFGA_LIST_OBJECTS_MAX_RESULTS); otherwise the heuristic that
// flags a streamed call as truncated will fire on the wrong threshold
// and either miss real truncations or false-positive on slow-but-
// complete responses.
type ListObjectsConfig struct {
	// Deadline mirrors OpenFGA's listObjectsDeadline. When a stream
	// completes within `Slack` of this value, the result is treated as
	// potentially truncated: the cache write is skipped and the
	// truncation counter is incremented. Defaults to 3 seconds (the
	// OpenFGA default) when unset.
	Deadline time.Duration
	// MaxResults mirrors OpenFGA's listObjectsMaxResults. When the
	// stream returns at least this many results, the response is also
	// treated as potentially truncated regardless of elapsed time.
	// Defaults to 1000 (the OpenFGA default) when unset.
	MaxResults int
	// Slack widens the deadline-side truncation heuristic to absorb
	// network jitter. A stream that completes between
	// (Deadline - Slack) and Deadline is still flagged as truncated.
	// Defaults to 200ms when unset.
	Slack time.Duration
}

// DefaultListObjectsConfig returns the truncation-guard defaults that
// match the OpenFGA server defaults shipped in instill-core.
func DefaultListObjectsConfig() ListObjectsConfig {
	return ListObjectsConfig{
		Deadline:   3 * time.Second,
		MaxResults: 1000,
		Slack:      200 * time.Millisecond,
	}
}

// resolved returns a ListObjectsConfig with zero-valued fields filled
// from DefaultListObjectsConfig so that the caller never has to
// remember which fields are required.
func (l ListObjectsConfig) resolved() ListObjectsConfig {
	d := DefaultListObjectsConfig()
	if l.Deadline <= 0 {
		l.Deadline = d.Deadline
	}
	if l.MaxResults <= 0 {
		l.MaxResults = d.MaxResults
	}
	if l.Slack < 0 {
		l.Slack = d.Slack
	}
	return l
}

// CacheTTLDuration returns the cache TTL as a time.Duration.
// Returns 60 seconds if TTL is not set or invalid.
func (c CacheConfig) CacheTTLDuration() time.Duration {
	if c.TTL <= 0 {
		return 60 * time.Second
	}
	return time.Duration(c.TTL) * time.Second
}
