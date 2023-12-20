//go:build docker

package utils

import (
	"embed"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/pkg/errors"
)

// InstallImpl installs mongoutil, mysqlutil, postgres in resourceDir.
func InstallImpl(resourceDir, utilDir, tarName, version string, _ embed.FS) error {
	preloadingDir := getPreloadingUtilDir(utilDir)
	if _, err := os.Stat(preloadingDir); err != nil {
		if os.IsNotExist(err) {
			// source file doesn't exists.'
			return errors.Errorf("preloadingDir %q does not exist", preloadingDir)
		}
		return errors.Wrapf(err, "preloadingDir %q error", preloadingDir)
	}
	if utilDir == preloadingDir {
		// they are same, just use it.
		return nil
	}

	// they are not same, create a symbolic link.
	if err := os.Symlink(preloadingDir, utilDir); err != nil {
		// panic if failed to create symbolic link
		return errors.Wrapf(err, "failed to create a symbolic link for utilDir %q", utilDir)
	}

	// create symbolic link success
	return nil
}

// getPreloadingUtilDir returns the preloading directory which decided at the build time for the given util directory.
func getPreloadingUtilDir(utilDir string) string {
	// These paths must be consistent with the Dockerfile where decompressing the txz files.
	// we use these magic paths because the resources can only be extracted at build time. But at runtime we will decide real path according to the user input.
	// So we extract resources to these specific paths during build and then symlink them to actual paths.
	if strings.Contains(utilDir, "mysql") {
		// we only support linux/amd64 or linux/arm64 now.
		return fmt.Sprintf("/var/opt/bytebase/resources/mysqlutil-8.0.33-linux-%s", runtime.GOARCH)
	}
	if strings.Contains(utilDir, "mongoutil") {
		// we only support linux/amd64 or linux/arm64 now.
		return fmt.Sprintf("/var/opt/bytebase/resources/mongoutil-1.6.1-linux-%s", runtime.GOARCH)
	}
	if strings.Contains(utilDir, "postgres") {
		// we only support linux/amd64 or linux/arm64 now.
		return fmt.Sprintf("/var/opt/bytebase/resources/postgres-linux-%s-16", runtime.GOARCH)
	}
	return utilDir
}
