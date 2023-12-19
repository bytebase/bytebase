//go:build docker_amd64 && !docker_arm64

package mongoutil

import "embed"

var resources embed.FS
