package model

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/blang/semver/v4"
	"github.com/pkg/errors"
)

// Version is the message representing the change history version.
type Version struct {
	Semantic bool
	Version  string
	Suffix   string
}

// nonSemanticPrefix is the prefix for non-semantic version.
const nonSemanticPrefix = "0000.0000.0000-"

// NewVersion converts stored version to version.
func NewVersion(storedVersion string) (Version, error) {
	if storedVersion == "" {
		return Version{}, nil
	}
	if strings.HasPrefix(storedVersion, nonSemanticPrefix) {
		return Version{
			Version: strings.TrimPrefix(storedVersion, nonSemanticPrefix),
		}, nil
	}

	idx := strings.Index(storedVersion, "-")
	if idx < 0 {
		return Version{}, errors.Errorf("invalid stored version %q, version should contain '-'", storedVersion)
	}
	prefix, suffix := storedVersion[:idx], storedVersion[idx+1:]
	parts := strings.Split(prefix, ".")
	if len(parts) != 3 {
		return Version{}, errors.Errorf("invalid stored version %q, version prefix %q should be in semantic version format", storedVersion, prefix)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return Version{}, errors.Errorf("invalid stored version %q, version prefix %q should be in semantic version format", storedVersion, prefix)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return Version{}, errors.Errorf("invalid stored version %q, version prefix %q should be in semantic version format", storedVersion, prefix)
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return Version{}, errors.Errorf("invalid stored version %q, version prefix %q should be in semantic version format", storedVersion, prefix)
	}
	if major >= 10000 || minor >= 10000 || patch >= 10000 {
		return Version{}, errors.Errorf("invalid stored version %q, major, minor, patch version of %q should be < 10000", storedVersion, prefix)
	}
	return Version{
		Semantic: true,
		Version:  fmt.Sprintf("%d.%d.%d", major, minor, patch),
		Suffix:   suffix,
	}, nil
}

// Marshal converts version to stored version format.
// Non-semantic version will have additional "0000.0000.0000-" prefix.
// Semantic version will add zero padding to MAJOR, MINOR, PATCH version with a timestamp suffix.
func (v *Version) Marshal() (string, error) {
	if v.Version == "" {
		return "", nil
	}
	if !v.Semantic {
		return fmt.Sprintf("%s%s", nonSemanticPrefix, v.Version), nil
	}
	sv, err := semver.Make(v.Version)
	if err != nil {
		return "", err
	}
	major, minor, patch := fmt.Sprintf("%d", sv.Major), fmt.Sprintf("%d", sv.Minor), fmt.Sprintf("%d", sv.Patch)
	if len(major) > 4 || len(minor) > 4 || len(patch) > 4 {
		return "", errors.Errorf("invalid version %q, major, minor, patch version should be < 10000", v.Version)
	}
	return fmt.Sprintf("%04s.%04s.%04s-%s", major, minor, patch, v.Suffix), nil
}
