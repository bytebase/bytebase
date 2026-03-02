package validation

import (
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/action/common"
	"github.com/bytebase/bytebase/action/world"
)

// ValidateFlags validates all the command line flags and environment variables.
func ValidateFlags(w *world.World) error {
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

	// Set platform if not specified
	if w.Platform == world.UnspecifiedPlatform {
		w.Platform = world.GetJobPlatform()
	}

	// Validate authentication: either access-token or service-account+secret
	if w.AccessToken == "" {
		if w.ServiceAccount == "" {
			return errors.Errorf("service-account is required and cannot be empty")
		}
		if w.ServiceAccountSecret == "" {
			return errors.Errorf("service-account-secret is required and cannot be empty")
		}
	}

	// Validate URL format
	u, err := url.Parse(w.URL)
	if err != nil {
		return errors.Wrapf(err, "invalid URL format: %s", w.URL)
	}
	w.URL = strings.TrimSuffix(u.String(), "/") // update the URL to the canonical form

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
			return errors.Errorf("invalid target format, must be instances/{instance}/databases/{database} or projects/{project}/databaseGroups/{databaseGroup}")
		}
	}
	if databaseTarget > 0 && databaseGroupTarget > 0 {
		return errors.Errorf("targets must be either databases or databaseGroups")
	}
	if databaseGroupTarget > 1 {
		return errors.Errorf("targets must be a single databaseGroup")
	}
	return nil
}

// IsCloudURL checks if the given URL is a Bytebase cloud URL.
func IsCloudURL(u *url.URL) bool {
	cloudURLPattern := regexp.MustCompile(`^[a-z0-9]+\.us-central1\.bytebase\.com$`)
	return cloudURLPattern.MatchString(u.Host)
}
