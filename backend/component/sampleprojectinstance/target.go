// Package sampleprojectinstance manages Cloud PostgreSQL targets for sample
// project instances.
package sampleprojectinstance

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/resources/postgres"
)

// Allocation identifies the physical PostgreSQL resources for one sample
// project instance.
type Allocation struct {
	Database string
	Role     string
	Password string
}

// InstanceConfig describes the registered instance connection and the exact
// database that must be discovered before preparation succeeds.
type InstanceConfig struct {
	AdminDataSource   *storepb.DataSource
	SyncDatabaseNames []string
}

// Target manages direct PostgreSQL operations for sample project instances.
type Target struct {
	config        *pgx.ConnConfig
	verifyTLS     bool
	sslCA         string
	provisionHook func(provisionStage) error
}

type provisionStage string

const (
	provisionStageRoleCreated         provisionStage = "role-created"
	provisionStageDatabaseCreated     provisionStage = "database-created"
	provisionStagePublicAccessRevoked provisionStage = "public-access-revoked"
	provisionStageRoleAccessGranted   provisionStage = "role-access-granted"
	provisionStageConnectionsEnabled  provisionStage = "connections-enabled"
	provisionStagePublicSchemaRevoked provisionStage = "public-schema-revoked"
	provisionStageRoleSchemaGranted   provisionStage = "role-schema-granted"
	provisionStageSeeded              provisionStage = "seeded"
	provisionStageCleanupDatabase     provisionStage = "cleanup-database"
	provisionStageCleanupRole         provisionStage = "cleanup-role"
)

type targetFailureKind string

const (
	targetFailureStatic      targetFailureKind = "static"
	targetFailureUnavailable targetFailureKind = "unavailable"
	targetFailureInvariant   targetFailureKind = "invariant"
)

type targetFailure struct {
	kind targetFailureKind
	err  error
}

type provisionOwnership struct {
	databaseCreated bool
	roleCreated     bool
}

type provisionFailure struct {
	err       error
	ownership provisionOwnership
}

func (e *provisionFailure) Error() string {
	return e.err.Error()
}

func (e *provisionFailure) Unwrap() error {
	return e.err
}

func provisionOwnershipOf(err error) (provisionOwnership, bool) {
	var failure *provisionFailure
	if errors.As(err, &failure) {
		return failure.ownership, true
	}
	return provisionOwnership{}, false
}

func (e *targetFailure) Error() string {
	if e.err == nil {
		return string(e.kind)
	}
	return e.err.Error()
}

func (e *targetFailure) Unwrap() error {
	return e.err
}

func newTargetFailure(kind targetFailureKind, err error) error {
	return &targetFailure{kind: kind, err: err}
}

func targetFailureKindOf(err error) targetFailureKind {
	var failure *targetFailure
	if errors.As(err, &failure) {
		return failure.kind
	}
	return targetFailureUnavailable
}

// NewTarget parses a direct PostgreSQL target URL. It intentionally accepts
// only URL-form password authentication so the configured target cannot use
// passfiles, service files, or credential rotation indirection.
func NewTarget(targetURL string) (*Target, error) {
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
	if len(query["sslmode"]) != 1 || query.Get("sslmode") != "verify-full" {
		return nil, staticTargetError("sample project instance target URL requires sslmode=verify-full")
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
		resolved, err := util.ResolveTLSMaterial(&storepb.DataSource{
			UseSsl:    true,
			SslCaPath: path,
		})
		if err != nil {
			return nil, staticTargetError("invalid sample project instance target sslrootcert")
		}
		sslCA = resolved.GetSslCa()
	}
	return newTargetFromConfig(config, true, sslCA), nil
}

// InstanceConfig builds the persisted admin datasource and its exact discovery
// filter for an allocation.
func (t *Target) InstanceConfig(allocation Allocation) (*InstanceConfig, error) {
	if err := validateProvisionAllocation(allocation); err != nil {
		return nil, err
	}
	return &InstanceConfig{
		AdminDataSource: &storepb.DataSource{
			Id:                   "admin",
			Type:                 storepb.DataSourceType_ADMIN,
			Host:                 t.config.Host,
			Port:                 fmt.Sprint(t.config.Port),
			Database:             allocation.Database,
			Username:             allocation.Role,
			Password:             allocation.Password,
			UseSsl:               true,
			VerifyTlsCertificate: t.verifyTLS,
			SslCa:                t.sslCA,
		},
		SyncDatabaseNames: []string{allocation.Database},
	}, nil
}

