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
	"time"

	"github.com/pkg/errors"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Container is a running engine and the admin handle its tests share. Tests
// reach one through the accessors in shared.go; the two constructors exported
// here are for backend/tests, whose containers are the Bytebase instances under
// test and so cannot be shared.
type Container struct {
	container testcontainers.Container
	host      string
	port      string
	username  string
	password  string
	db        *sql.DB
	tlsDir    string
	tlsCAPath string
}

func (c *Container) GetHost() string     { return c.host }
func (c *Container) GetPort() string     { return c.port }
func (c *Container) GetUsername() string { return c.username }
func (c *Container) GetPassword() string { return c.password }

// GetDB is the admin connection, nil for an engine with no database/sql driver.
func (c *Container) GetDB() *sql.DB { return c.db }

// GetTLSCAPath returns the CA certificate path for a TLS-enabled PostgreSQL
// container.
func (c *Container) GetTLSCAPath() string { return c.tlsCAPath }

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

// spec is what an engine needs beyond its container request: the port to map,
// the credentials its tests connect with, and the driver and DSN for the admin
// handle. An engine with no database/sql driver leaves driver empty, and its
// Container carries the credentials instead of a handle.
type spec struct {
	port     string
	username string
	password string
	driver   string
	dsn      func(*Container) string
}

// start runs the container and returns once the engine answers, so the first
// statement against it does not race the boot. A container that fails any step
// here is terminated rather than left behind.
func start(ctx context.Context, req testcontainers.ContainerRequest, s spec) (retC *Container, retErr error) {
	req.ExposedPorts = []string{s.port}
	raw, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}

	c := &Container{container: raw, username: s.username, password: s.password}
	defer func() {
		if retErr != nil {
			c.Close(ctx)
		}
	}()

	if c.host, err = raw.Host(ctx); err != nil {
		return nil, err
	}
	port, err := raw.MappedPort(ctx, s.port)
	if err != nil {
		return nil, err
	}
	c.port = port.Port()

	if s.driver == "" {
		return c, nil
	}
	if c.db, err = sql.Open(s.driver, s.dsn(c)); err != nil {
		return nil, err
	}
	if err := waitDBPing(ctx, c.db); err != nil {
		return nil, err
	}
	return c, nil
}

