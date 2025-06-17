package testcontainer

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"testing"
	"time"

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

// GetMySQLContainer creates a MySQL container for testing
func GetMySQLContainer(ctx context.Context) (retc *Container, retErr error) {
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