func newTargetFromConfig(config *pgx.ConnConfig, verifyTLS bool, sslCA string) *Target {
	return &Target{
		config:    config,
		verifyTLS: verifyTLS,
		sslCA:     sslCA,
	}
}

// Validate verifies the target is reachable and has the static isolation
// baseline required before allocating shared sample databases.
func (t *Target) Validate(ctx context.Context) error {
	if err := t.validateCapabilities(ctx, true); err != nil {
		return err
	}
	return t.validateIsolationBaseline(ctx)
}

// ValidateForCleanup verifies the connectivity and target capabilities needed
// to remove existing allocations. Unlike Validate, it does not require the
// provisioning isolation baseline because a partially provisioned allocation
// may need cleanup before that baseline can be restored.
func (t *Target) ValidateForCleanup(ctx context.Context) error {
	return t.validateCapabilities(ctx, false)
}

func (t *Target) validateCapabilities(ctx context.Context, requireCreateDB bool) error {
	conn, err := t.connect(ctx, "", "", "")
	if err != nil {
		return unavailableTargetError("failed to connect to sample project instance target")
	}
	defer conn.Close(ctx)

	var canCreateDB, canCreateRole, canSignal bool
	if err := conn.QueryRow(ctx, `
		SELECT rolcreatedb, rolcreaterole, pg_has_role(current_user, 'pg_signal_backend', 'USAGE')
		FROM pg_roles
		WHERE rolname = current_user
	`).Scan(&canCreateDB, &canCreateRole, &canSignal); err != nil {
		return unavailableTargetError("failed to inspect sample project instance target role")
	}
	if requireCreateDB && (!canCreateDB || !canCreateRole || !canSignal) {
		return staticTargetError("sample project instance target role requires CREATEDB, CREATEROLE, and pg_signal_backend")
	}
	if !requireCreateDB && (!canCreateRole || !canSignal) {
		return staticTargetError("sample project instance target role requires CREATEROLE and pg_signal_backend for cleanup")
	}
	return nil
}

func (t *Target) validateIsolationBaseline(ctx context.Context) error {
	conn, err := t.connect(ctx, "", "", "")
	if err != nil {
		return unavailableTargetError("failed to connect to sample project instance target")
	}
	defer conn.Close(ctx)

	rows, err := conn.Query(ctx, `
		SELECT
			datname,
			datallowconn,
			EXISTS (
				SELECT 1
				FROM aclexplode(COALESCE(datacl, acldefault('d', datdba))) AS privilege
				WHERE privilege.grantee = 0
					AND privilege.privilege_type IN ('CONNECT', 'TEMPORARY')
			)
		FROM pg_database
	`)
	if err != nil {
		return unavailableTargetError("failed to inspect sample project instance target database privileges")
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var allowConnections, publicAccess bool
		if err := rows.Scan(&name, &allowConnections, &publicAccess); err != nil {
			return unavailableTargetError("failed to read sample project instance target database privileges")
		}
		switch name {
		case "postgres", "template1":
			if publicAccess {
				return staticTargetError("sample project instance target requires postgres and template1 to deny PUBLIC access")
			}
		case "template0":
			if allowConnections {
				return staticTargetError("sample project instance target requires template0 to deny connections")
			}
		case "cloudsqladmin":
			// Cloud SQL reserves this exact database for its administration
			// service. It is not a tenant-accessible sample database.
		default:
			if allowConnections && publicAccess {
				return staticTargetError("sample project instance target has a connectable database with PUBLIC access")
			}
		}
	}
	if err := rows.Err(); err != nil {
		return unavailableTargetError("failed to inspect sample project instance target database privileges")
	}
	return nil
}

