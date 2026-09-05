package pg

import (
	"context"
	"sync"
	"testing"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
)

// The package shares one PostgreSQL container, started on first use.
//
// Sharing it is worth a container start, and this package took two of them.
// Starting it lazily is what keeps that saving from turning into a tax on the
// tests that need no engine at all, which would otherwise pay for a container
// they never open.
var (
	pgOnce      sync.Once
	pgContainer *testcontainer.Container
	pgErr       error
)

func TestMain(m *testing.M) {
	defer func() {
		if pgContainer != nil {
			pgContainer.Close(context.Background())
		}
	}()
	m.Run()
}

// sharedPgContainer returns the package's PostgreSQL container, starting it on
// the first call. Tests keep isolating themselves with a per-test database,
// which is what makes one container enough.
func sharedPgContainer(t *testing.T) *testcontainer.Container {
	t.Helper()

	pgOnce.Do(func() {
		container, err := testcontainer.GetPgContainer(context.Background())
		if err != nil {
			pgErr = err
			return
		}
		pgContainer = container
	})
	if pgErr != nil {
		t.Fatalf("failed to start the shared PostgreSQL container: %v", pgErr)
	}
	return pgContainer
}
