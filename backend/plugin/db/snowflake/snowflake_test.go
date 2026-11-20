package snowflake

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"testing"
	"time"

	odriver "github.com/bytebase/omni/driver"
	snowflakedriver "github.com/bytebase/omni/snowflake/driver"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func TestSnowflakeConfig(t *testing.T) {
	for _, tc := range []struct {
		name   string
		config db.ConnectionConfig
		want   odriver.Config
	}{
		{
			name: "account host and password login",
			config: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:     "nb47110.ap-southeast-1",
					Port:     "443",
					Username: "bytebase",
				},
				Password: "pwd",
			},
			want: odriver.Config{
				Host:     "nb47110.ap-southeast-1",
				Port:     "443",
				User:     "bytebase",
				Password: "pwd",
			},
		},
		{
			name: "account@proxy host with database",
			config: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:     "nb47110.ap-southeast-1@10.0.0.1",
					Port:     "4182",
					Username: "bytebase",
				},
				Password: "pwd",
				ConnectionContext: db.ConnectionContext{
					DatabaseName: "SAMPLE_DB",
				},
			},
			want: odriver.Config{
				Host:     "nb47110.ap-southeast-1@10.0.0.1",
				Port:     "4182",
				Database: "SAMPLE_DB",
				User:     "bytebase",
				Password: "pwd",
			},
		},
		{
			name: "unencrypted private key in Params without passphrase",
			config: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:                     "nb47110.ap-southeast-1",
					Username:                 "bytebase",
					AuthenticationPrivateKey: "-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----",
				},
				ConnectionContext: db.ConnectionContext{
					DatabaseName: "SAMPLE_DB",
				},
			},
			want: odriver.Config{
				Host:     "nb47110.ap-southeast-1",
				Database: "SAMPLE_DB",
				User:     "bytebase",
				Params: map[string]string{
					snowflakedriver.ParamPrivateKey: "-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----",
				},
			},
		},
		{
			name: "encrypted private key in Params with passphrase",
			config: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:                               "nb47110.ap-southeast-1",
					Username:                           "bytebase",
					AuthenticationPrivateKey:           "-----BEGIN ENCRYPTED PRIVATE KEY-----\n...\n-----END ENCRYPTED PRIVATE KEY-----",
					AuthenticationPrivateKeyPassphrase: "secret",
				},
			},
			want: odriver.Config{
				Host: "nb47110.ap-southeast-1",
				User: "bytebase",
				Params: map[string]string{
					snowflakedriver.ParamPrivateKey:           "-----BEGIN ENCRYPTED PRIVATE KEY-----\n...\n-----END ENCRYPTED PRIVATE KEY-----",
					snowflakedriver.ParamPrivateKeyPassphrase: "secret",
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, snowflakeConfig(tc.config))
		})
	}
}

func TestOpenAndCapabilities(t *testing.T) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	pkcs8DER, err := x509.MarshalPKCS8PrivateKey(rsaKey)
	require.NoError(t, err)
	validPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8DER}))

	driver, err := newDriver().Open(context.Background(), storepb.Engine_SNOWFLAKE, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:                     "nb47110.ap-southeast-1",
			Port:                     "443",
			Username:                 "bytebase",
			AuthenticationPrivateKey: validPEM,
		},
		ConnectionContext: db.ConnectionContext{DatabaseName: "SAMPLE_DB"},
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, driver.Close(context.Background())) })

	require.True(t, db.OwnsQuerySession(driver))
	require.Nil(t, db.SQLDB(driver))
}

