package testcontainer

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"encoding/pem"
	"fmt"
	"log/slog"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/pkg/errors"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type Container struct {
	container testcontainers.Container
	host      string
	port      string
	db        *sql.DB
	tlsDir    string
	tlsCAPath string
}

func (c *Container) GetHost() string {
	return c.host
}

func (c *Container) GetPort() string {
	return c.port
}

func (c *Container) GetDB() *sql.DB {
	return c.db
}

// GetTLSCAPath returns the CA certificate path for a TLS-enabled PostgreSQL
// container.
func (c *Container) GetTLSCAPath() string {
	return c.tlsCAPath
}

func (c *Container) Close(ctx context.Context) {
	if c == nil {
		return
	}
	if c.db != nil {
		if err := c.db.Close(); err != nil {
			slog.Error("close db error")
		}
	}
	if c.container != nil {
		if err := c.container.Terminate(ctx, testcontainers.StopTimeout(1*time.Millisecond)); err != nil {
			slog.Error("close container error")
		}
	}
	if c.tlsDir != "" {
		if err := os.RemoveAll(c.tlsDir); err != nil {
			slog.Error("remove TLS directory error")
		}
	}
}

// GetTestMySQLContainer creates a MySQL container for testing
func GetTestMySQLContainer(ctx context.Context) (retc *Container, retErr error) {
	req := testcontainers.ContainerRequest{
		Image: "mysql:8.0.33",
		Env: map[string]string{
			"MYSQL_ROOT_PASSWORD": "root-password",
		},
		ExposedPorts: []string{"3306/tcp"},
		WaitingFor:   wait.ForLog("ready for connections").WithOccurrence(2).WithStartupTimeout(5 * time.Minute),
	}
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}

	host, err := c.Host(ctx)
	if err != nil {
		return nil, err
	}
	port, err := c.MappedPort(ctx, "3306/tcp")
	if err != nil {
		return nil, err
	}

	dsn := fmt.Sprintf("root:root-password@tcp(%s:%s)/?multiStatements=true", host, port.Port())
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	defer func() {
		if retErr != nil {
			db.Close()
		}
	}()

	if err := waitDBPing(ctx, db); err != nil {
		return nil, err
	}

	return &Container{
		container: c,
		host:      host,
		port:      port.Port(),
		db:        db,
	}, nil
}

// GetPgContainer creates a PostgreSQL 16 container for testing.
func GetPgContainer(ctx context.Context) (*Container, error) {
	return getPgContainerWithImage(ctx, "postgres:16-alpine")
}

// GetTLSPgContainer creates a TLS-enabled PostgreSQL 16 container. Clients
// can connect with sslmode=verify-full and the CA returned by GetTLSCAPath.
func GetTLSPgContainer(ctx context.Context) (*Container, error) {
	return getTLSPgContainerWithImage(ctx, "postgres:16-alpine")
}

// GetPg17Container creates a PostgreSQL 17 container for testing. PG17 is required
// for features absent in 16 — notably MERGE ... RETURNING.
func GetPg17Container(ctx context.Context) (*Container, error) {
	return getPgContainerWithImage(ctx, "postgres:17-alpine")
}

func getPgContainerWithImage(ctx context.Context, image string) (retC *Container, retErr error) {
	req := testcontainers.ContainerRequest{
		Image: image,
		Env: map[string]string{
			"LANG":              "en_US.UTF-8",
			"POSTGRES_PASSWORD": "root-password",
		},
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor:   wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(5 * time.Minute),
	}

	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}

	host, err := c.Host(ctx)
	if err != nil {
		return nil, err
	}
	port, err := c.MappedPort(ctx, "5432/tcp")
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("pgx", fmt.Sprintf("host=%s port=%s user=postgres password=root-password database=postgres", host, port.Port()))
	if err != nil {
		return nil, err
	}
	defer func() {
		if retErr != nil {
			db.Close()
		}
	}()

	if err := waitDBPing(ctx, db); err != nil {
		return nil, err
	}

	return &Container{
		container: c,
		host:      host,
		port:      port.Port(),
		db:        db,
	}, nil
}

