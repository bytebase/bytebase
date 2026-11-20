// Package cockroachdb is Bytebase's CockroachDB driver: an adapter over
// github.com/bytebase/omni/cockroachdb/driver that maps Bytebase's connection
// config onto the driver contract and the contract's results back into the
// product's. The conversation with the server — connection configuration,
// session setup, metadata sync, statement classification, execution, result
// decoding and the plan diagram — is the driver's.
//
// CockroachDB is a PostgreSQL fork in Bytebase and its own tree in omni, so the
// adapter follows pg's shape where the two overlap: Config carries libpq's host
// and port lists, a TLS func bound to each host the driver dials, and the SSH
// tunnel as Config.Dial; QueryOptions.Schema is the session's search_path. Two
// parts of the pg path are not copied: Config.Params stays unset, because this
// data source has no extra connection parameters, and there is no tenant-mode
// prelude, because tenant mode does not apply to CockroachDB and its Open takes
// no engine options.
//
// QueryConn runs on one session held for the driver's lifetime and serializes
// its calls, as the editor stream's one pooled connection did: what a statement
// sets there (search_path, a temporary table) holds for the stream's next call.
// The caller's *sql.Conn is unused.
//
// Execute maps the script's transaction directive and passes neither the
// isolation directive nor MaximumRetries: this engine ignored both, and honoring
// them would change what a script does.
package cockroachdb

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"

	omnicockroach "github.com/bytebase/omni/cockroachdb/driver"
	omnidriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	"go.uber.org/multierr"
	"golang.org/x/crypto/ssh"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

var (
	_ db.Driver               = (*Driver)(nil)
	_ db.RowEstimator         = (*Driver)(nil)
	_ db.TaskRunSessionReader = (*Driver)(nil)
)

func init() {
	db.Register(storepb.Engine_COCKROACHDB, newDriver)
}

// Driver is Bytebase's CockroachDB driver.
type Driver struct {
	inner         *omnicockroach.Driver
	connectionCtx db.ConnectionContext

	// sshClient is the tunnel the driver dials through, closed with it.
	sshClient *ssh.Client

	// sessionMu serializes QueryConn and guards session: the editor keeps one
	// driver for a stream, and one pooled connection served its statements one
	// at a time.
	sessionMu sync.Mutex
	session   omnidriver.Session
}

func newDriver() db.Driver {
	return &Driver{}
}

// Open maps the connection config onto the driver contract. It does not reach
// the server: Ping does.
func (d *Driver) Open(ctx context.Context, _ storepb.Engine, config db.ConnectionConfig) (db.Driver, error) {
	cfg, err := cockroachConfig(config)
	if err != nil {
		return nil, err
	}

	if config.DataSource.GetSshHost() != "" {
		sshClient, err := util.GetSSHClient(config.DataSource)
		if err != nil {
			return nil, err
		}
		d.sshClient = sshClient
		cfg.Dial = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return util.DialSSHTunnel(ctx, sshClient, network, addr)
		}
	}

	inner, err := omnicockroach.Open(ctx, cfg)
	if err != nil {
		if d.sshClient != nil {
			_ = d.sshClient.Close()
			d.sshClient = nil
		}
		return nil, err
	}
	d.inner = inner
	d.connectionCtx = config.ConnectionContext
	return d, nil
}

// cockroachConfig resolves what Bytebase knows about reaching a server into the
// plain values the contract's CockroachDB driver takes.
func cockroachConfig(config db.ConnectionConfig) (omnidriver.Config, error) {
	dataSource := config.DataSource
	// The old driver refused half a client certificate whether or not TLS was
	// asked for, so the mistake is reported here rather than left to a
	// connection that never presents it.
	if (dataSource.GetSslCert() == "") != (dataSource.GetSslKey() == "") {
		return omnidriver.Config{}, errors.New("ssl-cert and ssl-key must be both set or unset")
	}
	database := config.ConnectionContext.DatabaseName
	if database == "" {
		database = dataSource.GetDatabase()
	}
	cfg := omnidriver.Config{
		Host:            dataSource.GetHost(),
		Port:            dataSource.GetPort(),
		Database:        database,
		User:            dataSource.GetUsername(),
		Password:        config.Password,
		ApplicationName: applicationName(config.ConnectionContext),
		ReadOnly:        config.ConnectionContext.ReadOnly,
		EngineVersion:   config.ConnectionContext.EngineVersion,
	}
	if dataSource.GetUseSsl() {
		// util.GetTLSConfig binds certificate verification to the data source's
		// host; ApplyPGTLSConfig rebinds it to the host of whichever server the
		// driver dials, so a multi-host list verifies each host's certificate
		// against its own name, as the old driver did.
		cfg.TLS = func(host string) (*tls.Config, error) {
			tlsConfig, err := util.GetTLSConfig(dataSource)
			if err != nil {
				return nil, err
			}
			if tlsConfig != nil {
				util.ApplyPGTLSConfig(tlsConfig, host, nil)
			}
			return tlsConfig, nil
		}
	}
	return cfg, nil
}

// applicationName identifies the connection to the server. A task run names
// itself so the server's activity views can find it.
func applicationName(connectionContext db.ConnectionContext) string {
	if connectionContext.TaskRunUID != nil {
		return fmt.Sprintf("bytebase-taskrun-%d", *connectionContext.TaskRunUID)
	}
	return "bytebase"
}

