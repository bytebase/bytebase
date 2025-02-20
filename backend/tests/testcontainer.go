package tests

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type Container struct {
	container testcontainers.Container
	host      string
	port      string
	db        *sql.DB
}

func getMySQLContainer(ctx context.Context) (*Container, error) {
	req := testcontainers.ContainerRequest{
		Image: "mysql:8.0.33",
		Env: map[string]string{
			"MYSQL_ROOT_PASSWORD": "root-password",
		},
		ExposedPorts: []string{"3306/tcp"},
		WaitingFor:   wait.ForListeningPort("3306/tcp"),
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

	return &Container{
		container: c,
		host:      host,
		port:      port.Port(),
		db:        db,
	}, nil
}
