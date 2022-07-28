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
	"github.com/bytebase/bytebase/common/log"
	server "github.com/bytebase/bytebase/sql-review-server"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	// Register pingcap parser driver.
	_ "github.com/pingcap/tidb/types/parser_driver"
	// Register fake advisor.
	_ "github.com/bytebase/bytebase/plugin/advisor/fake"
	// Register mysql advisor.
	_ "github.com/bytebase/bytebase/plugin/advisor/mysql"
	// Register postgresql advisor.
	_ "github.com/bytebase/bytebase/plugin/advisor/pg"

	// Register postgres parser driver.
	_ "github.com/bytebase/bytebase/plugin/parser/engine/pg"
)

// -----------------------------------Global constant BEGIN----------------------------------------.
const (

	// greetingBanner is the greeting banner.
	// http://patorjk.com/software/taag/#p=display&f=ANSI%20Shadow&t=Bytebase%20SQL%20Review
	greetingBanner = `
██████╗ ██████╗     ███████╗ ██████╗ ██╗         ██████╗ ███████╗██╗   ██╗██╗███████╗██╗    ██╗
██╔══██╗██╔══██╗    ██╔════╝██╔═══██╗██║         ██╔══██╗██╔════╝██║   ██║██║██╔════╝██║    ██║
██████╔╝██████╔╝    ███████╗██║   ██║██║         ██████╔╝█████╗  ██║   ██║██║█████╗  ██║ █╗ ██║
██╔══██╗██╔══██╗    ╚════██║██║▄▄ ██║██║         ██╔══██╗██╔══╝  ╚██╗ ██╔╝██║██╔══╝  ██║███╗██║
██████╔╝██████╔╝    ███████║╚██████╔╝███████╗    ██║  ██║███████╗ ╚████╔╝ ██║███████╗╚███╔███╔╝
╚═════╝ ╚═════╝     ╚══════╝ ╚══▀▀═╝ ╚══════╝    ╚═╝  ╚═╝╚══════╝  ╚═══╝  ╚═╝╚══════╝ ╚══╝╚══╝

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

// -----------------------------------Command Line Config BEGIN------------------------------------.
var (
	flags struct {
		// Used for Bytebase command line config
		host    string
		port    int
		dataDir string
		debug   bool
	}
	rootCmd = &cobra.Command{
		Use:   "sql-review",
		Short: "Bytebase SQL Review is a database schema change and version control tool",
		Run: func(_ *cobra.Command, _ []string) {
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
	rootCmd.PersistentFlags().StringVar(&flags.dataDir, "data", ".", "directory where Bytebase stores data. If relative path is supplied, then the path is relative to the directory where Bytebase is under")
	rootCmd.PersistentFlags().BoolVar(&flags.debug, "debug", false, "whether to enable debug level logging")
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
		return fmt.Errorf("unable to access --data %s, %w", flags.dataDir, err)
	}

	return nil
}

func start() {
	if flags.debug {
		log.SetLevel(zap.DebugLevel)
	}
	defer log.Sync()

	// check flags
	if !common.HasPrefixes(flags.host, "http://", "https://") {
		log.Error(fmt.Sprintf("--host %s must start with http:// or https://", flags.host))
		return
	}
	if err := checkDataDir(); err != nil {
		log.Error(err.Error())
		return
	}

	activeProfile := activeProfile()

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
		log.Info(fmt.Sprintf("%s received.", sig.String()))
		if s != nil {
			_ = s.Shutdown(ctx)
		}
		cancel()
	}()

	s, err := server.NewServer(ctx, activeProfile)
	if err != nil {
		fmt.Printf("cannot new server, error: %v\n", err)
		return
	}
	fmt.Printf(greetingBanner, fmt.Sprintf("Version %s has started at %s:%d", activeProfile.Version, activeProfile.BackendHost, activeProfile.BackendPort))
	// Execute program.
	if err := s.Run(ctx); err != nil {
		if err != http.ErrServerClosed {
			log.Error(err.Error())
			_ = s.Shutdown(ctx)
			cancel()
		}
	}

	// Wait for CTRL-C.
	<-ctx.Done()
}
