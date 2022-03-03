package resources

import (
	"database/sql"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	// install mysql driver
	_ "github.com/go-sql-driver/mysql"
)

// MysqlInstance is MySQL instance installed by bytebase for testing.
type MysqlInstance struct {
	// basedir is the directory where the mysql binary is installed.
	basedir string
	// datadir is the directory where the mysql data is stored.
	datadir string
	// port is the port of the mysql instance.
	port int
	// proc is the process of the mysql instance.
	proc *os.Process
}

// Port returns the port of the mysql instance.
func (i MysqlInstance) Port() int { return i.port }

// Start starts the mysql instance on the given port, outputs to stdout and stderr.
// Waits at most `waitSec` seconds for the mysql instance to ready for connection.
// If `waitSec` is 0, it returns immediately.
func (i *MysqlInstance) Start(port int, stdout, stderr io.Writer, waitSec int) (err error) {
	i.port = port
	if i.port == 0 {
		i.port, err = randomUnusedPort()
		if err != nil {
			return err
		}
	}

	cmd := exec.Command(filepath.Join(i.basedir, "bin", "mysqld"),
		fmt.Sprintf("--defaults-file=%s", filepath.Join(i.basedir, "my.cnf")),
		fmt.Sprintf("--port=%d", i.port),
	)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	i.proc = cmd.Process

	// wait for mysql to start
	db, err := sql.Open("mysql", fmt.Sprintf("root@tcp(localhost:%d)/mysql", i.port))
	if err != nil {
		return err
	}
	for retry := 0; waitSec > 0; retry++ {
		if err := db.Ping(); err == nil {
			break
		}
		if retry > waitSec {
			return fmt.Errorf("failed to connect to mysql, error: %w", err)
		}
		time.Sleep(time.Second)
	}

	return nil
}

// Stop stops the mysql instance, outputs to stdout and stderr.
func (i *MysqlInstance) Stop(stdout, stderr io.Writer) error { return i.proc.Kill() }

// InstallMysql installs mysql on basedir, prepares the data directory and default user.
func InstallMysql(basedir, datadir, user string) (*MysqlInstance, error) {
	var tarName, version string
	// Mysql uses both tar.gz and tar.xz, so we use this ugly hack.
	var extractFn func(io.Reader, string) error
	switch {
	case runtime.GOOS == "darwin" && runtime.GOARCH == "arm64":
		tarName = "mysql-8.0.28-macos11-arm64.tar.gz"
		version = "mysql-8.0.28-macos11-arm64"
		extractFn = extractTarGz
	case runtime.GOOS == "linux" && runtime.GOARCH == "amd64":
		tarName = "mysql-8.0.28-linux-glibc2.17-x86_64-minimal.tar.xz"
		version = "mysql-8.0.28-linux-glibc2.17-x86_64-minimal"
		extractFn = extractTarXz
	default:
		return nil, fmt.Errorf("unsupported os %q and arch %q", runtime.GOOS, runtime.GOARCH)
	}

	tarF, err := resources.Open(tarName)
	if err != nil {
		return nil, fmt.Errorf("failed to open mysql dist %q, error: %w", tarName, err)
	}
	defer tarF.Close()

	if err := extractFn(tarF, basedir); err != nil {
		return nil, fmt.Errorf("failed to extract mysql distribution %q, error: %w", tarName, err)
	}

	basedir = filepath.Join(basedir, version)

	socket, err := os.Create(filepath.Join(basedir, "mysql.sock"))
	if err != nil {
		return nil, err
	}
	socket.Close()

	pidFile, err := os.Create(filepath.Join(basedir, "mysql.pid"))
	if err != nil {
		return nil, err
	}
	pidFile.Close()

	const configFmt = `[mysqld]
basedir=%s
datadir=%s
pid-file=mysql.pid
socket=mysql.sock
user=%s
`
	defaultCfgFile, err := os.Create(filepath.Join(basedir, "my.cnf"))
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(defaultCfgFile, configFmt, basedir, datadir, user)
	defaultCfgFile.Close()

	args := []string{
		fmt.Sprintf("--defaults-file=%s", defaultCfgFile.Name()),
		"--initialize-insecure",
	}

	cmd := exec.Command(filepath.Join(basedir, "bin", "mysqld"), args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to initialize mysql, error: %w", err)
	}

	return &MysqlInstance{
		basedir: basedir,
		datadir: datadir,
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
