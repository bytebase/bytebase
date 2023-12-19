//go:build !docker_amd64 && !docker_arm64

package postgres

import "embed"

//go:embed postgres-linux-amd64.txz
var resources embed.FS
