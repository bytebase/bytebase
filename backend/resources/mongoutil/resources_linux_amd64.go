//go:build !docker_amd64 && !docker_arm64

package mongoutil

import "embed"

//go:embed mongoutil-1.6.1-linux-amd64.txz
var resources embed.FS
