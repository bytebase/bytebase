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
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const (
	header = `
SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

`

	setDefaultTableSpace        = "SET default_tablespace = '';\n\n"
	setDefaultTableAccessMethod = "SET default_table_access_method = heap;\n\n"
)

// Dump dumps the database.
func (*Driver) Dump(_ context.Context, out io.Writer, metadata *storepb.DatabaseSchemaMetadata) error {
	if len(metadata.Schemas) == 0 {
		return nil
	}

	if _, err := io.WriteString(out, header); err != nil {
		return err
	}

	// Construct schemas.
	for _, schema := range metadata.Schemas {
		if err := writeSchema(out, schema); err != nil {
			return err
		}
	}

	// Construct extensions.
	for _, extension := range metadata.Extensions {
		if err := writeExtension(out, extension); err != nil {
			return err
		}
	}

	// Construct functions.
	for _, schema := range metadata.Schemas {
		for _, function := range schema.Functions {
			if err := writeFunction(out, function); err != nil {
				return err
			}
		}
	}

	// Mapping from table ID to sequence metadata.
	// Construct none owner column sequences first.
	sequenceMap := make(map[string][]*storepb.SequenceMetadata)
	for _, schema := range metadata.Schemas {
		for _, sequence := range schema.Sequences {
			if sequence.OwnerTable == "" || sequence.OwnerColumn == "" {
				if err := writeCreateSequence(out, schema.Name, sequence); err != nil {
					return err
				}
				continue
			}
			tableID := getTableID(schema.Name, sequence.OwnerTable)
			sequenceMap[tableID] = append(sequenceMap[tableID], sequence)
		}
	}

	if _, err := io.WriteString(out, setDefaultTableSpace); err != nil {
		return err
	}

	if _, err := io.WriteString(out, setDefaultTableAccessMethod); err != nil {
		return err
	}

	// Construct tables.
	for _, schema := range metadata.Schemas {
		for _, table := range schema.Tables {
			if err := writeTable(out, schema.Name, table, sequenceMap[getTableID(schema.Name, table.Name)]); err != nil {
				return err
			}
		}
	}

	graph := NewGraph()
	viewMap := make(map[string]*storepb.ViewMetadata)
	materializedViewMap := make(map[string]*storepb.MaterializedViewMetadata)

	// Build the graph for topological sort.
	for _, schema := range metadata.Schemas {
		for _, view := range schema.Views {
			viewID := getTableID(schema.Name, view.Name)
			viewMap[viewID] = view
			graph.AddNode(viewID)
			for _, dependency := range view.DependentColumns {
				dependencyID := getTableID(dependency.Schema, dependency.Table)
				graph.AddEdge(dependencyID, viewID)
			}
		}
		for _, view := range schema.MaterializedViews {
			viewID := getTableID(schema.Name, view.Name)
			materializedViewMap[viewID] = view
			graph.AddNode(viewID)
			for _, dependency := range view.DependentColumns {
				dependencyID := getTableID(dependency.Schema, dependency.Table)
				graph.AddEdge(dependencyID, viewID)
			}
		}
	}

	orderedList, err := graph.GetTopoSort()
	if err != nil {
		return errors.Wrap(err, "failed to get topological sort")
	}

	for _, viewID := range orderedList {
		if view, ok := viewMap[viewID]; ok {
			if err := writeView(out, getSchemaNameFromID(viewID), view); err != nil {
				return err
			}
			delete(viewMap, viewID)
			continue
		}
		if view, ok := materializedViewMap[viewID]; ok {
			if err := writeMaterializedView(out, getSchemaNameFromID(viewID), view); err != nil {
				return err
			}
			delete(materializedViewMap, viewID)
		}
	}

	return nil
}

func writeMaterializedView(out io.Writer, schema string, view *storepb.MaterializedViewMetadata) error {
	if _, err := io.WriteString(out, `CREATE MATERIALIZED VIEW "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, view.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\" AS \n"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, view.Definition); err != nil {
		return err
	}
	_, err := io.WriteString(out, "\n  WITH NO DATA;\n\n")
	return err
}

func writeView(out io.Writer, schema string, view *storepb.ViewMetadata) error {
	if _, err := io.WriteString(out, `CREATE VIEW "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, view.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\" AS \n"); err != nil {
		return err
	}
	if _, err := io.WriteString(out, view.Definition); err != nil {
		return err
	}
	_, err := io.WriteString(out, ";\n\n")
	return err
}

func getSchemaNameFromID(id string) string {
	parts := strings.Split(id, ".")
	if len(parts) != 2 {
		return ""
	}
	return parts[0]
}

