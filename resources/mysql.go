package resources

import (
	"archive/tar"
	"compress/gzip"
	"database/sql"
	"embed"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"time"

	// install mysql driver
	_ "github.com/go-sql-driver/mysql"
)

// MysqlInstance represents a mysql instance.
type MysqlInstance struct {
	dir  string
	port int
	proc *os.Process
}

// CreateMysqlInstance creates a mysql instance.
// Must call Purge() to remove the mysql instance after use.
func CreateMysqlInstance() (*MysqlInstance, error) {
	basedir, err := installMysql8()
	if err != nil {
		return nil, err
	}
	if err := initMysql8(basedir); err != nil {
		return nil, err
	}
	return startMysql8(basedir)
}

// Port returns the port of the mysql instance.
func (instance MysqlInstance) Port() int { return instance.port }

// Purge removes the mysql instance.
func (instance *MysqlInstance) Purge() error {
	if err := instance.proc.Kill(); err != nil {
		return err
	}
	return os.RemoveAll(instance.dir)
}

//go:embed mysql-8.0.28-macos11-arm64.tar.gz
var mysqlResources embed.FS

// installMysql8 extracts mysql distrubution to a temporary directory,
// returns the directory where the mysql is installed.
func installMysql8() (string, error) {
	var _os string
	switch runtime.GOOS {
	case "darwin":
		_os = "macos11"
	default:
		return "", fmt.Errorf("unsupported os %q", runtime.GOOS)
	}
	var arch string
	switch runtime.GOARCH {
	case "arm64":
		arch = "arm64"
	default:
		return "", fmt.Errorf("unsupported arch %q", runtime.GOARCH)
	}
	distName := fmt.Sprintf("mysql-8.0.28-%s-%s", _os, arch)

	tarGzF, err := mysqlResources.Open(distName + ".tar.gz")
	if err != nil {
		return "", fmt.Errorf("failed to open mysql dist %q, error: %w", distName, err)
	}
	defer tarGzF.Close()

	tempDir, err := os.MkdirTemp("", "bytebase-mysql-instance-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory, error: %w", err)
	}

	if err := extractTarGz(tarGzF, tempDir); err != nil {
		return "", fmt.Errorf("failed to extract mysql distribution %q, error: %w", distName, err)
	}

	return filepath.Join(tempDir, distName), nil
}

// extractTarGz extracts the given file as .tar.gz format to the given directory.
func extractTarGz(tarGzF io.Reader, targetDir string) error {
	gzipR, err := gzip.NewReader(tarGzF)
	if err != nil {
		return err
	}
	defer gzipR.Close()
	tarR := tar.NewReader(gzipR)

	for {
		header, err := tarR.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		targetPath := filepath.Join(targetDir, header.Name)
		if err := os.MkdirAll(path.Dir(targetPath), os.ModePerm); err != nil {
			return fmt.Errorf("failed to create directory %q, error: %w", header.Name, err)
		}

		switch header.Typeflag {
		case tar.TypeReg:
			outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			defer outFile.Close()

			if _, err := io.Copy(outFile, tarR); err != nil {
				return err
			}
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeSymlink:
			if err := os.Symlink(header.Linkname, targetPath); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported type flag %d", header.Typeflag)
		}
	}

	return nil
}

const configFmt = `[mysqld]
basedir=%s
datadir=data
pid-file=mysql.pid
socket=mysql.sock
user=root
`

func initMysql8(basedir string) error {
	datadir := filepath.Join(basedir, "data")
	if err := os.MkdirAll(datadir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create data directory %q, error: %w", datadir, err)
	}

	socket, err := os.Create(filepath.Join(basedir, "mysql.sock"))
	if err != nil {
		return err
	}
	socket.Close()

	pidFile, err := os.Create(filepath.Join(basedir, "mysql.pid"))
	if err != nil {
		return err
	}
	pidFile.Close()

	defaultCfgFile, err := os.Create(filepath.Join(basedir, "my.cnf"))
	if err != nil {
		return err
	}
	fmt.Fprintf(defaultCfgFile, configFmt, basedir)
	defaultCfgFile.Close()

	args := []string{
		fmt.Sprintf("--defaults-file=%s", defaultCfgFile.Name()),
		"--initialize-insecure",
	}

	cmd := exec.Command(filepath.Join(basedir, "bin", "mysqld"), args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to initialize mysql, error: %w", err)
	}

	return nil
}

func startMysql8(installedDir string) (*MysqlInstance, error) {
	port, err := randomUnusedPort()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(filepath.Join(installedDir, "bin", "mysqld"),
		fmt.Sprintf("--defaults-file=%s", filepath.Join(installedDir, "my.cnf")),
		fmt.Sprintf("--port=%d", port),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	db, err := sql.Open("mysql", fmt.Sprintf("root@tcp(localhost:%d)/mysql", port))
	if err != nil {
		return nil, err
	}
	for retry := 0; true; retry++ {
		if retry > 60 {
			return nil, fmt.Errorf("failed to connect to mysql after 60 seconds")
		}
		if err := db.Ping(); err == nil {
			break
		}
		time.Sleep(time.Second)
	}

	return &MysqlInstance{
		dir:  installedDir,
		port: port,
		proc: cmd.Process,
	}, nil
}

// randomUnusedPort returns a random port that is not in use.
func randomUnusedPort() (int, error) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer l.Close()

	return l.Addr().(*net.TCPAddr).Port, nil
}