func getTLSPgContainerWithImage(ctx context.Context, image string) (retC *Container, retErr error) {
	tlsDir, ca, certificate, key, err := createPostgreSQLTLSMaterial()
	if err != nil {
		return nil, err
	}
	defer func() {
		if retErr != nil {
			_ = os.RemoveAll(tlsDir)
		}
	}()

	req := testcontainers.ContainerRequest{
		Image: image,
		Env: map[string]string{
			"LANG":              "en_US.UTF-8",
			"POSTGRES_PASSWORD": "root-password",
		},
		ExposedPorts: []string{"5432/tcp"},
		Entrypoint: []string{
			"/bin/sh",
			"-c",
			"chown postgres:postgres /tmp/server.key && exec /usr/local/bin/docker-entrypoint.sh \"$@\"",
			"--",
		},
		Cmd: []string{
			"postgres",
			"-c", "ssl=on",
			"-c", "ssl_cert_file=/tmp/server.crt",
			"-c", "ssl_key_file=/tmp/server.key",
		},
		Files: []testcontainers.ContainerFile{
			{Reader: bytes.NewReader(certificate), ContainerFilePath: "/tmp/server.crt", FileMode: 0o644},
			{Reader: bytes.NewReader(key), ContainerFilePath: "/tmp/server.key", FileMode: 0o600},
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(5 * time.Minute),
	}
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}

	host, err := c.Host(ctx)
	if err != nil {
		return nil, err
	}
	port, err := c.MappedPort(ctx, "5432/tcp")
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("pgx", fmt.Sprintf("host=%s port=%s user=postgres password=root-password database=postgres sslmode=disable", host, port.Port()))
	if err != nil {
		return nil, err
	}
	defer func() {
		if retErr != nil {
			_ = db.Close()
			_ = c.Terminate(ctx)
		}
	}()
	if err := waitDBPing(ctx, db); err != nil {
		return nil, err
	}

	return &Container{
		container: c,
		host:      host,
		port:      port.Port(),
		db:        db,
		tlsDir:    tlsDir,
		tlsCAPath: ca,
	}, nil
}

func createPostgreSQLTLSMaterial() (string, string, []byte, []byte, error) {
	tlsDir, err := os.MkdirTemp("", "bytebase-postgres-tls-*")
	if err != nil {
		return "", "", nil, nil, err
	}
	fail := func(err error) (string, string, []byte, []byte, error) {
		_ = os.RemoveAll(tlsDir)
		return "", "", nil, nil, err
	}

	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fail(err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fail(err)
	}
	now := time.Now()
	caTemplate := x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "Bytebase PostgreSQL Test CA"},
		NotBefore:             now.Add(-time.Minute),
		NotAfter:              now.Add(24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, &caTemplate, &caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return fail(err)
	}
	ca, err := x509.ParseCertificate(caDER)
	if err != nil {
		return fail(err)
	}
	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fail(err)
	}
	serial, err = rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fail(err)
	}
	serverTemplate := x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "localhost"},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
		NotBefore:    now.Add(-time.Minute),
		NotAfter:     now.Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	serverDER, err := x509.CreateCertificate(rand.Reader, &serverTemplate, ca, &serverKey.PublicKey, caKey)
	if err != nil {
		return fail(err)
	}
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	certificate := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverDER})
	key := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(serverKey)})
	caPath := filepath.Join(tlsDir, "ca.crt")
	if err := os.WriteFile(caPath, caPEM, 0o644); err != nil {
		return fail(err)
	}
	return tlsDir, caPath, certificate, key, nil
}

func waitDBPing(ctx context.Context, db *sql.DB) error {
	started := time.Now()
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	timeout := time.After(10 * time.Minute)
outerLoop:
	for {
		select {
		case <-ticker.C:
			if err := db.PingContext(ctx); err == nil {
				if time.Since(started) > 1*time.Minute {
					fmt.Printf("Total wait time: %s\n", time.Since(started))
				}
				break outerLoop
			}
		case <-timeout:
			return errors.Errorf("start container timeout reached")
		}
	}
	return nil
}

