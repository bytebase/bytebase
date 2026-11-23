package mongodb

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	omnidriver "github.com/bytebase/omni/driver"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

// TestConnectionConfig pins the mapping from a data source onto the driver
// contract. The connection string the contract builds from it is omni's own
// TestConnectionURI, whose cases are these.
func TestConnectionConfig(t *testing.T) {
	tests := []struct {
		name       string
		connConfig db.ConnectionConfig
		want       omnidriver.Config
		wantSRV    bool
	}{
		{
			name: "host and port",
			connConfig: db.ConnectionConfig{
				DataSource: &storepb.DataSource{Host: "localhost", Port: "27017"},
			},
			want: omnidriver.Config{Host: "localhost:27017", Port: "27017"},
		},
		{
			name: "direct connection",
			connConfig: db.ConnectionConfig{
				DataSource: &storepb.DataSource{Host: "localhost", Port: "27017", DirectConnection: true},
			},
			want: omnidriver.Config{
				Host:   "localhost:27017",
				Port:   "27017",
				Params: map[string]string{"directConnection": "true"},
			},
		},
		{
			name: "database",
			connConfig: db.ConnectionConfig{
				DataSource:        &storepb.DataSource{Host: "localhost", Port: "27017"},
				ConnectionContext: db.ConnectionContext{DatabaseName: "sampleDB"},
			},
			want: omnidriver.Config{Host: "localhost:27017", Port: "27017", Database: "sampleDB"},
		},
		{
			name: "srv",
			connConfig: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:     "cluster0.sample.mongodb.net",
					Username: "bytebase",
					Srv:      true,
				},
				ConnectionContext: db.ConnectionContext{DatabaseName: "sampleDB"},
				Password:          "passwd",
			},
			wantSRV: true,
			want: omnidriver.Config{
				Host:     "cluster0.sample.mongodb.net",
				Database: "sampleDB",
				User:     "bytebase",
				Password: "passwd",
			},
		},
		{
			name: "connection string options",
			connConfig: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:     "cluster0.sample.mongodb.net",
					Username: "bytebase",
					Srv:      true,
					ExtraConnectionParameters: map[string]string{
						"readPreference":     "secondary",
						"readPreferenceTags": "dc:ny",
					},
				},
				ConnectionContext: db.ConnectionContext{DatabaseName: "sampleDB"},
				Password:          "passwd",
			},
			wantSRV: true,
			want: omnidriver.Config{
				Host:     "cluster0.sample.mongodb.net",
				Database: "sampleDB",
				User:     "bytebase",
				Password: "passwd",
				Params: map[string]string{
					"readPreference":     "secondary",
					"readPreferenceTags": "dc:ny",
				},
			},
		},
		{
			name: "authentication database",
			connConfig: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:                   "cluster0.sample.mongodb.net",
					Username:               "bytebase",
					AuthenticationDatabase: "bytebase",
					Srv:                    true,
				},
				ConnectionContext: db.ConnectionContext{DatabaseName: "sampleDB"},
				Password:          "passwd",
			},
			wantSRV: true,
			want: omnidriver.Config{
				Host:     "cluster0.sample.mongodb.net",
				Database: "sampleDB",
				User:     "bytebase",
				Password: "passwd",
				Params:   map[string]string{"authSource": "bytebase"},
			},
		},
		{
			name: "additional addresses carry their own ports",
			connConfig: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:     "node1.cluster0.sample.mongodb.net",
					Port:     "27017",
					Username: "bytebase",
					AdditionalAddresses: []*storepb.DataSource_Address{
						{Host: "node2.cluster0.sample.mongodb.net", Port: "27017"},
						{Host: "node3.cluster0.sample.mongodb.net", Port: "27018"},
					},
					ReplicaSet:             "rs0",
					AuthenticationDatabase: "admin",
				},
				ConnectionContext: db.ConnectionContext{DatabaseName: "sampleDB"},
				Password:          "passwd",
			},
			want: omnidriver.Config{
				Host:     "node1.cluster0.sample.mongodb.net:27017,node2.cluster0.sample.mongodb.net:27017,node3.cluster0.sample.mongodb.net:27018",
				Port:     "27017",
				Database: "sampleDB",
				User:     "bytebase",
				Password: "passwd",
				Params:   map[string]string{"authSource": "admin", "replicaSet": "rs0"},
			},
		},
		{
			name: "TLS follows use_ssl",
			connConfig: db.ConnectionConfig{
				DataSource: &storepb.DataSource{Host: "localhost", Port: "27017", UseSsl: true},
			},
			want: omnidriver.Config{Host: "localhost:27017", Port: "27017"},
		},
		{
			name: "IPv6 host with port",
			connConfig: db.ConnectionConfig{
				DataSource: &storepb.DataSource{Host: "2001:db8::1", Port: "27017"},
			},
			want: omnidriver.Config{Host: "[2001:db8::1]:27017", Port: "27017"},
		},
		{
			name: "already-bracketed IPv6 host with port",
			connConfig: db.ConnectionConfig{
				DataSource: &storepb.DataSource{Host: "[2001:db8::1]", Port: "27017"},
			},
			want: omnidriver.Config{Host: "[2001:db8::1]:27017", Port: "27017"},
		},
		{
			name: "IPv6 host without port",
			connConfig: db.ConnectionConfig{
				DataSource: &storepb.DataSource{Host: "2001:db8::1"},
			},
			want: omnidriver.Config{Host: "[2001:db8::1]"},
		},
		{
			name: "multi-host IPv6 addresses",
			connConfig: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host: "2001:db8::1",
					Port: "27017",
					AdditionalAddresses: []*storepb.DataSource_Address{
						{Host: "2001:db8::2", Port: "27018"},
					},
				},
			},
			want: omnidriver.Config{Host: "[2001:db8::1]:27017,[2001:db8::2]:27018", Port: "27017"},
		},
	}

	a := require.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(*testing.T) {
			got, options, err := toDriverConfig(tt.connConfig)
			a.NoError(err)
			if tt.connConfig.DataSource.GetUseSsl() {
				a.NotNil(got.TLS)
				got.TLS = nil
			}
			a.Equal(tt.want, got)
			a.Equal(tt.wantSRV, options.SRV)
		})
	}

	_, _, err := toDriverConfig(db.ConnectionConfig{})
	a.ErrorContains(err, "a data source is required")
}