func TestOpenRefusesInvalidConfigsWithoutDialing(t *testing.T) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	pkcs1PEM := string(pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(rsaKey),
	}))
	_, edKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	edDER, err := x509.MarshalPKCS8PrivateKey(edKey)
	require.NoError(t, err)
	edPEM := string(pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: edDER,
	}))
	encryptedPEM := string(pem.EncodeToMemory(&pem.Block{
		Type:  "ENCRYPTED PRIVATE KEY",
		Bytes: []byte("ciphertext"),
	}))

	for _, tc := range []struct {
		name    string
		config  db.ConnectionConfig
		wantErr string
	}{
		{
			name: "missing user",
			config: db.ConnectionConfig{
				DataSource: &storepb.DataSource{Host: "nb47110.ap-southeast-1"},
				Password:   "pwd",
			},
			wantErr: "user must be set",
		},
		{
			name: "missing host",
			config: db.ConnectionConfig{
				DataSource: &storepb.DataSource{Username: "bytebase"},
				Password:   "pwd",
			},
			wantErr: "host must be set",
		},
		{
			name: "malformed account@host@extra",
			config: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:     "nb47110@10.0.0.1@extra",
					Username: "bytebase",
				},
				Password: "pwd",
			},
			wantErr: "expected one @ in the host at most",
		},
		{
			name: "account without proxy host after @",
			config: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:     "nb47110@",
					Username: "bytebase",
				},
				Password: "pwd",
			},
			wantErr: "must name both an account and a host",
		},
		{
			name: "non-numeric port",
			config: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:     "nb47110",
					Port:     "https",
					Username: "bytebase",
				},
				Password: "pwd",
			},
			wantErr: `port "https" is not a number`,
		},
		{
			name: "invalid PEM private key",
			config: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:                     "nb47110",
					Username:                 "bytebase",
					AuthenticationPrivateKey: "rsa_key.p8",
				},
			},
			wantErr: "failed to get private key PEM block",
		},
		{
			name: "PKCS#1 RSA PRIVATE KEY block rejected",
			config: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:                     "nb47110",
					Username:                 "bytebase",
					AuthenticationPrivateKey: pkcs1PEM,
				},
			},
			wantErr: "unsupported pem block type: RSA PRIVATE KEY",
		},
		{
			name: "non-RSA PKCS#8 private key rejected",
			config: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:                     "nb47110",
					Username:                 "bytebase",
					AuthenticationPrivateKey: edPEM,
				},
			},
			wantErr: "expected RSA private key",
		},
		{
			name: "encrypted private key without passphrase",
			config: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:                     "nb47110",
					Username:                 "bytebase",
					AuthenticationPrivateKey: encryptedPEM,
				},
			},
			wantErr: "private key is encrypted, but no passphrase was provided",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			driver, err := newDriver().Open(context.Background(), storepb.Engine_SNOWFLAKE, tc.config)
			if driver != nil {
				_ = driver.Close(context.Background())
			}
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

type fakeSession struct {
	gotScript string
	gotOpts   odriver.QueryOptions
	results   []*odriver.Result
	err       error
	closed    bool
}

func (s *fakeSession) Query(_ context.Context, script string, opts odriver.QueryOptions) ([]*odriver.Result, error) {
	s.gotScript = script
	s.gotOpts = opts
	return s.results, s.err
}

func (s *fakeSession) Close() error {
	s.closed = true
	return nil
}

func TestQueryConnForwardsOptionsAndConvertsResults(t *testing.T) {
	session := &fakeSession{
		results: []*odriver.Result{
			{
				Statement: `EXPLAIN USING JSON SELECT 1`,
				Columns:   []odriver.Column{{Name: "plan", TypeName: "TEXT"}},
				Rows:      [][]odriver.Value{{{Kind: odriver.String, String: `{"GlobalStats":{}}`}}},
				Plan:      &odriver.Plan{Format: "JSON", Executed: false},
			},
		},
	}
	d := &Driver{session: session}

	got, err := d.QueryConn(context.Background(), nil, "SELECT 1", db.QueryContext{
		Schema:               "ANALYTICS",
		Limit:                100,
		Explain:              true,
		Option:               &v1pb.QueryOption{ExplainFormat: v1pb.QueryOption_JSON},
		MaskingEnabled:       true,
		MaximumSQLResultSize: 1 << 20,
	})
	require.NoError(t, err)
	require.Equal(t, "SELECT 1", session.gotScript)
	require.Equal(t, odriver.QueryOptions{
		Schema:         "ANALYTICS",
		Limit:          100,
		Explain:        true,
		ExplainFormat:  "JSON",
		MaskingEnabled: true,
		MaxResultBytes: 1 << 20,
	}, session.gotOpts)
	require.Len(t, got, 1)
	require.Equal(t, []string{"plan"}, got[0].ColumnNames)
	require.NotNil(t, got[0].QueryPlan)
	require.Equal(t, v1pb.QueryOption_JSON, got[0].QueryPlan.Format)
	require.False(t, got[0].QueryPlan.Executed)
	require.False(t, session.closed)
	require.NoError(t, d.Close(context.Background()))
	require.True(t, session.closed)
}

func TestQueryConnDropsTheSessionAScriptCouldNotRun(t *testing.T) {
	session := &fakeSession{err: errors.New("connection lost")}
	d := &Driver{session: session}

	_, err := d.QueryConn(context.Background(), nil, "SELECT 1", db.QueryContext{MaximumSQLResultSize: 1024})
	require.ErrorContains(t, err, "connection lost")
	require.True(t, session.closed)
	require.Nil(t, d.session)
}

