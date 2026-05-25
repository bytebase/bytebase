// Package cmd implements the cobra CLI for Bytebase server.
package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/jackc/pgconn"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/server"
)

// -----------------------------------Global constant BEGIN----------------------------------------.
const (

	// greetingBanner is the greeting banner.
	// http://patorjk.com/software/taag/#p=display&f=ANSI%20Shadow&t=Bytebase
	greetingBanner = `
___________________________________________________________________________________________

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

// -----------------------------------Command Line Config BEGIN------------------------------------.
var (
	flags struct {
		// Used for Bytebase command line config
		port        int
		externalURL string
		dataDir     string
		ha          bool
		saas        bool
		debug       bool
		// output logs in json format
		enableJSONLogging bool
		// memoryProfileThreshold is the threshold of memory usage in bytes to trigger a memory profile.
		memoryProfileThreshold uint64
	}

	rootCmd = &cobra.Command{
		Use:   "bytebase",
		Short: "Bytebase is a database schema change and version control tool",
		Run: func(_ *cobra.Command, _ []string) {
			start()

			fmt.Printf("%s", byeBanner)
		},
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Set default data directory based on environment.
	// In Docker, default to /var/opt/bytebase to match the Dockerfile's CMD.
	// This ensures the data directory is correct even when users add other flags,
	// which would otherwise cause Docker to replace the entire CMD and lose --data.
	defaultDataDir := "."
	if isDocker() {
		defaultDataDir = "/var/opt/bytebase"
	}

	// In the release build, Bytebase bundles frontend and backend together and runs on a single port as a mono server.
	// During development, Bytebase frontend runs on a separate port.
	rootCmd.PersistentFlags().IntVar(&flags.port, "port", 8080, "port where Bytebase server runs. Default to 80")
	// When running the release build in production, most of the time, users would not expose Bytebase directly to the public.
	// Instead they would configure a gateway to forward the traffic to Bytebase. Users need to set --external-url to the address
	// exposed on that gateway accordingly.
	rootCmd.PersistentFlags().StringVar(&flags.externalURL, "external-url", "", "the external URL where user visits Bytebase, must start with http:// or https://")
	rootCmd.PersistentFlags().StringVar(&flags.dataDir, "data", defaultDataDir, "not recommended for production. Directory where Bytebase stores data if PG_URL is not specified. If relative path is supplied, then the path is relative to the directory where Bytebase is under")
	rootCmd.PersistentFlags().BoolVar(&flags.ha, "ha", false, "run in HA mode")
	rootCmd.PersistentFlags().BoolVar(&flags.saas, "saas", false, "run in SaaS mode")
	rootCmd.PersistentFlags().BoolVar(&flags.debug, "debug", false, "enable debug level log")
	rootCmd.PersistentFlags().BoolVar(&flags.enableJSONLogging, "enable-json-logging", false, "enable output logs in bytebase in json format")
	rootCmd.PersistentFlags().Uint64Var(&flags.memoryProfileThreshold, "memory-profile-threshold", 0, "the threshold of memory usage in bytes to trigger a memory profile")
}

// -----------------------------------Command Line Config END--------------------------------------

func checkDataDir() error {
	// Clean data directory path.
	flags.dataDir = filepath.Clean(flags.dataDir)

	// Convert to absolute path if relative path is supplied.
	if !filepath.IsAbs(flags.dataDir) {
		absDir, err := filepath.Abs(filepath.Dir(os.Args[0]) + "/" + flags.dataDir)
		if err != nil {
			return err
		}
		flags.dataDir = absDir
	}

	if _, err := os.Stat(flags.dataDir); err != nil {
		return errors.Wrapf(err, "unable to access --data directory %s", flags.dataDir)
	}

	return nil
}

// Check the port availability by trying to bind and immediately release it.
func checkPort(port int) error {
	l, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		return err
	}
	return l.Close()
}

func start() {
	if flags.saas || flags.enableJSONLogging {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{AddSource: true, Level: log.LogLevel, ReplaceAttr: log.Replace})))
	} else {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{AddSource: true, Level: log.LogLevel, ReplaceAttr: log.Replace})))
	}

	var err error

	if flags.externalURL != "" {
		flags.externalURL, err = common.NormalizeExternalURL(flags.externalURL)
		if err != nil {
			slog.Error("invalid --external-url", log.BBError(err))
			return
		}
	}

	if err := checkDataDir(); err != nil {
		slog.Error(err.Error())
		return
	}

	profile := activeProfile(flags.dataDir)

	fmt.Printf("Starting Bytebase %s(%s)...\n", profile.Version, profile.GitCommit)

	// The ideal bootstrap order is:
	// 1. Connect to the metadb
	// 2. Start echo server
	// 3. Start various background runners
	//
	// Strangely, when the port is unavailable, echo server would return OK response for /healthz
	// and then complain unable to bind port. Thus we cannot rely on checking /healthz. As a
	// workaround, we check whether the port is available here.
	if err := checkPort(flags.port); err != nil {
		slog.Error(fmt.Sprintf("server port %d is not available", flags.port), log.BBError(err))
		return
	}
	if profile.UseEmbedDB() {
		if err := checkPort(profile.DatastorePort); err != nil {
			slog.Error(fmt.Sprintf("database port %d is not available", profile.DatastorePort), log.BBError(err))
			return
		}
	}

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
		slog.Info(fmt.Sprintf("%s received.", sig.String()))
		if s != nil {
			_ = s.Shutdown(ctx)
		}
		cancel()
	}()

	s, err = server.NewServer(ctx, profile)
	if err != nil {
		var pge *pgconn.PgError
		if errors.As(err, &pge) {
			slog.Error("Cannot new server", log.BBError(err), "detail", pge.Detail, "hint", pge.Hint)
			return
		}
		slog.Error("Cannot new server", log.BBError(err))
		return
	}

	fmt.Printf(greetingBanner, fmt.Sprintf("Version %s(%s) has started on port %d 🚀", profile.Version, profile.GitCommit, flags.port))

	// Execute program.
	if err := s.Run(ctx, flags.port); err != nil {
		slog.Error(err.Error())
		_ = s.Shutdown(ctx)
		cancel()
	}

	// Wait for CTRL-C.
	<-ctx.Done()
}
