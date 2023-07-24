package utils

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/pkg/errors"
)

// GetTarNameAndVersion construct tar file name, and file name without extension part.
// @param ext is filename extension begin with ".", i.e. ".txz".
func GetTarNameAndVersion(resKey, ext string) (tarname string, version string, err error) {
	var tarName string
	switch runtime.GOOS {
	case "linux", "darwin":
		if runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64" {
			tarName = fmt.Sprintf("%s-%s-%s%s", resKey, runtime.GOOS, runtime.GOARCH, ext)
			break
		}
		fallthrough
	default:
		return "", "", errors.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	return tarName, strings.TrimSuffix(tarName, ext), nil
}
