package validation

import (
	"net"
	"net/url"
	"os"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/action/common"
	"github.com/bytebase/bytebase/action/world"
)

// The audience the GitOps page presets on a new workload identity.
const defaultWorkloadIdentityAudience = "bytebase"

// ValidateFlags validates all the command line flags and environment variables.
func ValidateFlags(w *world.World) error {
	// Read before the environment fills the field: only the flag outranks a
	// service account.
	workloadIdentityFlag := w.WorkloadIdentity != ""

	// Set access token from environment if not provided
	if w.AccessToken == "" {
		w.AccessToken = os.Getenv("BYTEBASE_ACCESS_TOKEN")
	}
	// Set service account and secret from environment if not provided
	if w.ServiceAccount == "" {
		w.ServiceAccount = os.Getenv("BYTEBASE_SERVICE_ACCOUNT")
	}
	if w.ServiceAccountSecret == "" {
		w.ServiceAccountSecret = os.Getenv("BYTEBASE_SERVICE_ACCOUNT_SECRET")
	}
	if w.WorkloadIdentity == "" {
		w.WorkloadIdentity = os.Getenv("BYTEBASE_WORKLOAD_IDENTITY")
	}
	if w.WorkloadIdentityAudience == "" {
		w.WorkloadIdentityAudience = os.Getenv("BYTEBASE_WORKLOAD_IDENTITY_AUDIENCE")
	}
	if w.WorkloadIdentityAudience == "" {
		w.WorkloadIdentityAudience = defaultWorkloadIdentityAudience
	}

	// Set platform if not specified
	if w.Platform == world.UnspecifiedPlatform {
		w.Platform = world.GetJobPlatform()
	}

	signIn, err := chooseSignIn(w, workloadIdentityFlag)
	if err != nil {
		return err
	}
	w.SignIn = signIn
	logSignIn(w)

	// Validate URL: must be a non-empty absolute URL.
	if w.URL == "" {
		return errors.Errorf("--url is required")
	}
	u, err := url.Parse(w.URL)
	if err != nil {
		return errors.Wrapf(err, "invalid URL format: %s", w.URL)
	}
	if u.Scheme == "" || u.Host == "" {
		return errors.Errorf("--url must be an absolute URL (e.g. https://bytebase.example.com), got %q", w.URL)
	}
	w.URL = strings.TrimSuffix(u.String(), "/") // update the URL to the canonical form
	if w.SignIn == world.SignInWorkloadIdentity && u.Scheme == "http" && !isLoopback(u.Hostname()) {
		w.Logger.Warn("the CI job's ID token is sent to Bytebase over http; use an https --url")
	}

	// Validate project format
	if !strings.HasPrefix(w.Project, "projects/") {
		return errors.Errorf("invalid project format, must be projects/{project}")
	}

	// Validate targets format
	return validateTargets(w.Targets)
}

// validateTargets validates the targets format and ensures consistency.
func validateTargets(targets []string) error {
	var databaseTarget, databaseGroupTarget int
	for _, target := range targets {
		if _, _, err := common.GetInstanceDatabaseID(target); err == nil {
			databaseTarget++
		} else if _, _, err := common.GetProjectIDDatabaseGroupID(target); err == nil {
			databaseGroupTarget++
		} else {
			return errors.Errorf("invalid target format %q; must be instances/{instance}/databases/{database}, projects/{project}/instances/{instance}/databases/{database}, or projects/{project}/databaseGroups/{databaseGroup}", target)
		}
	}
	if databaseTarget > 0 && databaseGroupTarget > 0 {
		return errors.Errorf("targets must be either database targets or a database group target")
	}
	if databaseGroupTarget > 1 {
		return errors.Errorf("targets must contain a single database group target")
	}
	return nil
}

// chooseSignIn picks how the run signs in. An existing job keeps the method it
// has today: BYTEBASE_WORKLOAD_IDENTITY is set by every generated workflow, so
// it ranks below the service account, while the new --workload-identity flag
// ranks above it.
func chooseSignIn(w *world.World, workloadIdentityFlag bool) (world.SignIn, error) {
	switch {
	case w.AccessToken != "":
		return world.SignInAccessToken, nil
	case workloadIdentityFlag:
		return world.SignInWorkloadIdentity, nil
	case w.ServiceAccount != "":
		if w.ServiceAccountSecret == "" {
			return world.SignInUnspecified, errors.Errorf("service-account-secret is required and cannot be empty")
		}
		return world.SignInServiceAccount, nil
	case w.WorkloadIdentity != "":
		return world.SignInWorkloadIdentity, nil
	default:
		return world.SignInUnspecified, errors.Errorf("no sign-in is set: use --workload-identity, --service-account with --service-account-secret, or --access-token")
	}
}

func logSignIn(w *world.World) {
	switch w.SignIn {
	case world.SignInAccessToken:
		w.Logger.Info("signing in with the access token")
		return
	case world.SignInServiceAccount:
		w.Logger.Info("signing in with a service account", "serviceAccount", w.ServiceAccount)
	case world.SignInWorkloadIdentity:
		w.Logger.Info("signing in with a workload identity", "workloadIdentity", w.WorkloadIdentity)
	default:
		return
	}
	if w.ServiceAccount != "" && w.WorkloadIdentity != "" {
		w.Logger.Warn("a service account and a workload identity are both set; the two are different actors",
			"serviceAccount", w.ServiceAccount, "workloadIdentity", w.WorkloadIdentity)
	}
}

func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
