package testcontainer

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/pkg/errors"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type Container struct {
	container testcontainers.Container
	host      string
	port      string
	db        *sql.DB
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

// GetPgContainer creates a PostgreSQL container for testing
func GetPgContainer(ctx context.Context) (retC *Container, retErr error) {
	req := testcontainers.ContainerRequest{
		Image: "postgres:16-alpine",
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
	dsn := fmt.Sprintf("oracle://testuser:testpass@%s:%s/FREEPDB1", host, port.Port())
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