// waitDBPing returns once the database accepts a connection. The container's
// log wait has already passed by the time this runs, so the first ping nearly
// always succeeds; the poll only covers the gap between "ready" in the log and
// the listener. A 3 s ticker here used to add 3 s to every container start.
func waitDBPing(ctx context.Context, db *sql.DB) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.After(10 * time.Minute)
	for {
		if err := db.PingContext(ctx); err == nil {
			return nil
		}
		select {
		case <-ticker.C:
		case <-timeout:
			return errors.Errorf("start container timeout reached")
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

var pgSpec = spec{
	port:     "5432/tcp",
	username: "postgres",
	password: "root-password",
	driver:   "pgx",
	dsn: func(c *Container) string {
		return fmt.Sprintf("host=%s port=%s user=%s password=%s database=postgres sslmode=disable",
			c.host, c.port, c.username, c.password)
	},
}

func pgRequest(image string) testcontainers.ContainerRequest {
	return testcontainers.ContainerRequest{
		Image: image,
		Env: map[string]string{
			"LANG":              "en_US.UTF-8",
			"POSTGRES_PASSWORD": pgSpec.password,
		},
		// A shared container carries a connection pool per server and per
		// instance on it; the default 100 runs out at twenty parallel tests, as
		// "server refused TLS connection".
		Cmd: []string{"postgres", "-c", "max_connections=1000"},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).WithStartupTimeout(5 * time.Minute),
	}
}

// GetPgContainer starts PostgreSQL 16. Tests share one through
// SharedPgContainer; this is exported for the instances backend/tests
// provisions per test.
func GetPgContainer(ctx context.Context) (*Container, error) {
	return start(ctx, pgRequest("postgres:16-alpine"), pgSpec)
}

// getPg17Container starts PostgreSQL 17, required for the features absent in
// 16 — MERGE ... RETURNING being the one.
func getPg17Container(ctx context.Context) (*Container, error) {
	return start(ctx, pgRequest("postgres:17-alpine"), pgSpec)
}

// getTLSPgContainer starts a PostgreSQL 16 that serves TLS, for clients
// connecting with sslmode=verify-full and the CA at GetTLSCAPath. The admin
// handle stays on plaintext.
func getTLSPgContainer(ctx context.Context) (retC *Container, retErr error) {
	tlsDir, ca, certificate, key, err := createPostgreSQLTLSMaterial()
	if err != nil {
		return nil, err
	}
	defer func() {
		if retErr != nil {
			_ = os.RemoveAll(tlsDir)
		}
	}()

	req := pgRequest("postgres:16-alpine")
	req.Entrypoint = []string{
		"/bin/sh",
		"-c",
		"chown postgres:postgres /tmp/server.key && exec /usr/local/bin/docker-entrypoint.sh \"$@\"",
		"--",
	}
	req.Cmd = append(req.Cmd,
		"-c", "ssl=on",
		"-c", "ssl_cert_file=/tmp/server.crt",
		"-c", "ssl_key_file=/tmp/server.key",
	)
	req.Files = []testcontainers.ContainerFile{
		{Reader: bytes.NewReader(certificate), ContainerFilePath: "/tmp/server.crt", FileMode: 0o644},
		{Reader: bytes.NewReader(key), ContainerFilePath: "/tmp/server.key", FileMode: 0o600},
	}

	c, err := start(ctx, req, pgSpec)
	if err != nil {
		return nil, err
	}
	c.tlsDir, c.tlsCAPath = tlsDir, ca
	return c, nil
}

// GetMySQLContainer starts MySQL. Exported for the same reason as GetPgContainer.
func GetMySQLContainer(ctx context.Context) (*Container, error) {
	const user, password = "root", "root-password"
	return start(ctx, testcontainers.ContainerRequest{
		Image:      "mysql:8.0.33",
		Env:        map[string]string{"MYSQL_ROOT_PASSWORD": password},
		WaitingFor: wait.ForLog("ready for connections").WithOccurrence(2).WithStartupTimeout(5 * time.Minute),
	}, spec{
		port:     "3306/tcp",
		username: user,
		password: password,
		driver:   "mysql",
		dsn: func(c *Container) string {
			return fmt.Sprintf("%s:%s@tcp(%s:%s)/?multiStatements=true", c.username, c.password, c.host, c.port)
		},
	})
}

func getMSSQLContainer(ctx context.Context) (*Container, error) {
	const user, password = "sa", "Test123!"
	return start(ctx, testcontainers.ContainerRequest{
		Image: "mcr.microsoft.com/mssql/server:2022-latest",
		Env: map[string]string{
			"ACCEPT_EULA": "Y",
			"SA_PASSWORD": password,
			"MSSQL_PID":   "Express",
		},
		WaitingFor: wait.ForLog("SQL Server is now ready for client connections").
			WithStartupTimeout(3 * time.Minute),
	}, spec{
		port:     "1433/tcp",
		username: user,
		password: password,
		driver:   "sqlserver",
		dsn: func(c *Container) string {
			return fmt.Sprintf("sqlserver://%s:%s@%s:%s?database=master", c.username, c.password, c.host, c.port)
		},
	})
}

// getTiDBContainer starts TiDB, which speaks the MySQL protocol with no
// password on root.
func getTiDBContainer(ctx context.Context) (*Container, error) {
	return start(ctx, testcontainers.ContainerRequest{
		Image:      "pingcap/tidb:v8.5.0",
		WaitingFor: wait.ForLog("server is running MySQL protocol").WithStartupTimeout(5 * time.Minute),
	}, spec{
		port:     "4000/tcp",
		username: "root",
		driver:   "mysql",
		dsn: func(c *Container) string {
			return fmt.Sprintf("%s@tcp(%s:%s)/?multiStatements=true&tls=false", c.username, c.host, c.port)
		},
	})
}

// getMongoDBContainer starts MongoDB, which has no database/sql driver: its
// tests open the Mongo driver themselves from the container's credentials.
func getMongoDBContainer(ctx context.Context) (*Container, error) {
	const user, password = "testuser", "testpass"
	return start(ctx, testcontainers.ContainerRequest{
		Image: "mongo:5",
		Env: map[string]string{
			"MONGO_INITDB_ROOT_USERNAME": user,
			"MONGO_INITDB_ROOT_PASSWORD": password,
		},
		WaitingFor: wait.ForLog("Waiting for connections").WithStartupTimeout(3 * time.Minute),
	}, spec{
		port:     "27017/tcp",
		username: user,
		password: password,
	})
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
