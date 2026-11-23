package trino

import (
	"context"
	"testing"

	omnidriver "github.com/bytebase/omni/driver"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

func openTrino(t *testing.T, password string, useSSL bool) error {
	t.Helper()
	driver, err := (&Driver{}).Open(context.Background(), storepb.Engine_TRINO, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:     "trino.example.com",
			Port:     "8080",
			Username: "bytebase",
			UseSsl:   useSSL,
		},
		Password: password,
	})
	if driver != nil {
		// Releases the custom HTTP client the SSL path registers globally.
		t.Cleanup(func() { _ = driver.Close(context.Background()) })
	}
	return err
}

// trino-go-client refuses a password on a plaintext connection, so such a data
// source cannot connect whatever we do. The guard exists for the message: it
// names the setting to change instead of reporting a DSN parse failure.
func TestOpenRefusesAPasswordWithoutSSL(t *testing.T) {
	err := openTrino(t, "secret", false)
	require.ErrorContains(t, err, "requires SSL to authenticate with a password")
	require.NotContains(t, err.Error(), "secret", "the error must not carry the password")
}

func TestOpenAcceptsTheConfigurationsThatCanConnect(t *testing.T) {
	for _, tc := range []struct {
		name     string
		password string
		useSSL   bool
	}{
		{"password over SSL", "secret", true},
		{"no password without SSL", "", false},
		{"no password over SSL", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := openTrino(t, tc.password, tc.useSSL)
			require.NoError(t, err)
		})
	}
}

type captureTrinoSession struct {
	gotOpts omnidriver.QueryOptions
	results []*omnidriver.Result
}

func (s *captureTrinoSession) Query(_ context.Context, _ string, opts omnidriver.QueryOptions) ([]*omnidriver.Result, error) {
	s.gotOpts = opts
	return s.results, nil
}

func (*captureTrinoSession) Close() error { return nil }

func TestQueryConnFillsZeroAffectedRowsAndMapsExplainFormat(t *testing.T) {
	session := &captureTrinoSession{
		results: []*omnidriver.Result{
			{Statement: "CREATE TABLE memory.default.t (id BIGINT)"},
		},
	}
	d := &Driver{}
	d.SetSession(session)

	results, err := d.QueryConn(context.Background(), "CREATE TABLE memory.default.t (id BIGINT)", db.QueryContext{
		Schema:               "default",
		Limit:                100,
		Explain:              true,
		Option:               &v1pb.QueryOption{ExplainFormat: v1pb.QueryOption_JSON},
		MaximumSQLResultSize: 1024,
		MaskingEnabled:       true,
	})
	require.NoError(t, err)
	require.Equal(t, "default", session.gotOpts.Schema)
	require.Equal(t, 100, session.gotOpts.Limit)
	require.True(t, session.gotOpts.Explain)
	require.Equal(t, "JSON", session.gotOpts.ExplainFormat)
	require.Equal(t, int64(1024), session.gotOpts.MaxResultBytes)
	require.True(t, session.gotOpts.MaskingEnabled)
	require.Len(t, results, 1)
	require.Equal(t, []string{"Affected Rows"}, results[0].ColumnNames)
	require.Equal(t, int64(0), results[0].Rows[0].Values[0].GetInt64Value())
}
