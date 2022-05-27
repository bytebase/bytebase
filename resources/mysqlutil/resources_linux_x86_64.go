//go:build linux && amd64
// +build linux,amd64

package mysqlutil

import "embed"

//go:embed mysqlutil-8.0.28-linux-glibc-2.17-x86_64.tar.gz
var resources embed.FS