func TestDirectiveShiftMapsEveryRemovedShape(t *testing.T) {
	for _, tc := range []struct {
		script string
		want   string
	}{
		{"-- txn-mode=off\nINSERT INTO t VALUES (1)", "INSERT INTO t VALUES (1)"},
		{"-- keep\n-- txn-mode=off\nINSERT INTO t VALUES (1)", "INSERT INTO t VALUES (1)"},
		{"-- txn-mode=off\n-- keep\nINSERT INTO t VALUES (1)", "-- keep\nINSERT INTO t VALUES (1)"},
		{"-- keep\n\n-- txn-isolation=serializable\n\nINSERT INTO t VALUES (1)", "\nINSERT INTO t VALUES (1)"},
		{"INSERT INTO t VALUES (1)", "INSERT INTO t VALUES (1)"},
		{"-- txn-mode=off\nSELECT 1;\n  SELECT 2", "SELECT 1;\n  SELECT 2"},
		{"-- txn-mode=off\nSELECT 1;\n  SELECT 2  ", "SELECT 1;\n  SELECT 2"},
		{"-- txn-mode=off\n  SELECT 1", "SELECT 1"},
		{"-- txn-mode=off\n\u00a0SELECT 1", "SELECT 1"},
	} {
		t.Run(tc.script, func(t *testing.T) {
			_, cleaned := base.ParseTransactionConfig(tc.script)
			start, end := directiveShift(tc.script, cleaned)(0, len(cleaned))
			require.Equal(t, tc.want, tc.script[start:end])
		})
	}
}

func TestExecuteLoggerMapsRangesAndTransactionControl(t *testing.T) {
	script := "-- txn-mode=on\nINSERT INTO t VALUES (1);\nINSERT INTO t VALUES (2);"
	_, cleaned := base.ParseTransactionConfig(script)

	var logs []*storepb.TaskRunLog
	opts := db.ExecuteOptions{
		MaximumRetries: 3,
		CreateTaskRunLog: func(_ time.Time, entry *storepb.TaskRunLog) error {
			logs = append(logs, entry)
			return nil
		},
	}
	logger := executeLogger{
		opts:        opts,
		sourceRange: directiveShift(script, cleaned),
	}

	firstEnd := len("INSERT INTO t VALUES (1);")
	logger.TransactionControl(odriver.Begin, nil)
	logger.CommandExecute(odriver.Command{
		Text:      cleaned[:firstEnd],
		ByteStart: 0,
		ByteEnd:   firstEnd,
	})
	logger.CommandResponse(1, []int64{1}, nil)
	logger.Retry(errors.New("transient lock timeout"), 1, 3)
	logger.CommandExecute(odriver.Command{
		Text:      cleaned[firstEnd+1:],
		ByteStart: firstEnd + 1,
		ByteEnd:   len(cleaned),
	})
	logger.CommandResponse(0, nil, errors.New("constraint violation"))
	logger.TransactionControl(odriver.Rollback, nil)
	logger.TransactionControl(odriver.Commit, errors.New("commit failed"))

	require.Len(t, logs, 8)

	require.Equal(t, storepb.TaskRunLog_TRANSACTION_CONTROL, logs[0].Type)
	require.Equal(t, storepb.TaskRunLog_TransactionControl_BEGIN, logs[0].GetTransactionControl().GetType())
	require.Empty(t, logs[0].GetTransactionControl().GetError())

	cmd1Range := logs[1].GetCommandExecute().GetRange()
	require.Equal(t, "INSERT INTO t VALUES (1);", script[cmd1Range.GetStart():cmd1Range.GetEnd()])

	require.EqualValues(t, 1, logs[2].GetCommandResponse().GetAffectedRows())
	require.Equal(t, []int64{1}, logs[2].GetCommandResponse().GetAllAffectedRows())
	require.Empty(t, logs[2].GetCommandResponse().GetError())

	require.Equal(t, storepb.TaskRunLog_RETRY_INFO, logs[3].Type)
	require.EqualValues(t, 1, logs[3].GetRetryInfo().GetRetryCount())
	require.EqualValues(t, 3, logs[3].GetRetryInfo().GetMaximumRetries())
	require.Equal(t, "transient lock timeout", logs[3].GetRetryInfo().GetError())

	cmd2Range := logs[4].GetCommandExecute().GetRange()
	require.Equal(t, "INSERT INTO t VALUES (2);", script[cmd2Range.GetStart():cmd2Range.GetEnd()])

	require.Equal(t, "constraint violation", logs[5].GetCommandResponse().GetError())

	require.Equal(t, storepb.TaskRunLog_TransactionControl_ROLLBACK, logs[6].GetTransactionControl().GetType())
	require.Empty(t, logs[6].GetTransactionControl().GetError())

	require.Equal(t, storepb.TaskRunLog_TransactionControl_COMMIT, logs[7].GetTransactionControl().GetType())
	require.Equal(t, "commit failed", logs[7].GetTransactionControl().GetError())
}
