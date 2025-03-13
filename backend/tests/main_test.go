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
	if err := startMain(ctx); err != nil {
		log.Fatal(err)
	}

	os.Exit(m.Run())
}

func startMain(ctx context.Context) error {
	resourceDir = os.TempDir()
	if _, err := postgres.Install(resourceDir); err != nil {
		return err
	}
	if _, err := mongoutil.Install(resourceDir); err != nil {
		return err
	}

	pgContainer, err := getPgContainer(ctx)
	defer func() {
		pgContainer.Close(ctx)
	}()
	if err != nil {
		return err
	}
	externalPgHost = pgContainer.host
	externalPgPort = pgContainer.port
	return nil
}
