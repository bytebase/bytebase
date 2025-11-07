package server

import (
	// This includes the first-class database, Postgres.

	// Drivers.
	_ "github.com/bytebase/bytebase/backend/plugin/db/pg"

	// Parsers.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/pg"

	// Schema designer.
	_ "github.com/bytebase/bytebase/backend/plugin/schema/pg"

	// Advisors.
	_ "github.com/bytebase/bytebase/backend/plugin/advisor/pg"

	// IM webhooks.
	_ "github.com/bytebase/bytebase/backend/plugin/webhook/dingtalk"
	_ "github.com/bytebase/bytebase/backend/plugin/webhook/feishu"
	_ "github.com/bytebase/bytebase/backend/plugin/webhook/slack"
	_ "github.com/bytebase/bytebase/backend/plugin/webhook/wecom"
)