// GetTestPgContainer is a helper function for tests that creates a PostgreSQL container
// and handles the error by failing the test if container creation fails
func GetTestPgContainer(ctx context.Context, t testing.TB) *Container {
	t.Helper()
	container, err := GetPgContainer(ctx)
	if err != nil {
		t.Fatalf("failed to create PostgreSQL container: %v", err)
	}
	return container
}

// GetTestTLSPgContainer creates a TLS-enabled PostgreSQL container, failing
// the test when the container cannot be started.
func GetTestTLSPgContainer(ctx context.Context, t testing.TB) *Container {
	t.Helper()
	container, err := GetTLSPgContainer(ctx)
	if err != nil {
		t.Fatalf("failed to create TLS PostgreSQL container: %v", err)
	}
	return container
}

// GetTestPg17Container creates a PostgreSQL 17 container for testing, failing the
// test on error.
func GetTestPg17Container(ctx context.Context, t testing.TB) *Container {
	t.Helper()
	container, err := GetPg17Container(ctx)
	if err != nil {
		t.Fatalf("failed to create PostgreSQL 17 container: %v", err)
	}
	return container
}

// GetOracleContainer creates an Oracle container for testing
func GetOracleContainer(ctx context.Context) (retC *Container, retErr error) {
	req := testcontainers.ContainerRequest{
		Image: "gvenzl/oracle-free:slim",
		Env: map[string]string{
			"ORACLE_PASSWORD":   "test123",
			"APP_USER":          "testuser",
			"APP_USER_PASSWORD": "testpass",
		},
		ExposedPorts: []string{"1521/tcp"},
		WaitingFor: wait.ForLog("DATABASE IS READY TO USE!").
			WithStartupTimeout(10 * time.Minute),
		HostConfigModifier: func(hc *container.HostConfig) {
			hc.ShmSize = 1 * 1024 * 1024 * 1024 // 1GB shared memory
		},
	}

	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}

	host, err := c.Host(ctx)
	if err != nil {
		return nil, err
	}
	port, err := c.MappedPort(ctx, "1521/tcp")
	if err != nil {
		return nil, err
	}

	// Oracle connection string format: oracle://username:password@host:port/service_name
	// Use SYSTEM account for privileged operations (like creating users), similar to MySQL root
	dsn := fmt.Sprintf("oracle://system:test123@%s:%s/FREEPDB1", host, port.Port())
	db, err := sql.Open("oracle", dsn)
	if err != nil {
		return nil, err
	}
	defer func() {
		if retErr != nil {
			db.Close()
		}
	}()

	if err := waitDBPing(ctx, db); err != nil {
		return nil, err
	}

	return &Container{
		container: c,
		host:      host,
		port:      port.Port(),
		db:        db,
	}, nil
}

// GetTestOracleContainer is a helper function for tests that creates an Oracle container
// and handles the error by failing the test if container creation fails
func GetTestOracleContainer(ctx context.Context, t testing.TB) *Container {
	t.Helper()
	container, err := GetOracleContainer(ctx)
	if err != nil {
		t.Fatalf("failed to create Oracle container: %v", err)
	}
	return container
}

// GetStarRocksContainer creates a StarRocks (allin1) container for testing. The allin1
// image bundles the FE and BE; information_schema is served by the BE, so this waits until
// the backend actually serves queries rather than only the FE port being open.
//
// NOTE: requires an amd64 host — the StarRocks BE has no working arm64 build (it fails to
// come alive under emulation). bytebase CI runs on amd64; locally, run without -short on an
// amd64 machine, or the test is skipped via testing.Short().
func GetStarRocksContainer(ctx context.Context) (retC *Container, retErr error) {
	req := testcontainers.ContainerRequest{
		Image:        "starrocks/allin1-ubuntu:3.4.10",
		ExposedPorts: []string{"9030/tcp"},
		WaitingFor:   wait.ForListeningPort("9030/tcp").WithStartupTimeout(5 * time.Minute),
	}
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}
	host, err := c.Host(ctx)
	if err != nil {
		return nil, err
	}
	port, err := c.MappedPort(ctx, "9030/tcp")
	if err != nil {
		return nil, err
	}
	dsn := fmt.Sprintf("root@tcp(%s:%s)/?multiStatements=true", host, port.Port())
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	defer func() {
		if retErr != nil {
			db.Close()
		}
	}()
	if err := waitStarRocksReady(ctx, db); err != nil {
		return nil, err
	}
	return &Container{
		container: c,
		host:      host,
		port:      port.Port(),
		db:        db,
	}, nil
}

