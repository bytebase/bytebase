package tests

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/bytebase/bytebase/backend/resources/mongoutil"

	"github.com/bytebase/bytebase/backend/resources/postgres"
)

func TestMain(m *testing.M) {
	ctx := context.Background()
	code, err := startMain(ctx, m)
	if err != nil {
		log.Fatal(err)
	}

	os.Exit(code)
}

func startMain(ctx context.Context, m *testing.M) (int, error) {
	resourceDir = os.TempDir()
	if _, err := postgres.Install(resourceDir); err != nil {
		return 0, err
	}
	if _, err := mongoutil.Install(resourceDir); err != nil {
		return 0, err
	}

	pgContainer, err := getPgContainer(ctx)
	defer func() {
		pgContainer.Close(ctx)
	}()
	if err != nil {
		return 0, err
	}
	externalPgHost = pgContainer.host
	externalPgPort = pgContainer.port

	return m.Run(), nil
}
