//go:build !docker_amd64
//go:build !docker_arm64

package mongoutil

import "embed"

//go:embed mongoutil-1.6.1-linux-arm64.txz
var resources embed.FS
