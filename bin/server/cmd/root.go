package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/server"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	// Register clickhouse driver.
	_ "github.com/bytebase/bytebase/plugin/db/clickhouse"
	// Register mysql driver.
	_ "github.com/bytebase/bytebase/plugin/db/mysql"
	// Register postgres driver.
	_ "github.com/bytebase/bytebase/plugin/db/pg"
	// Register snowflake driver.
	_ "github.com/bytebase/bytebase/plugin/db/snowflake"
	// Register sqlite driver.
	_ "github.com/bytebase/bytebase/plugin/db/sqlite"

	// Register pingcap parser driver.
	_ "github.com/pingcap/tidb/types/parser_driver"
	// Register fake advisor.
	_ "github.com/bytebase/bytebase/plugin/advisor/fake"
	// Register mysql advisor.
	_ "github.com/bytebase/bytebase/plugin/advisor/mysql"
)

// -----------------------------------Global constant BEGIN----------------------------------------
const (

	// greetingBanner is the greeting banner.
	// http://patorjk.com/software/taag/#p=display&f=ANSI%20Shadow&t=Bytebase
	greetingBanner = `
██████╗ ██╗   ██╗████████╗███████╗██████╗  █████╗ ███████╗███████╗
██╔══██╗╚██╗ ██╔╝╚══██╔══╝██╔════╝██╔══██╗██╔══██╗██╔════╝██╔════╝
██████╔╝ ╚████╔╝    ██║   █████╗  ██████╔╝███████║███████╗█████╗
██╔══██╗  ╚██╔╝     ██║   ██╔══╝  ██╔══██╗██╔══██║╚════██║██╔══╝
██████╔╝   ██║      ██║   ███████╗██████╔╝██║  ██║███████║███████╗
╚═════╝    ╚═╝      ╚═╝   ╚══════╝╚═════╝ ╚═╝  ╚═╝╚══════╝╚══════╝

%s
___________________________________________________________________________________________

`
	// byeBanner is the bye banner.
	// http://patorjk.com/software/taag/#p=display&f=ANSI%20Shadow&t=BYE
	byeBanner = `
██████╗ ██╗   ██╗███████╗
██╔══██╗╚██╗ ██╔╝██╔════╝
██████╔╝ ╚████╔╝ █████╗
██╔══██╗  ╚██╔╝  ██╔══╝
██████╔╝   ██║   ███████╗
╚═════╝    ╚═╝   ╚══════╝

`
)

// -----------------------------------Global constant END------------------------------------------

// -----------------------------------Command Line Config BEGIN------------------------------------
var (
	flags struct {
		// Used for bytebase command line config
		host         string
		port         int
		frontendHost string
		frontendPort int
		dataDir      string
		// When we are running in readonly mode:
		// - The data file will be opened in readonly mode, no applicable migration or seeding will be applied.
		// - Requests other than GET will be rejected
		// - Any operations involving mutation will not start (e.g. Background schema syncer, task scheduler)
		readonly bool
		demo     bool
		debug    bool
		// pgURL must follow PostgreSQL connection URIs pattern.
		// https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING
		pgURL string
	}
	rootCmd = &cobra.Command{
		Use:   "bytebase",
		Short: "Bytebase is a database schema change and version control tool",
		Run: func(_ *cobra.Command, _ []string) {
			if flags.frontendHost == "" {
				flags.frontendHost = flags.host
			}
			if flags.frontendPort == 0 {
				flags.frontendPort = flags.port
			}

			start()

			fmt.Print(byeBanner)
		},
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flags.host, "host", "http://localhost", "host where Bytebase backend is accessed from, must start with http:// or https://. This is used by Bytebase to create the webhook callback endpoint for VCS integration")
	rootCmd.PersistentFlags().IntVar(&flags.port, "port", 80, "port where Bytebase backend is accessed from. This is also used by Bytebase to create the webhook callback endpoint for VCS integration")
	rootCmd.PersistentFlags().StringVar(&flags.frontendHost, "frontend-host", "", "host where Bytebase frontend is accessed from, must start with http:// or https://. This is used by Bytebase to compose the frontend link when posting the webhook event. Default is the same as --host")
	rootCmd.PersistentFlags().IntVar(&flags.frontendPort, "frontend-port", 0, "port where Bytebase frontend is accessed from. This is used by Bytebase to compose the frontend link when posting the webhook event. Default is the same as --port")
	rootCmd.PersistentFlags().StringVar(&flags.dataDir, "data", ".", "directory where Bytebase stores data. If relative path is supplied, then the path is relative to the directory where bytebase is under")
	rootCmd.PersistentFlags().BoolVar(&flags.readonly, "readonly", false, "whether to run in read-only mode")
	rootCmd.PersistentFlags().BoolVar(&flags.demo, "demo", false, "whether to run using demo data")
	rootCmd.PersistentFlags().BoolVar(&flags.debug, "debug", false, "whether to enable debug level logging")
	// TODO(tianzhou): this needs more bake time. There are couple blocking issues:
	// 1. Currently, we will create a separate bytebase database to store the migration_history table, we need to put it inside the specified database here.
	// 2. We need to move the logic of creating bytebase metadata db logic outside. Because with --pg option, the db has already been created.
	rootCmd.PersistentFlags().StringVar(&flags.pgURL, "pg", "", "optional external PostgreSQL instance connection url(must provide dbname); for example postgresql://user:secret@masterhost:5432/dbname?sslrootcert=cert")
}

