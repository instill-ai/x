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
	// Enabled indicates whether permission caching is enabled.
	Enabled bool
	// TTL is the cache time-to-live in seconds.
	TTL int
}

// DefaultCacheConfig returns the default cache configuration.
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		Enabled: true,
		TTL:     60, // 60 seconds default TTL
	}
}

// CacheTTLDuration returns the cache TTL as a time.Duration.
// Returns 60 seconds if TTL is not set or invalid.
func (c CacheConfig) CacheTTLDuration() time.Duration {
	if c.TTL <= 0 {
		return 60 * time.Second
	}
	return time.Duration(c.TTL) * time.Second
}
