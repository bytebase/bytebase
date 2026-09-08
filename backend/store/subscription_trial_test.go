package store_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func newTrialLicenseFixture(t *testing.T) (context.Context, *store.Store) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(cancel)
	_, stores, _ := testcontainer.NewMetadataDBWithCache(t, true)

	_, err := stores.GetDB().ExecContext(ctx, `INSERT INTO workspace (resource_id) VALUES ('default')`)
	require.NoError(t, err)
	_, err = stores.UpsertSetting(ctx, &store.SettingMessage{
		Name:      storepb.SettingName_SYSTEM,
		Workspace: "default",
		Value:     &storepb.SystemSetting{},
	})
	require.NoError(t, err)
	return ctx, stores
}

func TestCreateTrialLicenseRejectsSubscriptionHistory(t *testing.T) {
	t.Parallel()
	ctx, stores := newTrialLicenseFixture(t)

	_, err := stores.UpsertSubscription(ctx, "default", &storepb.SubscriptionPayload{
		Status: storepb.SubscriptionPayload_CANCELED,
	})
	require.NoError(t, err)

	err = stores.CreateTrialLicense(ctx, "default", "trial-license")
	require.ErrorIs(t, err, store.ErrTrialNotEligible)

	setting, err := stores.GetSystemSettingUncached(ctx, "default")
	require.NoError(t, err)
	require.Empty(t, setting.License)
}

func TestCreateTrialLicenseRejectsExistingLicense(t *testing.T) {
	t.Parallel()
	ctx, stores := newTrialLicenseFixture(t)
	require.NoError(t, stores.UpdateLicense(ctx, "default", "existing-license"))

	err := stores.CreateTrialLicense(ctx, "default", "trial-license")
	require.ErrorIs(t, err, store.ErrTrialNotEligible)

	setting, err := stores.GetSystemSettingUncached(ctx, "default")
	require.NoError(t, err)
	require.Equal(t, "existing-license", setting.License)
}

func TestCreateTrialLicenseConcurrentCallsCreateOnce(t *testing.T) {
	t.Parallel()
	ctx, stores := newTrialLicenseFixture(t)

	type result struct {
		license string
		err     error
	}
	start := make(chan struct{})
	results := make(chan result, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	for _, license := range []string{"trial-license-a", "trial-license-b"} {
		go func(license string) {
			ready.Done()
			<-start
			err := stores.CreateTrialLicense(ctx, "default", license)
			results <- result{license: license, err: err}
		}(license)
	}
	ready.Wait()
	close(start)

	var winner string
	for range 2 {
		select {
		case result := <-results:
			if result.err == nil {
				require.Empty(t, winner, "only one trial may be created")
				winner = result.license
			} else {
				require.ErrorIs(t, result.err, store.ErrTrialNotEligible)
			}
		case <-time.After(15 * time.Second):
			t.Fatal("trial creation did not complete; possible deadlock")
		}
	}
	require.NotEmpty(t, winner)

	setting, err := stores.GetSystemSettingUncached(ctx, "default")
	require.NoError(t, err)
	require.Equal(t, winner, setting.License)
	cached, err := stores.GetSystemSetting(ctx, "default")
	require.NoError(t, err)
	require.Equal(t, winner, cached.License)
}
