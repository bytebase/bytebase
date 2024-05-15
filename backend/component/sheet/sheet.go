package sheet

import (
	"context"
	"log/slog"
	"strings"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func CreateSheet(ctx context.Context, s *store.Store, sheet *store.SheetMessage) (*store.SheetMessage, error) {
	if sheet.Payload == nil {
		sheet.Payload = &storepb.SheetPayload{}
	}
	sheet.Payload.Commands = getSheetCommands(sheet.Payload.Engine, sheet.Statement)

	return s.CreateSheet(ctx, sheet)
}

func getSheetCommands(engine storepb.Engine, statement string) []*storepb.SheetCommand {
	// Do get sheet commands if
	// - engine=MYSQL and sql is not large
	// - engine=POSTGRES
	switch engine {
	case storepb.Engine_MYSQL:
		if len(statement) > common.MaxSheetCheckSize {
			return nil
		}
	case storepb.Engine_POSTGRES:
	default:
		return nil
	}

	singleSQLs, err := base.SplitMultiSQL(engine, statement)
	if err != nil {
		if !strings.Contains(err.Error(), "not supported") {
			slog.Warn("failed to split multi sql", "engine", engine.String(), "statement", statement)
		}
		return nil
	}

	var sheetCommands []*storepb.SheetCommand
	p := 0
	for _, s := range singleSQLs {
		np := p + len(s.Text)
		sheetCommands = append(sheetCommands, &storepb.SheetCommand{
			Start: int32(p),
			End:   int32(np),
		})
		p = np
	}
	return sheetCommands
}
