package sampleprojectinstance

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
)

func TestNewTargetRejectsUnsafeConfiguration(t *testing.T) {
	for _, test := range []struct {
		name      string
		targetURL string
	}{
		{name: "empty", targetURL: ""},
		{name: "keyword form", targetURL: "user=postgres password=secret host=db.example.com port=5432 dbname=postgres sslmode=require"},
		{name: "missing port", targetURL: "postgres://postgres:secret@db.example.com/postgres?sslmode=require"},
		{name: "insecure TLS", targetURL: "postgres://postgres:secret@db.example.com:5432/postgres?sslmode=disable"},
		{name: "unauthenticated TLS", targetURL: "postgres://postgres:secret@db.example.com:5432/postgres?sslmode=require"},
		{name: "missing password", targetURL: "postgres://postgres@db.example.com:5432/postgres?sslmode=require"},
		{name: "passfile", targetURL: "postgres://postgres:secret@db.example.com:5432/postgres?sslmode=require&passfile=/tmp/password"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewTarget(test.targetURL)
			require.Error(t, err)
			require.NotContains(t, err.Error(), "secret")
			require.Equal(t, targetFailureStatic, targetFailureKindOf(err))
		})
	}
}

func TestNewTargetBuildsRegistrationConfig(t *testing.T) {
	caPath := t.TempDir() + "/ca.pem"
	caPEM := `-----BEGIN CERTIFICATE-----
MIIDOTCCAiGgAwIBAgIQSRJrEpBGFc7tNb1fb5pKFzANBgkqhkiG9w0BAQsFADAS
MRAwDgYDVQQKEwdBY21lIENvMCAXDTcwMDEwMTAwMDAwMFoYDzIwODQwMTI5MTYw
MDAwWjASMRAwDgYDVQQKEwdBY21lIENvMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8A
MIIBCgKCAQEA6Gba5tHV1dAKouAaXO3/ebDUU4rvwCUg/CNaJ2PT5xLD4N1Vcb8r
bFSW2HXKq+MPfVdwIKR/1DczEoAGf/JWQTW7EgzlXrCd3rlajEX2D73faWJekD0U
aUgz5vtrTXZ90BQL7WvRICd7FlEZ6FPOcPlumiyNmzUqtwGhO+9ad1W5BqJaRI6P
YfouNkwR6Na4TzSj5BrqUfP0FwDizKSJ0XXmh8g8G9mtwxOSN3Ru1QFc61Xyeluk
POGKBV/q6RBNklTNe0gI8usUMlYyoC7ytppNMW7X2vodAelSu25jgx2anj9fDVZu
h7AXF5+4nJS4AAt0n1lNY7nGSsdZas8PbQIDAQABo4GIMIGFMA4GA1UdDwEB/wQE
AwICpDATBgNVHSUEDDAKBggrBgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MB0GA1Ud
DgQWBBStsdjh3/JCXXYlQryOrL4Sh7BW5TAuBgNVHREEJzAlggtleGFtcGxlLmNv
bYcEfwAAAYcQAAAAAAAAAAAAAAAAAAAAATANBgkqhkiG9w0BAQsFAAOCAQEAxWGI
5NhpF3nwwy/4yB4i/CwwSpLrWUa70NyhvprUBC50PxiXav1TeDzwzLx/o5HyNwsv
cxv3HdkLW59i/0SlJSrNnWdfZ19oTcS+6PtLoVyISgtyN6DpkKpdG1cOkW3Cy2P2
+tK/tKHRP1Y/Ra0RiDpOAmqn0gCOFGz8+lqDIor/T7MTpibL3IxqWfPrvfVRHL3B
grw/ZQTTIVjjh4JBSW3WyWgNo/ikC1lrVxzl4iPUGptxT36Cr7Zk2Bsg0XqwbOvK
5d+NTDREkSnUbie4GeutujmX3Dsx88UiV6UY/4lHJa6I5leHUNOHahRbpbWeOfs/
WkBKOclmOV2xlTVuPw==
-----END CERTIFICATE-----`
	require.NoError(t, os.WriteFile(caPath, []byte(caPEM), 0o600))

	target, err := NewTarget("postgres://control:secret@db.example.com:5432/postgres?sslmode=verify-full&sslrootcert=" + url.QueryEscape(caPath))
	require.NoError(t, err)
	require.NotNil(t, target.config.TLSConfig)
	require.False(t, target.config.TLSConfig.InsecureSkipVerify)
	require.Equal(t, "db.example.com", target.config.TLSConfig.ServerName)

	config, err := target.InstanceConfig(Allocation{
		Database: "sample_database",
		Role:     "sample_role",
		Password: "sample-password",
	})
	require.NoError(t, err)
	require.Equal(t, "admin", config.AdminDataSource.Id)
	require.Equal(t, "db.example.com", config.AdminDataSource.Host)
	require.Equal(t, "5432", config.AdminDataSource.Port)
	require.Equal(t, "sample_database", config.AdminDataSource.Database)
	require.Equal(t, "sample_role", config.AdminDataSource.Username)
	require.Equal(t, "sample-password", config.AdminDataSource.Password)
	require.True(t, config.AdminDataSource.UseSsl)
	require.True(t, config.AdminDataSource.VerifyTlsCertificate)
	require.Equal(t, caPEM, config.AdminDataSource.SslCa)
	require.Equal(t, []string{"sample_database"}, config.SyncDatabaseNames)

	_, err = target.InstanceConfig(Allocation{Database: "sample_database", Role: "sample_role"})
	require.Error(t, err)
}

