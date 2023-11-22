package server

import (
	// This includes the first-class database, Postgres.

	// Drivers.
	_ "github.com/bytebase/bytebase/backend/plugin/db/mongodb"
	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"

	// Parsers.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/pg"

	// Advisors.
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/pg"

	// Editors.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/sql/engine/pg"
)
