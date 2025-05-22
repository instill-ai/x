package temporal

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
	"google.golang.org/grpc/metadata"

	qt "github.com/frankban/quicktest"
	commonpb "go.temporal.io/api/common/v1"
)

var propagatedValues = []keyValuePair{
	{Key: "k1", Value: "v1"},
	{Key: "k2", Value: "v2"},
	{Key: "k3", Value: "v3"},
}

func Test_CtxPropWorkflow(t *testing.T) {
	c := qt.New(t)

	ts := new(testsuite.WorkflowTestSuite)
	payload, _ := converter.GetDefaultDataConverter().ToPayload(propagatedValues)
	ts.SetHeader(&commonpb.Header{
		Fields: map[string]*commonpb.Payload{
			headerKey: payload,
		},
	})

	env := ts.NewTestWorkflowEnvironment()
	env.SetContextPropagators([]workflow.ContextPropagator{NewContextPropagator()})
	env.RegisterActivity(SampleActivity)

	got := make([]string, 0, len(propagatedValues))
	env.SetOnActivityStartedListener(func(activityInfo *activity.Info, ctx context.Context, args converter.EncodedValues) {
		// The key should be propagated by custom context propagator.
		for _, kv := range propagatedValues {
			got = append(got, ValueFromPropagatedContext(ctx, kv.Key))
		}
	})

	env.ExecuteWorkflow(CtxPropWorkflow)
	c.Check(env.IsWorkflowCompleted(), qt.IsTrue)
	c.Check(env.GetWorkflowError(), qt.IsNil)

	c.Check(got, qt.HasLen, len(propagatedValues))

	for i, want := range propagatedValues {
		c.Check(got[i], qt.Equals, want.Value)
	}
}

func Test_PropagateValue(t *testing.T) {
	c := qt.New(t)

	ctx := context.Background()
	for _, kv := range propagatedValues {
		ctx = PropagateValue(ctx, kv.Key, kv.Value)
	}

	for i, want := range propagatedValues {
		// Test case-insensitivity.
		k := want.Key
		if i == 1 {
			k = strings.ToUpper(k)
		}

		got := ValueFromPropagatedContext(ctx, k)
		c.Check(got, qt.Equals, want.Value)
	}
}

func Test_PropagateMetadata(t *testing.T) {
	c := qt.New(t)

	metadataMap := map[string]string{
		"foo": "bar",
		"pim": "pam",
		"tic": "tac",
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.New(metadataMap))
	ctx = PropagateMetadata(ctx)

	for k, want := range metadataMap {
		got := ValueFromPropagatedContext(ctx, k)
		c.Check(got, qt.Equals, want)
	}
}
func SampleActivity(_ context.Context) error {
	return nil
}

func CtxPropWorkflow(ctx workflow.Context) error {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 100 * time.Millisecond,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	for _, want := range propagatedValues {
		got := ValueFromPropagatedContext(ctx, want.Key)
		if got != want.Value {
			return fmt.Errorf("unexpected propagated value for %s: %s", want.Key, got)
		}
	}

	return workflow.ExecuteActivity(ctx, SampleActivity).Get(ctx, nil)
}
