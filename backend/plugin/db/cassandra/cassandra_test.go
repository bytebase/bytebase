package cassandra

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"testing"
	"time"

	omnidriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

func TestConvertValue(t *testing.T) {
	timestamp := time.Date(2026, 11, 18, 7, 30, 0, 0, time.UTC)
	zone := time.FixedZone("PDT", -7*60*60)
	for _, tc := range []struct {
		name  string
		value omnidriver.Value
		want  *v1pb.RowValue
	}{
		{
			name:  "null",
			value: omnidriver.Value{Kind: omnidriver.Null},
			want:  util.NullRowValue,
		},
		{
			name:  "bool",
			value: omnidriver.Value{Kind: omnidriver.Bool, Bool: true},
			want:  &v1pb.RowValue{Kind: &v1pb.RowValue_BoolValue{BoolValue: true}},
		},
		{
			name:  "blob",
			value: omnidriver.Value{Kind: omnidriver.Bytes, Bytes: []byte("bytes")},
			want:  &v1pb.RowValue{Kind: &v1pb.RowValue_BytesValue{BytesValue: []byte("bytes")}},
		},
		{
			// A CQL float is 32-bit, and Bytebase showed it as a FloatValue.
			name:  "float",
			value: omnidriver.Value{Kind: omnidriver.Float32, Float64: 1.5},
			want:  &v1pb.RowValue{Kind: &v1pb.RowValue_FloatValue{FloatValue: 1.5}},
		},
		{
			name:  "double",
			value: omnidriver.Value{Kind: omnidriver.Float64, Float64: 3.25},
			want:  &v1pb.RowValue{Kind: &v1pb.RowValue_DoubleValue{DoubleValue: 3.25}},
		},
		{
			// smallint and tinyint widened to Int32Value, int and bigint to Int64Value.
			name:  "small int",
			value: omnidriver.Value{Kind: omnidriver.Int32, Int64: 7},
			want:  &v1pb.RowValue{Kind: &v1pb.RowValue_Int32Value{Int32Value: 7}},
		},
		{
			name:  "big int",
			value: omnidriver.Value{Kind: omnidriver.Int64, Int64: 1 << 40},
			want:  &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{Int64Value: 1 << 40}},
		},
		{
			name:  "unsigned int",
			value: omnidriver.Value{Kind: omnidriver.Uint32, Uint64: 7},
			want:  &v1pb.RowValue{Kind: &v1pb.RowValue_Uint32Value{Uint32Value: 7}},
		},
		{
			name:  "unsigned big int",
			value: omnidriver.Value{Kind: omnidriver.Uint64, Uint64: 1 << 40},
			want:  &v1pb.RowValue{Kind: &v1pb.RowValue_Uint64Value{Uint64Value: 1 << 40}},
		},
		{
			// text, ascii, varchar, uuid, inet, decimal, varint, time and duration.
			name:  "string",
			value: omnidriver.Value{Kind: omnidriver.String, String: "text"},
			want:  &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: "text"}},
		},
		{
			name:  "structured",
			value: omnidriver.Value{Kind: omnidriver.Structured, Structured: []any{1.0, "a"}},
			want:  &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: `[1,"a"]`}},
		},
		{
			name:  "timestamp",
			value: omnidriver.Value{Kind: omnidriver.Timestamp, Time: timestamp, Precision: 3},
			want: &v1pb.RowValue{Kind: &v1pb.RowValue_TimestampValue{TimestampValue: &v1pb.RowValue_Timestamp{
				GoogleTimestamp: timestamppb.New(timestamp),
				Accuracy:        3,
			}}},
		},
		{
			name:  "timestamp with zone",
			value: omnidriver.Value{Kind: omnidriver.TimestampTZ, Time: timestamp.In(zone), Precision: 3},
			want: &v1pb.RowValue{Kind: &v1pb.RowValue_TimestampTzValue{TimestampTzValue: &v1pb.RowValue_TimestampTZ{
				GoogleTimestamp: timestamppb.New(timestamp.In(zone)),
				Zone:            "PDT",
				Offset:          -7 * 60 * 60,
				Accuracy:        3,
			}}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, convertValue(tc.value))
		})
	}
}

func toQueryResult(result *omnidriver.Result, queryContext db.QueryContext) *v1pb.QueryResult {
	return util.OmniResultToQueryResultWith(result, queryContext.MaximumSQLResultSize, util.QueryResultOptions{
		SkipPlan:      true,
		SkipMessages:  true,
		TruncatedWins: true,
		ConvertValue:  convertValue,
	})
}