func TestTargetProvisionAndRemove(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping PostgreSQL testcontainer test in short mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(context.Background()) })

	target := newLocalTarget(t, container)
	admin := connectLocal(ctx, t, container, "postgres", "postgres", "root-password")
	defer admin.Close(ctx)
	require.NoError(t, prepareBaseline(ctx, admin))
	require.NoError(t, target.Validate(ctx))

	allocation := Allocation{
		Database: "sample_target_test",
		Role:     "sample_target_role",
		Password: "sample-target-password",
	}
	require.NoError(t, target.Provision(ctx, allocation))

	sampleAdmin := connectLocal(ctx, t, container, allocation.Database, "postgres", "root-password")
	var databaseOwner string
	require.NoError(t, sampleAdmin.QueryRow(ctx, `
		SELECT pg_get_userbyid(datdba)
		FROM pg_database
		WHERE datname = current_database()
	`).Scan(&databaseOwner))
	require.Equal(t, "postgres", databaseOwner)

	var owner string
	require.NoError(t, sampleAdmin.QueryRow(ctx, `
		SELECT tableowner
		FROM pg_tables
		WHERE schemaname = 'public' AND tablename = 'employee'
	`).Scan(&owner))
	require.Equal(t, allocation.Role, owner)
	var publicCanCreate bool
	require.NoError(t, sampleAdmin.QueryRow(ctx, `
		SELECT NOT EXISTS (
			SELECT 1
			FROM aclexplode(COALESCE(nspacl, acldefault('n', nspowner))) AS privilege
			WHERE privilege.grantee = 0 AND privilege.privilege_type = 'CREATE'
		)
		FROM pg_namespace
		WHERE nspname = 'public'
	`).Scan(&publicCanCreate))
	require.True(t, publicCanCreate)
	require.NoError(t, sampleAdmin.Close(ctx))

	var canLogin, isSuperuser, canCreateDB, canCreateRole, canReplicate, bypassRLS bool
	require.NoError(t, admin.QueryRow(ctx, `
		SELECT rolcanlogin, rolsuper, rolcreatedb, rolcreaterole, rolreplication, rolbypassrls
		FROM pg_roles
		WHERE rolname = $1
	`, allocation.Role).Scan(&canLogin, &isSuperuser, &canCreateDB, &canCreateRole, &canReplicate, &bypassRLS))
	require.True(t, canLogin)
	require.False(t, isSuperuser)
	require.False(t, canCreateDB)
	require.False(t, canCreateRole)
	require.False(t, canReplicate)
	require.False(t, bypassRLS)

	foreignRole := "sample_target_foreign"
	_, err := admin.Exec(ctx, fmt.Sprintf("CREATE ROLE %s LOGIN PASSWORD %s", quoteIdentifier(foreignRole), quoteLiteral("foreign-password")))
	require.NoError(t, err)
	foreignConfig, err := pgx.ParseConfig(fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		foreignRole,
		"foreign-password",
		container.GetHost(),
		container.GetPort(),
		allocation.Database,
	))
	require.NoError(t, err)
	foreignConn, err := pgx.ConnectConfig(ctx, foreignConfig)
	require.Error(t, err)
	require.Nil(t, foreignConn)
	_, err = admin.Exec(ctx, fmt.Sprintf("DROP ROLE %s", quoteIdentifier(foreignRole)))
	require.NoError(t, err)

	crossDatabaseConfig, err := pgx.ParseConfig(fmt.Sprintf(
		"postgres://%s:%s@%s:%s/postgres?sslmode=disable",
		allocation.Role,
		allocation.Password,
		container.GetHost(),
		container.GetPort(),
	))
	require.NoError(t, err)
	crossDatabaseConn, err := pgx.ConnectConfig(ctx, crossDatabaseConfig)
	require.Error(t, err)
	require.Nil(t, crossDatabaseConn)

	sample := connectLocal(ctx, t, container, allocation.Database, allocation.Role, allocation.Password)
	_, err = sample.Exec(ctx, "ALTER TABLE employee ADD COLUMN target_owned boolean NOT NULL DEFAULT false")
	require.NoError(t, err)
	require.NoError(t, sample.Close(ctx))

	_, err = admin.Exec(ctx, "CREATE DATABASE other_target_test")
	require.NoError(t, err)
	_, err = admin.Exec(ctx, fmt.Sprintf("GRANT CONNECT ON DATABASE %s TO %s", quoteIdentifier("other_target_test"), quoteIdentifier(allocation.Role)))
	require.NoError(t, err)
	crossDatabaseSession := connectLocal(ctx, t, container, "other_target_test", allocation.Role, allocation.Password)
	_, err = admin.Exec(ctx, fmt.Sprintf("REVOKE CONNECT ON DATABASE %s FROM %s", quoteIdentifier("other_target_test"), quoteIdentifier(allocation.Role)))
	require.NoError(t, err)

	cleanupAllocation := Allocation{
		Database: allocation.Database,
		Role:     allocation.Role,
	}
	require.NoError(t, target.Remove(ctx, cleanupAllocation))
	require.NoError(t, crossDatabaseSession.Close(ctx))

	var exists bool
	require.NoError(t, admin.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", allocation.Database).Scan(&exists))
	require.False(t, exists)
	require.NoError(t, admin.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)", allocation.Role).Scan(&exists))
	require.False(t, exists)
	require.NoError(t, target.Remove(ctx, cleanupAllocation))
	_, err = admin.Exec(ctx, "DROP DATABASE other_target_test")
	require.NoError(t, err)
}

func TestTargetValidateRejectsPublicAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping PostgreSQL testcontainer test in short mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(context.Background()) })

	require.Error(t, newLocalTarget(t, container).Validate(ctx))
}

func TestTargetValidateAllowsCloudSQLAdminPublicAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping PostgreSQL testcontainer test in short mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(context.Background()) })

	admin := connectLocal(ctx, t, container, "postgres", "postgres", "root-password")
	defer admin.Close(ctx)
	require.NoError(t, prepareBaseline(ctx, admin))
	_, err := admin.Exec(ctx, "CREATE DATABASE cloudsqladmin")
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), "DROP DATABASE cloudsqladmin")
	})

	require.NoError(t, newLocalTarget(t, container).Validate(ctx))
}

func TestTargetValidateRejectsUnexpectedPublicDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping PostgreSQL testcontainer test in short mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(context.Background()) })

	admin := connectLocal(ctx, t, container, "postgres", "postgres", "root-password")
	defer admin.Close(ctx)
	require.NoError(t, prepareBaseline(ctx, admin))
	_, err := admin.Exec(ctx, "CREATE DATABASE unexpected_public_database")
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), "DROP DATABASE unexpected_public_database")
	})

	err = newLocalTarget(t, container).Validate(ctx)
	require.Error(t, err)
	require.Equal(t, targetFailureStatic, targetFailureKindOf(err))
}

