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

	"github.com/bytebase/bytebase/backend/plugin/db/util"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"
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
			if ExcludedDatabaseList[n.Name] {
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
	var dbConnPairs []string
	dbConnPairs = append(dbConnPairs, fmt.Sprintf("user=%s", driver.config.Username))
	if driver.config.Password == "" {
		args = append(args, "--no-password")
	}
	if driver.sshClient == nil {
		dbConnPairs = append(dbConnPairs, fmt.Sprintf("host=%s", driver.config.Host))
		dbConnPairs = append(dbConnPairs, fmt.Sprintf("port=%s", driver.config.Port))
	} else {
		localPort := <-util.PortFIFO
		defer func() {
			util.PortFIFO <- localPort
		}()
		dbConnPairs = append(dbConnPairs, "host=localhost")
		dbConnPairs = append(dbConnPairs, fmt.Sprintf("port=%d", localPort))
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
	dbConnPairs = append(dbConnPairs, fmt.Sprintf("dbname=%s", database))

	if driver.config.TLSConfig.SslCert != "" {
		sslCertFile, err := os.CreateTemp(os.TempDir(), "pgsslcert")
		if err != nil {
			return errors.Wrap(err, "failed to create temporary file to store PG SSL Cert")
		}
		defer os.Remove(sslCertFile.Name())
		if err := sslCertFile.Chmod(0400); err != nil {
			return errors.Wrap(err, "failed to chmod SSL Cert file to 0400")
		}
		if _, err := sslCertFile.WriteString(driver.config.TLSConfig.SslCert); err != nil {
			return errors.Wrap(err, "failed to write SSL Cert to temporary file")
		}
		if err := sslCertFile.Close(); err != nil {
			return errors.Wrap(err, "failed to close SSL Cert temporary file")
		}
		dbConnPairs = append(dbConnPairs, fmt.Sprintf("sslcert=%s", sslCertFile.Name()))
	}
	if driver.config.TLSConfig.SslCA != "" {
		sslRootCertFile, err := os.CreateTemp(os.TempDir(), "pgsslrootcert")
		if err != nil {
			return errors.Wrap(err, "failed to create temporary file to store PG SSL CA")
		}
		defer os.Remove(sslRootCertFile.Name())
		if err := sslRootCertFile.Chmod(0400); err != nil {
			return errors.Wrap(err, "failed to chmod SSL CA file to 0400")
		}
		if _, err := sslRootCertFile.WriteString(driver.config.TLSConfig.SslCA); err != nil {
			return errors.Wrap(err, "failed to write SSL CA to temporary file")
		}
		if err := sslRootCertFile.Close(); err != nil {
			return errors.Wrap(err, "failed to close SSL CA temporary file")
		}
		dbConnPairs = append(dbConnPairs, fmt.Sprintf("sslrootcert=%s", sslRootCertFile.Name()))
	}
	if driver.config.TLSConfig.SslKey != "" {
		sslKeyFile, err := os.CreateTemp(os.TempDir(), "pgsslkey")
		if err != nil {
			return errors.Wrap(err, "failed to create temporary file to store PG SSL Key")
		}
		defer os.Remove(sslKeyFile.Name())
		if err := sslKeyFile.Chmod(0400); err != nil {
			return errors.Wrap(err, "failed to chmod SSL Key file to 0400")
		}
		if _, err := sslKeyFile.WriteString(driver.config.TLSConfig.SslKey); err != nil {
			return errors.Wrap(err, "failed to write SSL Key to temporary file")
		}
		if err := sslKeyFile.Close(); err != nil {
			return errors.Wrap(err, "failed to close SSL Key temporary file")
		}
		dbConnPairs = append(dbConnPairs, fmt.Sprintf("sslkey=%s", sslKeyFile.Name()))
	}

	pgDumpPath := filepath.Join(driver.dbBinDir, "pg_dump")
	dbConnStr := strings.Join(dbConnPairs, " ")
	args = append([]string{dbConnStr}, args...)

	cmd := exec.CommandContext(ctx, pgDumpPath, args...)
	// Unlike MySQL, PostgreSQL does not support specifying password in commands, we can do this by means of environment variables.
	if driver.config.Password != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("PGPASSWORD=%s", driver.config.Password))
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
	txn, err := driver.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer txn.Rollback()

	owner, err := driver.GetCurrentDatabaseOwner()
	if err != nil {
		return errors.Wrapf(err, "failed to get the OWNER of the current database")
	}

	if _, err := txn.ExecContext(ctx, fmt.Sprintf("SET LOCAL ROLE \"%s\";", owner)); err != nil {
		return errors.Wrapf(err, "failed to set role to %q", owner)
	}

	f := func(stmt string) error {
		// CREATE EVENT TRIGGER statement only supports EXECUTE PROCEDURE in version 10 and before, while newer version supports both EXECUTE { FUNCTION | PROCEDURE }.
		// Since we use pg_dump version 14, the dump uses new style even for old version of PostgreSQL.
		// We should convert EXECUTE FUNCTION to EXECUTE PROCEDURE to make the restore to work on old versions.
		// https://www.postgresql.org/docs/14/sql-createeventtrigger.html
		if strings.Contains(strings.ToUpper(stmt), "CREATE EVENT TRIGGER") {
			stmt = strings.ReplaceAll(stmt, "EXECUTE FUNCTION", "EXECUTE PROCEDURE")
		}
		if isSuperuserStatement(stmt) {
			stmt = fmt.Sprintf("SET LOCAL ROLE NONE;%sSET LOCAL ROLE \"%s\";", stmt, owner)
		}
		if isIgnoredStatement(stmt) {
			return nil
		}
		if _, err := txn.Exec(stmt); err != nil {
			return err
		}
		return nil
	}

	if _, err := parser.SplitMultiSQLStream(parser.Postgres, sc, f); err != nil {
		return err
	}

	if _, err := txn.ExecContext(ctx, "SET LOCAL ROLE NONE;"); err != nil {
		return errors.Wrap(err, "failed to reset role")
	}

	return txn.Commit()
}