func TestToQueryResult(t *testing.T) {
	result := toQueryResult(&omnidriver.Result{
		Statement: "SELECT * FROM t",
		Columns: []omnidriver.Column{
			{Name: "id", TypeName: "int"},
			{Name: "name", TypeName: "text"},
		},
		Rows: [][]omnidriver.Value{
			{{Kind: omnidriver.Int32, Int64: 1}, {Kind: omnidriver.String, String: "a"}},
			{{Kind: omnidriver.Int32, Int64: 2}, {Kind: omnidriver.Null}},
		},
	}, db.QueryContext{})

	require.Equal(t, "SELECT * FROM t", result.Statement)
	require.Equal(t, []string{"id", "name"}, result.ColumnNames)
	require.Equal(t, []string{"int", "text"}, result.ColumnTypeNames)
	require.Equal(t, int64(2), result.RowsCount)
	require.Empty(t, result.Error)
	require.Equal(t, []*v1pb.QueryRow{
		{Values: []*v1pb.RowValue{
			{Kind: &v1pb.RowValue_Int32Value{Int32Value: 1}},
			{Kind: &v1pb.RowValue_StringValue{StringValue: "a"}},
		}},
		{Values: []*v1pb.RowValue{
			{Kind: &v1pb.RowValue_Int32Value{Int32Value: 2}},
			util.NullRowValue,
		}},
	}, result.Rows)
}

func TestToQueryResultReportsTruncationAndErrors(t *testing.T) {
	truncated := toQueryResult(&omnidriver.Result{Statement: "SELECT * FROM t", Truncated: true}, db.QueryContext{MaximumSQLResultSize: 1 << 20})
	require.Equal(t, util.FormatMaximumSQLResultSizeMessage(1<<20), truncated.Error)

	failed := toQueryResult(&omnidriver.Result{Statement: "SELECT * FROM t", Err: errTest}, db.QueryContext{})
	require.Equal(t, "failed: refused", failed.Error)
}

var errTest = &testError{}

type testError struct{}

func (*testError) Error() string { return "failed: refused" }

func TestExecuteLoggerReportsEachStep(t *testing.T) {
	var logs []*storepb.TaskRunLog
	opts := db.ExecuteOptions{
		LogCommandStatement: true,
		CreateTaskRunLog: func(_ time.Time, log *storepb.TaskRunLog) error {
			logs = append(logs, log)
			return nil
		},
	}
	logger := util.NewExecuteLogger(opts)
	logger.CommandExecute(omnidriver.Command{Text: "SELECT 1", ByteStart: 0, ByteEnd: 8})
	logger.CommandResponse(0, nil, errTest)
	logger.TransactionControl(omnidriver.Begin, nil)
	logger.Retry(errTest, 1, 3)

	require.Len(t, logs, 4)
	require.Equal(t, storepb.TaskRunLog_COMMAND_EXECUTE, logs[0].Type)
	require.Equal(t, "SELECT 1", logs[0].GetCommandExecute().GetStatement())
	require.Equal(t, storepb.TaskRunLog_COMMAND_RESPONSE, logs[1].Type)
	require.Equal(t, "failed: refused", logs[1].GetCommandResponse().GetError())
	require.Equal(t, storepb.TaskRunLog_TRANSACTION_CONTROL, logs[2].Type)
	require.Equal(t, storepb.TaskRunLog_TransactionControl_BEGIN, logs[2].GetTransactionControl().GetType())
	require.Equal(t, storepb.TaskRunLog_RETRY_INFO, logs[3].Type)
	require.Equal(t, int32(1), logs[3].GetRetryInfo().GetRetryCount())
}

func TestExecuteLoggerLogsTheCommandRange(t *testing.T) {
	var logs []*storepb.TaskRunLog
	opts := db.ExecuteOptions{CreateTaskRunLog: func(_ time.Time, log *storepb.TaskRunLog) error {
		logs = append(logs, log)
		return nil
	}}
	util.NewExecuteLogger(opts).CommandExecute(omnidriver.Command{Text: "SELECT 1", ByteStart: 3, ByteEnd: 11})

	require.Len(t, logs, 1)
	require.Equal(t, &storepb.Range{Start: 3, End: 11}, logs[0].GetCommandExecute().GetRange())
}

