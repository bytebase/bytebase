//go:build docker_amd64
//go:build !docker_arm64

package postgres

import "embed"

var resources embed.FS
