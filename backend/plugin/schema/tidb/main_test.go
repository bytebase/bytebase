package tidb

import (
	"context"
	"sync"
	"testing"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
)

// The package shares one TiDB container, started on first use.
//
// Sharing it is worth a container start, and this package took four of them.
// Starting it lazily is what keeps that saving from turning into a tax on the
// tests that need no engine at all, which would otherwise pay for a container
// they never open.
var (
	tidbOnce      sync.Once
	tidbContainer *testcontainer.Container
	tidbErr       error
)

func TestMain(m *testing.M) {
	defer func() {
		if tidbContainer != nil {
			tidbContainer.Close(context.Background())
		}
	}()
	m.Run()
}

// sharedTiDBContainer returns the package's TiDB container, starting it on
// the first call. Tests keep isolating themselves with a per-test database,
// which is what makes one container enough.
func sharedTiDBContainer(t *testing.T) *testcontainer.Container {
	t.Helper()

	tidbOnce.Do(func() {
		container, err := testcontainer.GetTiDBContainer(context.Background())
		if err != nil {
			tidbErr = err
			return
		}
		tidbContainer = container
	})
	if tidbErr != nil {
		t.Fatalf("failed to start the shared TiDB container: %v", tidbErr)
	}
	return tidbContainer
}
