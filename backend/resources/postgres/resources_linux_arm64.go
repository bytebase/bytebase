//go:build !docker_amd64
//go:build !docker_arm64

package postgres

import "embed"

//go:embed postgres-linux-arm64.txz
var resources embed.FS
