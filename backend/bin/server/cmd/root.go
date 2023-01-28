// Package cmd implements the cobra CLI for Bytebase server.
package cmd

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/server"
)

// -----------------------------------Global constant BEGIN----------------------------------------.
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

************* External Visiting URL (--external-url) *************

%s

******************************************************************
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
		// When we are running in readonly mode:
		// - The data file will be opened in readonly mode, no applicable migration or seeding will be applied.
		// - Requests other than GET will be rejected
		// - Any operations involving mutation will not start (e.g. Background schema syncer, task scheduler)
		readonly bool
		// demoName is the name of the demo and should be one of the subpath name in the ./store/demo/ directory.
		// empty means no demo.
		demoName string
		debug    bool
		// pgURL must follow PostgreSQL connection URIs pattern.
		// https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING
		pgURL string
		// disableMetric is the flag to disable the metric collector.
		disableMetric bool

		// Cloud backup configs.
		backupRegion     string
		backupBucket     string
		backupCredential string
	}

	rootCmd = &cobra.Command{
		Use:   "bytebase",
		Short: "Bytebase is a database schema change and version control tool",
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
	// In the release build, Bytebase bundles frontend and backend together and runs on a single port as a mono server.
	// During development, Bytebase frontend runs on a separate port.
	rootCmd.PersistentFlags().IntVar(&flags.port, "port", 80, "port where Bytebase server runs. Default to 80")
	// When running the release build in production, most of the time, users would not expose Bytebase directly to the public.
	// Instead they would configure a gateway to forward the traffic to Bytebase. Users need to set --external-url to the address
	// exposed on that gateway accordingly.
	//
	// It's important to set the correct --external-url. This is used for:
	// 1. Constructing the correct callback URL when configuring the VCS provider. The callback URL points to the frontend.
	// 2. Creating the correct webhook endpoint when configuring the project GitOps workflow. The webhook endpoint points to the backend.
	// Since frontend and backend are bundled and run on the same address in the release build, thus we just need to specify a single external URL.
	rootCmd.PersistentFlags().StringVar(&flags.externalURL, "external-url", common.ExternalURLDocsLink, "the external URL where user visits Bytebase, must start with http:// or https://")
	rootCmd.PersistentFlags().StringVar(&flags.dataDir, "data", ".", "directory where Bytebase stores data. If relative path is supplied, then the path is relative to the directory where Bytebase is under")
	rootCmd.PersistentFlags().BoolVar(&flags.readonly, "readonly", false, "whether to run in read-only mode")
	// Must be one of the subpath name in the ./store/demo/ directory
	rootCmd.PersistentFlags().StringVar(&flags.demoName, "demo", "", "name of the demo to use. Empty means not running in demo mode.")
	rootCmd.PersistentFlags().BoolVar(&flags.debug, "debug", false, "whether to enable debug level logging")
	// Support environment variable for deploying to render.com using its blueprint file.
	// Render blueprint allows to specify a postgres database along with a service.
	// It allows to pass the postgres connection string as an ENV to the service.
	rootCmd.PersistentFlags().StringVar(&flags.pgURL, "pg", os.Getenv("PG_URL"), "optional external PostgreSQL instance connection url(must provide dbname); for example postgresql://user:secret@masterhost:5432/dbname?sslrootcert=cert")
	rootCmd.PersistentFlags().BoolVar(&flags.disableMetric, "disable-metric", false, "disable the metric collector")

	// Cloud backup related flags.
	// TODO(dragonly): Add GCS usages when it's supported.
	rootCmd.PersistentFlags().StringVar(&flags.backupBucket, "backup-bucket", "", "bucket where Bytebase stores backup data, e.g., s3://example-bucket. When provided, Bytebase will store data to the S3 bucket.")
	rootCmd.PersistentFlags().StringVar(&flags.backupRegion, "backup-region", "", "region of the backup bucket, e.g., us-west-2 for AWS S3.")
	rootCmd.PersistentFlags().StringVar(&flags.backupCredential, "backup-credential", "", "credentials file to use for the backup bucket. It should be the same format as the AWS/GCP credential files.")
}

// -----------------------------------Command Line Config END--------------------------------------

