package tests

import (
	"database/sql"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/bytebase/bytebase/resources/mysql"
	"github.com/github/gh-ost/go/base"
	"github.com/github/gh-ost/go/logic"
	ghostsql "github.com/github/gh-ost/go/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/openark/golib/log"
)

// ghostRun runs gh-ost
// ghostRun is the original gh-ost main function with
// some simplications since we are using as a library
func ghostRun(args []string) error {
	os.Args = args
	migrationContext := base.NewMigrationContext()
	flag.StringVar(&migrationContext.InspectorConnectionConfig.Key.Hostname, "host", "127.0.0.1", "MySQL hostname (preferably a replica, not the master)")
	flag.StringVar(&migrationContext.AssumeMasterHostname, "assume-master-host", "", "(optional) explicitly tell gh-ost the identity of the master. Format: some.host.com[:port] This is useful in master-master setups where you wish to pick an explicit master, or in a tungsten-replicator where gh-ost is unable to determine the master")
	flag.IntVar(&migrationContext.InspectorConnectionConfig.Key.Port, "port", 3306, "MySQL port (preferably a replica, not the master)")
	flag.Float64Var(&migrationContext.InspectorConnectionConfig.Timeout, "mysql-timeout", 0.0, "Connect, read and write timeout for MySQL")
	flag.StringVar(&migrationContext.CliUser, "user", "", "MySQL user")
	flag.StringVar(&migrationContext.CliPassword, "password", "", "MySQL password")
	flag.StringVar(&migrationContext.CliMasterUser, "master-user", "", "MySQL user on master, if different from that on replica. Requires --assume-master-host")
	flag.StringVar(&migrationContext.CliMasterPassword, "master-password", "", "MySQL password on master, if different from that on replica. Requires --assume-master-host")
	flag.StringVar(&migrationContext.ConfigFile, "conf", "", "Config file")

	flag.BoolVar(&migrationContext.UseTLS, "ssl", false, "Enable SSL encrypted connections to MySQL hosts")
	flag.StringVar(&migrationContext.TLSCACertificate, "ssl-ca", "", "CA certificate in PEM format for TLS connections to MySQL hosts. Requires --ssl")
	flag.StringVar(&migrationContext.TLSCertificate, "ssl-cert", "", "Certificate in PEM format for TLS connections to MySQL hosts. Requires --ssl")
	flag.StringVar(&migrationContext.TLSKey, "ssl-key", "", "Key in PEM format for TLS connections to MySQL hosts. Requires --ssl")
	flag.BoolVar(&migrationContext.TLSAllowInsecure, "ssl-allow-insecure", false, "Skips verification of MySQL hosts' certificate chain and host name. Requires --ssl")

	flag.StringVar(&migrationContext.DatabaseName, "database", "", "database name (mandatory)")
	flag.StringVar(&migrationContext.OriginalTableName, "table", "", "table name (mandatory)")
	flag.StringVar(&migrationContext.AlterStatement, "alter", "", "alter statement (mandatory)")
	flag.BoolVar(&migrationContext.CountTableRows, "exact-rowcount", false, "actually count table rows as opposed to estimate them (results in more accurate progress estimation)")
	flag.BoolVar(&migrationContext.ConcurrentCountTableRows, "concurrent-rowcount", true, "(with --exact-rowcount), when true (default): count rows after row-copy begins, concurrently, and adjust row estimate later on; when false: first count rows, then start row copy")
	flag.BoolVar(&migrationContext.AllowedRunningOnMaster, "allow-on-master", false, "allow this migration to run directly on master. Preferably it would run on a replica")
	flag.BoolVar(&migrationContext.AllowedMasterMaster, "allow-master-master", false, "explicitly allow running in a master-master setup")
	flag.BoolVar(&migrationContext.NullableUniqueKeyAllowed, "allow-nullable-unique-key", false, "allow gh-ost to migrate based on a unique key with nullable columns. As long as no NULL values exist, this should be OK. If NULL values exist in chosen key, data may be corrupted. Use at your own risk!")
	flag.BoolVar(&migrationContext.ApproveRenamedColumns, "approve-renamed-columns", false, "in case your `ALTER` statement renames columns, gh-ost will note that and offer its interpretation of the rename. By default gh-ost does not proceed to execute. This flag approves that gh-ost's interpretation is correct")
	flag.BoolVar(&migrationContext.SkipRenamedColumns, "skip-renamed-columns", false, "in case your `ALTER` statement renames columns, gh-ost will note that and offer its interpretation of the rename. By default gh-ost does not proceed to execute. This flag tells gh-ost to skip the renamed columns, i.e. to treat what gh-ost thinks are renamed columns as unrelated columns. NOTE: you may lose column data")
	flag.BoolVar(&migrationContext.IsTungsten, "tungsten", false, "explicitly let gh-ost know that you are running on a tungsten-replication based topology (you are likely to also provide --assume-master-host)")
	flag.BoolVar(&migrationContext.DiscardForeignKeys, "discard-foreign-keys", false, "DANGER! This flag will migrate a table that has foreign keys and will NOT create foreign keys on the ghost table, thus your altered table will have NO foreign keys. This is useful for intentional dropping of foreign keys")
	flag.BoolVar(&migrationContext.SkipForeignKeyChecks, "skip-foreign-key-checks", false, "set to 'true' when you know for certain there are no foreign keys on your table, and wish to skip the time it takes for gh-ost to verify that")
	flag.BoolVar(&migrationContext.SkipStrictMode, "skip-strict-mode", false, "explicitly tell gh-ost binlog applier not to enforce strict sql mode")
	flag.BoolVar(&migrationContext.AliyunRDS, "aliyun-rds", false, "set to 'true' when you execute on Aliyun RDS.")
	flag.BoolVar(&migrationContext.GoogleCloudPlatform, "gcp", false, "set to 'true' when you execute on a 1st generation Google Cloud Platform (GCP).")
	flag.BoolVar(&migrationContext.AzureMySQL, "azure", false, "set to 'true' when you execute on Azure Database on MySQL.")

	executeFlag := flag.Bool("execute", false, "actually execute the alter & migrate the table. Default is noop: do some tests and exit")
	flag.BoolVar(&migrationContext.TestOnReplica, "test-on-replica", false, "Have the migration run on a replica, not on the master. At the end of migration replication is stopped, and tables are swapped and immediately swap-revert. Replication remains stopped and you can compare the two tables for building trust")
	flag.BoolVar(&migrationContext.TestOnReplicaSkipReplicaStop, "test-on-replica-skip-replica-stop", false, "When --test-on-replica is enabled, do not issue commands stop replication (requires --test-on-replica)")
	flag.BoolVar(&migrationContext.MigrateOnReplica, "migrate-on-replica", false, "Have the migration run on a replica, not on the master. This will do the full migration on the replica including cut-over (as opposed to --test-on-replica)")

	flag.BoolVar(&migrationContext.OkToDropTable, "ok-to-drop-table", false, "Shall the tool drop the old table at end of operation. DROPping tables can be a long locking operation, which is why I'm not doing it by default. I'm an online tool, yes?")
	flag.BoolVar(&migrationContext.InitiallyDropOldTable, "initially-drop-old-table", false, "Drop a possibly existing OLD table (remains from a previous run?) before beginning operation. Default is to panic and abort if such table exists")
	flag.BoolVar(&migrationContext.InitiallyDropGhostTable, "initially-drop-ghost-table", false, "Drop a possibly existing Ghost table (remains from a previous run?) before beginning operation. Default is to panic and abort if such table exists")
	flag.BoolVar(&migrationContext.TimestampOldTable, "timestamp-old-table", false, "Use a timestamp in old table name. This makes old table names unique and non conflicting cross migrations")
	cutOver := flag.String("cut-over", "atomic", "choose cut-over type (default|atomic, two-step)")
	flag.BoolVar(&migrationContext.ForceNamedCutOverCommand, "force-named-cut-over", false, "When true, the 'unpostpone|cut-over' interactive command must name the migrated table")
	flag.BoolVar(&migrationContext.ForceNamedPanicCommand, "force-named-panic", false, "When true, the 'panic' interactive command must name the migrated table")

	flag.BoolVar(&migrationContext.SwitchToRowBinlogFormat, "switch-to-rbr", false, "let this tool automatically switch binary log format to 'ROW' on the replica, if needed. The format will NOT be switched back. I'm too scared to do that, and wish to protect you if you happen to execute another migration while this one is running")
	flag.BoolVar(&migrationContext.AssumeRBR, "assume-rbr", false, "set to 'true' when you know for certain your server uses 'ROW' binlog_format. gh-ost is unable to tell, event after reading binlog_format, whether the replication process does indeed use 'ROW', and restarts replication to be certain RBR setting is applied. Such operation requires SUPER privileges which you might not have. Setting this flag avoids restarting replication and you can proceed to use gh-ost without SUPER privileges")
	flag.BoolVar(&migrationContext.CutOverExponentialBackoff, "cut-over-exponential-backoff", false, "Wait exponentially longer intervals between failed cut-over attempts. Wait intervals obey a maximum configurable with 'exponential-backoff-max-interval').")
	exponentialBackoffMaxInterval := flag.Int64("exponential-backoff-max-interval", 64, "Maximum number of seconds to wait between attempts when performing various operations with exponential backoff.")
	chunkSize := flag.Int64("chunk-size", 1000, "amount of rows to handle in each iteration (allowed range: 10-100,000)")
	dmlBatchSize := flag.Int64("dml-batch-size", 10, "batch size for DML events to apply in a single transaction (range 1-100)")
	defaultRetries := flag.Int64("default-retries", 60, "Default number of retries for various operations before panicking")
	cutOverLockTimeoutSeconds := flag.Int64("cut-over-lock-timeout-seconds", 3, "Max number of seconds to hold locks on tables while attempting to cut-over (retry attempted when lock exceeds timeout)")
	niceRatio := flag.Float64("nice-ratio", 0, "force being 'nice', imply sleep time per chunk time; range: [0.0..100.0]. Example values: 0 is aggressive. 1: for every 1ms spent copying rows, sleep additional 1ms (effectively doubling runtime); 0.7: for every 10ms spend in a rowcopy chunk, spend 7ms sleeping immediately after")

	maxLagMillis := flag.Int64("max-lag-millis", 1500, "replication lag at which to throttle operation")
	replicationLagQuery := flag.String("replication-lag-query", "", "Deprecated. gh-ost uses an internal, subsecond resolution query")
	throttleControlReplicas := flag.String("throttle-control-replicas", "", "List of replicas on which to check for lag; comma delimited. Example: myhost1.com:3306,myhost2.com,myhost3.com:3307")
	throttleQuery := flag.String("throttle-query", "", "when given, issued (every second) to check if operation should throttle. Expecting to return zero for no-throttle, >0 for throttle. Query is issued on the migrated server. Make sure this query is lightweight")
	throttleHTTP := flag.String("throttle-http", "", "when given, gh-ost checks given URL via HEAD request; any response code other than 200 (OK) causes throttling; make sure it has low latency response")
	ignoreHTTPErrors := flag.Bool("ignore-http-errors", false, "ignore HTTP connection errors during throttle check")
	heartbeatIntervalMillis := flag.Int64("heartbeat-interval-millis", 100, "how frequently would gh-ost inject a heartbeat value")
	flag.StringVar(&migrationContext.ThrottleFlagFile, "throttle-flag-file", "", "operation pauses when this file exists; hint: use a file that is specific to the table being altered")
	flag.StringVar(&migrationContext.ThrottleAdditionalFlagFile, "throttle-additional-flag-file", "/tmp/gh-ost.throttle", "operation pauses when this file exists; hint: keep default, use for throttling multiple gh-ost operations")
	flag.StringVar(&migrationContext.PostponeCutOverFlagFile, "postpone-cut-over-flag-file", "", "while this file exists, migration will postpone the final stage of swapping tables, and will keep on syncing the ghost table. Cut-over/swapping would be ready to perform the moment the file is deleted.")
	flag.StringVar(&migrationContext.PanicFlagFile, "panic-flag-file", "", "when this file is created, gh-ost will immediately terminate, without cleanup")

	flag.BoolVar(&migrationContext.DropServeSocket, "initially-drop-socket-file", false, "Should gh-ost forcibly delete an existing socket file. Be careful: this might drop the socket file of a running migration!")
	flag.StringVar(&migrationContext.ServeSocketFile, "serve-socket-file", "", "Unix socket file to serve on. Default: auto-determined and advertised upon startup")
	flag.Int64Var(&migrationContext.ServeTCPPort, "serve-tcp-port", 0, "TCP port to serve on. Default: disabled")

	flag.StringVar(&migrationContext.HooksPath, "hooks-path", "", "directory where hook files are found (default: empty, ie. hooks disabled). Hook files found on this path, and conforming to hook naming conventions will be executed")
	flag.StringVar(&migrationContext.HooksHintMessage, "hooks-hint", "", "arbitrary message to be injected to hooks via GH_OST_HOOKS_HINT, for your convenience")
	flag.StringVar(&migrationContext.HooksHintOwner, "hooks-hint-owner", "", "arbitrary name of owner to be injected to hooks via GH_OST_HOOKS_HINT_OWNER, for your convenience")
	flag.StringVar(&migrationContext.HooksHintToken, "hooks-hint-token", "", "arbitrary token to be injected to hooks via GH_OST_HOOKS_HINT_TOKEN, for your convenience")
	flag.Int64Var(&migrationContext.HooksStatusIntervalSec, "hooks-status-interval", 60, "how many seconds to wait between calling onStatus hook")

	flag.UintVar(&migrationContext.ReplicaServerId, "replica-server-id", 99999, "server id used by gh-ost process. Default: 99999")

	maxLoad := flag.String("max-load", "", "Comma delimited status-name=threshold. e.g: 'Threads_running=100,Threads_connected=500'. When status exceeds threshold, app throttles writes")
	criticalLoad := flag.String("critical-load", "", "Comma delimited status-name=threshold, same format as --max-load. When status exceeds threshold, app panics and quits")
	flag.Int64Var(&migrationContext.CriticalLoadIntervalMilliseconds, "critical-load-interval-millis", 0, "When 0, migration immediately bails out upon meeting critical-load. When non-zero, a second check is done after given interval, and migration only bails out if 2nd check still meets critical load")
	flag.Int64Var(&migrationContext.CriticalLoadHibernateSeconds, "critical-load-hibernate-seconds", 0, "When non-zero, critical-load does not panic and bail out; instead, gh-ost goes into hibernation for the specified duration. It will not read/write anything from/to any server")
	quiet := flag.Bool("quiet", false, "quiet")
	verbose := flag.Bool("verbose", false, "verbose")
	debug := flag.Bool("debug", false, "debug mode (very verbose)")
	stack := flag.Bool("stack", false, "add stack trace upon error")
	help := flag.Bool("help", false, "Display usage")
	checkFlag := flag.Bool("check-flag", false, "Check if another flag exists/supported. This allows for cross-version scripting. Exits with 0 when all additional provided flags exist, nonzero otherwise. You must provide (dummy) values for flags that require a value. Example: gh-ost --check-flag --cut-over-lock-timeout-seconds --nice-ratio 0")
	flag.StringVar(&migrationContext.ForceTmpTableName, "force-table-names", "", "table name prefix to be used on the temporary tables")
	flag.CommandLine.SetOutput(os.Stdout)

	flag.Parse()

	if *checkFlag {
		return nil
	}
	if *help {
		fmt.Fprintf(os.Stdout, "Usage of gh-ost:\n")
		flag.PrintDefaults()
		return nil
	}

	migrationContext.Log.SetLevel(log.ERROR)
	if *verbose {
		migrationContext.Log.SetLevel(log.INFO)
	}
	if *debug {
		migrationContext.Log.SetLevel(log.DEBUG)
	}
	if *stack {
		migrationContext.Log.SetPrintStackTrace(*stack)
	}
	if *quiet {
		// Override!!
		migrationContext.Log.SetLevel(log.ERROR)
	}

	if migrationContext.AlterStatement == "" {
		return fmt.Errorf("--alter must be provided and statement must not be empty")
	}
	parser := ghostsql.NewParserFromAlterStatement(migrationContext.AlterStatement)
	migrationContext.AlterStatementOptions = parser.GetAlterStatementOptions()

	if migrationContext.DatabaseName == "" {
		if parser.HasExplicitSchema() {
			migrationContext.DatabaseName = parser.GetExplicitSchema()
		} else {
			return fmt.Errorf("--database must be provided and database name must not be empty, or --alter must specify database name")
		}
	}

	if err := flag.Set("database", url.QueryEscape(migrationContext.DatabaseName)); err != nil {
		return err
	}

	if migrationContext.OriginalTableName == "" {
		if parser.HasExplicitTable() {
			migrationContext.OriginalTableName = parser.GetExplicitTable()
		} else {
			return fmt.Errorf("--table must be provided and table name must not be empty, or --alter must specify table name")
		}
	}
	migrationContext.Noop = !(*executeFlag)
	if migrationContext.AllowedRunningOnMaster && migrationContext.TestOnReplica {
		return fmt.Errorf("--allow-on-master and --test-on-replica are mutually exclusive")
	}
	if migrationContext.AllowedRunningOnMaster && migrationContext.MigrateOnReplica {
		return fmt.Errorf("--allow-on-master and --migrate-on-replica are mutually exclusive")
	}
	if migrationContext.MigrateOnReplica && migrationContext.TestOnReplica {
		return fmt.Errorf("--migrate-on-replica and --test-on-replica are mutually exclusive")
	}
	if migrationContext.SwitchToRowBinlogFormat && migrationContext.AssumeRBR {
		return fmt.Errorf("--switch-to-rbr and --assume-rbr are mutually exclusive")
	}
	if migrationContext.TestOnReplicaSkipReplicaStop {
		if !migrationContext.TestOnReplica {
			return fmt.Errorf("--test-on-replica-skip-replica-stop requires --test-on-replica to be enabled")
		}
		fmt.Println("--test-on-replica-skip-replica-stop enabled. We will not stop replication before cut-over. Ensure you have a plugin that does this.")
	}
	if migrationContext.CliMasterUser != "" && migrationContext.AssumeMasterHostname == "" {
		return fmt.Errorf("--master-user requires --assume-master-host")
	}
	if migrationContext.CliMasterPassword != "" && migrationContext.AssumeMasterHostname == "" {
		return fmt.Errorf("--master-password requires --assume-master-host")
	}
	if migrationContext.TLSCACertificate != "" && !migrationContext.UseTLS {
		return fmt.Errorf("--ssl-ca requires --ssl")
	}
	if migrationContext.TLSCertificate != "" && !migrationContext.UseTLS {
		return fmt.Errorf("--ssl-cert requires --ssl")
	}
	if migrationContext.TLSKey != "" && !migrationContext.UseTLS {
		return fmt.Errorf("--ssl-key requires --ssl")
	}
	if migrationContext.TLSAllowInsecure && !migrationContext.UseTLS {
		return fmt.Errorf("--ssl-allow-insecure requires --ssl")
	}
	if *replicationLagQuery != "" {
		fmt.Println("--replication-lag-query is deprecated")
	}

	switch *cutOver {
	case "atomic", "default", "":
		migrationContext.CutOverType = base.CutOverAtomic
	case "two-step":
		migrationContext.CutOverType = base.CutOverTwoStep
	default:
		return fmt.Errorf("Unknown cut-over: %s", *cutOver)
	}
	if err := migrationContext.ReadConfigFile(); err != nil {
		return err
	}
	if err := migrationContext.ReadThrottleControlReplicaKeys(*throttleControlReplicas); err != nil {
		return err
	}
	if err := migrationContext.ReadMaxLoad(*maxLoad); err != nil {
		return err
	}
	if err := migrationContext.ReadCriticalLoad(*criticalLoad); err != nil {
		return err
	}
	if migrationContext.ServeSocketFile == "" {
		migrationContext.ServeSocketFile = fmt.Sprintf("/tmp/gh-ost.%s.%s.sock", migrationContext.DatabaseName, migrationContext.OriginalTableName)
	}
	migrationContext.SetHeartbeatIntervalMilliseconds(*heartbeatIntervalMillis)
	migrationContext.SetNiceRatio(*niceRatio)
	migrationContext.SetChunkSize(*chunkSize)
	migrationContext.SetDMLBatchSize(*dmlBatchSize)
	migrationContext.SetMaxLagMillisecondsThrottleThreshold(*maxLagMillis)
	migrationContext.SetThrottleQuery(*throttleQuery)
	migrationContext.SetThrottleHTTP(*throttleHTTP)
	migrationContext.SetIgnoreHTTPErrors(*ignoreHTTPErrors)
	migrationContext.SetDefaultNumRetries(*defaultRetries)
	migrationContext.ApplyCredentials()
	if err := migrationContext.SetupTLS(); err != nil {
		return err
	}
	if err := migrationContext.SetCutOverLockTimeoutSeconds(*cutOverLockTimeoutSeconds); err != nil {
		return err
	}
	if err := migrationContext.SetExponentialBackoffMaxInterval(*exponentialBackoffMaxInterval); err != nil {
		return err
	}

	migrator := logic.NewMigrator(migrationContext)
	err := migrator.Migrate()
	if err != nil {
		_ = migrator.ExecOnFailureHook()
		return err
	}
	fmt.Fprintf(os.Stdout, "# Done\n")
	return nil
}