// Provision creates a sample role and database, applies the isolation policy,
// and seeds the employee data as the sample role.
func (t *Target) Provision(ctx context.Context, allocation Allocation) (retErr error) {
	if err := validateProvisionAllocation(allocation); err != nil {
		return err
	}

	admin, err := t.connect(ctx, "", "", "")
	if err != nil {
		return errors.New("failed to connect to sample project instance target")
	}
	defer admin.Close(ctx)

	var roleCreated, databaseCreated bool
	defer func() {
		if retErr == nil {
			return
		}
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), compensationDeadline)
		defer cancel()
		remaining, err := t.cleanupProvisionAttempt(cleanupCtx, admin, allocation, provisionOwnership{
			databaseCreated: databaseCreated,
			roleCreated:     roleCreated,
		})
		if err != nil {
			retErr = &provisionFailure{
				err:       unavailableTargetError("failed to clean up sample project instance provisioning attempt"),
				ownership: remaining,
			}
		}
	}()

	if _, err := admin.Exec(ctx, fmt.Sprintf(
		"CREATE ROLE %s LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD %s",
		quoteIdentifier(allocation.Role),
		quoteLiteral(allocation.Password),
	)); err != nil {
		return classifyProvisionError(err, "failed to create sample project instance role")
	}
	roleCreated = true
	if err := t.runProvisionHook(provisionStageRoleCreated); err != nil {
		return err
	}
	if _, err := admin.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s WITH ALLOW_CONNECTIONS false", quoteIdentifier(allocation.Database))); err != nil {
		return classifyProvisionError(err, "failed to create sample project instance database")
	}
	databaseCreated = true
	if err := t.runProvisionHook(provisionStageDatabaseCreated); err != nil {
		return err
	}
	if _, err := admin.Exec(ctx, fmt.Sprintf("REVOKE ALL PRIVILEGES ON DATABASE %s FROM PUBLIC", quoteIdentifier(allocation.Database))); err != nil {
		return errors.New("failed to revoke PUBLIC database privileges")
	}
	if err := t.runProvisionHook(provisionStagePublicAccessRevoked); err != nil {
		return err
	}
	if _, err := admin.Exec(ctx, fmt.Sprintf(
		"GRANT CONNECT, CREATE, TEMPORARY ON DATABASE %s TO %s",
		quoteIdentifier(allocation.Database),
		quoteIdentifier(allocation.Role),
	)); err != nil {
		return errors.New("failed to grant sample project instance database privileges")
	}
	if err := t.runProvisionHook(provisionStageRoleAccessGranted); err != nil {
		return err
	}
	if _, err := admin.Exec(ctx, fmt.Sprintf("ALTER DATABASE %s ALLOW_CONNECTIONS true", quoteIdentifier(allocation.Database))); err != nil {
		return errors.New("failed to enable sample project instance database connections")
	}
	if err := t.runProvisionHook(provisionStageConnectionsEnabled); err != nil {
		return err
	}

	sampleAdmin, err := t.connect(ctx, allocation.Database, "", "")
	if err != nil {
		return errors.New("failed to connect to sample project instance database")
	}
	defer sampleAdmin.Close(ctx)
	if _, err := sampleAdmin.Exec(ctx, "REVOKE CREATE ON SCHEMA public FROM PUBLIC"); err != nil {
		return errors.New("failed to revoke PUBLIC schema creation")
	}
	if err := t.runProvisionHook(provisionStagePublicSchemaRevoked); err != nil {
		return err
	}
	if _, err := sampleAdmin.Exec(ctx, fmt.Sprintf("GRANT USAGE, CREATE ON SCHEMA public TO %s", quoteIdentifier(allocation.Role))); err != nil {
		return errors.New("failed to grant sample project instance schema privileges")
	}
	if err := t.runProvisionHook(provisionStageRoleSchemaGranted); err != nil {
		return err
	}

	sample, err := t.connect(ctx, allocation.Database, allocation.Role, allocation.Password)
	if err != nil {
		return errors.New("failed to connect as sample project instance role")
	}
	defer sample.Close(ctx)

	seed, err := postgres.LoadSampleData()
	if err != nil {
		return errors.New("failed to load sample project instance seed data")
	}
	tx, err := sample.Begin(ctx)
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
	return t.runProvisionHook(provisionStageSeeded)
}

func (t *Target) runProvisionHook(stage provisionStage) error {
	if t.provisionHook == nil {
		return nil
	}
	return t.provisionHook(stage)
}

func (t *Target) cleanupProvisionAttempt(
	ctx context.Context,
	admin *pgx.Conn,
	allocation Allocation,
	ownership provisionOwnership,
) (provisionOwnership, error) {
	if ownership.databaseCreated {
		if err := t.runProvisionHook(provisionStageCleanupDatabase); err != nil {
			return ownership, err
		}
		if _, err := admin.Exec(ctx, fmt.Sprintf("DROP DATABASE %s WITH (FORCE)", quoteIdentifier(allocation.Database))); err != nil {
			return ownership, err
		}
		ownership.databaseCreated = false
	}
	if ownership.roleCreated {
		if err := t.runProvisionHook(provisionStageCleanupRole); err != nil {
			return ownership, err
		}
		if _, err := admin.Exec(ctx, fmt.Sprintf("DROP ROLE %s", quoteIdentifier(allocation.Role))); err != nil {
			return ownership, err
		}
		ownership.roleCreated = false
	}
	return ownership, nil
}

