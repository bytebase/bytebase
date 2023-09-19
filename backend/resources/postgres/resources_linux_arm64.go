package postgres

import "embed"

//go:embed postgres-linux-arm64.txz
var resources embed.FS
