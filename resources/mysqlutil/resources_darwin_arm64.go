//go:build darwin && arm64
// +build darwin,arm64

package mysqlutil

import "embed"

//go:embed mysqlutil-8.0.28-macos11-arm64.tar.gz
var resources embed.FS