// GetTestStarRocksContainer creates a StarRocks container and fails the test on error. It
// skips on non-amd64 hosts: the StarRocks all-in-one BE has no working arm64 build (it never
// comes alive under emulation), so the readiness wait would otherwise time out.
func GetTestStarRocksContainer(ctx context.Context, t testing.TB) *Container {
	t.Helper()
	if runtime.GOARCH != "amd64" {
		t.Skipf("StarRocks requires an amd64 host; the all-in-one BE has no working arm64 build (GOARCH=%s)", runtime.GOARCH)
	}
	container, err := GetStarRocksContainer(ctx)
	if err != nil {
		t.Fatalf("failed to create StarRocks container: %v", err)
	}
	return container
}

// waitStarRocksReady blocks until at least one StarRocks backend is registered and alive,
// which is required before tablet-allocating DDL (CREATE TABLE) succeeds. The check is
// read-only on purpose: issuing DDL during allin1's first-boot disrupts backend
// registration, so poll SHOW BACKENDS rather than probing with CREATE TABLE.
func waitStarRocksReady(ctx context.Context, db *sql.DB) error {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	timeout := time.After(10 * time.Minute)
	for {
		select {
		case <-ticker.C:
			if starRocksBackendAlive(ctx, db) {
				return nil
			}
		case <-timeout:
			return errors.Errorf("StarRocks backend did not become ready")
		}
	}
}

// starRocksBackendAlive reports whether SHOW BACKENDS lists a backend with Alive=true.
func starRocksBackendAlive(ctx context.Context, db *sql.DB) bool {
	rows, err := db.QueryContext(ctx, "SHOW BACKENDS")
	if err != nil {
		return false
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return false
	}
	aliveIdx := -1
	for i, c := range cols {
		if strings.EqualFold(c, "Alive") {
			aliveIdx = i
			break
		}
	}
	if aliveIdx < 0 {
		return false
	}
	alive := false
	for rows.Next() {
		vals := make([]sql.RawBytes, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return false
		}
		if strings.EqualFold(string(vals[aliveIdx]), "true") {
			alive = true
			break
		}
	}
	if err := rows.Err(); err != nil {
		return false
	}
	return alive
}

// GetMSSQLContainer creates a Microsoft SQL Server container for testing
func GetMSSQLContainer(ctx context.Context) (retC *Container, retErr error) {
	req := testcontainers.ContainerRequest{
		Image: "mcr.microsoft.com/mssql/server:2022-latest",
		Env: map[string]string{
			"ACCEPT_EULA": "Y",
			"SA_PASSWORD": "Test123!",
			"MSSQL_PID":   "Express",
		},
		ExposedPorts: []string{"1433/tcp"},
		WaitingFor: wait.ForLog("SQL Server is now ready for client connections").
			WithStartupTimeout(3 * time.Minute),
	}

	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}

	host, err := c.Host(ctx)
	if err != nil {
		return nil, err
	}
	port, err := c.MappedPort(ctx, "1433/tcp")
	if err != nil {
		return nil, err
	}

	// MSSQL connection string format
	dsn := fmt.Sprintf("sqlserver://sa:Test123!@%s:%s?database=master", host, port.Port())
	db, err := sql.Open("sqlserver", dsn)
	if err != nil {
		return nil, err
	}
	defer func() {
		if retErr != nil {
			db.Close()
		}
	}()

	if err := waitDBPing(ctx, db); err != nil {
		return nil, err
	}

	return &Container{
		container: c,
		host:      host,
		port:      port.Port(),
		db:        db,
	}, nil
}

