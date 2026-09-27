package saas

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/component/sample"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

type allocation struct {
	database string
	role     string
	password string
}

type instanceConfig struct {
	adminDataSource   *storepb.DataSource
	syncDatabaseNames []string
}

type postgresTarget struct {
	config        *pgx.ConnConfig
	useSSL        bool
	verifyTLS     bool
	sslCA         string
	provisionHook func(provisionStage) error
}

type provisionStage string

const (
	provisionStageRoleCreated     provisionStage = "role-created"
	provisionStageDatabaseCreated provisionStage = "database-created"
	provisionStageCleanupDatabase provisionStage = "cleanup-database"
	provisionStageCleanupRole     provisionStage = "cleanup-role"
)

type staticTargetFailure struct {
	err error
}

func (e *staticTargetFailure) Error() string { return e.err.Error() }
func (e *staticTargetFailure) Unwrap() error { return e.err }

func newPostgresTarget(targetURL string) (*postgresTarget, error) {
	u, err := url.Parse(targetURL)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") {
		return nil, staticTargetError("sample project instance target must be a PostgreSQL URL")
	}
	if u.Hostname() == "" || u.Port() == "" || u.User == nil || u.User.Username() == "" || strings.TrimPrefix(u.EscapedPath(), "/") == "" {
		return nil, staticTargetError("sample project instance target URL requires host, port, user, and database")
	}
	port, err := strconv.ParseUint(u.Port(), 10, 16)
	if err != nil || port == 0 {
		return nil, staticTargetError("sample project instance target URL requires a valid port")
	}
	password, hasPassword := u.User.Password()
	if !hasPassword || password == "" {
		return nil, staticTargetError("sample project instance target URL requires password authentication")
	}

	query := u.Query()
	if len(query["sslmode"]) > 1 {
		return nil, staticTargetError("sample project instance target URL requires at most one sslmode")
	}
	sslMode := query.Get("sslmode")
	switch sslMode {
	case "":
		sslMode = "disable"
		query.Set("sslmode", sslMode)
		u.RawQuery = query.Encode()
		targetURL = u.String()
	case "disable", "require", "verify-full":
	default:
		return nil, staticTargetError("sample project instance target URL has unsupported sslmode")
	}
	for key := range query {
		if key != "sslmode" && key != "sslrootcert" {
			return nil, staticTargetError("sample project instance target URL only supports sslmode and sslrootcert")
		}
	}
	if roots, ok := query["sslrootcert"]; ok && (len(roots) != 1 || roots[0] == "") {
		return nil, staticTargetError("sample project instance target URL requires a single non-empty sslrootcert")
	}

	config, err := pgx.ParseConfig(targetURL)
	if err != nil {
		return nil, staticTargetError("invalid sample project instance target URL")
	}
	if len(config.Fallbacks) != 0 {
		return nil, staticTargetError("sample project instance target URL must use one host")
	}
	var sslCA string
	if path := query.Get("sslrootcert"); path != "" {
		resolved, err := util.ResolveTLSMaterial(&storepb.DataSource{UseSsl: true, SslCaPath: path})
		if err != nil {
			return nil, staticTargetError("invalid sample project instance target sslrootcert")
		}
		sslCA = resolved.GetSslCa()
	}
	return newPostgresTargetFromConfig(config, sslMode == "verify-full", sslCA), nil
}

func newPostgresTargetFromConfig(config *pgx.ConnConfig, verifyTLS bool, sslCA string) *postgresTarget {
	return &postgresTarget{
		config:    config,
		useSSL:    config.TLSConfig != nil,
		verifyTLS: verifyTLS,
		sslCA:     sslCA,
	}
}

func (t *postgresTarget) instanceConfig(allocation allocation) (*instanceConfig, error) {
	if err := validateProvisionAllocation(allocation); err != nil {
		return nil, err
	}
	return &instanceConfig{
		adminDataSource: &storepb.DataSource{
			Id:                   "admin",
			Type:                 storepb.DataSourceType_ADMIN,
			Host:                 t.config.Host,
			Port:                 fmt.Sprint(t.config.Port),
			Database:             allocation.database,
			Username:             allocation.role,
			Password:             allocation.password,
			UseSsl:               t.useSSL,
			VerifyTlsCertificate: t.verifyTLS,
			SslCa:                t.sslCA,
		},
		syncDatabaseNames: []string{allocation.database},
	}, nil
}

func (t *postgresTarget) validate(ctx context.Context) error {
	return t.validateCapabilities(ctx, true)
}

func (t *postgresTarget) validateForCleanup(ctx context.Context) error {
	return t.validateCapabilities(ctx, false)
}

