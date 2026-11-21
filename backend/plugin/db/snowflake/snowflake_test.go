package snowflake

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	sqldriver "database/sql/driver"
	"encoding/pem"
	"io"
	"testing"
	"time"

	odriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	snowflakedriver "github.com/bytebase/omni/snowflake/driver"
	snowflakeparser "github.com/bytebase/omni/snowflake/parser"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
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

type fakeImpl struct {
	sessions     []*fakeSession
	sessionCalls int
	closed       bool

	executeScript string
	executeOpts   odriver.ExecuteOptions
	executeFn     func(script string, opts odriver.ExecuteOptions) (int64, error)

	instance    *odriver.Instance
	instanceErr error
	schema      *metadatapb.DatabaseSchemaMetadata
	schemaErr   error
	dumpDDL     string
}

func (*fakeImpl) Ping(context.Context) error { return nil }

func (f *fakeImpl) Close() error {
	f.closed = true
	return nil
}

func (f *fakeImpl) Session(context.Context) (odriver.Session, error) {
	idx := f.sessionCalls
	f.sessionCalls++
	if idx < len(f.sessions) {
		return f.sessions[idx], nil
	}
	return &fakeSession{}, nil
}

func (f *fakeImpl) Execute(_ context.Context, script string, opts odriver.ExecuteOptions) (int64, error) {
	f.executeScript = script
	f.executeOpts = opts
	if f.executeFn != nil {
		return f.executeFn(script, opts)
	}
	return 0, nil
}

func (f *fakeImpl) SyncInstance(context.Context) (*odriver.Instance, error) {
	return f.instance, f.instanceErr
}

func (f *fakeImpl) SyncDatabase(context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	return f.schema, f.schemaErr
}

func (f *fakeImpl) Dump(_ context.Context, w io.Writer, _ *metadatapb.DatabaseSchemaMetadata) error {
	_, err := io.WriteString(w, f.dumpDDL)
	return err
}

func newFakeDriver(impl *fakeImpl, databaseName string) *Driver {
	return &Driver{
		OmniAdapter: util.OmniAdapter{
			Driver:               impl,
			WithInstanceMetadata: true,
			NotFoundResource:     "database",
			DatabaseName:         databaseName,
		},
	}
}

func newFakeSessionDriver(session odriver.Session) *Driver {
	d := &Driver{}
	d.SetSession(session)
	return d
}

func TestQueryConnLazySessionAndExplainOptions(t *testing.T) {
	s1 := &fakeSession{
		results: []*odriver.Result{
			{
				Statement: `EXPLAIN USING JSON SELECT 1`,
				Columns:   []odriver.Column{{Name: "plan", TypeName: "TEXT"}},
				Rows:      [][]odriver.Value{{{Kind: odriver.String, String: `{"GlobalStats":{}}`}}},
				// snowflake/analysis.DescribeExplain returns lowercase "json".
				Plan: &odriver.Plan{Format: "json", Executed: false},
			},
		},
	}
	impl := &fakeImpl{sessions: []*fakeSession{s1}}
	d := newFakeDriver(impl, "SAMPLE_DB")
	require.Equal(t, 0, impl.sessionCalls, "session must not be opened before first QueryConn")

	// Explain: true leaves QueryPlan nil so SQLService stamps the requested format.
	gotExplain, err := d.QueryConn(context.Background(), "SELECT 1", db.QueryContext{
		Schema:               "ANALYTICS",
		Limit:                100,
		Explain:              true,
		Option:               &v1pb.QueryOption{ExplainFormat: v1pb.QueryOption_JSON},
		MaskingEnabled:       true,
		MaximumSQLResultSize: 1 << 20,
	})
	require.NoError(t, err)
	require.Equal(t, 1, impl.sessionCalls)
	require.Equal(t, odriver.QueryOptions{
		Schema:         "ANALYTICS",
		Limit:          100,
		Explain:        true,
		ExplainFormat:  "JSON",
		MaskingEnabled: true,
		MaxResultBytes: 1 << 20,
	}, s1.gotOpts)
	require.Len(t, gotExplain, 1)
	require.Nil(t, gotExplain[0].QueryPlan)

	// User-written EXPLAIN (Explain: false) reuses the held session and maps
	// lowercase "json" onto QueryOption_JSON.
	gotUserExplain, err := d.QueryConn(context.Background(), "EXPLAIN USING JSON SELECT 1", db.QueryContext{
		MaximumSQLResultSize: 1 << 20,
	})
	require.NoError(t, err)
	require.Equal(t, 1, impl.sessionCalls, "subsequent QueryConn must reuse held session")
	require.Len(t, gotUserExplain, 1)
	require.Equal(t, &v1pb.QueryResult_QueryPlan{
		Format:   v1pb.QueryOption_JSON,
		Executed: false,
	}, gotUserExplain[0].QueryPlan)

	require.NoError(t, d.Close(context.Background()))
	require.True(t, s1.closed)
	require.True(t, impl.closed)
}