func normalizeExternalURL(url string) (string, error) {
	r := strings.TrimSpace(url)
	r = strings.TrimSuffix(r, "/")
	if !common.HasPrefixes(r, "http://", "https://") {
		return "", errors.Errorf("%s must start with http:// or https://", url)
	}
	parts := strings.Split(r, ":")
	if len(parts) > 3 {
		return "", errors.Errorf("%s malformed", url)
	}
	if len(parts) == 3 {
		port, err := strconv.Atoi(parts[2])
		if err != nil {
			return "", errors.Errorf("%s has non integer port", url)
		}
		// The external URL is used as the redirectURL in the get token process of OAuth, and the
		// RedirectURL needs to be consistent with the RedirectURL in the get code process.
		// The frontend gets it through window.location.origin in the get code
		// process, so port 80/443 need to be cropped.
		if port == 80 || port == 443 {
			r = strings.Join(parts[0:2], ":")
		}
	}
	return r, nil
}

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

func checkCloudBackupFlags() error {
	if flags.backupBucket == "" {
		return nil
	}
	if !strings.HasPrefix(flags.backupBucket, "s3://") {
		return errors.Errorf("only support bucket URI starting with s3://")
	}
	flags.backupBucket = strings.TrimPrefix(flags.backupBucket, "s3://")
	if flags.backupCredential == "" {
		return errors.Errorf("must specify --backup-credential when --backup-bucket is present")
	}
	if flags.backupRegion == "" {
		return errors.Errorf("must specify --backup-region for AWS S3 backup")
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
	if flags.debug {
		log.SetLevel(zap.DebugLevel)
	}
	defer log.Sync()

	var err error
	flags.externalURL, err = normalizeExternalURL(flags.externalURL)
	if err != nil {
		log.Error("invalid --external-url", zap.Error(err))
		return
	}
	if err := checkDataDir(); err != nil {
		log.Error(err.Error())
		return
	}

	if err := checkCloudBackupFlags(); err != nil {
		log.Error("invalid flags for cloud backup", zap.Error(err))
		return
	}

	profile := activeProfile(flags.dataDir)

	// The ideal bootstrap order is:
	// 1. Connect to the metadb
	// 2. Start echo server
	// 3. Start various background runners
	//
	// Strangely, when the port is unavailable, echo server would return OK response for /healthz
	// and then complain unable to bind port. Thus we cannot rely on checking /healthz. As a
	// workaround, we check whether the port is available here.
	if err := checkPort(flags.port); err != nil {
		log.Error(fmt.Sprintf("server port %d is not available", flags.port), zap.Error(err))
		return
	}
	if profile.UseEmbedDB() {
		if err := checkPort(profile.DatastorePort); err != nil {
			log.Error(fmt.Sprintf("database port %d is not available", profile.DatastorePort), zap.Error(err))
			return
		}
	}
	if err := checkPort(profile.GrpcPort); err != nil {
		log.Error(fmt.Sprintf("gRPC server port %d is not available", profile.GrpcPort), zap.Error(err))
		return
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
		log.Info(fmt.Sprintf("%s received.", sig.String()))
		if s != nil {
			_ = s.Shutdown(ctx)
		}
		cancel()
	}()

	s, err = server.NewServer(ctx, profile)
	if err != nil {
		log.Error("Cannot new server", zap.Error(err))
		return
	}

	externalAddr := profile.ExternalURL
	if profile.ExternalURL == common.ExternalURLDocsLink {
		externalAddr = fmt.Sprintf("!!! You have not set --external-url. If you want to make Bytebase\n!!! externally accessible, follow:\n\n%s", common.ExternalURLDocsLink)
	}
	fmt.Printf(greetingBanner, fmt.Sprintf("Version %s (schema version %v) has started on port %d", profile.Version, s.SchemaVersion, flags.port), externalAddr)

	// Execute program.
	if err := s.Run(ctx, flags.port); err != nil {
		if err != http.ErrServerClosed {
			log.Error(err.Error())
			_ = s.Shutdown(ctx)
			cancel()
		}
	}

	// Wait for CTRL-C.
	<-ctx.Done()
}