func (t *postgresTarget) validateCapabilities(ctx context.Context, requireCreateDB bool) error {
	conn, err := t.connect(ctx, "", "", "")
	if err != nil {
		return errors.New("failed to connect to sample project instance target")
	}
	defer conn.Close(ctx)

	var canCreateDB, canCreateRole, canSignal bool
	if err := conn.QueryRow(ctx, `
		SELECT rolcreatedb, rolcreaterole, pg_has_role(current_user, 'pg_signal_backend', 'USAGE')
		FROM pg_roles
		WHERE rolname = current_user
	`).Scan(&canCreateDB, &canCreateRole, &canSignal); err != nil {
		return errors.New("failed to inspect sample project instance target role")
	}
	if requireCreateDB && (!canCreateDB || !canCreateRole || !canSignal) {
		return staticTargetError("sample project instance target role requires CREATEDB, CREATEROLE, and pg_signal_backend")
	}
	if !requireCreateDB && (!canCreateRole || !canSignal) {
		return staticTargetError("sample project instance target role requires CREATEROLE and pg_signal_backend for cleanup")
	}
	return nil
}

func (t *postgresTarget) provision(ctx context.Context, allocation allocation) error {
	if err := validateProvisionAllocation(allocation); err != nil {
		return err
	}
	admin, err := t.connect(ctx, "", "", "")
	if err != nil {
		return errors.New("failed to connect to sample project instance target")
	}
	defer admin.Close(ctx)

	if _, err := admin.Exec(ctx, fmt.Sprintf(
		"CREATE ROLE %s LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD %s",
		quoteIdentifier(allocation.role), quoteLiteral(allocation.password),
	)); err != nil {
		return errors.New("failed to create sample project instance role")
	}
	if err := t.runProvisionHook(provisionStageRoleCreated); err != nil {
		return errors.New("failed after creating sample project instance role")
	}
	if _, err := admin.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s WITH ALLOW_CONNECTIONS false", quoteIdentifier(allocation.database))); err != nil {
		return errors.New("failed to create sample project instance database")
	}
	if err := t.runProvisionHook(provisionStageDatabaseCreated); err != nil {
		return errors.New("failed after creating sample project instance database")
	}
	if _, err := admin.Exec(ctx, fmt.Sprintf("REVOKE ALL PRIVILEGES ON DATABASE %s FROM PUBLIC", quoteIdentifier(allocation.database))); err != nil {
		return errors.New("failed to revoke PUBLIC database privileges")
	}
	if _, err := admin.Exec(ctx, fmt.Sprintf(
		"GRANT CONNECT, CREATE, TEMPORARY ON DATABASE %s TO %s",
		quoteIdentifier(allocation.database), quoteIdentifier(allocation.role),
	)); err != nil {
		return errors.New("failed to grant sample project instance database privileges")
	}
	if _, err := admin.Exec(ctx, fmt.Sprintf("ALTER DATABASE %s ALLOW_CONNECTIONS true", quoteIdentifier(allocation.database))); err != nil {
		return errors.New("failed to enable sample project instance database connections")
	}

	sampleAdmin, err := t.connect(ctx, allocation.database, "", "")
	if err != nil {
		return errors.New("failed to connect to sample project instance database")
	}
	defer sampleAdmin.Close(ctx)
	if _, err := sampleAdmin.Exec(ctx, "REVOKE CREATE ON SCHEMA public FROM PUBLIC"); err != nil {
		return errors.New("failed to revoke PUBLIC schema creation")
	}
	if _, err := sampleAdmin.Exec(ctx, fmt.Sprintf("GRANT USAGE, CREATE ON SCHEMA public TO %s", quoteIdentifier(allocation.role))); err != nil {
		return errors.New("failed to grant sample project instance schema privileges")
	}

	sampleConn, err := t.connect(ctx, allocation.database, allocation.role, allocation.password)
	if err != nil {
		return errors.New("failed to connect as sample project instance role")
	}
	defer sampleConn.Close(ctx)
	seed, err := sample.LoadSeedData()
	if err != nil {
		return errors.New("failed to load sample project instance seed data")
	}
	tx, err := sampleConn.Begin(ctx)
	if err != nil {
		return errors.New("failed to begin sample project instance seed transaction")
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, seed, pgx.QueryExecModeSimpleProtocol); err != nil {
		return errors.New("failed to seed sample project instance database")
	}
	if err := tx.Commit(ctx); err != nil {
		return errors.New("failed to commit sample project instance seed transaction")
	}
	return nil
}

func (t *postgresTarget) runProvisionHook(stage provisionStage) error {
	if t.provisionHook == nil {
		return nil
	}
	return t.provisionHook(stage)
}

