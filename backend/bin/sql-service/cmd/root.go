// Package cmd provides the cobra CLI implementation for SQL review service.
package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	server "github.com/bytebase/bytebase/backend/sql-server"

	// Register pingcap parser driver.
	_ "github.com/pingcap/tidb/types/parser_driver"
	// Register fake advisor.
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/fake"
	// Register mysql advisor.
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/mysql"
	// Register postgresql advisor.
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/pg"
	// Register oracle advisor.
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/oracle"
	// Register snowflake advisor.
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/snowflake"
	// Register mssql advisor.
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/mssql"

	// Register postgres parser driver.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/sql/engine/pg"
)

// -----------------------------------Global constant BEGIN----------------------------------------.
const (

	// greetingBanner is the greeting banner.
	// http://patorjk.com/software/taag/#p=display&f=ANSI%20Shadow&t=BB%20SQL%20Service
	greetingBanner = `
██████╗ ██████╗     ███████╗ ██████╗ ██╗         ███████╗███████╗██████╗ ██╗   ██╗██╗ ██████╗███████╗
██╔══██╗██╔══██╗    ██╔════╝██╔═══██╗██║         ██╔════╝██╔════╝██╔══██╗██║   ██║██║██╔════╝██╔════╝
██████╔╝██████╔╝    ███████╗██║   ██║██║         ███████╗█████╗  ██████╔╝██║   ██║██║██║     █████╗
██╔══██╗██╔══██╗    ╚════██║██║▄▄ ██║██║         ╚════██║██╔══╝  ██╔══██╗╚██╗ ██╔╝██║██║     ██╔══╝
██████╔╝██████╔╝    ███████║╚██████╔╝███████╗    ███████║███████╗██║  ██║ ╚████╔╝ ██║╚██████╗███████╗
╚═════╝ ╚═════╝     ╚══════╝ ╚══▀▀═╝ ╚══════╝    ╚══════╝╚══════╝╚═╝  ╚═╝  ╚═══╝  ╚═╝ ╚═════╝╚══════╝

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
		host        string
		port        int
		debug       bool
		workspaceID string
	}
	rootCmd = &cobra.Command{
		Use:   "sql",
		Short: "Bytebase SQL Service is the API service for Bytebase SQL Review",
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
	rootCmd.PersistentFlags().StringVar(&flags.host, "host", "http://localhost", "host where Bytebase SQL service backend is accessed from, must start with http:// or https://.")
	rootCmd.PersistentFlags().IntVar(&flags.port, "port", 80, "port where Bytebase SQL service backend is accessed from.")
	rootCmd.PersistentFlags().BoolVar(&flags.debug, "debug", false, "whether to enable debug level logging")
	rootCmd.PersistentFlags().StringVar(&flags.workspaceID, "workspace-id", "", "the identifier for SQL service")
}

// -----------------------------------Command Line Config END--------------------------------------

// -----------------------------------Main Entry Point---------------------------------------------

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
		log.Error("Cannot new server", zap.Error(err))
		return
	}
	fmt.Printf(greetingBanner, fmt.Sprintf("Version %s has started at %s:%d", activeProfile.Version, activeProfile.BackendHost, activeProfile.BackendPort))
	// Execute program.
	if err := s.Run(); err != nil {
		if err != http.ErrServerClosed {
			log.Error(err.Error())
			_ = s.Shutdown(ctx)
			cancel()
		}
	}

	// Wait for CTRL-C.
	<-ctx.Done()
}
