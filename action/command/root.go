package command

import (
	"context"
	"log/slog"
	"net/http"
	"net/textproto"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/net/http/httpguts"

	"github.com/bytebase/bytebase/action/command/validation"
	"github.com/bytebase/bytebase/action/world"
)

func NewRootCommand(w *world.World) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "bytebase-action",
		Short:             "Bytebase action",
		PersistentPreRunE: rootPreRun(w),
		// XXX: PersistentPostRunE is not called when the command fails
		// So we call it manually in the commands
	}
	// bytebase-action flags
	cmd.PersistentFlags().StringVar(&w.Output, "output", "", "Output file location. The output file is a JSON file with the created resource names")
	cmd.PersistentFlags().StringVar(&w.URL, "url", "https://demo.bytebase.com", "Bytebase URL")
	cmd.PersistentFlags().DurationVar(&w.Timeout, "timeout", 120*time.Second, "HTTP timeout for API requests (e.g. 120s, 5m)")
	cmd.PersistentFlags().StringVar(&w.ServiceAccount, "service-account", "", "Bytebase Service account")
	cmd.PersistentFlags().StringVar(&w.ServiceAccountSecret, "service-account-secret", "", "Bytebase Service account secret")
	cmd.PersistentFlags().StringVar(&w.AccessToken, "access-token", "", "Bytebase access token (alternative to service account auth, e.g. from workload identity exchange)")
	cmd.PersistentFlags().Var(newCustomHeaderFlag(&w.CustomHeaders, &w.CustomHeaderError), "custom-header", "Custom HTTP header for Bytebase API requests, in 'Name: value' format. Can be specified multiple times")
	cmd.PersistentFlags().StringVar(&w.Project, "project", "projects/hr", "Bytebase project")
	cmd.PersistentFlags().StringSliceVar(&w.Targets, "targets", []string{"instances/test-sample-instance/databases/hr_test", "instances/prod-sample-instance/databases/hr_prod"}, "Bytebase targets. Either one or more databases or a single databaseGroup")
	cmd.PersistentFlags().StringVar(&w.FilePattern, "file-pattern", "", "File pattern to glob migration files")
	cmd.PersistentFlags().BoolVar(&w.Declarative, "declarative", false, "Whether to use declarative mode. (experimental)")

	cmd.AddCommand(NewCheckCommand(w))
	cmd.AddCommand(NewRolloutCommand(w))
	return cmd
}

func rootPreRun(w *world.World) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		w.Logger = slog.New(slog.NewTextHandler(cmd.ErrOrStderr(), nil))

		if w.CustomHeaderError != nil {
			return w.CustomHeaderError
		}

		// Validate all flags and environment variables
		if err := validation.ValidateFlags(w); err != nil {
			return errors.Wrapf(err, "failed to validate flags")
		}

		return nil
	}
}

// newClientFromWorld creates a client using the authentication configured in World.
func newClientFromWorld(w *world.World) (*client, error) {
	options := defaultClientOptions()
	if w.Timeout > 0 {
		options.timeout = w.Timeout
	}
	options.customHeaders = w.CustomHeaders

	// Access tokens take precedence when both auth modes are populated.
	if w.AccessToken != "" {
		return newClient(w.URL, w.AccessToken, "", "", options)
	}
	return newClient(w.URL, "", w.ServiceAccount, w.ServiceAccountSecret, options)
}

type customHeaderFlag struct {
	headers *http.Header
	err     *error
}

func newCustomHeaderFlag(headers *http.Header, err *error) *customHeaderFlag {
	if *headers == nil {
		*headers = http.Header{}
	}
	return &customHeaderFlag{headers: headers, err: err}
}

func (f *customHeaderFlag) Set(value string) error {
	name, headerValue, ok := strings.Cut(value, ":")
	if !ok {
		*f.err = errors.Errorf("invalid custom-header, must be in 'Name: value' format")
		return nil
	}

	name = textproto.CanonicalMIMEHeaderKey(strings.TrimSpace(name))
	if !httpguts.ValidHeaderFieldName(name) {
		*f.err = errors.Errorf("invalid custom-header name")
		return nil
	}
	if strings.EqualFold(name, "Authorization") {
		*f.err = errors.Errorf("custom-header Authorization is not allowed because bytebase-action manages Bytebase authorization")
		return nil
	}
	if !httpguts.ValidHeaderFieldValue(headerValue) {
		*f.err = errors.Errorf("invalid custom-header value")
		return nil
	}

	f.headers.Add(name, strings.TrimSpace(headerValue))
	return nil
}

func (f *customHeaderFlag) String() string {
	if f == nil || f.headers == nil {
		return ""
	}
	if len(*f.headers) == 0 {
		return ""
	}
	return "[redacted]"
}

func (*customHeaderFlag) Type() string {
	return "header"
}

func checkVersionCompatibility(w *world.World, client *client, cliVersion string) {
	if cliVersion == "unknown" {
		w.Logger.Warn("CLI version unknown, unable to check compatibility")
		return
	}

	actuatorInfo, err := client.getActuatorInfo(context.Background())
	if err != nil {
		w.Logger.Warn("Unable to get server version for compatibility check", "error", err)
		return
	}

	serverVersion := actuatorInfo.Version
	if serverVersion == "" {
		w.Logger.Warn("Server version is empty, unable to check compatibility")
		return
	}

	if cliVersion == "latest" {
		w.Logger.Warn("Using 'latest' CLI version. It is recommended to use a specific version like bytebase-action:" + serverVersion + " to match your Bytebase server version " + serverVersion)
		return
	}

	if cliVersion != serverVersion {
		w.Logger.Warn("CLI version mismatch", "cliVersion", cliVersion, "serverVersion", serverVersion, "recommendation", "use bytebase-action:"+serverVersion+" to match your Bytebase server")
	} else {
		w.Logger.Info("CLI version matches server version", "version", cliVersion)
	}
}