func (t *postgresTarget) remove(ctx context.Context, allocation allocation) error {
	if err := validateCleanupAllocation(allocation); err != nil {
		return err
	}
	if allocation.database != "" && t.config.Database == allocation.database {
		return staticTargetError("sample project instance target cannot remove its configured database")
	}
	admin, err := t.connect(ctx, "", "", "")
	if err != nil {
		return errors.New("failed to connect to sample project instance target")
	}
	defer admin.Close(ctx)

	roleExists, err := roleExists(ctx, admin, allocation.role)
	if err != nil {
		return errors.New("failed to inspect sample project instance role")
	}
	if roleExists {
		if _, err := admin.Exec(ctx, fmt.Sprintf("ALTER ROLE %s NOLOGIN", quoteIdentifier(allocation.role))); err != nil {
			return errors.New("failed to disable sample project instance role")
		}
	}
	databaseExists, err := databaseExists(ctx, admin, allocation.database)
	if err != nil {
		return errors.New("failed to inspect sample project instance database")
	}
	if databaseExists && roleExists {
		if _, err := admin.Exec(ctx, fmt.Sprintf("REVOKE ALL PRIVILEGES ON DATABASE %s FROM %s", quoteIdentifier(allocation.database), quoteIdentifier(allocation.role))); err != nil {
			return errors.New("failed to revoke sample project instance database privileges")
		}
	}
	if err := terminateAndDrain(ctx, admin, allocation); err != nil {
		return errors.New("failed to terminate sample project instance sessions")
	}
	if databaseExists {
		if err := t.runProvisionHook(provisionStageCleanupDatabase); err != nil {
			return errors.New("failed while removing sample project instance database")
		}
		if _, err := admin.Exec(ctx, fmt.Sprintf("DROP DATABASE %s", quoteIdentifier(allocation.database))); err != nil {
			return errors.New("failed to drop sample project instance database")
		}
	}
	if roleExists {
		if err := t.runProvisionHook(provisionStageCleanupRole); err != nil {
			return errors.New("failed while removing sample project instance role")
		}
		if _, err := admin.Exec(ctx, fmt.Sprintf("DROP ROLE %s", quoteIdentifier(allocation.role))); err != nil {
			return errors.New("failed to drop sample project instance role")
		}
	}
	return nil
}

func (t *postgresTarget) connect(ctx context.Context, database, user, password string) (*pgx.Conn, error) {
	config := t.config.Copy()
	if database != "" {
		config.Database = database
	}
	if user != "" {
		config.User = user
		config.Password = password
	}
	return pgx.ConnectConfig(ctx, config)
}

func validateProvisionAllocation(value allocation) error {
	if value.database == "" || value.role == "" || strings.ContainsRune(value.database, 0) || strings.ContainsRune(value.role, 0) {
		return staticTargetError("sample project instance provisioning requires database and role")
	}
	if value.password == "" || strings.ContainsRune(value.password, 0) {
		return staticTargetError("sample project instance provisioning requires password")
	}
	return nil
}

func validateCleanupAllocation(value allocation) error {
	if (value.database == "" && value.role == "") || strings.ContainsRune(value.database, 0) || strings.ContainsRune(value.role, 0) {
		return staticTargetError("sample project instance cleanup requires a database or role")
	}
	return nil
}

func quoteIdentifier(value string) string { return pgx.Identifier{value}.Sanitize() }

func quoteLiteral(value string) string {
	return "E'" + strings.NewReplacer(`\`, `\\`, `'`, `\'`).Replace(value) + "'"
}

func staticTargetError(message string) error {
	return &staticTargetFailure{err: errors.New(message)}
}

func isStaticTargetError(err error) bool {
	var failure *staticTargetFailure
	return errors.As(err, &failure)
}

func roleExists(ctx context.Context, conn *pgx.Conn, role string) (bool, error) {
	var exists bool
	err := conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)", role).Scan(&exists)
	return exists, err
}

func databaseExists(ctx context.Context, conn *pgx.Conn, database string) (bool, error) {
	var exists bool
	err := conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", database).Scan(&exists)
	return exists, err
}

func terminateAndDrain(ctx context.Context, conn *pgx.Conn, value allocation) error {
	for {
		rows, err := conn.Query(ctx, `
			SELECT pid FROM pg_stat_activity
			WHERE (usename = $1 OR datname = $2) AND pid <> pg_backend_pid()
		`, value.role, value.database)
		if err != nil {
			return errors.New("failed to list sample project instance sessions")
		}
		var pids []int32
		for rows.Next() {
			var pid int32
			if err := rows.Scan(&pid); err != nil {
				rows.Close()
				return errors.New("failed to read sample project instance session")
			}
			pids = append(pids, pid)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return errors.New("failed to list sample project instance sessions")
		}
		rows.Close()
		if len(pids) == 0 {
			return nil
		}
		for _, pid := range pids {
			if _, err := conn.Exec(ctx, "SELECT pg_terminate_backend($1)", pid); err != nil {
				return errors.New("failed to terminate sample project instance session")
			}
		}
		timer := time.NewTimer(250 * time.Millisecond)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return errors.New("timed out waiting for sample project instance sessions to drain")
		case <-timer.C:
		}
	}
}