func TestQueryConnNormalizesResults(t *testing.T) {
	t.Run("columnless command with nil AffectedRows answers Affected Rows 0", func(t *testing.T) {
		session := &fakeSession{
			results: []*odriver.Result{
				{
					Statement: "CREATE TABLE t (id INT)",
					Messages:  []odriver.Message{{Level: "INFO", Content: "created"}},
				},
			},
		}
		d := newFakeSessionDriver(session)
		got, err := d.QueryConn(context.Background(), "CREATE TABLE t (id INT)", db.QueryContext{})
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, []string{"Affected Rows"}, got[0].ColumnNames)
		require.Equal(t, []string{"INT"}, got[0].ColumnTypeNames)
		require.EqualValues(t, 1, got[0].RowsCount)
		require.Len(t, got[0].Rows, 1)
		require.EqualValues(t, 0, got[0].Rows[0].Values[0].GetInt64Value())
		require.Len(t, got[0].Messages, 1)
	})

	t.Run("zero-row query with columns keeps empty rows", func(t *testing.T) {
		session := &fakeSession{
			results: []*odriver.Result{
				{
					Statement: "SELECT id FROM t WHERE 1 = 0",
					Columns:   []odriver.Column{{Name: "ID", TypeName: "FIXED"}},
				},
			},
		}
		d := newFakeSessionDriver(session)
		got, err := d.QueryConn(context.Background(), "SELECT id FROM t WHERE 1 = 0", db.QueryContext{})
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, []string{"ID"}, got[0].ColumnNames)
		require.Empty(t, got[0].Rows)
		require.Zero(t, got[0].RowsCount)
	})

	t.Run("failed EXPLAIN clears QueryPlan", func(t *testing.T) {
		session := &fakeSession{
			results: []*odriver.Result{
				{
					Statement: "EXPLAIN SELECT * FROM missing",
					Err:       errors.New("object does not exist"),
					Plan:      &odriver.Plan{Format: "text"},
				},
			},
		}
		d := newFakeSessionDriver(session)
		got, err := d.QueryConn(context.Background(), "EXPLAIN SELECT * FROM missing", db.QueryContext{})
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, "object does not exist", got[0].Error)
		require.Nil(t, got[0].QueryPlan)
		require.Empty(t, got[0].ColumnNames)
	})

	t.Run("timestamps outside protobuf range convert to strings", func(t *testing.T) {
		session := &fakeSession{
			results: []*odriver.Result{
				{
					Statement: "SELECT ts_ntz, ts_tz, valid_ts",
					Columns: []odriver.Column{
						{Name: "TS_NTZ", TypeName: "TIMESTAMP_NTZ"},
						{Name: "TS_TZ", TypeName: "TIMESTAMP_TZ"},
						{Name: "VALID_TS", TypeName: "TIMESTAMP_NTZ"},
					},
					Rows: [][]odriver.Value{{
						{Kind: odriver.Timestamp, Time: time.Date(0, 1, 1, 0, 0, 0, 7000000, time.UTC), Precision: 3},
						{Kind: odriver.TimestampTZ, Time: time.Date(10000, 1, 2, 3, 4, 5, 0, time.UTC), Precision: 0},
						{Kind: odriver.Timestamp, Time: time.Date(2026, 5, 11, 12, 34, 56, 0, time.UTC), Precision: 0},
					}},
				},
			},
		}
		d := newFakeSessionDriver(session)
		got, err := d.QueryConn(context.Background(), "SELECT ts_ntz, ts_tz, valid_ts", db.QueryContext{})
		require.NoError(t, err)
		require.Len(t, got, 1)
		_, err = protojson.Marshal(got[0])
		require.NoError(t, err)
		require.Equal(t, "0000-01-01 00:00:00.007", got[0].Rows[0].Values[0].GetStringValue())
		require.Equal(t, "10000-01-02T03:04:05Z", got[0].Rows[0].Values[1].GetStringValue())
		require.NotNil(t, got[0].Rows[0].Values[2].GetTimestampValue())
	})

	t.Run("zero MaximumSQLResultSize clamps session cap to 1 byte instead of unbounded", func(t *testing.T) {
		session := &fakeSession{
			results: []*odriver.Result{
				{
					Statement: "SELECT 1",
					Columns:   []odriver.Column{{Name: "1", TypeName: "FIXED"}},
					Rows:      [][]odriver.Value{{{Kind: odriver.Int64, Int64: 1}}},
					Truncated: true,
				},
			},
		}
		d := newFakeSessionDriver(session)
		got, err := d.QueryConn(context.Background(), "SELECT 1", db.QueryContext{})
		require.NoError(t, err)
		require.EqualValues(t, 1, session.gotOpts.MaxResultBytes)
		require.Len(t, got, 1)
		require.Contains(t, got[0].Error, "max allowed output size of 0MB")
	})
}