// TestExecuteLogsParseFailure pins BYT-9950: when the statement fails to
// split/parse, Execute must still emit a COMMAND_EXECUTE / COMMAND_RESPONSE
// pair carrying the parse error so the failure is visible in task run logs,
// and must report failure.
func TestExecuteLogsParseFailure(t *testing.T) {
	const statement = "db.users.find();\nthis is not mongosh"

	run := func(t *testing.T, logStatement bool) []*storepb.TaskRunLog {
		var logs []*storepb.TaskRunLog
		opts := db.ExecuteOptions{
			CreateTaskRunLog: func(_ time.Time, l *storepb.TaskRunLog) error {
				logs = append(logs, l)
				return nil
			},
			LogCommandStatement: logStatement,
		}

		// Open does not reach the server, and the parse fails before a
		// statement is sent, so this test needs none.
		opened, err := (&Driver{}).Open(context.Background(), storepb.Engine_MONGODB, db.ConnectionConfig{
			DataSource: &storepb.DataSource{Host: "localhost", Port: "27017"},
		})
		require.NoError(t, err)
		_, err = opened.Execute(context.Background(), statement, opts)
		require.Error(t, err)
		require.Contains(t, err.Error(), "syntax error")

		require.Len(t, logs, 2)
		require.Equal(t, storepb.TaskRunLog_COMMAND_EXECUTE, logs[0].Type)
		require.Equal(t, storepb.TaskRunLog_COMMAND_RESPONSE, logs[1].Type)
		require.Contains(t, logs[1].CommandResponse.Error, "syntax error")
		return logs
	}

	t.Run("range logging", func(t *testing.T) {
		logs := run(t, false)
		require.Empty(t, logs[0].CommandExecute.Statement)
		require.Equal(t, int32(0), logs[0].CommandExecute.Range.GetStart())
		require.Equal(t, int32(len(statement)), logs[0].CommandExecute.Range.GetEnd())
	})

	t.Run("statement logging", func(t *testing.T) {
		logs := run(t, true)
		require.Nil(t, logs[0].CommandExecute.Range)
		require.Equal(t, statement, logs[0].CommandExecute.GetStatement())
	})
}

