//go:build darwin && amd64
// +build darwin,amd64

package mysqlutil

import "embed"

//go:embed mysqlutil-8.0.28-macos11-x86_64.tar.gz
var resources embed.FS
