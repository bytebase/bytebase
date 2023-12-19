//go:build !docker_amd64 && !docker_arm64

package mysqlutil

import "embed"

//go:embed mysqlutil-8.0.33-linux-arm64.tar.gz
var resources embed.FS