type rawError string

func (e rawError) Error() string { return string(e) }

func TestQueryConnDropsSessionOnErrorOrCanceledContext(t *testing.T) {
	t.Run("recoverable USE SCHEMA and pre-execution errors keep session for next request", func(t *testing.T) {
		for _, recErr := range []error{
			rawError("002043 (02000): SQL compilation error:\nObject does not exist, or operation cannot be performed."),
			rawError("003001 (42501): SQL access control error:\nInsufficient privileges to operate on schema 'SECRET'."),
			rawError("390201 (-1): Object does not exist, or operation cannot be performed."),
			&snowflakeparser.ParseError{Message: "syntax error"},
			rawError("EXPLAIN plans a single statement"),
			rawError("EXPLAIN needs a statement to plan"),
		} {
			s1 := &fakeSession{err: recErr}
			s2 := &fakeSession{}
			impl := &fakeImpl{sessions: []*fakeSession{s1, s2}}
			d := newFakeDriver(impl, "SAMPLE_DB")

			_, err := d.QueryConn(context.Background(), "SELECT * FROM temp_tbl", db.QueryContext{
				Schema:               "MISSING_SCHEMA",
				MaximumSQLResultSize: 1024,
			})
			require.ErrorIs(t, err, recErr)
			require.False(t, s1.closed, "recoverable error %v must not close held session", recErr)
			require.Same(t, s1, d.Session())

			s1.err = nil
			s1.results = []*odriver.Result{{
				Statement: "SELECT * FROM temp_tbl",
				Columns:   []odriver.Column{{Name: "ID", TypeName: "FIXED"}},
				Rows:      [][]odriver.Value{{{Kind: odriver.Int64, Int64: 42}}},
			}}
			got, err := d.QueryConn(context.Background(), "SELECT * FROM temp_tbl", db.QueryContext{
				Schema:               "PUBLIC",
				MaximumSQLResultSize: 1024,
			})
			require.NoError(t, err)
			require.Equal(t, 1, impl.sessionCalls, "next request must reuse the existing session")
			require.Len(t, got, 1)
			require.EqualValues(t, 42, got[0].Rows[0].Values[0].GetInt64Value())
		}
	})

	t.Run("unrecoverable connection or session-gone errors discard session", func(t *testing.T) {
		for _, fatalErr := range []error{
			errors.New("connection lost"),
			rawError("000604 (57014): SQL execution canceled"),
			rawError("261000 (08006): failed to POST. HTTP: 503, URL: https://account.snowflakecomputing.com/queries/v1/query-request"),
			rawError("390111 (-1): Session no longer exists. New login required to access the service."),
			rawError("390112 (-1): Your session has expired. Please login again."),
			sql.ErrConnDone,
			sqldriver.ErrBadConn,
		} {
			s1 := &fakeSession{err: fatalErr}
			s2 := &fakeSession{
				results: []*odriver.Result{{
					Statement: "SELECT 1",
					Columns:   []odriver.Column{{Name: "1", TypeName: "FIXED"}},
					Rows:      [][]odriver.Value{{{Kind: odriver.Int64, Int64: 1}}},
				}},
			}
			impl := &fakeImpl{sessions: []*fakeSession{s1, s2}}
			d := newFakeDriver(impl, "SAMPLE_DB")

			_, err := d.QueryConn(context.Background(), "SELECT 1", db.QueryContext{MaximumSQLResultSize: 1024})
			require.ErrorIs(t, err, fatalErr)
			require.True(t, s1.closed, "fatal error %v must close dead session", fatalErr)
			require.Nil(t, d.Session())

			got, err := d.QueryConn(context.Background(), "SELECT 1", db.QueryContext{MaximumSQLResultSize: 1024})
			require.NoError(t, err)
			require.Equal(t, 2, impl.sessionCalls)
			require.Len(t, got, 1)
		}
	})

	t.Run("canceled context discards session even on SQL-shaped error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		session := &fakeSession{
			err: rawError("002043 (02000): SQL compilation error"),
		}
		d := newFakeSessionDriver(session)

		_, err := d.QueryConn(ctx, "SELECT 1", db.QueryContext{Schema: "MISSING", MaximumSQLResultSize: 1024})
		require.Error(t, err)
		require.True(t, session.closed)
		require.Nil(t, d.Session())
	})

	t.Run("canceled context with partial results discards session", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		session := &fakeSession{
			results: []*odriver.Result{{Statement: "SELECT 1", Err: context.Canceled}},
		}
		d := newFakeSessionDriver(session)

		got, err := d.QueryConn(ctx, "SELECT 1", db.QueryContext{MaximumSQLResultSize: 1024})
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.True(t, session.closed)
		require.Nil(t, d.Session())
	})
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
			start, end := util.DirectiveShift(tc.script, cleaned)(0, len(cleaned))
			require.Equal(t, tc.want, tc.script[start:end])
		})
	}
}