func TestTargetValidateForCleanupAllowsUnexpectedPublicDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping PostgreSQL testcontainer test in short mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(context.Background()) })

	admin := connectLocal(ctx, t, container, "postgres", "postgres", "root-password")
	defer admin.Close(ctx)
	require.NoError(t, prepareBaseline(ctx, admin))
	_, err := admin.Exec(ctx, "CREATE DATABASE unexpected_public_cleanup_database")
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), "DROP DATABASE unexpected_public_cleanup_database")
	})

	target := newLocalTarget(t, container)
	require.Error(t, target.Validate(ctx))
	require.NoError(t, target.ValidateForCleanup(ctx))
}

func TestTargetProvisionRemovesNewRoleWhenDatabaseExists(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping PostgreSQL testcontainer test in short mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	container := testcontainer.GetTestPgContainer(ctx, t)
	t.Cleanup(func() { container.Close(context.Background()) })

	admin := connectLocal(ctx, t, container, "postgres", "postgres", "root-password")
	defer admin.Close(ctx)
	require.NoError(t, prepareBaseline(ctx, admin))

	allocation := Allocation{
		Database: "sample_target_existing_database",
		Role:     "sample_target_partial_role",
		Password: "sample-target-password",
	}
	_, err := admin.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", quoteIdentifier(allocation.Database)))
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), fmt.Sprintf("DROP DATABASE %s", quoteIdentifier(allocation.Database)))
	})

	err = newLocalTarget(t, container).Provision(ctx, allocation)
	require.Error(t, err)
	require.Equal(t, targetFailureInvariant, targetFailureKindOf(err))

	var databasePresent, rolePresent bool
	require.NoError(t, admin.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", allocation.Database).Scan(&databasePresent))
	require.True(t, databasePresent)
	require.NoError(t, admin.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)", allocation.Role).Scan(&rolePresent))
	require.False(t, rolePresent)
}

func newLocalTarget(t *testing.T, container *testcontainer.Container) *Target {
	t.Helper()
	config, err := pgx.ParseConfig(fmt.Sprintf(
		"postgres://postgres:root-password@%s:%s/postgres?sslmode=disable",
		container.GetHost(),
		container.GetPort(),
	))
	require.NoError(t, err)
	return newTargetFromConfig(config, false, "")
}

func connectLocal(ctx context.Context, t *testing.T, container *testcontainer.Container, database, user, password string) *pgx.Conn {
	t.Helper()
	config, err := pgx.ParseConfig(fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user,
		password,
		container.GetHost(),
		container.GetPort(),
		database,
	))
	require.NoError(t, err)
	conn, err := pgx.ConnectConfig(ctx, config)
	require.NoError(t, err)
	return conn
}

func prepareBaseline(ctx context.Context, conn *pgx.Conn) error {
	for _, statement := range []string{
		"REVOKE ALL PRIVILEGES ON DATABASE postgres FROM PUBLIC",
		"REVOKE ALL PRIVILEGES ON DATABASE template1 FROM PUBLIC",
	} {
		if _, err := conn.Exec(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}
