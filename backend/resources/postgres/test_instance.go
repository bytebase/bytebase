package postgres

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
)

// SetupTestInstance installs and starts a postgresql instance for testing,
// returns the stop function.
func SetupTestInstance(t *testing.T, port int, resourceDir string) func() {
	dataDir := t.TempDir()
	t.Log("Installing PostgreSQL...")
	binDir, err := Install(resourceDir)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("InitDB...")
	if err := InitDB(binDir, dataDir, "root"); err != nil {
		t.Fatal(err)
	}
	t.Log("Starting PostgreSQL...")
	if err := startForTest(port, binDir, dataDir); err != nil {
		t.Fatal(err)
	}

	stopFn := func() {
		t.Log("Stopping PostgreSQL...")
		if err := Stop(binDir, dataDir); err != nil {
			t.Fatal(err)
		}
	}

	return stopFn
}

// startForTest starts a postgres instance on localhost given port, outputs to stdout and stderr.
// If port is 0, then it will choose a random unused port.
func startForTest(port int, pgBinDir, pgDataDir string) (err error) {
	pgbin := filepath.Join(pgBinDir, "pg_ctl")

	// See -p -k -h option definitions in the link below.
	// https://www.postgresql.org/docs/current/app-postgres.html
	p := exec.Command(pgbin, "start", "-w",
		"-D", pgDataDir,
		"-o", fmt.Sprintf(`-p %d -k %s`, port, common.GetPostgresSocketDir()))

	// Suppress log spam
	p.Stdout = nil
	p.Stderr = os.Stderr
	if err := p.Run(); err != nil {
		return errors.Wrapf(err, "failed to start postgres %q", p.String())
	}

	return nil
}
