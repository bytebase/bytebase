package postgres

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/bytebase/bytebase/resources/utils"
)

// Instance is a postgres instance installed by bytebase
// for backend storage or testing.
type Instance struct {
	// basedir is the directory where the postgres binary is installed.
	basedir string
	// datadir is the directory where the postgres data is stored.
	datadir string
	// port is the port number of the postgres instance.
	port int
}

// Port returns the port number of the postgres instance.
func (i Instance) Port() int { return i.port }

// Start starts a postgres instance on given port, outputs to stdout and stderr.
//
// If port is 0, then it will choose a random unused port.
//
// If waitSec > 0, watis at most `waitSec` seconds for the postgres instance to start.
// Otherwise, returns immediately.
func (i *Instance) Start(port int, stdout, stderr io.Writer, waitSec int) (err error) {
	pgbin := filepath.Join(i.basedir, "bin", "pg_ctl")

	i.port = port

	p := exec.Command(pgbin, "start", "-w",
		"-D", i.datadir,
		"-o", fmt.Sprintf(`"-p %d"`, i.port))

	p.Stdout = stdout
	p.Stderr = stderr
	uid, gid, sameUser, err := getBytebaseUser()
	if err != nil {
		return err
	}
	if !sameUser {
		p.SysProcAttr = &syscall.SysProcAttr{
			Credential: &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid), NoSetGroups: true},
		}
	}

	if err := p.Run(); err != nil {
		return fmt.Errorf("failed to start postgres %q, error %v", p.String(), err)
	}

	// TODO
	for retry := 0; waitSec > 0; retry++ {
		break
	}

	return nil
}

// Stop stops a postgres instance, outputs to stdout and stderr.
func (i *Instance) Stop(stdout, stderr io.Writer) error {
	pgbin := filepath.Join(i.basedir, "bin", "pg_ctl")
	p := exec.Command(pgbin, "stop", "-w",
		"-D", i.datadir)

	p.Stderr = stderr
	p.Stdout = stdout
	uid, gid, sameUser, err := getBytebaseUser()
	if err != nil {
		return err
	}
	if !sameUser {
		p.SysProcAttr = &syscall.SysProcAttr{
			Credential: &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid), NoSetGroups: true},
		}
	}

	if err := p.Run(); err != nil {
		return err
	}

	return nil
}

// Install returns the postgres binary depending on the OS.
func Install(resourceDir, pgDataDir, pgUser string) (*Instance, error) {
	uid, gid, _, err := getBytebaseUser()
	if err != nil {
		return nil, err
	}
	// Create resource directory if not exists.
	if _, err := os.Stat(resourceDir); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to check resource directory path %q, error: %w", resourceDir, err)
		}
		if err := os.MkdirAll(resourceDir, os.ModePerm); err != nil {
			return nil, err
		}
		if err := os.Chown(resourceDir, uid, gid); err != nil {
			return nil, fmt.Errorf("failed to change owner to bytebase of resource directory %q, error: %w", resourceDir, err)
		}
	}

	var tarName string
	switch runtime.GOOS {
	case "darwin":
		tarName = "postgres-darwin-x86_64.txz"
	case "linux":
		if isAlpineLinux() {
			tarName = "postgres-linux-x86_64-alpine_linux.txz"
		} else {
			tarName = "postgres-linux-x86_64.txz"
		}
	default:
		return nil, fmt.Errorf("OS %q is not supported", runtime.GOOS)
	}
	log.Printf("Installing Postgres OS %q Arch %q txz %q\n", runtime.GOOS, runtime.GOARCH, tarName)
	f, err := resources.Open(tarName)
	if err != nil {
		return nil, fmt.Errorf("failed to find %q in embedded resources, error: %v", tarName, err)
	}
	defer f.Close()
	version := strings.TrimRight(tarName, ".txz")

	pgBinDir := path.Join(resourceDir, version)
	if _, err := os.Stat(pgBinDir); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to check binary directory path %q, error: %w", pgBinDir, err)
		}
		// Install if not exist yet.
		// The ordering below made Postgres installation atomic.
		tmpDir := path.Join(resourceDir, fmt.Sprintf("tmp-%s", version))
		if err := os.RemoveAll(tmpDir); err != nil {
			return nil, fmt.Errorf("failed to remove postgres binary temp directory %q, error: %w", tmpDir, err)
		}
		if err := utils.ExtractTarXz(f, tmpDir); err != nil {
			return nil, fmt.Errorf("failed to extract txz file, error: %w", err)
		}
		if err := filepath.Walk(tmpDir, func(path string, f os.FileInfo, err error) error {
			if err := os.Chown(path, uid, gid); err != nil {
				return fmt.Errorf("failed to change owner to bytebase of directory %q, error: %w", path, err)
			}
			return nil
		}); err != nil {
			return nil, err
		}

		if err := os.Rename(tmpDir, pgBinDir); err != nil {
			return nil, fmt.Errorf("failed to rename postgres binary directory from %q to %q, error: %w", tmpDir, pgBinDir, err)
		}
	}

	if err := initDB(pgBinDir, pgDataDir, pgUser); err != nil {
		return nil, err
	}

	return &Instance{
		basedir: pgBinDir,
		datadir: pgDataDir,
	}, nil
}

