//go:build linux && amd64
// +build linux,amd64

package mysqlbinlog

import (
	"embed"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/bytebase/bytebase/resources/utils"
)

//go:embed mysqlbinlog-8.0.28-linux-glibc-2.17-x86_64.tar.gz
var resources embed.FS

// TestBootMySQLBinlog tests whether mysqlbinlog can be started on the target platform
// to check whether the lib extraction is correct.
func TestBootMySQLBinlog(t *testing.T) {
	tarName := "mysqlbinlog-8.0.28-linux-glibc-2.17-x86_64.tar.gz"
	version := "mysqlbinlog-8.0.28-linux-glibc-2.17-x86_64"
	tarF, err := resources.Open(tarName)
	if err != nil {
		t.Errorf("failed to open mysqlbinlog dist %q, error: %w", tarName, err)
	}
	defer tarF.Close()

	tmpDir := t.TempDir()
	if err := utils.ExtractTarGz(tarF, tmpDir); err != nil {
		t.Errorf("failed to extract mysqlbinlog distribution %q, error: %v", tarName, err)
	}

	mysqlbinlogBinPath := filepath.Join(tmpDir, version, "bin", "mysqlbinlog")
	cmd := exec.Command(mysqlbinlogBinPath, "-V")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		t.Errorf("failed to boot mysqlbinlog, cmd: %s, err: %w", cmd.String(), err)
	}
}
