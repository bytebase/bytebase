package taskrun

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/github/gh-ost/go/logic"
	gomysql "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/ghost"
	"github.com/bytebase/bytebase/backend/component/state"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

// NewSchemaUpdateGhostExecutor creates a schema update (gh-ost) task executor.
func NewSchemaUpdateGhostExecutor(s *store.Store, dbFactory *dbfactory.DBFactory, license *enterprise.LicenseService, stateCfg *state.State, schemaSyncer *schemasync.Syncer, profile *config.Profile) Executor {
	return &SchemaUpdateGhostExecutor{
		s:            s,
		dbFactory:    dbFactory,
		license:      license,
		stateCfg:     stateCfg,
		schemaSyncer: schemaSyncer,
		profile:      profile,
	}
}

// SchemaUpdateGhostExecutor is the schema update (gh-ost) task executor.
type SchemaUpdateGhostExecutor struct {
	s            *store.Store
	dbFactory    *dbfactory.DBFactory
	license      *enterprise.LicenseService
	stateCfg     *state.State
	schemaSyncer *schemasync.Syncer
	profile      *config.Profile
}

func (exec *SchemaUpdateGhostExecutor) RunOnce(ctx context.Context, driverCtx context.Context, task *store.TaskMessage, taskRunUID int) (bool, *storepb.TaskRunResult, error) {
	sheetID := int(task.Payload.GetSheetId())
	statement, err := exec.s.GetSheetStatementByID(ctx, sheetID)
	if err != nil {
		return true, nil, err
	}
	flags := task.Payload.GetFlags()

	instance, err := exec.s.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &task.InstanceID})
	if err != nil {
		return true, nil, err
	}
	if instance == nil {
		return true, nil, errors.Errorf("instance %s not found", task.InstanceID)
	}
	database, err := exec.s.GetDatabaseV2(ctx, &store.FindDatabaseMessage{InstanceID: &task.InstanceID, DatabaseName: task.DatabaseName})
	if err != nil {
		return true, nil, err
	}
	if database == nil {
		return true, nil, errors.Errorf("database not found")
	}

	execFunc := func(execCtx context.Context, execStatement string, driver db.Driver, _ db.ExecuteOptions) error {
		// set buffer size to 1 to unblock the sender because there is no listener if the task is canceled.
		migrationError := make(chan error, 1)

		statement := strings.TrimSpace(execStatement)
		// Trim trailing semicolons.
		statement = strings.TrimRight(statement, ";")

		tableName, err := ghost.GetTableNameFromStatement(statement)
		if err != nil {
			return err
		}

		adminDataSource := utils.DataSourceFromInstanceWithType(instance, storepb.DataSourceType_ADMIN)
		if adminDataSource == nil {
			return common.Errorf(common.Internal, "admin data source not found for instance %s", instance.ResourceID)
		}

		migrationContext, err := ghost.NewMigrationContext(ctx, task.ID, database, adminDataSource, tableName, fmt.Sprintf("_%d", time.Now().Unix()), execStatement, false, flags, 10000000)
		if err != nil {
			return errors.Wrap(err, "failed to init migrationContext for gh-ost")
		}
		defer func() {
			// Use migrationContext.Uuid as the tls_config_key by convention.
			// We need to deregister it when gh-ost exits.
			// https://github.com/bytebase/gh-ost2/pull/4
			gomysql.DeregisterTLSConfig(migrationContext.Uuid)
		}()

		// Pre-flight validation: Test binlog access before starting migration
		if err := validateBinlogAccess(execCtx, driver, adminDataSource); err != nil {
			return errors.Wrap(err, "pre-flight check failed: cannot access binlog. Please verify that the user has REPLICATION SLAVE privilege and authentication is correct")
		}

		migrator := logic.NewMigrator(migrationContext, "bb")

		defer func() {
			if err := func() error {
				ctx := context.Background()

				// Use the backup database name of MySQL as the ghost database name.
				ghostDBName := common.BackupDatabaseNameOfEngine(storepb.Engine_MYSQL)
				sql := fmt.Sprintf("DROP TABLE IF EXISTS `%s`.`%s`; DROP TABLE IF EXISTS `%s`.`%s`;",
					ghostDBName,
					migrationContext.GetGhostTableName(),
					ghostDBName,
					migrationContext.GetChangelogTableName(),
				)

				if _, err := driver.GetDB().ExecContext(ctx, sql); err != nil {
					return errors.Wrapf(err, "failed to drop gh-ost temp tables")
				}
				return nil
			}(); err != nil {
				slog.Warn("failed to cleanup gh-ost temp tables", log.BBError(err))
			}
		}()

		go func() {
			if err := migrator.Migrate(); err != nil {
				slog.Error("failed to run gh-ost migration", log.BBError(err))
				migrationError <- err
				return
			}
			migrationError <- nil
		}()

		select {
		case err := <-migrationError:
			if err != nil {
				return err
			}
			return nil
		case <-execCtx.Done():
			migrationContext.PanicAbort <- errors.New("task canceled")
			return errors.New("task canceled")
		}
	}

	return runMigrationWithFunc(ctx, driverCtx, exec.s, exec.dbFactory, exec.stateCfg, exec.schemaSyncer, exec.profile, task, taskRunUID, statement, task.Payload.GetSchemaVersion(), &sheetID, execFunc)
}

