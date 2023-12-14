package pg

import (
	"bufio"
	"context"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/multierr"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

// sslCAThreshold is the block size for splitting sslCA.
// we use 120kb as the threshold to avoid argument list too long error.
// https://stackoverflow.com/questions/46897008/why-am-i-getting-e2big-from-exec-when-im-accounting-for-the-arguments-and-the
const sslCAThreshold = 120 * 1024

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

	sslCAs := splitSslCA(driver.config.TLSConfig.SslCA)
	dumpSuccess := false
	var errs error
	for _, sslCA := range sslCAs {
		if err := driver.execPgDump(ctx, args, out, sslCA); err != nil {
			errs = multierr.Append(errs, err)
			slog.Warn("Failed to exec pg_dump", log.BBError(err))
		} else {
			dumpSuccess = true
			slog.Info("pg dump successfully")
			break
		}
	}
	if !dumpSuccess {
		return errors.Errorf("Failed to exec pg_dump, err: %v", errs)
	}
	return nil
}

func (driver *Driver) execPgDump(ctx context.Context, args []string, out io.Writer, sslCA string) error {
	pgDumpPath := filepath.Join(driver.dbBinDir, "pg_dump")
	cmd := exec.CommandContext(ctx, pgDumpPath, args...)
	// Unlike MySQL, PostgreSQL does not support specifying commands in commands, we can do this by means of environment variables.
	if driver.config.Password != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("PGPASSWORD=%s", driver.config.Password))
	}
	if driver.config.TLSConfig.SslCert != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("PGSSLCERT=%s", driver.config.TLSConfig.SslCert))
	}
	if sslCA != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("PGSSLROOTCERT=%s", sslCA))
	}
	if driver.config.TLSConfig.SslKey != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("PGSSLKEY=%s", driver.config.TLSConfig.SslKey))
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

	if _, err := pgparser.SplitMultiSQLStream(sc, f); err != nil {
		return err
	}

	if _, err := txn.ExecContext(ctx, "SET LOCAL ROLE NONE;"); err != nil {
		return errors.Wrap(err, "failed to reset role")
	}

	return txn.Commit()
}

// split large sslCA to multiple smaller sslCAs.
func splitSslCA(sslca string) []string {
	if len(sslca) < sslCAThreshold {
		return []string{sslca}
	}

	var certs []string
	var cert string
	for block, rest := pem.Decode([]byte(sslca)); block != nil; block, rest = pem.Decode(rest) {
		switch block.Type {
		case "CERTIFICATE":
			curCert := string(pem.EncodeToMemory(block))
			if len(cert)+len(curCert) > sslCAThreshold {
				certs = append(certs, cert)
				cert = curCert
			} else {
				cert += curCert
			}
		default:
			slog.Warn("unknown block type when spliting sslca")
		}
	}

	if len(cert) > 0 {
		certs = append(certs, cert)
	}
	return certs
}
