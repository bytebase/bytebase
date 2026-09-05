package oracle

import (
	"context"
	"sync"
	"testing"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
)

// The package shares one Oracle container, started on first use.
//
// Sharing it is worth a container start: the metadata and definition tests took
// one each, and an Oracle Free start is around 12 s. Starting it lazily is what
// keeps that saving from turning into a tax on everything else -- the
// generate-migration goldens, the omni parser cases and the topological-order
// cases need no engine at all, and a package-wide eager start would have made
// `go test -run TestGenerateMigration` pay 12 s for a container it never opens.
var (
	oracleOnce      sync.Once
	oracleContainer *testcontainer.Container
	oracleErr       error
)

func TestMain(m *testing.M) {
	defer func() {
		if oracleContainer != nil {
			oracleContainer.Close(context.Background())
		}
	}()
	m.Run()
}

// sharedOracleContainer returns the package's Oracle container, starting it on
// the first call. Tests keep isolating themselves with a per-test Oracle user,
// which is what makes one container enough.
func sharedOracleContainer(t *testing.T) *testcontainer.Container {
	t.Helper()

	oracleOnce.Do(func() {
		container, err := testcontainer.GetOracleContainer(context.Background())
		if err != nil {
			oracleErr = err
			return
		}
		oracleContainer = container
	})
	if oracleErr != nil {
		t.Fatalf("failed to start the shared Oracle container: %v", oracleErr)
	}
	return oracleContainer
}
