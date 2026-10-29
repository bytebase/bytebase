package store

import (
	"context"
	"time"

	"github.com/pkg/errors"
)

// NewWithOutOfBandCacheLifetimeForTest is New with another lifetime for the
// caches that expire after OutOfBandCacheLifetime. expirable.LRU never expires
// a lifetime of zero and panics on one under 100ns, so it must be at least a
// millisecond.
func NewWithOutOfBandCacheLifetimeForTest(ctx context.Context, pgURL string, enableCache bool, lifetime time.Duration) (*Store, error) {
	if lifetime < time.Millisecond {
		return nil, errors.Errorf("lifetime %v is shorter than a millisecond", lifetime)
	}
	return newStore(ctx, pgURL, enableCache, lifetime)
}

// SetSettingPublishHookForTest installs a hook that runs between a setting
// update's commit and its cache publish — test-only, to deterministically
// expose the publish-ordering window.
func (s *Store) SetSettingPublishHookForTest(hook func()) {
	s.settingPublishHookForTest = hook
}

// SetSettingMissHookForTest installs a hook that runs after a setting read
// misses the cache and before it waits for the publish mutex.
func (s *Store) SetSettingMissHookForTest(hook func()) {
	s.settingMissHookForTest = hook
}

// SetSettingFillHookForTest installs a hook that runs right before a setting
// cache-miss fill reads the database.
func (s *Store) SetSettingFillHookForTest(hook func()) {
	s.settingFillHookForTest = hook
}
