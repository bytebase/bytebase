package postgres

import "embed"

// HACK: tmp hack to use darwin/amd64 plaform binaries, assumed Rosetta2 has been installed
//
//go:embed postgres-darwin-amd64.txz
var resources embed.FS
