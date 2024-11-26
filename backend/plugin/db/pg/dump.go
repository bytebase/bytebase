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

	"github.com/blang/semver/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/db/util"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// Dump dumps the database.
func (driver *Driver) Dump(ctx context.Context, out io.Writer, _ *storepb.DatabaseSchemaMetadata) error {
	// We don't support pg_dump for CloudSQL, because pg_dump not support IAM & instance name for authentication.
	// To dump schema for CloudSQL, you need to run the cloud-sql-proxy with IAM to get the host and port.
	// Learn more: https://linear.app/bytebase/issue/BYT-5401/support-iam-authentication-for-gcp-and-aws
	if driver.config.AuthenticationType == storepb.DataSourceOptions_GOOGLE_CLOUD_SQL_IAM {
		return nil
	}
	// pg_dump -d dbName --schema-only+

	// Find all dumpable databases
	databases, err := driver.getDatabases(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get databases")
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
			return errors.Errorf("database %s not found", driver.databaseName)
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
		if err := driver.dumpOneDatabaseWithPgDump(ctx, dbName, out); err != nil {
			return err
		}
	}

	return nil
}

func (driver *Driver) dumpOneDatabaseWithPgDump(ctx context.Context, database string, out io.Writer) error {
	var args []string
	var host, port string
	if driver.sshClient == nil {
		host = driver.config.Host
		port = driver.config.Port
	} else {
		localPort := <-util.PortFIFO
		defer func() {
			util.PortFIFO <- localPort
		}()
		host = "localhost"
		port = fmt.Sprintf("%d", localPort)
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", localPort))
		if err != nil {
			return err
		}
		defer listener.Close()
		databaseAddress := fmt.Sprintf("%s:%s", driver.config.Host, driver.config.Port)
		go util.ProxyConnection(driver.sshClient, listener, databaseAddress)
	}

	password := driver.config.Password
	if driver.config.AuthenticationType == storepb.DataSourceOptions_AWS_RDS_IAM {
		rdsPassword, err := getRDSConnectionPassword(ctx, driver.config)
		if err != nil {
			return err
		}
		password = rdsPassword
	}

	if password == "" {
		args = append(args, "--no-password")
	}
	connectionString := buildPostgreSQLKeywordValueConnectionString(host, port, driver.config.Username, password, database)
	args = append(args, connectionString)
	args = append(args, "--schema-only")
	args = append(args, "--inserts")
	args = append(args, "--use-set-session-authorization")
	// Avoid pg_dump v15 generate "ALTER SCHEMA public OWNER TO" statement.
	args = append(args, "--no-owner")
	// Avoid pg_dump v15 generate REVOKE/GRANT statement.
	args = append(args, "--no-privileges")

	if err := driver.execPgDump(ctx, args, out); err != nil {
		return errors.Wrapf(err, "failed to exec pg_dump")
	}
	return nil
}

func (driver *Driver) execPgDump(ctx context.Context, args []string, out io.Writer) error {
	version, err := driver.getVersion(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to get version")
	}
	semVersion, err := semver.Make(version)
	if err != nil {
		return errors.Wrapf(err, "failed to parse version %s to semantic version", version)
	}
	atLeast10_0_0 := semVersion.GE(semver.MustParse("10.0.0"))

	pgDumpPath := filepath.Join(driver.dbBinDir, "pg_dump")
	cmd := exec.CommandContext(ctx, pgDumpPath, args...)

	sslMode := getSSLMode(driver.config.TLSConfig, driver.config.SSHConfig)

	// Unfortunately, pg_dump doesn't directly support use system certificate directly. Instead, it supprots
	// specify one cert file path in PGSSLROOTCERT environment variable.
	// MacOS(dev-env):
	// 1. The system certs are stored in the Keychain utility, and it is not recommended to access them outside of Keychain.
	// 2. The user self-signed ca should be add in Keychain utility and mark it is trusted.
	// Debian(our docker image based):
	// 1. The system certs are mostly stored in the /etc/ssl/certs/ca-certificates.crt, and some are stored in the /usr/share/ca-certificates/xxx.crt.
	// 2. The user self-signed ca should be added in /usr/share/ca-certificates/xxx.crt.
	// We should expose this option to user. For now, using require ssl mode to trust server certificate anyway.
	if driver.config.TLSConfig.UseSSL {
		sslMode = SSLModeRequire
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
	}
	cmd.Env = append(cmd.Env, fmt.Sprintf("PGSSLMODE=%s", sslMode))
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
			if !atLeast10_0_0 && strings.Contains(line, "CREATE EVENT TRIGGER") {
				// CREATE EVENT TRIGGER statement only supports EXECUTE PROCEDURE in version 10 and before, while newer version supports both EXECUTE { FUNCTION | PROCEDURE }.
				// Since we use pg_dump >= version 10, the dump uses a new style even for an old version of PostgreSQL.
				// We should convert EXECUTE FUNCTION to EXECUTE PROCEDURE to make the restoration work on old versions.
				// https://www.postgresql.org/docs/14/sql-createeventtrigger.html
				line = strings.ReplaceAll(line, "EXECUTE FUNCTION", "EXECUTE PROCEDURE")
			}
			// Skip "COMMENT ON EXTENSION" till we can support it.
			// Extensions created in AWS Aurora PostgreSQL are owned by rdsadmin.
			// We don't have privileges to comment on the extension and have to ignore it.
			if strings.HasPrefix(line, "COMMENT ON EXTENSION ") {
				previousLineEmpty = true
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

// Learn more: https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING-KEYWORD-VALUE
func buildPostgreSQLKeywordValueConnectionString(host, port, username, password, database string) string {
	pairs := make(map[string]string)
	pairs["user"] = escapeConnectionStringValue(username)
	pairs["password"] = escapeConnectionStringValue(password)
	pairs["host"] = escapeConnectionStringValue(host)
	pairs["port"] = escapeConnectionStringValue(port)
	pairs["dbname"] = escapeConnectionStringValue(database)
	pairs["application_name"] = escapeConnectionStringValue("bytebase dump")

	var items []string
	for key, value := range pairs {
		items = append(items, fmt.Sprintf("%s=%s", key, value))
	}
	return strings.Join(items, " ")
}

// Single quotes and backslashes within a value must be escaped with a backslash, i.e., \' and \\.
func escapeConnectionStringValue(value string) string {
	newValue := strings.ReplaceAll(value, `\`, `\\`)
	newValue = strings.ReplaceAll(newValue, `'`, `\'`)
	return fmt.Sprintf(`'%s'`, newValue)
}
