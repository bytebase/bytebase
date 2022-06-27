//go:build !release
// +build !release

package api

// The idea of dependency gate is to use compiler check to prevent introducing unintended dependency.
// In particular:
// 1. the plugin package should NOT depend on the api package as the plugin package should be self-contained.

import (
	// TODO(rebelice): fix the incorrect dependency and uncomment these

	_ "github.com/bytebase/bytebase/plugin/advisor"
	_ "github.com/bytebase/bytebase/plugin/catalog"
	_ "github.com/bytebase/bytebase/plugin/db"
	_ "github.com/bytebase/bytebase/plugin/metric"
	_ "github.com/bytebase/bytebase/plugin/vcs"
	_ "github.com/bytebase/bytebase/plugin/webhook"
)
