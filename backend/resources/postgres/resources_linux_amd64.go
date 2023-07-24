package postgres

import "embed"

//go:embed postgres-linux-amd64.txz
var resources embed.FS
