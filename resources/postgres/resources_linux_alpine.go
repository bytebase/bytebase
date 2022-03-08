//go:build linux_alpine
// +build linux_alpine

package postgres

import "embed"

//go:embed postgres-linux-x86_64-alpine_linux.txz
var resources embed.FS