func (d *Driver) Close(context.Context) error {
	d.sessionMu.Lock()
	session := d.session
	d.session = nil
	d.sessionMu.Unlock()

	var err error
	if session != nil {
		err = multierr.Append(err, session.Close())
	}
	if d.inner != nil {
		err = multierr.Append(err, d.inner.Close())
	}
	if d.sshClient != nil {
		err = multierr.Append(err, d.sshClient.Close())
		d.sshClient = nil
	}
	return err
}

// Ping reaches the server.
func (d *Driver) Ping(ctx context.Context) error {
	return d.inner.Ping(ctx)
}

// TaskRunSession reads the running session by application name from
// CockroachDB's own catalog. CockroachDB's pg_stat_activity compatibility view
// answers no rows and it has no pg_blocking_pids, so neither PostgreSQL's
// query nor its blocking relationships apply.
func (d *Driver) TaskRunSession(ctx context.Context, applicationName string) (*v1pb.TaskRunSession, error) {
	if d.inner == nil {
		return nil, errors.New("cockroachdb: driver is closed")
	}
	session, err := d.inner.Session(ctx)
	if err != nil {
		return nil, err
	}
	defer session.Close()

	query := fmt.Sprintf(`
		SELECT
			q.session_id,
			q.query,
			q.phase,
			q.database,
			q.user_name,
			q.application_name,
			q.client_address,
			s.session_start,
			q.start
		FROM crdb_internal.cluster_queries AS q
		LEFT JOIN crdb_internal.cluster_sessions AS s
			ON s.session_id = q.session_id
		WHERE q.application_name = '%s'
		ORDER BY q.session_id
	`, strings.ReplaceAll(applicationName, "'", "''"))

	results, err := session.Query(ctx, query, omnidriver.QueryOptions{})
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return &v1pb.TaskRunSession{
			Session: &v1pb.TaskRunSession_Postgres_{
				Postgres: &v1pb.TaskRunSession_Postgres{},
			},
		}, nil
	}
	if results[0].Err != nil {
		return nil, results[0].Err
	}
	return taskRunSessionFromResult(results[0], applicationName), nil
}

func taskRunSessionFromResult(result *omnidriver.Result, applicationName string) *v1pb.TaskRunSession {
	ss := &v1pb.TaskRunSession_Postgres{}
	for _, row := range result.Rows {
		if len(row) < 9 {
			continue
		}
		s := v1pb.TaskRunSession_Postgres_Session{
			Pid:             valueString(row[0]),
			Query:           valueString(row[1]),
			State:           optionalValueString(row[2]),
			Datname:         optionalValueString(row[3]),
			Usename:         optionalValueString(row[4]),
			ApplicationName: valueString(row[5]),
			ClientAddr:      optionalValueString(row[6]),
			BackendStart:    optionalValueTimestamp(row[7]),
			QueryStart:      optionalValueTimestamp(row[8]),
		}
		if s.ApplicationName == applicationName {
			ss.Session = &s
		}
	}
	return &v1pb.TaskRunSession{
		Session: &v1pb.TaskRunSession_Postgres_{
			Postgres: ss,
		},
	}
}

func valueString(v omnidriver.Value) string {
	if v.Kind == omnidriver.Null {
		return ""
	}
	return v.String
}

func optionalValueString(v omnidriver.Value) *string {
	if v.Kind == omnidriver.Null {
		return nil
	}
	s := v.String
	return &s
}

func optionalValueTimestamp(v omnidriver.Value) *timestamppb.Timestamp {
	if v.Kind != omnidriver.Timestamp && v.Kind != omnidriver.TimestampTZ {
		return nil
	}
	return timestamppb.New(v.Time)
}

// SyncInstance reads the server version, its user databases and its roles.
func (d *Driver) SyncInstance(ctx context.Context) (*db.InstanceMetadata, error) {
	instance, err := d.inner.SyncInstance(ctx)
	if err != nil {
		return nil, err
	}
	roles := make([]*storepb.InstanceRole, 0, len(instance.Roles))
	for _, role := range instance.Roles {
		roles = append(roles, &storepb.InstanceRole{Name: role.Name, Attribute: role.Attribute})
	}
	return &db.InstanceMetadata{
		Version:   instance.Version,
		Databases: instance.Databases,
		Metadata:  &storepb.Instance{Roles: roles},
	}, nil
}

// SyncDBSchema reads the schema of the database the driver was opened on. A
// database the server no longer has is reported as not found.
func (d *Driver) SyncDBSchema(ctx context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	metadata, err := d.inner.SyncDatabase(ctx)
	if err != nil {
		if errors.Is(err, omnidriver.ErrNotFound) {
			return nil, common.Errorf(common.NotFound, "database %q not found", d.connectionCtx.DatabaseName)
		}
		return nil, err
	}
	return metadata, nil
}

// Dump writes the server's own DDL for the database. The synced metadata is not
// rendered and does not narrow what the server states.
func (d *Driver) Dump(ctx context.Context, out io.Writer, dbMetadata *metadatapb.DatabaseSchemaMetadata) error {
	return d.inner.Dump(ctx, out, dbMetadata)
}

// CountAffectedRows returns the planner's estimate of the rows a statement
// modifies. plancheck reads it through db.RowEstimator.
func (d *Driver) CountAffectedRows(ctx context.Context, statement string) (int64, error) {
	return d.inner.CountAffectedRows(ctx, statement)
}