// Remove revokes access, terminates every role or database session, and drops
// the allocation's database before its role. Missing resources are successful.
func (t *Target) Remove(ctx context.Context, allocation Allocation) error {
	if err := validateCleanupAllocation(allocation); err != nil {
		return err
	}
	if allocation.Database != "" && t.config.Database == allocation.Database {
		return errors.New("sample project instance target cannot remove its configured database")
	}

	admin, err := t.connect(ctx, "", "", "")
	if err != nil {
		return errors.New("failed to connect to sample project instance target")
	}
	defer admin.Close(ctx)

	roleExists, err := roleExists(ctx, admin, allocation.Role)
	if err != nil {
		return errors.New("failed to inspect sample project instance role")
	}
	if roleExists {
		if _, err := admin.Exec(ctx, fmt.Sprintf("ALTER ROLE %s NOLOGIN", quoteIdentifier(allocation.Role))); err != nil {
			return errors.New("failed to disable sample project instance role")
		}
	}

	databaseExists, err := databaseExists(ctx, admin, allocation.Database)
	if err != nil {
		return errors.New("failed to inspect sample project instance database")
	}
	if databaseExists && roleExists {
		if _, err := admin.Exec(ctx, fmt.Sprintf("REVOKE ALL PRIVILEGES ON DATABASE %s FROM %s", quoteIdentifier(allocation.Database), quoteIdentifier(allocation.Role))); err != nil {
			return errors.New("failed to revoke sample project instance database privileges")
		}
	}
	if err := terminateAndDrain(ctx, admin, allocation); err != nil {
		return err
	}
	if databaseExists {
		if _, err := admin.Exec(ctx, fmt.Sprintf("DROP DATABASE %s WITH (FORCE)", quoteIdentifier(allocation.Database))); err != nil {
			return errors.New("failed to drop sample project instance database")
		}
	}
	if roleExists {
		if _, err := admin.Exec(ctx, fmt.Sprintf("DROP ROLE %s", quoteIdentifier(allocation.Role))); err != nil {
			return errors.New("failed to drop sample project instance role")
		}
	}
	return nil
}

func (t *Target) connect(ctx context.Context, database, user, password string) (*pgx.Conn, error) {
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

func validateProvisionAllocation(allocation Allocation) error {
	if allocation.Database == "" || allocation.Role == "" ||
		strings.ContainsRune(allocation.Database, 0) || strings.ContainsRune(allocation.Role, 0) {
		return errors.New("sample project instance provisioning requires database and role")
	}
	if allocation.Password == "" || strings.ContainsRune(allocation.Password, 0) {
		return errors.New("sample project instance provisioning requires password")
	}
	return nil
}

func validateCleanupAllocation(allocation Allocation) error {
	if (allocation.Database == "" && allocation.Role == "") ||
		strings.ContainsRune(allocation.Database, 0) || strings.ContainsRune(allocation.Role, 0) {
		return errors.New("sample project instance cleanup requires a database or role")
	}
	return nil
}

func quoteIdentifier(value string) string {
	return pgx.Identifier{value}.Sanitize()
}

func quoteLiteral(value string) string {
	return "E'" + strings.NewReplacer(`\`, `\\`, `'`, `\'`).Replace(value) + "'"
}

func staticTargetError(message string) error {
	return newTargetFailure(targetFailureStatic, errors.New(message))
}

func unavailableTargetError(message string) error {
	return newTargetFailure(targetFailureUnavailable, errors.New(message))
}

func classifyProvisionError(err error, message string) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && (pgErr.Code == "42710" || pgErr.Code == "42P04") {
		return newTargetFailure(targetFailureInvariant, errors.New(message))
	}
	return unavailableTargetError(message)
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

func terminateAndDrain(ctx context.Context, conn *pgx.Conn, allocation Allocation) error {
	for {
		rows, err := conn.Query(ctx, `
			SELECT pid
			FROM pg_stat_activity
			WHERE (usename = $1 OR datname = $2)
				AND pid <> pg_backend_pid()
		`, allocation.Role, allocation.Database)
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
