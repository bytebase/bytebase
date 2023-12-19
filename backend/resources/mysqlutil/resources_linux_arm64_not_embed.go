//go:build !docker_amd64 && docker_arm64

package mysqlutil

import "embed"

var resources embed.FS
