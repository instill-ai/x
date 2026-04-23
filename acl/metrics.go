package acl

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// instrumentationName is the OTel meter scope under which x/acl
// instruments are registered. Stable across versions because dashboards
// and alerts in the OTel collector are keyed on it.
const instrumentationName = "github.com/instill-ai/x/acl"

// Lazy-initialised so the counter binds to whichever MeterProvider has
// been installed by the consumer service's `otel.SetupMetrics` call.
// init()-time creation would bind permanently to the no-op provider
// because x/acl is imported before main() runs SetupMetrics.
var (
	truncCounterOnce sync.Once
	truncCounter     metric.Int64Counter
)

// recordListObjectsTruncated bumps the truncation counter for the given
// (object_type, relation) pair. Wrapped so the call site does not have
// to import OTel; the lazy init keeps the call cheap (single atomic
// load on the hot path after first use) and survives a no-op global
// MeterProvider when running under tests that never wire one up.
//
// Counter name: `acl.list_objects_truncated` (OTel snake-case style;
// the collector remaps to `acl_list_objects_truncated_total` for any
// Prometheus exporter wired downstream of the collector).
//
// Attributes:
//
//   - object_type — the FGA object type (e.g. "file", "collection").
//     Bounded cardinality: matches the FGA schema, currently <20 types.
//   - relation    — the relation queried (e.g. "viewer", "executor").
//     Bounded cardinality: matches the relations defined per type.
func recordListObjectsTruncated(ctx context.Context, objectType, relation string) {
	truncCounterOnce.Do(func() {
		c, err := otel.Meter(instrumentationName).Int64Counter(
			"acl.list_objects_truncated",
			metric.WithDescription("Number of StreamedListObjects responses flagged as likely-truncated by the x/acl truncation guard. A non-zero value means OpenFGA's listObjectsDeadline / listObjectsMaxResults capped a response that the caller asked to enumerate; cache writes were skipped for those responses to prevent stale data propagation."),
			metric.WithUnit("1"),
		)
		if err != nil {
			// Defensive: never break the call site over telemetry. The
			// truncation log line in client.go already carries the
			// operational signal; the counter is only there to make
			// alerts cheap.
			return
		}
		truncCounter = c
	})
	if truncCounter == nil {
		return
	}
	truncCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("object_type", objectType),
		attribute.String("relation", relation),
	))
}
