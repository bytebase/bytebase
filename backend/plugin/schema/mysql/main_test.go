package mysql

import (
	"context"
	"sync"
	"testing"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
)

// The package shares one MySQL container, started on first use.
//
// Sharing it is worth a container start, and this package took three of them.
// Starting it lazily is what keeps that saving from turning into a tax on the
// tests that need no engine at all, which would otherwise pay for a container
// they never open.
var (
	mysqlOnce      sync.Once
	mysqlContainer *testcontainer.Container
	mysqlErr       error
)

func TestMain(m *testing.M) {
	defer func() {
		if mysqlContainer != nil {
			mysqlContainer.Close(context.Background())
		}
	}()
	m.Run()
}

// sharedMySQLContainer returns the package's MySQL container, starting it on
// the first call. Tests keep isolating themselves with a per-test database,
// which is what makes one container enough.
func sharedMySQLContainer(t *testing.T) *testcontainer.Container {
	t.Helper()

	mysqlOnce.Do(func() {
		container, err := testcontainer.GetTestMySQLContainer(context.Background())
		if err != nil {
			mysqlErr = err
			return
		}
		mysqlContainer = container
	})
	if mysqlErr != nil {
		t.Fatalf("failed to start the shared MySQL container: %v", mysqlErr)
	}
	return mysqlContainer
}