func writeCreateSequence(out io.Writer, schema string, sequence *storepb.SequenceMetadata) error {
	if _, err := io.WriteString(out, `CREATE SEQUENCE "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, sequence.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\"\n    "); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "AS "); err != nil {
		return err
	}
	if _, err := io.WriteString(out, sequence.DataType); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\n	START WITH "); err != nil {
		return err
	}
	if _, err := io.WriteString(out, sequence.Start); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\n	INCREMENT BY "); err != nil {
		return err
	}
	if _, err := io.WriteString(out, sequence.Increment); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\n	MINVALUE "); err != nil {
		return err
	}
	if _, err := io.WriteString(out, sequence.MinValue); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\n	MAXVALUE "); err != nil {
		return err
	}
	if _, err := io.WriteString(out, sequence.MaxValue); err != nil {
		return err
	}
	if sequence.Cycle {
		if _, err := io.WriteString(out, "\n	CYCLE"); err != nil {
			return err
		}
	} else {
		if _, err := io.WriteString(out, "\n	NO CYCLE"); err != nil {
			return err
		}
	}
	_, err := io.WriteString(out, ";\n\n")
	return err
}

func writeAlterSequenceOwnedBy(out io.Writer, schema string, sequence *storepb.SequenceMetadata) error {
	if _, err := io.WriteString(out, `ALTER SEQUENCE "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, sequence.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, "\" OWNED BY \""); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, sequence.OwnerTable); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, sequence.OwnerColumn); err != nil {
		return err
	}
	_, err := io.WriteString(out, "\";\n\n")
	return err
}

func getTableID(schema string, table string) string {
	var buf strings.Builder
	_, _ = buf.WriteString(schema)
	_, _ = buf.WriteString(".")
	_, _ = buf.WriteString(table)
	return buf.String()
}

func writeCreateTable(out io.Writer, schema string, tableName string, columns []*storepb.ColumnMetadata) error {
	if _, err := io.WriteString(out, `CREATE TABLE "`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, tableName); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `" (`); err != nil {
		return err
	}

	for i, column := range columns {
		if i > 0 {
			if _, err := io.WriteString(out, ","); err != nil {
				return err
			}
		}

		if _, err := io.WriteString(out, "\n    "); err != nil {
			return err
		}

		if _, err := io.WriteString(out, `"`); err != nil {
			return err
		}

		if _, err := io.WriteString(out, column.Name); err != nil {
			return err
		}

		if _, err := io.WriteString(out, `" `); err != nil {
			return err
		}

		if _, err := io.WriteString(out, column.Type); err != nil {
			return err
		}

		if !column.Nullable {
			if _, err := io.WriteString(out, ` NOT NULL`); err != nil {
				return err
			}
		}

		if column.DefaultValue != nil {
			if defaultValue, ok := column.DefaultValue.(*storepb.ColumnMetadata_DefaultExpression); ok {
				if _, err := io.WriteString(out, ` DEFAULT `); err != nil {
					return err
				}
				if _, err := io.WriteString(out, defaultValue.DefaultExpression); err != nil {
					return err
				}
			}
		}
	}

	_, err := io.WriteString(out, "\n)")
	return err
}

func writeTable(out io.Writer, schema string, table *storepb.TableMetadata, sequences []*storepb.SequenceMetadata) error {
	for _, sequence := range sequences {
		if err := writeCreateSequence(out, schema, sequence); err != nil {
			return err
		}
	}

	if err := writeCreateTable(out, schema, table.Name, table.Columns); err != nil {
		return err
	}

	if len(table.Partitions) > 0 {
		if err := writePartitionClause(out, table.Partitions[0]); err != nil {
			return err
		}
	}

	if _, err := io.WriteString(out, ";\n\n"); err != nil {
		return err
	}

	for _, sequence := range sequences {
		if err := writeAlterSequenceOwnedBy(out, schema, sequence); err != nil {
			return err
		}
	}

	// Construct comments.
	if len(table.Comment) > 0 {
		if err := writeTableComment(out, schema, table); err != nil {
			return err
		}
	}

	for _, column := range table.Columns {
		if len(column.Comment) > 0 {
			if err := writeColumnComment(out, schema, table.Name, column); err != nil {
				return err
			}
		}
	}

	// Construct partition tables.
	for _, partition := range table.Partitions {
		if err := writePartitionTable(out, schema, table, partition); err != nil {
			return err
		}
	}

	for _, partition := range table.Partitions {
		if err := writeAttachPartition(out, schema, table, partition); err != nil {
			return err
		}
	}

	// Construct Primary Key.
	for _, index := range table.Indexes {
		if index.Primary {
			if err := writePrimaryKey(out, schema, table.Name, index); err != nil {
				return err
			}
		}
	}

	// Construct Partition table primary key.
	for _, partition := range table.Partitions {
		if err := writePartitionPrimaryKey(out, schema, partition); err != nil {
			return err
		}
	}

	return nil
}

