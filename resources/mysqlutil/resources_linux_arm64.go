//go:build linux && arm64
// +build linux,arm64

package mysqlutil

import "embed"

//TODO(zp): need test for arm64
//go:embed mysqlbinlog-8.0.28-linux-glibc-2.17-x86_64.tar.gz
var resources embed.FS