func TestExecuteLoggerIsSilentWithoutACallback(t *testing.T) {
	require.NotPanics(t, func() {
		logger := util.NewExecuteLogger(db.ExecuteOptions{})
		logger.CommandExecute(omnidriver.Command{Text: "SELECT 1"})
		logger.CommandResponse(0, nil, nil)
		logger.TransactionControl(omnidriver.Commit, nil)
		logger.Retry(nil, 1, 1)
	})
}

func TestFormatAddress(t *testing.T) {
	require.Equal(t, "node1.example.com:9042", formatAddress("node1.example.com", "9042"))
	require.Equal(t, "node1.example.com", formatAddress("node1.example.com", ""))
	require.Equal(t, "[2001:db8::1]:9042", formatAddress("2001:db8::1", "9042"))
	require.Equal(t, "[2001:db8::1]:9042", formatAddress("[2001:db8::1]", "9042"))
}

func TestTLSConfigFuncVerifiesDialedHost(t *testing.T) {
	require.Nil(t, util.HostTLSFunc(&storepb.DataSource{Host: "node1.example.com"}))

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	require.NoError(t, err)
	caPEM := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER}))

	signLeaf := func(serial int64, host string) []byte {
		leafKey, genErr := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, genErr)
		tmpl := &x509.Certificate{
			SerialNumber: big.NewInt(serial),
			Subject:      pkix.Name{CommonName: host},
			DNSNames:     []string{host},
			NotBefore:    time.Now().Add(-time.Hour),
			NotAfter:     time.Now().Add(time.Hour),
			KeyUsage:     x509.KeyUsageDigitalSignature,
			ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		}
		der, createErr := x509.CreateCertificate(rand.Reader, tmpl, caTemplate, &leafKey.PublicKey, caKey)
		require.NoError(t, createErr)
		return der
	}

	node1Cert := signLeaf(2, "node1.example.com")
	node2Cert := signLeaf(3, "node2.example.com")

	fn := util.HostTLSFunc(&storepb.DataSource{
		Host:                 "node1.example.com",
		UseSsl:               true,
		VerifyTlsCertificate: true,
		SslCa:                caPEM,
	})
	require.NotNil(t, fn)

	tlsCfg, err := fn("node2.example.com")
	require.NoError(t, err)
	require.Equal(t, "node2.example.com", tlsCfg.ServerName)
	require.NoError(t, tlsCfg.VerifyPeerCertificate([][]byte{node2Cert}, nil))
	require.Error(t, tlsCfg.VerifyPeerCertificate([][]byte{node1Cert}, nil))
}

func TestTunnelConnRemoteAddr(t *testing.T) {
	addr := remoteTCPAddr("127.0.0.1:9042")
	require.Equal(t, net.ParseIP("127.0.0.1"), addr.IP)
	require.Equal(t, 9042, addr.Port)

	client, server := net.Pipe()
	t.Cleanup(func() {
		_ = client.Close()
		_ = server.Close()
	})
	wrapped := &tunnelConn{Conn: client, remote: addr}
	tcpAddr, ok := wrapped.RemoteAddr().(*net.TCPAddr)
	require.True(t, ok)
	require.Equal(t, 9042, tcpAddr.Port)
}

func TestOpenWithUnreachableSSHHost(t *testing.T) {
	_, err := newDriver().Open(context.Background(), storepb.Engine_CASSANDRA, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:    "127.0.0.1",
			Port:    "9042",
			SshHost: "127.0.0.1",
			SshPort: "1",
		},
	})
	require.ErrorContains(t, err, "127.0.0.1:1")
}

type fakeCassandraSession struct {
	queries []string
	opts    []omnidriver.QueryOptions
	queryFn func(script string) ([]*omnidriver.Result, error)
	closed  bool
}

func (s *fakeCassandraSession) Query(_ context.Context, script string, opts omnidriver.QueryOptions) ([]*omnidriver.Result, error) {
	s.queries = append(s.queries, script)
	s.opts = append(s.opts, opts)
	if s.queryFn != nil {
		return s.queryFn(script)
	}
	return []*omnidriver.Result{{Statement: script}}, nil
}

func (s *fakeCassandraSession) Close() error {
	s.closed = true
	return nil
}