func TestExecuteModesAndLogging(t *testing.T) {
	t.Run("default mode runs transactional and maps ranges back to caller script", func(t *testing.T) {
		script := "-- txn-mode=on\nINSERT INTO t VALUES (1);\nINSERT INTO t VALUES (2);"
		var logs []*storepb.TaskRunLog
		impl := &fakeImpl{
			executeFn: func(cleaned string, opts odriver.ExecuteOptions) (int64, error) {
				firstEnd := len("INSERT INTO t VALUES (1);")
				opts.Log.TransactionControl(odriver.Begin, nil)
				opts.Log.CommandExecute(odriver.Command{Text: cleaned[:firstEnd], ByteStart: 0, ByteEnd: firstEnd})
				opts.Log.CommandResponse(1, []int64{1}, nil)
				opts.Log.Retry(errors.New("transient lock timeout"), 1, 3)
				opts.Log.CommandExecute(odriver.Command{Text: cleaned[firstEnd+1:], ByteStart: firstEnd + 1, ByteEnd: len(cleaned)})
				opts.Log.CommandResponse(2, []int64{2}, nil)
				opts.Log.TransactionControl(odriver.Commit, nil)
				return 3, nil
			},
		}
		d := newFakeDriver(impl, "")

		affected, err := d.Execute(context.Background(), script, db.ExecuteOptions{
			CreateDatabase: true,
			MaximumRetries: 3,
			CreateTaskRunLog: func(_ time.Time, entry *storepb.TaskRunLog) error {
				logs = append(logs, entry)
				return nil
			},
		})
		require.NoError(t, err)
		require.EqualValues(t, 3, affected)
		require.Equal(t, "INSERT INTO t VALUES (1);\nINSERT INTO t VALUES (2);", impl.executeScript)
		require.Equal(t, odriver.Transactional, impl.executeOpts.Transaction)
		require.True(t, impl.executeOpts.CreateDatabase)
		require.Equal(t, 3, impl.executeOpts.MaxRetries)
		require.Len(t, logs, 7)

		require.Equal(t, storepb.TaskRunLog_TransactionControl_BEGIN, logs[0].GetTransactionControl().GetType())
		r1 := logs[1].GetCommandExecute().GetRange()
		require.Equal(t, "INSERT INTO t VALUES (1);", script[r1.GetStart():r1.GetEnd()])
		require.EqualValues(t, 1, logs[2].GetCommandResponse().GetAffectedRows())
		require.Equal(t, "transient lock timeout", logs[3].GetRetryInfo().GetError())
		r2 := logs[4].GetCommandExecute().GetRange()
		require.Equal(t, "INSERT INTO t VALUES (2);", script[r2.GetStart():r2.GetEnd()])
		require.Equal(t, storepb.TaskRunLog_TransactionControl_COMMIT, logs[6].GetTransactionControl().GetType())
	})

	t.Run("txn-mode=off selects AutoCommit and logs errors and rollback", func(t *testing.T) {
		script := "-- txn-mode=off\nALTER TABLE t ADD COLUMN c INT;"
		var logs []*storepb.TaskRunLog
		impl := &fakeImpl{
			executeFn: func(cleaned string, opts odriver.ExecuteOptions) (int64, error) {
				opts.Log.CommandExecute(odriver.Command{Text: cleaned, ByteStart: 0, ByteEnd: len(cleaned)})
				opts.Log.CommandResponse(0, nil, errors.New("syntax error"))
				opts.Log.TransactionControl(odriver.Rollback, errors.New("rollback failed"))
				return 0, errors.New("syntax error")
			},
		}
		d := newFakeDriver(impl, "")

		_, err := d.Execute(context.Background(), script, db.ExecuteOptions{
			CreateTaskRunLog: func(_ time.Time, entry *storepb.TaskRunLog) error {
				logs = append(logs, entry)
				return nil
			},
		})
		require.ErrorContains(t, err, "syntax error")
		require.Equal(t, odriver.AutoCommit, impl.executeOpts.Transaction)
		require.Len(t, logs, 3)
		r := logs[0].GetCommandExecute().GetRange()
		require.Equal(t, "ALTER TABLE t ADD COLUMN c INT;", script[r.GetStart():r.GetEnd()])
		require.Equal(t, "syntax error", logs[1].GetCommandResponse().GetError())
		require.Equal(t, storepb.TaskRunLog_TransactionControl_ROLLBACK, logs[2].GetTransactionControl().GetType())
		require.Equal(t, "rollback failed", logs[2].GetTransactionControl().GetError())
	})
}

