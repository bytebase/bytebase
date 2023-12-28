//go:build !minidemo

package server

import (
	// This includes more databases such as MySQL, SQLite, TiDB.

	// Drivers.
	_ "github.com/bytebase/bytebase/backend/plugin/db/mysql"
	_ "github.com/bytebase/bytebase/backend/plugin/db/sqlite"
	_ "github.com/bytebase/bytebase/backend/plugin/db/tidb"

	// Parsers.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/standard"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/tidb"

	// Schema designer.
	_ "github.com/bytebase/bytebase/backend/plugin/schema/mysql"
	_ "github.com/bytebase/bytebase/backend/plugin/schema/tidb"

	// Advisors.
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/mysql"
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/tidb"

	// Transformers.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/sql/transform/mysql"
)
