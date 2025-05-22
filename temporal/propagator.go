package temporal

import (
	"context"
	"strings"

	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/workflow"
	"google.golang.org/grpc/metadata"
)

type (
	// contextKey is an unexported type used as key for items stored in the
	// Context object.
	contextKey struct{}

	// propagator implements a custom context propagator for Temporal.
	propagator struct{}

	keyValuePair struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
)

const headerKey = "instill-metadata"

// NewContextPropagator returns a context propagator that propagates a set of
// string key-value pairs across a workflow.
func NewContextPropagator() workflow.ContextPropagator {
	return &propagator{}
}

// Inject injects values from context into headers for propagation.
func (s *propagator) Inject(ctx context.Context, writer workflow.HeaderWriter) error {
	value := ctx.Value(contextKey{})
	payload, err := converter.GetDefaultDataConverter().ToPayload(value)
	if err != nil {
		return err
	}
	writer.Set(headerKey, payload)
	return nil
}

// InjectFromWorkflow injects values from context into headers for propagation.
func (s *propagator) InjectFromWorkflow(ctx workflow.Context, writer workflow.HeaderWriter) error {
	value := ctx.Value(contextKey{})
	payload, err := converter.GetDefaultDataConverter().ToPayload(value)
	if err != nil {
		return err
	}
	writer.Set(headerKey, payload)
	return nil
}

// Extract extracts values from headers and puts them into context.
func (s *propagator) Extract(ctx context.Context, reader workflow.HeaderReader) (context.Context, error) {
	if raw, ok := reader.Get(headerKey); ok {
		var values []keyValuePair
		if err := converter.GetDefaultDataConverter().FromPayload(raw, &values); err != nil {
			return ctx, nil
		}
		ctx = context.WithValue(ctx, contextKey{}, values)
	}

	return ctx, nil
}

// ExtractToWorkflow extracts values from headers and puts them into context.
func (s *propagator) ExtractToWorkflow(ctx workflow.Context, reader workflow.HeaderReader) (workflow.Context, error) {
	if raw, ok := reader.Get(headerKey); ok {
		var values []keyValuePair
		if err := converter.GetDefaultDataConverter().FromPayload(raw, &values); err != nil {
			return ctx, nil
		}
		ctx = workflow.WithValue(ctx, contextKey{}, values)
	}

	return ctx, nil
}

type valueGetter interface {
	Value(key any) any
}

// ValueFromPropagatedContext returns the value in the propagated
// context.Context corresponding to the provided key. Keys are matched in a
// case insensitive manner.
func ValueFromPropagatedContext(ctx valueGetter, key string) string {
	propagatedValues, ok := ctx.Value(contextKey{}).([]keyValuePair)
	if !ok {
		return ""
	}

	for _, kv := range propagatedValues {
		if strings.EqualFold(key, kv.Key) {
			return kv.Value
		}
	}

	return ""
}

// PropagateValue adds a key-value pair to the propagation key in the context.
func PropagateValue(ctx context.Context, k string, v string) context.Context {
	propagatedValues, _ := ctx.Value(contextKey{}).([]keyValuePair)

	newValues := make([]keyValuePair, len(propagatedValues)+1)
	copy(newValues, propagatedValues)
	newValues[len(newValues)-1] = keyValuePair{Key: strings.ToLower(k), Value: v}

	return context.WithValue(ctx, contextKey{}, newValues)
}

// PropagateMetadata adds the key-value pairs present in the context metadata
// (coming from gRPC or HTTP requests via gRPC Gateway) to the propagation key
// in the context. If a key has more than one value, only the first one will be
// copied to replicate the behaviour in the x/resource package.
func PropagateMetadata(ctx context.Context) context.Context {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx
	}

	propagatedValues, _ := ctx.Value(contextKey{}).([]keyValuePair)
	newValues := make([]keyValuePair, len(propagatedValues)+len(md))
	copy(newValues, propagatedValues)

	i := len(propagatedValues)
	for k, v := range md {
		newValues[i] = keyValuePair{Key: k, Value: v[0]}
		i++
	}

	return context.WithValue(ctx, contextKey{}, newValues)
}