func TestSyncAndDumpMapping(t *testing.T) {
	attr := "ACCOUNTADMIN, SYSADMIN"
	impl := &fakeImpl{
		instance: &odriver.Instance{
			Version:   "9.12.1",
			Databases: []*metadatapb.DatabaseSchemaMetadata{{Name: "SAMPLE_DB"}},
			Roles:     []*odriver.Role{{Name: "BYTEBASE", Attribute: &attr}},
		},
		schema:  &metadatapb.DatabaseSchemaMetadata{Name: "SAMPLE_DB"},
		dumpDDL: "create table PUBLIC.T (ID NUMBER);\n",
	}
	d := newFakeDriver(impl, "SAMPLE_DB")

	inst, err := d.SyncInstance(context.Background())
	require.NoError(t, err)
	require.Equal(t, "9.12.1", inst.Version)
	require.Len(t, inst.Databases, 1)
	require.Equal(t, "SAMPLE_DB", inst.Databases[0].Name)
	require.Len(t, inst.Metadata.GetRoles(), 1)
	require.Equal(t, "BYTEBASE", inst.Metadata.GetRoles()[0].GetName())
	require.Equal(t, attr, inst.Metadata.GetRoles()[0].GetAttribute())

	schema, err := d.SyncDBSchema(context.Background())
	require.NoError(t, err)
	require.Equal(t, "SAMPLE_DB", schema.Name)

	impl.schemaErr = errors.Wrapf(odriver.ErrNotFound, "database %q", "SAMPLE_DB")
	_, err = d.SyncDBSchema(context.Background())
	require.Error(t, err)
	var bbErr *common.Error
	require.ErrorAs(t, err, &bbErr)
	require.Equal(t, common.NotFound, bbErr.Code)
	require.Contains(t, err.Error(), `database "SAMPLE_DB" not found`)

	var out bytes.Buffer
	require.NoError(t, d.Dump(context.Background(), &out, schema))
	require.Equal(t, "create table PUBLIC.T (ID NUMBER);\n", out.String())
}
