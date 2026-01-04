package utils // nolint:revive

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/store"
)

const setupExternalURLError = "external URL isn't setup yet, see https://docs.bytebase.com/get-started/self-host/external-url"

// GetEffectiveExternalURL returns the external URL to use, preferring the command-line flag over the database setting.
// This ensures that when the --external-url flag is set, it takes precedence over any value stored in the database.
func GetEffectiveExternalURL(ctx context.Context, stores *store.Store, profile *config.Profile) (string, error) {
	// Use command-line flag value if set, otherwise use database value
	externalURL := profile.ExternalURL
	if externalURL == "" {
		setting, err := stores.GetWorkspaceProfileSetting(ctx)
		if err != nil {
			return "", connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get workspace setting"))
		}
		externalURL = setting.ExternalUrl
	}

	if externalURL == "" {
		return "", connect.NewError(connect.CodeFailedPrecondition, errors.Errorf(setupExternalURLError))
	}
	return externalURL, nil
}
