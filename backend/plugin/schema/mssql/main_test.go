package mssql

import (
	"context"
	"sync"
	"testing"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
)

// The package shares one SQL Server container, started on first use.
//
// Sharing it is worth a container start, and this package took three of them.
// Starting it lazily is what keeps that saving from turning into a tax on the
// tests that need no engine at all, which would otherwise pay for a container
// they never open.
var (
	mssqlOnce      sync.Once
	mssqlContainer *testcontainer.Container
	mssqlErr       error
)

func TestMain(m *testing.M) {
	defer func() {
		if mssqlContainer != nil {
			mssqlContainer.Close(context.Background())
		}
	}()
	m.Run()
}

// sharedMSSQLContainer returns the package's SQL Server container, starting it on
// the first call. Tests keep isolating themselves with a per-test database,
// which is what makes one container enough.
func sharedMSSQLContainer(t *testing.T) *testcontainer.Container {
	t.Helper()

	mssqlOnce.Do(func() {
		container, err := testcontainer.GetMSSQLContainer(context.Background())
		if err != nil {
			mssqlErr = err
			return
		}
		mssqlContainer = container
	})
	if mssqlErr != nil {
		t.Fatalf("failed to start the shared SQL Server container: %v", mssqlErr)
	}
	return mssqlContainer
}
