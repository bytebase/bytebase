package server

import (
	// Drivers.
	_ "github.com/pingcap/tidb/types/parser_driver"

	_ "github.com/bytebase/bytebase/backend/plugin/db/sqlite"

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