// validateBinlogAccess tests whether the user has proper permissions for binlog replication
func validateBinlogAccess(ctx context.Context, driver db.Driver, adminDataSource *storepb.DataSource) error {
	// Test 1: Check if we can read the binary log status
	// This requires REPLICATION CLIENT or SUPER privilege
	if _, err := driver.GetDB().ExecContext(ctx, "SHOW MASTER STATUS"); err != nil {
		// Try SHOW BINARY LOG STATUS for MySQL 8.2.0+ which replaced SHOW MASTER STATUS
		if _, err2 := driver.GetDB().ExecContext(ctx, "SHOW BINARY LOG STATUS"); err2 != nil {
			slog.Error("binlog access validation failed",
				slog.String("host", adminDataSource.GetHost()),
				slog.String("user", adminDataSource.GetUsername()),
				log.BBError(err))
			return errors.New("cannot access binary logs - SHOW MASTER STATUS/SHOW BINARY LOG STATUS failed")
		}
	}

	// Test 2: Check if binary logging is enabled
	var logBin string
	row := driver.GetDB().QueryRowContext(ctx, "SELECT @@log_bin")
	if err := row.Scan(&logBin); err != nil {
		return errors.Wrap(err, "failed to check if binary logging is enabled")
	}
	if logBin != "1" && logBin != "ON" {
		return errors.New("binary logging is not enabled on this MySQL instance")
	}

	// Test 3: Check if we have REPLICATION SLAVE privilege
	// This is the critical permission for gh-ost to work
	rows, err := driver.GetDB().QueryContext(ctx, "SHOW GRANTS")
	if err != nil {
		return errors.Wrap(err, "failed to check user grants")
	}
	defer rows.Close()

	hasReplicationSlave := false
	var grants []string
	for rows.Next() {
		var grant string
		if err := rows.Scan(&grant); err != nil {
			continue
		}
		grants = append(grants, grant)
		if strings.Contains(strings.ToUpper(grant), "REPLICATION SLAVE") ||
			strings.Contains(strings.ToUpper(grant), "ALL PRIVILEGES") {
			hasReplicationSlave = true
			break
		}
	}
	if err := rows.Err(); err != nil {
		return errors.Wrap(err, "error reading grants")
	}

	if !hasReplicationSlave {
		slog.Error("missing REPLICATION SLAVE privilege",
			slog.String("host", adminDataSource.GetHost()),
			slog.String("user", adminDataSource.GetUsername()),
			slog.Any("grants", grants))
		return errors.New("user does not have REPLICATION SLAVE privilege required for gh-ost")
	}

	// Test 4: Check binlog format (should be ROW or MIXED)
	var binlogFormat string
	row = driver.GetDB().QueryRowContext(ctx, "SELECT @@binlog_format")
	if err := row.Scan(&binlogFormat); err != nil {
		return errors.Wrap(err, "failed to check binlog format")
	}
	if strings.ToUpper(binlogFormat) == "STATEMENT" {
		return errors.Errorf("binlog_format is STATEMENT, but gh-ost requires ROW or MIXED format")
	}

	slog.Info("binlog access validation passed",
		slog.String("host", adminDataSource.GetHost()),
		slog.String("user", adminDataSource.GetUsername()),
		slog.String("binlog_format", binlogFormat))

	return nil
}