func TestGhostSimpleNoop(t *testing.T) {
	basedir := t.TempDir()
	datadir := filepath.Join(basedir, "data")
	if err := os.Mkdir(datadir, 0755); err != nil {
		t.Fatal(err)
	}

	mysql, err := mysql.Install(basedir, datadir, "root")
	if err != nil {
		t.Fatalf("failed to start MySQL: %v", err)
	}
	if err := mysql.Start(13306, os.Stdout, os.Stderr, 60); err != nil {
		t.Fatalf("failed to start MySQL: %v", err)
	}

	db, err := sql.Open("mysql", fmt.Sprintf("root@tcp(localhost:%d)/mysql", mysql.Port()))
	if err != nil {
		t.Fatalf("failed to open MySQL: %v", err)
	}
	defer db.Close()

	_, err = db.Exec("CREATE DATABASE gh_ost_test_db")
	if err != nil {
		t.Fatalf("failed to CREATE DATABASE gh_ost_test_db")
	}

	_, err = db.Exec("USE gh_ost_test_db")
	if err != nil {
		t.Fatalf("failed to USE gh_ost_test_db")
	}

	_, err = db.Exec("CREATE TABLE tbl (id int primary key, data int)")
	if err != nil {
		t.Fatalf("failed to CREATE TABLE tbl (id int primary key, data int)")
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("failed to start a transaction: %v", err)
	}
	for i := 1; i <= 100000; i++ {
		_, err := tx.Exec(fmt.Sprintf("INSERT INTO tbl values (%v, %v)", i, i))
		if err != nil {
			t.Fatalf("failed to insert: %v", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	err = ghostRun([]string{"gh-ost", "--host=127.0.0.1", "--port=13306", "--allow-on-master", "--user=root", "--database=gh_ost_test_db", "--table=tbl", "--alter=add name varchar(64)"})

	if err != nil {
		t.Fatalf("failed to run gh-ost: %v", err)
	}

	rows, err := db.Query(`SELECT * FROM tbl`)
	if err != nil {
		t.Fatalf("failed to SELECT * FROM tbl: %v", err)
	}

	var (
		id   int
		data int
	)

	for rows.Next() {
		if err := rows.Scan(&id, &data); err != nil {
			t.Fatalf("failed to scan: %v", err)
		}
		if id != data {
			t.Errorf("data mismatch, expect id: %v, data: %v, get id: %v, data: %v", id, id, id, data)
		}
	}
	rows.Close()

	if err := mysql.Stop(os.Stdout, os.Stderr); err != nil {
		t.Fatalf("failed to stop MySQL: %v", err)
	}
}
