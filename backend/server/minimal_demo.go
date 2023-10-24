//go:build !minidemo

package server

import (
	// This includes more databases such as MySQL, SQLite, TiDB.

	// Drivers.
	_ "github.com/bytebase/bytebase/backend/plugin/db/mysql"
	_ "github.com/bytebase/bytebase/backend/plugin/db/sqlite"

	// Parsers.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/standard"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/tidb"

	// Advisors.
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/mysql"

	// Transformers.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/sql/transform/mysql"
)
