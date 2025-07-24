package tests

import (
	"context"
	"testing"
)

func TestMain(m *testing.M) {
	ctx := context.Background()
	code, err := startMain(ctx, m)
	if err != nil {
		panic(err)
	}

	if code != 0 {
		panic("tests failed")
	}
}

func startMain(ctx context.Context, m *testing.M) (int, error) {
	pgContainer, err := getPgContainer(ctx)
	if err != nil {
		return 0, err
	}
	defer func() {
		if pgContainer != nil {
			pgContainer.Close(ctx)
		}
	}()
	externalPgHost = pgContainer.host
	externalPgPort = pgContainer.port

	return m.Run(), nil
}
