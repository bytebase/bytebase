//go:build linux && arm64
// +build linux,arm64

package mysqlutil

import "embed"

//TODO(zp): need test for arm64
//go:embed mysqlutil-8.0.28-linux-glibc2.17-x86_64.tar.gz
var resources embed.FS
