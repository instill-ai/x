package errmsg

import (
	"errors"
	"fmt"
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestAddAndExtractMessage(t *testing.T) {
	c := qt.New(t)

	testcases := []struct {
		name    string
		wantMsg string
		wantErr string
		err     error
	}{
		{
			name:    "no message",
			wantMsg: "boom",
			wantErr: "boom",
			err:     errors.New("boom"),
		},
		{
			name:    "message on top of stack",
			wantMsg: "Something went wrong.",
			wantErr: "boom",
			err:     AddMessage(errors.New("boom"), "Something went wrong."),
		},
		{
			name:    "message in wrapped error (fmt)",
			wantMsg: "Something went wrong.",
			wantErr: "bang: boom",
			err: fmt.Errorf(
				"bang: %w",
				AddMessage(errors.New("boom"), "Something went wrong."),
			),
		},
		{
			name:    "message in joint error",
			wantMsg: "Something went wrong.",
			wantErr: "bang\nboom",
			err: errors.Join(
				errors.New("bang"),
				AddMessage(errors.New("boom"), "Something went wrong."),
			),
		},
		{
			name:    "multi-message error",
			wantMsg: "Please check your input. Something went wrong in condition 1. Condition 2 failed.",
			wantErr: "checking conditions: evaluating condition 1: boom\nbang",
			err: func() error {
				cond2Err := AddMessage(fmt.Errorf("bang"), "Condition 2 failed.")
				cond1Err := fmt.Errorf(
					"evaluating condition 1: %w",
					AddMessage(fmt.Errorf("boom"), "Something went wrong in condition 1."),
				)

				jointErr := fmt.Errorf("checking conditions: %w", errors.Join(cond1Err, cond2Err))
				return AddMessage(jointErr, "Please check your input.")
			}(),
		},
	}

	for _, tc := range testcases {
		c.Run(tc.name, func(c *qt.C) {
			c.Check(MessageOrErr(tc.err), qt.Equals, tc.wantMsg)
			c.Check(tc.err, qt.ErrorMatches, tc.wantErr)
		})
	}
}
