//go:build !alpine
// +build !alpine

package postgres

import "embed"

//go:embed postgres-linux-arm_64.txz
var resources embed.FS
