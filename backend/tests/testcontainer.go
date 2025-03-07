package tests

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/pkg/errors"
	"github.com/testcontainers/testcontainers-go"
)

type Container struct {
	container testcontainers.Container
	host      string
	port      string
	db        *sql.DB
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

func getMySQLContainer(ctx context.Context) (retc *Container, retErr error) {
	req := testcontainers.ContainerRequest{
		Image: "mysql:8.0.33",
		Env: map[string]string{
			"MYSQL_ROOT_PASSWORD": "root-password",
		},
		ExposedPorts: []string{"3306/tcp"},
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

func getPgContainer(ctx context.Context) (retC *Container, retErr error) {
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
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	timeout := time.After(2 * time.Minute)
outerLoop:
	for {
		select {
		case <-ticker.C:
			if err := db.PingContext(ctx); err == nil {
				break outerLoop
			}
		case <-timeout:
			return errors.Errorf("start container timeout reached")
		}
	}
	return nil
}
