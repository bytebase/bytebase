package testcontainer

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"sync"

	"github.com/testcontainers/testcontainers-go"
)

// Postgres containers start from an image whose data directory is already
// initialized. On the official image initdb is most of the start: a container
// answers 0.18 s after `docker run` from the prebuilt image against 1.25 s from
// postgres:16-alpine, and the suite starts about 200 of them. The image is
// built once per machine, tagged with the hash of its Dockerfile so that an
// edit here rebuilds it, and found by tag by every process after that.
const pgImageDockerfile = `FROM postgres:16-alpine
ENV PGDATA=/var/lib/postgresql/pgdata LANG=en_US.UTF-8 POSTGRES_PASSWORD=root-password
RUN docker-ensure-initdb.sh
`

const (
	pgOfficialImage = "postgres:16-alpine"
	pgImageRepo     = "bytebase-test-postgres"
)

var (
	pgImageOnce sync.Once
	pgImageName string
	// pgImageReady is how many times the server logs "ready to accept
	// connections" before it is up: twice on the official image, whose initdb
	// runs a temporary server first, and once on the prebuilt one.
	pgImageReady int
)

// pgImage returns the image Postgres containers start from and the log
// occurrence that marks it ready. When the prebuilt image cannot be built it
// falls back to the official one and says so.
func pgImage(ctx context.Context) (string, int) {
	pgImageOnce.Do(func() {
		pgImageName, pgImageReady = pgOfficialImage, 2
		name, err := buildPgImage(ctx)
		if err != nil {
			slog.Warn("falling back to the official postgres image", slog.String("error", err.Error()))
			return
		}
		pgImageName, pgImageReady = name, 1
	})
	return pgImageName, pgImageReady
}

func buildPgImage(ctx context.Context) (string, error) {
	sum := sha256.Sum256([]byte(pgImageDockerfile))
	tag := "16-" + hex.EncodeToString(sum[:6])
	name := pgImageRepo + ":" + tag

	provider, err := testcontainers.NewDockerProvider()
	if err != nil {
		return "", err
	}
	defer func() { _ = provider.Close() }()
	if _, err := provider.Client().ImageInspect(ctx, name); err == nil {
		return name, nil
	}

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	if err := tw.WriteHeader(&tar.Header{Name: "Dockerfile", Mode: 0o644, Size: int64(len(pgImageDockerfile))}); err != nil {
		return "", err
	}
	if _, err := tw.Write([]byte(pgImageDockerfile)); err != nil {
		return "", err
	}
	if err := tw.Close(); err != nil {
		return "", err
	}
	return provider.BuildImage(ctx, &testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			ContextArchive: bytes.NewReader(buf.Bytes()),
			Dockerfile:     "Dockerfile",
			Repo:           pgImageRepo,
			Tag:            tag,
			KeepImage:      true,
		},
	})
}