type fakeCassandraDriver struct {
	session       *fakeCassandraSession
	executeScript string
	executeFn     func(script string, opts omnidriver.ExecuteOptions) (int64, error)
	instance      *omnidriver.Instance
	schema        *metadatapb.DatabaseSchemaMetadata
	schemaErr     error
}

func (*fakeCassandraDriver) Ping(context.Context) error { return nil }
func (*fakeCassandraDriver) Close() error               { return nil }
func (f *fakeCassandraDriver) Session(context.Context) (omnidriver.Session, error) {
	return f.session, nil
}

func (f *fakeCassandraDriver) Execute(_ context.Context, script string, opts omnidriver.ExecuteOptions) (int64, error) {
	f.executeScript = script
	if f.executeFn != nil {
		return f.executeFn(script, opts)
	}
	return 0, nil
}

func (f *fakeCassandraDriver) SyncInstance(context.Context) (*omnidriver.Instance, error) {
	return f.instance, nil
}

func (f *fakeCassandraDriver) SyncDatabase(context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	return f.schema, f.schemaErr
}

func (*fakeCassandraDriver) Dump(context.Context, io.Writer, *metadatapb.DatabaseSchemaMetadata) error {
	return nil
}

func TestCassandraAdapterSeam(t *testing.T) {
	ctx := context.Background()
	sess := &fakeCassandraSession{
		queryFn: func(script string) ([]*omnidriver.Result, error) {
			if script == "SELECT bad FROM t" {
				return []*omnidriver.Result{{Statement: script, Err: errTest}}, nil
			}
			return []*omnidriver.Result{{
				Statement: script,
				Columns:   []omnidriver.Column{{Name: "id", TypeName: "int"}},
				Rows:      [][]omnidriver.Value{{{Kind: omnidriver.Int64, Int64: 1}}},
			}}, nil
		},
	}
	var logs []*storepb.TaskRunLog
	impl := &fakeCassandraDriver{
		session: sess,
		executeFn: func(script string, opts omnidriver.ExecuteOptions) (int64, error) {
			opts.Log.CommandExecute(omnidriver.Command{Text: script, ByteStart: 0, ByteEnd: len(script)})
			opts.Log.CommandResponse(0, nil, nil)
			return 0, nil
		},
		instance: &omnidriver.Instance{
			Version:   "5.0.0",
			Databases: []*metadatapb.DatabaseSchemaMetadata{{Name: "ks"}},
		},
		schema: &metadatapb.DatabaseSchemaMetadata{Name: "ks"},
	}
	d := &Driver{OmniAdapter: util.OmniAdapter{Driver: impl}}

	// QueryConn splits CQL statements, forwards Limit and MaxResultBytes, and stops at the first statement error.
	got, err := d.QueryConn(ctx, "SELECT id FROM t; SELECT bad FROM t; SELECT never FROM t;", db.QueryContext{
		Limit:                5,
		MaximumSQLResultSize: 1024,
	})
	require.NoError(t, err)
	require.True(t, sess.closed)
	require.Len(t, got, 2)
	require.Equal(t, []string{"SELECT id FROM t", "SELECT bad FROM t"}, sess.queries)
	require.Equal(t, omnidriver.QueryOptions{Limit: 5, MaxResultBytes: 1024}, sess.opts[0])
	require.Empty(t, got[0].Error)
	require.Equal(t, "failed: refused", got[1].Error)

	// Execute delegates and reports steps through ExecuteLogger.
	affected, err := d.Execute(ctx, "INSERT INTO t (id) VALUES (1);", db.ExecuteOptions{
		CreateTaskRunLog: func(_ time.Time, entry *storepb.TaskRunLog) error {
			logs = append(logs, entry)
			return nil
		},
	})
	require.NoError(t, err)
	require.Zero(t, affected)
	require.Equal(t, "INSERT INTO t (id) VALUES (1);", impl.executeScript)
	require.Len(t, logs, 2)

	inst, err := d.SyncInstance(ctx)
	require.NoError(t, err)
	require.Equal(t, "5.0.0", inst.Version)

	schema, err := d.SyncDBSchema(ctx)
	require.NoError(t, err)
	require.Equal(t, "ks", schema.GetName())

	var out bytes.Buffer
	require.NoError(t, d.Dump(ctx, &out, schema))
	require.Empty(t, out.String())

	impl.schemaErr = omnidriver.ErrNotFound
	_, err = d.SyncDBSchema(ctx)
	require.ErrorIs(t, err, omnidriver.ErrNotFound)
}
