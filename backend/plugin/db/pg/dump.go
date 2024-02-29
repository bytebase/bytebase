package pg

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

// Dump dumps the database.
func (driver *Driver) Dump(ctx context.Context, out io.Writer, schemaOnly bool) (string, error) {
	// pg_dump -d dbName --schema-only+

	// Find all dumpable databases
	databases, err := driver.getDatabases(ctx)
	if err != nil {
		return "", errors.Wrap(err, "failed to get databases")
	}

	var dumpableDbNames []string
	if driver.databaseName != "" {
		exist := false
		for _, n := range databases {
			if n.Name == driver.databaseName {
				exist = true
				break
			}
		}
		if !exist {
			return "", errors.Errorf("database %s not found", driver.databaseName)
		}
		dumpableDbNames = []string{driver.databaseName}
	} else {
		for _, n := range databases {
			if pgparser.IsSystemDatabase(n.Name) {
				continue
			}
			dumpableDbNames = append(dumpableDbNames, n.Name)
		}
	}

	for _, dbName := range dumpableDbNames {
		if err := driver.dumpOneDatabaseWithPgDump(ctx, dbName, out, schemaOnly); err != nil {
			return "", err
		}
	}

	return "", nil
}

func (driver *Driver) dumpOneDatabaseWithPgDump(ctx context.Context, database string, out io.Writer, schemaOnly bool) error {
	var args []string
	args = append(args, fmt.Sprintf("--username=%s", driver.config.Username))
	if driver.config.Password == "" {
		args = append(args, "--no-password")
	}
	if driver.sshClient == nil {
		args = append(args, fmt.Sprintf("--host=%s", driver.config.Host))
		args = append(args, fmt.Sprintf("--port=%s", driver.config.Port))
	} else {
		localPort := <-util.PortFIFO
		defer func() {
			util.PortFIFO <- localPort
		}()
		args = append(args, fmt.Sprintf("--host=%s", "localhost"))
		args = append(args, fmt.Sprintf("--port=%d", localPort))
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", localPort))
		if err != nil {
			return err
		}
		defer listener.Close()
		databaseAddress := fmt.Sprintf("%s:%s", driver.config.Host, driver.config.Port)
		go util.ProxyConnection(driver.sshClient, listener, databaseAddress)
	}
	if schemaOnly {
		args = append(args, "--schema-only")
	}
	args = append(args, "--inserts")
	args = append(args, "--use-set-session-authorization")
	// Avoid pg_dump v15 generate "ALTER SCHEMA public OWNER TO" statement.
	args = append(args, "--no-owner")
	// Avoid pg_dump v15 generate REVOKE/GRANT statement.
	args = append(args, "--no-privileges")
	args = append(args, database)

	if err := driver.execPgDump(ctx, args, out); err != nil {
		return errors.Wrapf(err, "failed to exec pg_dump")
	}
	return nil
}

func (driver *Driver) execPgDump(ctx context.Context, args []string, out io.Writer) error {
	pgDumpPath := filepath.Join(driver.dbBinDir, "pg_dump")
	cmd := exec.CommandContext(ctx, pgDumpPath, args...)
	// Unlike MySQL, PostgreSQL does not support specifying password in commands, we can do this by means of environment variables.
	if driver.config.Password != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("PGPASSWORD=%s", driver.config.Password))
	}
	sslMode := getSSLMode(driver.config.TLSConfig, driver.config.SSHConfig)
	cmd.Env = append(cmd.Env, fmt.Sprintf("PGSSLMODE=%s", sslMode))

	if driver.config.TLSConfig.SslCA != "" {
		caTempFile, err := os.CreateTemp("", "pg-ssl-ca-")
		if err != nil {
			return err
		}
		defer os.Remove(caTempFile.Name())
		if _, err := caTempFile.WriteString(driver.config.TLSConfig.SslCA); err != nil {
			return err
		}
		if err := caTempFile.Close(); err != nil {
			return err
		}
		cmd.Env = append(cmd.Env, fmt.Sprintf("PGSSLROOTCERT=%s", caTempFile.Name()))
	}
	if driver.config.TLSConfig.SslCert != "" {
		certTempFile, err := os.CreateTemp("", "pg-ssl-cert-")
		if err != nil {
			return err
		}
		defer os.Remove(certTempFile.Name())
		if _, err := certTempFile.WriteString(driver.config.TLSConfig.SslCert); err != nil {
			return err
		}
		if err := certTempFile.Close(); err != nil {
			return err
		}
		cmd.Env = append(cmd.Env, fmt.Sprintf("PGSSLCERT=%s", certTempFile.Name()))
	}
	if driver.config.TLSConfig.SslKey != "" {
		keyTempFile, err := os.CreateTemp("", "pg-ssl-key-")
		if err != nil {
			return err
		}
		defer os.Remove(keyTempFile.Name())
		if _, err := keyTempFile.WriteString(driver.config.TLSConfig.SslKey); err != nil {
			return err
		}
		if err := keyTempFile.Close(); err != nil {
			return err
		}
		cmd.Env = append(cmd.Env, fmt.Sprintf("PGSSLKEY=%s", keyTempFile.Name()))
	}
	cmd.Env = append(cmd.Env, "OPENSSL_CONF=/etc/ssl/")
	outPipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	defer outPipe.Close()
	outReader := bufio.NewReader(outPipe)

	errPipe, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	defer errPipe.Close()
	errReader := bufio.NewReader(errPipe)

	if err := cmd.Start(); err != nil {
		return err
	}
	previousLineComment := false
	previousLineEmpty := false
	for {
		line, readErr := outReader.ReadString('\n')
		if readErr != nil && readErr != io.EOF {
			return readErr
		}

		if err := func() error {
			// Skip "SET SESSION AUTHORIZATION" till we can support it.
			if strings.HasPrefix(line, "SET SESSION AUTHORIZATION ") {
				return nil
			}
			// Skip comment lines.
			if strings.HasPrefix(line, "--") {
				previousLineComment = true
				return nil
			}
			if previousLineComment && line == "" {
				previousLineComment = false
				return nil
			}
			previousLineComment = false
			// Skip extra empty lines.
			if strings.TrimSpace(line) == "" {
				if previousLineEmpty {
					return nil
				}
				previousLineEmpty = true
			} else {
				previousLineEmpty = false
			}

			if _, err := io.WriteString(out, line); err != nil {
				return err
			}

			return nil
		}(); err != nil {
			return err
		}

		if readErr == io.EOF {
			break
		}
	}

	// We init allMsg with 1024 bytes cap to avoid \x00 in the error message.
	allMsg := make([]byte, 0, 1024)
	for {
		errorMsg := make([]byte, 1024)
		readSize, readErr := errReader.Read(errorMsg)
		if readSize > 0 {
			slog.Warn(string(errorMsg))
			allMsg = append(allMsg, errorMsg[0:readSize]...)
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return err
		}
		// We may store the error message in meta store, due to the performance concern, we only store the first 1024 bytes.
		if len(allMsg) >= 1024 {
			break
		}
	}
	err = cmd.Wait()
	if err != nil {
		return errors.Wrapf(err, "error message: %s", allMsg)
	}
	return nil
}

// Restore restores a database.
func (driver *Driver) Restore(ctx context.Context, sc io.Reader) error {
	buf := new(strings.Builder)
	if _, err := io.Copy(buf, sc); err != nil {
		return err
	}
	if _, err := driver.Execute(ctx, buf.String(), db.ExecuteOptions{}); err != nil {
		return err
	}
	return nil
}
