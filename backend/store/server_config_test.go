package store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
)

func TestGetAuthSecretKeepsFirstSuccessfulRead(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	// NewMetadataDB disables the store cache, as HA does.
	db, s, _ := testcontainer.NewMetadataDB(t)

	_, err := db.ExecContext(ctx, `DELETE FROM server_config`)
	require.NoError(t, err)
	_, err = s.GetAuthSecret(ctx)
	require.Error(t, err)

	_, err = db.ExecContext(ctx, `INSERT INTO server_config (payload) VALUES ('{"authSecret": "first"}')`)
	require.NoError(t, err)
	secret, err := s.GetAuthSecret(ctx)
	require.NoError(t, err)
	require.Equal(t, "first", secret)

	_, err = db.ExecContext(ctx, `UPDATE server_config SET payload = '{"authSecret": "second"}'`)
	require.NoError(t, err)
	secret, err = s.GetAuthSecret(ctx)
	require.NoError(t, err)
	require.Equal(t, "first", secret)
}
