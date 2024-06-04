package sheet

import (
	"context"
	"io"
	"log/slog"
	"strings"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/db/mssql"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/plsql"
	tsqlbatch "github.com/bytebase/bytebase/backend/plugin/parser/tsql/batch"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// Manager is the coordinator for all sheets and SQL statements.
type Manager struct {
	store *store.Store
}

// NewManager creates a new sheet manager.
func NewManager(store *store.Store) *Manager {
	return &Manager{
		store: store,
	}
}

func (sm *Manager) CreateSheet(ctx context.Context, sheet *store.SheetMessage) (*store.SheetMessage, error) {
	if sheet.Payload == nil {
		sheet.Payload = &storepb.SheetPayload{}
	}
	sheet.Payload.Commands = getSheetCommands(sheet.Payload.Engine, sheet.Statement)

	return sm.store.CreateSheet(ctx, sheet)
}

func getSheetCommands(engine storepb.Engine, statement string) []*storepb.SheetCommand {
	switch engine {
	case storepb.Engine_MYSQL:
		if len(statement) > common.MaxSheetCheckSize {
			return nil
		}
	case storepb.Engine_POSTGRES:
	case storepb.Engine_ORACLE:
	case storepb.Engine_MSSQL:
	case storepb.Engine_DYNAMODB:
	default:
		return nil
	}

	switch engine {
	case storepb.Engine_ORACLE:
		return getSheetCommandsForOracle(statement)
	case storepb.Engine_MSSQL:
		return getSheetCommandsForMSSQL(statement)
	default:
		return getSheetCommandsGeneral(engine, statement)
	}
}

func getSheetCommandsGeneral(engine storepb.Engine, statement string) []*storepb.SheetCommand {
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

func getSheetCommandsForOracle(statement string) []*storepb.SheetCommand {
	singleSQLs, err := plsql.SplitSQL(statement)
	if err != nil {
		if !strings.Contains(err.Error(), "not supported") {
			slog.Warn("failed to get sheet command for oracle", "statement", statement)
		}
		return nil
	}

	var sheetCommands []*storepb.SheetCommand
	for _, s := range singleSQLs {
		sheetCommands = append(sheetCommands, &storepb.SheetCommand{
			Start: int32(s.ByteOffsetStart),
			End:   int32(s.ByteOffsetEnd),
		})
	}
	return sheetCommands
}

func getSheetCommandsForMSSQL(statement string) []*storepb.SheetCommand {
	var sheetCommands []*storepb.SheetCommand
	p := 0

	batch := mssql.NewBatch(statement)
	for {
		command, err := batch.Next()
		if err == io.EOF {
			np := p + len(batch.String())
			sheetCommands = append(sheetCommands, &storepb.SheetCommand{
				Start: int32(p),
				End:   int32(np),
			})
			break
		}
		if err != nil {
			slog.Warn("failed to get sheet commands for mssql", "statement", statement)
			return nil
		}
		if command == nil {
			continue
		}
		switch command.(type) {
		case *tsqlbatch.GoCommand:
			np := p + len(batch.String())
			sheetCommands = append(sheetCommands, &storepb.SheetCommand{
				Start: int32(p),
				End:   int32(np),
			})
			p = np
		default:
		}
	}
	return sheetCommands
}
