package acl

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// listObjectsTruncatedTotal counts every StreamedListObjects response
// that the truncation guard flagged as likely-truncated. The counter
// is exposed under the canonical name `acl_list_objects_truncated_total`
// so dashboards and alerts can be wired against it without the caller
// having to import the symbol.
//
// Labels:
//
//   - object_type — the FGA object type (e.g. "file", "collection").
//     Bounded cardinality: matches the FGA schema, currently <20 types.
//   - role        — the relation queried (e.g. "viewer", "executor").
//     Bounded cardinality: matches the relations defined per type.
//
// The counter is intentionally registered against the default
// Prometheus registry. Services that scrape /metrics from the default
// registry pick it up with no extra wiring; services that use a custom
// registry must wrap promhttp.HandlerFor with the default registry as
// well, or expose this counter by calling ListObjectsTruncatedTotal()
// during their own collector wiring.
var listObjectsTruncatedTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "acl_list_objects_truncated_total",
		Help: "Number of StreamedListObjects responses flagged as likely-truncated by the x/acl truncation guard. A non-zero value means OpenFGA's listObjectsDeadline / listObjectsMaxResults capped a response that the caller asked to enumerate; cache writes were skipped for those responses to prevent stale data propagation.",
	},
	[]string{"object_type", "role"},
)

// recordListObjectsTruncated bumps the truncation counter for the
// given (object_type, role) pair. Wrapped in a helper so callers do
// not need to import prometheus directly.
func recordListObjectsTruncated(objectType, role string) {
	listObjectsTruncatedTotal.WithLabelValues(objectType, role).Inc()
}

// ListObjectsTruncatedTotal returns the underlying CounterVec so that
// services using a custom Prometheus registry can register it
// themselves. Tests use this to read the counter value via the
// testutil package.
func ListObjectsTruncatedTotal() *prometheus.CounterVec {
	return listObjectsTruncatedTotal
}
