package stripe

import (
	"testing"

	"github.com/pkg/errors"
	stripego "github.com/stripe/stripe-go/v85"
)

func TestIsResourceMissingError(t *testing.T) {
	testCases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "plain error",
			err:  errors.New("boom"),
			want: false,
		},
		{
			name: "stripe resource_missing error",
			err:  &stripego.Error{Code: stripego.ErrorCodeResourceMissing},
			want: true,
		},
		{
			name: "stripe error with other code",
			err:  &stripego.Error{Code: stripego.ErrorCodeCardDeclined},
			want: false,
		},
		{
			// Mirrors CancelSubscription, which wraps the Stripe error with errors.Wrapf.
			name: "wrapped stripe resource_missing error",
			err:  errors.Wrap(&stripego.Error{Code: stripego.ErrorCodeResourceMissing}, "failed to cancel stripe subscription"),
			want: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsResourceMissingError(tc.err); got != tc.want {
				t.Errorf("IsResourceMissingError() = %v, want %v", got, tc.want)
			}
		})
	}
}