func writePartitionPrimaryKey(out io.Writer, schema string, partition *storepb.TablePartitionMetadata) error {
	for _, index := range partition.Indexes {
		if index.Primary {
			if err := writePrimaryKey(out, schema, partition.Name, index); err != nil {
				return err
			}
		}
	}

	for _, subpartition := range partition.Subpartitions {
		if err := writePartitionPrimaryKey(out, schema, subpartition); err != nil {
			return err
		}
	}
	return nil
}

func writePrimaryKey(out io.Writer, schema string, table string, index *storepb.IndexMetadata) error {
	if _, err := io.WriteString(out, `ALTER TABLE ONLY "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, table); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `" ADD CONSTRAINT "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, index.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `" PRIMARY KEY (`); err != nil {
		return err
	}
	for i, expression := range index.Expressions {
		if i > 0 {
			if _, err := io.WriteString(out, ", "); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(out, expression); err != nil {
			return err
		}
	}
	_, err := io.WriteString(out, ");\n\n")
	return err
}

func writeColumnComment(out io.Writer, schema string, table string, column *storepb.ColumnMetadata) error {
	if _, err := io.WriteString(out, `COMMENT ON COLUMN "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, table); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, column.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `" IS '`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, column.Comment); err != nil {
		return err
	}
	_, err := io.WriteString(out, "';\n\n")
	return err

}

func writeTableComment(out io.Writer, schema string, table *storepb.TableMetadata) error {
	if _, err := io.WriteString(out, `COMMENT ON TABLE "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, table.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `" IS '`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, table.Comment); err != nil {
		return err
	}
	_, err := io.WriteString(out, "';\n\n")
	return err
}

func writePartitionClause(out io.Writer, partition *storepb.TablePartitionMetadata) error {
	if _, err := io.WriteString(out, " PARTITION BY "); err != nil {
		return err
	}
	_, err := io.WriteString(out, partition.Expression)
	return err
}

func writeAttachPartition(out io.Writer, schema string, table *storepb.TableMetadata, partition *storepb.TablePartitionMetadata) error {
	if _, err := io.WriteString(out, `ALTER TABLE ONLY "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, table.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `" ATTACH PARTITION "`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, schema); err != nil {
		return err
	}
	if _, err := io.WriteString(out, `"."`); err != nil {
		return err
	}
	if _, err := io.WriteString(out, partition.Name); err != nil {
		return err
	}
	if _, err := io.WriteString(out, " "); err != nil {
		return err
	}
	if _, err := io.WriteString(out, partition.Value); err != nil {
		return err
	}
	_, err := io.WriteString(out, ";\n\n")
	return err
}

func writePartitionTable(out io.Writer, schema string, table *storepb.TableMetadata, partition *storepb.TablePartitionMetadata) error {
	if err := writeCreateTable(out, schema, partition.Name, table.Columns); err != nil {
		return err
	}

	if len(partition.Subpartitions) > 0 {
		if err := writePartitionClause(out, partition.Subpartitions[0]); err != nil {
			return err
		}
	}

	if _, err := io.WriteString(out, ";\n\n"); err != nil {
		return err
	}

	// Construct subpartition tables.
	for _, subpartition := range partition.Subpartitions {
		if err := writePartitionTable(out, schema, table, subpartition); err != nil {
			return err
		}
	}

	for _, subpartition := range partition.Subpartitions {
		if err := writeAttachPartition(out, schema, table, subpartition); err != nil {
			return err
		}
	}

	return nil
}

func writeFunction(out io.Writer, function *storepb.FunctionMetadata) error {
	if _, err := io.WriteString(out, function.Definition); err != nil {
		return err
	}

	_, err := io.WriteString(out, "\n\n")
	return err
}

func writeExtension(out io.Writer, extension *storepb.ExtensionMetadata) error {
	if _, err := io.WriteString(out, `CREATE EXTENSION IF NOT EXISTS "`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, extension.Name); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `" WITH SCHEMA "`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, extension.Schema); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `";`); err != nil {
		return err
	}

	_, err := io.WriteString(out, "\n\n")
	return err
}

func writeSchema(out io.Writer, schema *storepb.SchemaMetadata) error {
	if schema.Name == "public" {
		return nil
	}

	if _, err := io.WriteString(out, `CREATE SCHEMA "`); err != nil {
		return err
	}

	if _, err := io.WriteString(out, schema.Name); err != nil {
		return err
	}

	if _, err := io.WriteString(out, `";`); err != nil {
		return err
	}

	_, err := io.WriteString(out, "\n\n")
	return err
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
