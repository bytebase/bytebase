package common

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v4"
)

const (
	// Just choose an empirical timeout value. The default 15 minutes seems too long.
	timeout = 5 * time.Minute
)

// Retry uses exponential backoff with timeout.
func Retry(ctx context.Context, fn func() error) error {
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = timeout
	bWithContext := backoff.WithContext(b, ctx)
	return backoff.Retry(fn, bWithContext)
}
