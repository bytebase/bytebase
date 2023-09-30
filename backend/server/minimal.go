package server

import (
	// This includes the first-class databases including MySQL, Postgres, SQLite, TiDB.

	// Drivers.
	_ "github.com/bytebase/bytebase/backend/plugin/db/mysql"
	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"
	_ "github.com/bytebase/bytebase/backend/plugin/db/sqlite"

	// Parsers.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/standard"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/tidb"

	// Advisors.
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/fake"
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/mysql"
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/pg"

	// Differs.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/sql/differ/mysql"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/sql/differ/pg"

	// Editors.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/sql/edit/mysql"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/sql/edit/pg"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/sql/engine/pg"

	// Transformers.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/sql/transform/mysql"
)
