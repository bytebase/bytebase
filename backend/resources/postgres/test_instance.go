package postgres

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
)

const (
	// Shared external PG server variables.
	TestPgUser = "root"
)

// SetupTestInstance installs and starts a postgresql instance for testing,
// returns the stop function.
func SetupTestInstance(pgBinDir, pgDataDir string, port int) func() {
	if err := initDB(pgBinDir, pgDataDir, TestPgUser); err != nil {
		panic(err)
	}
	if err := startForTest(port, pgBinDir, pgDataDir); err != nil {
		panic(err)
	}

	stopFn := func() {
		if err := stop(pgBinDir, pgDataDir); err != nil {
			panic(err)
		}
		if err := os.RemoveAll(pgDataDir); err != nil {
			panic(err)
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