type stubMongoDriver struct {
	session *stubMongoSession
}

func (*stubMongoDriver) Ping(context.Context) error { return nil }
func (*stubMongoDriver) Close() error               { return nil }
func (d *stubMongoDriver) Session(context.Context) (omnidriver.Session, error) {
	return d.session, nil
}

type stubMongoSession struct {
	gotStatement string
	gotOpts      omnidriver.QueryOptions
	results      []*omnidriver.Result
	closed       bool
}

func (s *stubMongoSession) Query(_ context.Context, statement string, opts omnidriver.QueryOptions) ([]*omnidriver.Result, error) {
	s.gotStatement = statement
	s.gotOpts = opts
	return s.results, nil
}

func (s *stubMongoSession) Close() error {
	s.closed = true
	return nil
}

// TestQueryConnAdapterSeam pins QueryConn's ephemeral session lifecycle, option
// forwarding, and conversion of omni Results (including typed QueryPlan).
func TestQueryConnAdapterSeam(t *testing.T) {
	sess := &stubMongoSession{
		results: []*omnidriver.Result{{
			Statement: `db.orders.find({total: {$gt: 500}}).explain("queryPlanner")`,
			Columns:   []omnidriver.Column{{Name: "result", TypeName: "TEXT"}},
			Rows:      [][]omnidriver.Value{{{Kind: omnidriver.String, String: `{"queryPlanner":{"winningPlan":{}}}`}}},
			Plan:      &omnidriver.Plan{Format: "JSON", Executed: false},
		}},
	}
	driver := &Driver{OmniAdapter: util.OmniAdapter{Driver: &stubMongoDriver{session: sess}}}

	results, err := driver.QueryConn(context.Background(), `db.orders.find({total: {$gt: 500}})`, db.QueryContext{
		Schema:               "app",
		Limit:                50,
		Explain:              true,
		MaximumSQLResultSize: 10 * 1024 * 1024,
		MaskingEnabled:       true,
	})
	require.NoError(t, err)
	require.True(t, sess.closed, "MongoDB QueryConn closes its ephemeral session")
	require.Equal(t, omnidriver.QueryOptions{
		Schema:         "app",
		Limit:          50,
		Explain:        true,
		MaxResultBytes: 10 * 1024 * 1024,
		MaskingEnabled: true,
	}, sess.gotOpts)
	require.Len(t, results, 1)
	require.Equal(t, []string{"result"}, results[0].GetColumnNames())
	require.Equal(t, []string{"TEXT"}, results[0].GetColumnTypeNames())
	require.Equal(t, v1pb.QueryOption_JSON, results[0].GetQueryPlan().GetFormat())
	require.False(t, results[0].GetQueryPlan().GetExecuted())
}

func TestToDriverConfigTLSVerifiesDialedHost(t *testing.T) {
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

	seedCert := signLeaf(2, "cluster.example.com")
	shardCert := signLeaf(3, "shard-01.cluster.example.com")

	got, _, err := toDriverConfig(db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:                 "cluster.example.com",
			Srv:                  true,
			UseSsl:               true,
			VerifyTlsCertificate: true,
			SslCa:                caPEM,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, got.TLS)

	tlsCfg, err := got.TLS("shard-01.cluster.example.com")
	require.NoError(t, err)
	require.Equal(t, "shard-01.cluster.example.com", tlsCfg.ServerName)
	require.NoError(t, tlsCfg.VerifyPeerCertificate([][]byte{shardCert}, nil))
	require.Error(t, tlsCfg.VerifyPeerCertificate([][]byte{seedCert}, nil))
}
