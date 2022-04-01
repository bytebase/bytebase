//go:build !alpine
// +build !alpine

package postgres

import "embed"

//go:embed postgres-linux-x86_64.txz
var resources embed.FS