func isAlpineLinux() bool {
	_, err := os.Stat("/etc/alpine-release")
	return err == nil
}

// initDB inits a postgres database.
func initDB(pgBinDir, pgDataDir, pgUser string) error {
	_, err := os.Stat(pgDataDir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to check data directory path %q, error: %w", pgDataDir, err)
	}
	// Skip initDB if setup already.
	if err == nil {
		return nil
	}

	if err := os.MkdirAll(pgDataDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to make postgres data directory %q, error: %w", pgDataDir, err)
	}

	args := []string{
		"-U", pgUser,
		"-D", pgDataDir,
	}
	initDBBinary := filepath.Join(pgBinDir, "bin", "initdb")
	p := exec.Command(initDBBinary, args...)
	p.Env = append(os.Environ(),
		"LC_ALL=en_US.UTF-8",
		"LC_CTYPE=en_US.UTF-8",
	)
	p.Stderr = os.Stderr
	p.Stdout = os.Stdout
	uid, gid, sameUser, err := getBytebaseUser()
	if err != nil {
		return err
	}
	if !sameUser {
		p.SysProcAttr = &syscall.SysProcAttr{
			Credential: &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid), NoSetGroups: true},
		}
		if err := os.Chown(pgDataDir, int(uid), int(gid)); err != nil {
			return fmt.Errorf("failed to change owner to bytebase of data directory %q, error: %w", pgDataDir, err)
		}
	}

	if err := p.Run(); err != nil {
		return fmt.Errorf("failed to initdb %q, error %v", p.String(), err)
	}

	return nil
}

func getBytebaseUser() (int, int, bool, error) {
	sameUser := true
	bytebaseUser, err := user.Current()
	if err != nil {
		return 0, 0, true, fmt.Errorf("failed to get current user, error: %w", err)
	}
	// If user runs bytebase as root user, we will attempt to run as user `bytebase`.
	// https://www.postgresql.org/docs/14/app-initdb.html
	if bytebaseUser.Username == "root" {
		bytebaseUser, err = user.Lookup("bytebase")
		if err != nil {
			return 0, 0, false, fmt.Errorf("Please run Bytebase as non-root user. You can use the following command to create a dedicated bytebase user to run the application: RUN addgroup -g 113 -S bytebase && adduser -u 113 -S -G bytebase bytebase")
		}
		sameUser = false
	}

	uid, err := strconv.Atoi(bytebaseUser.Uid)
	if err != nil {
		return 0, 0, false, err
	}
	gid, err := strconv.Atoi(bytebaseUser.Gid)
	if err != nil {
		return 0, 0, false, err
	}
	return int(uid), int(gid), sameUser, nil
}
