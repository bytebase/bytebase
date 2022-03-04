//go:build release
// +build release

package postgres

import "embed"

//go:embed postgres-linux-x86_64-alpine_linux.txz
var resources embed.FS
