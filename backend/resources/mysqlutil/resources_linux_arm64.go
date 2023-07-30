package mysqlutil

import "embed"

// TODO(zp): We cheat go build here, we don't provide mysqlutil on linux arm64 now.
//
//go:embed mysqlutil-8.0.33-linux-glibc2.17-x86_64.tar.gz
var resources embed.FS