// -----------------------------------Command Line Config END--------------------------------------

// -----------------------------------Main Entry Point---------------------------------------------

func checkDataDir() error {
	// Convert to absolute path if relative path is supplied.
	if !filepath.IsAbs(flags.dataDir) {
		absDir, err := filepath.Abs(filepath.Dir(os.Args[0]) + "/" + flags.dataDir)
		if err != nil {
			return err
		}
		flags.dataDir = absDir
	}

	// Trim trailing / in case user supplies
	flags.dataDir = strings.TrimRight(flags.dataDir, "/")

	if _, err := os.Stat(flags.dataDir); err != nil {
		error := fmt.Errorf("unable to access --data %s, %w", flags.dataDir, err)
		return error
	}

	return nil
}

// GetLogger will return a logger.
func GetLogger() (*zap.Logger, *zap.AtomicLevel, error) {
	atom := zap.NewAtomicLevelAt(zap.InfoLevel)
	if flags.debug {
		atom.SetLevel(zap.DebugLevel)
	}
	logger := zap.New(zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
		zapcore.Lock(os.Stdout),
		atom,
	))
	return logger, &atom, nil
}

func start() {
	logger, level, err := GetLogger()
	if err != nil {
		panic(fmt.Errorf("failed to create logger, %w", err))
	}
	defer logger.Sync()

	// check flags
	if !common.HasPrefixes(flags.host, "http://", "https://") {
		logger.Error(fmt.Sprintf("--host %s must start with http:// or https://", flags.host))
		return
	}
	if err := checkDataDir(); err != nil {
		logger.Error(err.Error())
		return
	}

	activeProfile := activeProfile(flags.dataDir)

	var s *server.Server
	// Setup signal handlers.
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	// Trigger graceful shutdown on SIGINT or SIGTERM.
	// The default signal sent by the `kill` command is SIGTERM,
	// which is taken as the graceful shutdown signal for many systems, eg., Kubernetes, Gunicorn.
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-c
		logger.Info(fmt.Sprintf("%s received.", sig.String()))
		if s != nil {
			_ = s.Shutdown(ctx)
		}
		cancel()
	}()

	s, err = server.NewServer(ctx, activeProfile, logger, level)
	if err != nil {
		fmt.Printf("cannot new server, error: %v\n", err)
		return
	}
	fmt.Printf(greetingBanner, fmt.Sprintf("Version %s has started at %s:%d", activeProfile.Version, activeProfile.BackendHost, activeProfile.BackendPort))
	// Execute program.
	if err := s.Run(ctx); err != nil {
		if err != http.ErrServerClosed {
			logger.Error(err.Error())
			_ = s.Shutdown(ctx)
			cancel()
		}
	}

	// Wait for CTRL-C.
	<-ctx.Done()
}