// GetTestMSSQLContainer is a helper function for tests that creates a MSSQL container
// and handles the error by failing the test if container creation fails
func GetTestMSSQLContainer(ctx context.Context, t testing.TB) *Container {
	t.Helper()
	container, err := GetMSSQLContainer(ctx)
	if err != nil {
		t.Fatalf("failed to create MSSQL container: %v", err)
	}
	return container
}

// GetTiDBContainer creates a TiDB container for testing
func GetTiDBContainer(ctx context.Context) (retC *Container, retErr error) {
	req := testcontainers.ContainerRequest{
		Image:        "pingcap/tidb:v8.5.0",
		ExposedPorts: []string{"4000/tcp"},
		WaitingFor:   wait.ForLog("server is running MySQL protocol").WithStartupTimeout(5 * time.Minute),
	}

	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}

	host, err := c.Host(ctx)
	if err != nil {
		return nil, err
	}
	port, err := c.MappedPort(ctx, "4000/tcp")
	if err != nil {
		return nil, err
	}

	// TiDB uses MySQL protocol, so we use MySQL driver
	dsn := fmt.Sprintf("root@tcp(%s:%s)/?multiStatements=true&tls=false", host, port.Port())
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	defer func() {
		if retErr != nil {
			db.Close()
		}
	}()

	if err := waitDBPing(ctx, db); err != nil {
		return nil, err
	}

	return &Container{
		container: c,
		host:      host,
		port:      port.Port(),
		db:        db,
	}, nil
}

// GetTestTiDBContainer is a helper function for tests that creates a TiDB container
// and handles the error by failing the test if container creation fails
func GetTestTiDBContainer(ctx context.Context, t testing.TB) *Container {
	t.Helper()
	container, err := GetTiDBContainer(ctx)
	if err != nil {
		t.Fatalf("failed to create TiDB container: %v", err)
	}
	return container
}

// MongoDBContainer represents a MongoDB container with its connection details
type MongoDBContainer struct {
	container testcontainers.Container
	host      string
	port      string
	username  string
	password  string
}

func (c *MongoDBContainer) GetHost() string {
	return c.host
}

func (c *MongoDBContainer) GetPort() string {
	return c.port
}

func (c *MongoDBContainer) GetUsername() string {
	return c.username
}

func (c *MongoDBContainer) GetPassword() string {
	return c.password
}

func (c *MongoDBContainer) Close(ctx context.Context) {
	if c == nil {
		return
	}
	if c.container != nil {
		if err := c.container.Terminate(ctx, testcontainers.StopTimeout(1*time.Millisecond)); err != nil {
			slog.Error("close MongoDB container error")
		}
	}
}

// GetMongoDBContainer creates a MongoDB container for testing
func GetMongoDBContainer(ctx context.Context) (*MongoDBContainer, error) {
	req := testcontainers.ContainerRequest{
		Image: "mongo:5",
		Env: map[string]string{
			"MONGO_INITDB_ROOT_USERNAME": "testuser",
			"MONGO_INITDB_ROOT_PASSWORD": "testpass",
		},
		ExposedPorts: []string{"27017/tcp"},
		WaitingFor:   wait.ForLog("Waiting for connections").WithStartupTimeout(3 * time.Minute),
	}

	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}

	host, err := c.Host(ctx)
	if err != nil {
		return nil, err
	}
	port, err := c.MappedPort(ctx, "27017/tcp")
	if err != nil {
		return nil, err
	}

	return &MongoDBContainer{
		container: c,
		host:      host,
		port:      port.Port(),
		username:  "testuser",
		password:  "testpass",
	}, nil
}

// GetTestMongoDBContainer is a helper function for tests that creates a MongoDB container
// and handles the error by failing the test if container creation fails
func GetTestMongoDBContainer(ctx context.Context, t testing.TB) *MongoDBContainer {
	t.Helper()
	container, err := GetMongoDBContainer(ctx)
	if err != nil {
		t.Fatalf("failed to create MongoDB container: %v", err)
	}
	return container
}
