package tests

import (
	"context"
	"log"
	"os"
	"testing"
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
