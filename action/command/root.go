package command

import (
	"context"
	"log/slog"
	"net/http"
	"net/textproto"
	"strconv"
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
	cmd.PersistentFlags().StringVar(&w.URL, "url", "", "Bytebase URL (required)")
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

func checkVersionCompatibility(w *world.World, client *client, cliVersion string) error {
	if cliVersion == "unknown" {
		w.Logger.Warn("CLI version unknown, unable to check compatibility")
		return nil
	}

	actuatorInfo, err := client.getActuatorInfo(context.Background())
	if err != nil {
		w.Logger.Warn("Unable to get server version for compatibility check", "error", err)
		return nil
	}

	serverVersion := actuatorInfo.Version
	if serverVersion == "" {
		w.Logger.Warn("Server version is empty, unable to check compatibility")
		return nil
	}

	serverParsedVersion, err := parseVersion(serverVersion)
	if err != nil {
		return errors.Errorf("unable to parse Bytebase server version %q for compatibility check: %v; expected a self-hosted version like 3.14.0 or a Cloud version like cloud-YYYYMMDD", serverVersion, err)
	}

	if cliVersion == "latest" {
		actionTag := recommendedActionTag(serverParsedVersion)
		w.Logger.Warn("Using 'latest' CLI version. It is recommended to use a specific version like bytebase-action:" + actionTag + " to match your Bytebase server version " + serverVersion)
		return nil
	}

	cliParsedVersion, err := parseVersion(cliVersion)
	if err != nil {
		return errors.Errorf("unable to parse CLI version %q for compatibility check: %v; expected a self-hosted version like 3.14.0 or a Cloud version like cloud-YYYYMMDD", cliVersion, err)
	}
	if cliParsedVersion.kind != serverParsedVersion.kind {
		return errors.Errorf("unable to compare CLI version %q and Bytebase server version %q for compatibility check; expected both versions to use self-hosted versions like 3.14.0 or Cloud versions like cloud-YYYYMMDD", cliVersion, serverVersion)
	}

	if cliVersion == serverVersion {
		w.Logger.Info("CLI version matches server version", "version", cliVersion)
		return nil
	}

	if cliParsedVersion.kind == versionKindCloud {
		// Cloud action/server builds are date-qualified and compatible when the
		// action build is from the server's latest 7 days. Newer action builds
		// may call APIs that do not exist on older or rolled-back servers.
		dateDiff := int(cliParsedVersion.date.Sub(serverParsedVersion.date).Hours() / 24)
		if dateDiff < -7 || dateDiff > 0 {
			actionTag := recommendedActionTag(serverParsedVersion)
			return errors.Errorf("CLI version %q is outside the compatibility window for Bytebase server version %q. Cloud compatibility requires both versions to use cloud-YYYYMMDD and be within 7 days. Use bytebase-action:%s to match your Bytebase server", cliVersion, serverVersion, actionTag)
		}

		actionTag := recommendedActionTag(serverParsedVersion)
		w.Logger.Warn("CLI version is within the compatibility window but does not match server version", "cliVersion", cliVersion, "serverVersion", serverVersion, "recommendation", "use bytebase-action:"+actionTag+" to match your Bytebase server")
		return nil
	}

	// Self-hosted action/server releases are compatible when the action version
	// is within the server's latest 2 minor releases. Newer action versions may
	// call APIs that do not exist on older servers, so reject them.
	minorDiff := cliParsedVersion.minor - serverParsedVersion.minor
	if cliParsedVersion.major != serverParsedVersion.major || minorDiff < -2 || minorDiff > 0 {
		actionTag := recommendedActionTag(serverParsedVersion)
		return errors.Errorf("CLI version %q is outside the compatibility window for Bytebase server version %q. Self-hosted compatibility requires the same major version and versions within 2 minor versions. Use bytebase-action:%s to match your Bytebase server", cliVersion, serverVersion, actionTag)
	}

	actionTag := recommendedActionTag(serverParsedVersion)
	w.Logger.Warn("CLI version is within the compatibility window but does not match server version", "cliVersion", cliVersion, "serverVersion", serverVersion, "recommendation", "use bytebase-action:"+actionTag+" to match your Bytebase server")
	return nil
}

type versionKind int

const (
	versionKindRelease versionKind = iota
	versionKindCloud
)

const cloudVersionPrefix = "cloud-"

type parsedVersion struct {
	kind  versionKind
	raw   string
	major int
	minor int
	date  time.Time
}

func parseVersion(version string) (parsedVersion, error) {
	if strings.HasPrefix(version, "cloud") {
		t, err := parseCloudVersionDate(version)
		if err != nil {
			return parsedVersion{}, err
		}
		return parsedVersion{kind: versionKindCloud, raw: version, date: t}, nil
	}
	major, minor, err := parseReleaseMajorMinor(version)
	if err != nil {
		return parsedVersion{}, err
	}
	return parsedVersion{kind: versionKindRelease, raw: version, major: major, minor: minor}, nil
}

func recommendedActionTag(version parsedVersion) string {
	if version.kind == versionKindCloud {
		return "cloud"
	}
	return version.raw
}

func parseCloudVersionDate(version string) (time.Time, error) {
	if len(version) != len("cloud-YYYYMMDD") || !strings.HasPrefix(version, cloudVersionPrefix) {
		return time.Time{}, errors.Errorf("expected cloud-YYYYMMDD")
	}
	for _, r := range strings.TrimPrefix(version, cloudVersionPrefix) {
		if r < '0' || r > '9' {
			return time.Time{}, errors.Errorf("expected cloud-YYYYMMDD")
		}
	}
	t, err := time.Parse("20060102", strings.TrimPrefix(version, cloudVersionPrefix))
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}

func parseReleaseMajorMinor(version string) (int, int, error) {
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return 0, 0, errors.Errorf("expected at least major and minor release segments")
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, errors.Wrap(err, "invalid major release segment")
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, errors.Wrap(err, "invalid minor release segment")
	}
	return major, minor, nil
}
