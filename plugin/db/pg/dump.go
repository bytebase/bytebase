package pg

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/parser"
)

// Dump dumps the database.
func (driver *Driver) Dump(ctx context.Context, database string, out io.Writer, schemaOnly bool) (string, error) {
	// pg_dump -d dbName --schema-only+

	// Find all dumpable databases
	databases, err := driver.getDatabases(ctx)
	if err != nil {
		return "", errors.Wrap(err, "failed to get databases")
	}

	var dumpableDbNames []string
	if database != "" {
		exist := false
		for _, n := range databases {
			if n.Name == database {
				exist = true
				break
			}
		}
		if !exist {
			return "", errors.Errorf("database %s not found", database)
		}
		dumpableDbNames = []string{database}
	} else {
		for _, n := range databases {
			if excludedDatabaseList[n.Name] {
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
	args = append(args, fmt.Sprintf("--host=%s", driver.config.Host))
	args = append(args, fmt.Sprintf("--port=%s", driver.config.Port))
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

	pgDumpPath := filepath.Join(driver.dbBinDir, "pg_dump")
	cmd := exec.CommandContext(ctx, pgDumpPath, args...)
	if driver.config.Password != "" {
		// Unlike MySQL, PostgreSQL does not support specifying commands in commands, we can do this by means of environment variables.
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
	readErr := error(nil)
	for {
		if readErr == io.EOF {
			break
		}
		var line string
		line, readErr = outReader.ReadString('\n')
		if readErr != nil && readErr != io.EOF {
			return readErr
		}
		// Skip "SET SESSION AUTHORIZATION" till we can support it.
		if strings.HasPrefix(line, "SET SESSION AUTHORIZATION ") {
			continue
		}
		// Skip comment lines.
		if strings.HasPrefix(line, "--") {
			previousLineComment = true
			continue
		}
		if previousLineComment && line == "" {
			previousLineComment = false
			continue
		}
		previousLineComment = false
		// Skip extra empty lines.
		if strings.TrimSpace(line) == "" {
			if previousLineEmpty {
				continue
			}
			previousLineEmpty = true
		} else {
			previousLineEmpty = false
		}

		if _, err := io.WriteString(out, line); err != nil {
			return err
		}
	}

	var errBuilder strings.Builder
	readErr = error(nil)
	for {
		if readErr == io.EOF {
			break
		}

		var line string
		line, readErr = errReader.ReadString('\n')
		if readErr != nil && readErr != io.EOF {
			return readErr
		}

		// Log the error, but return the first 1024 characters in the error to users.
		log.Warn(line)
		if errBuilder.Len() < 1024 {
			if _, err := errBuilder.WriteString(line); err != nil {
				return err
			}
		}
	}

	err = cmd.Wait()
	if err != nil {
		return errors.Wrapf(err, "error message: %s", errBuilder.String())
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
