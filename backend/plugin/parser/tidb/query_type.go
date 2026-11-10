package tidb

import (
	"github.com/bytebase/omni/tidb/ast"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// getQueryType classifies an omni TiDB statement. The second return value is
// true for EXPLAIN ANALYZE over a read, which runs the wrapped statement. omni
// has no DML/DDL marker interface (pingcap's DMLNode/DDLNode), so the
// statement families are enumerated.
func getQueryType(node ast.Node, allSystem bool) (base.QueryType, bool) {
	switch n := node.(type) {
	case *ast.SelectStmt, *ast.TableStmt, *ast.ValuesStmt:
		if allSystem {
			return base.SelectInfoSchema, false
		}
		return base.Select, false
	case *ast.ExplainStmt:
		if n.Analyze {
			t, _ := getQueryType(n.Stmt, allSystem)
			return t, true
		}
		return base.Explain, false
	case *ast.ShowStmt:
		return base.SelectInfoSchema, false
	case *ast.SetStmt, *ast.SetConfigStmt, *ast.SetResourceGroupStmt, *ast.SetRoleStmt,
		*ast.SetBindingStmt, *ast.SetDefaultRoleStmt, *ast.SetTransactionStmt:
		// Treat SAFE SET as a select statement.
		return base.Select, false
	case *ast.CreateDatabaseStmt, *ast.CreateTableStmt, *ast.CreateViewStmt, *ast.CreateIndexStmt,
		*ast.CreateSequenceStmt, *ast.CreateUserStmt, *ast.CreateRoleStmt, *ast.CreateEventStmt,
		*ast.CreateTriggerStmt, *ast.CreateFunctionStmt, *ast.CreateBindingStmt,
		*ast.CreatePlacementPolicyStmt, *ast.CreateResourceGroupStmt, *ast.CreateStatisticsStmt,
		*ast.CreateSpatialRefSysStmt,
		*ast.AlterDatabaseStmt, *ast.AlterTableStmt, *ast.AlterViewStmt, *ast.AlterSequenceStmt,
		*ast.AlterUserStmt, *ast.AlterInstanceStmt, *ast.AlterEventStmt, *ast.AlterRoutineStmt,
		*ast.AlterResourceGroupStmt, *ast.AlterPlacementPolicyStmt, *ast.AlterRangeStmt,
		*ast.DropDatabaseStmt, *ast.DropTableStmt, *ast.DropViewStmt, *ast.DropIndexStmt,
		*ast.DropSequenceStmt, *ast.DropUserStmt, *ast.DropRoleStmt, *ast.DropEventStmt,
		*ast.DropTriggerStmt, *ast.DropRoutineStmt, *ast.DropStatisticsStmt, *ast.DropStatsStmt,
		*ast.DropBindingStmt, *ast.DropPlacementPolicyStmt, *ast.DropResourceGroupStmt,
		*ast.DropSpatialRefSysStmt, *ast.DropQueryWatchStmt,
		*ast.RenameTableStmt, *ast.RenameUserStmt, *ast.TruncateStmt, *ast.RecoverTableStmt,
		*ast.FlashbackTableStmt, *ast.FlashbackDatabaseStmt, *ast.FlashbackToTimestampStmt,
		*ast.GrantStmt, *ast.RevokeStmt, *ast.GrantRoleStmt, *ast.RevokeRoleStmt,
		*ast.AddQueryWatchStmt, *ast.CloneStmt, *ast.InstallPluginStmt, *ast.UninstallPluginStmt,
		*ast.InstallComponentStmt, *ast.UninstallComponentStmt:
		return base.DDL, false
	case *ast.InsertStmt, *ast.UpdateStmt, *ast.DeleteStmt, *ast.LoadDataStmt, *ast.ImportIntoStmt,
		*ast.BatchStmt, *ast.SplitRegionStmt, *ast.CallStmt, *ast.DoStmt, *ast.PrepareStmt,
		*ast.ExecuteStmt, *ast.DeallocateStmt, *ast.BeginStmt, *ast.CommitStmt, *ast.RollbackStmt,
		*ast.SavepointStmt, *ast.LockTablesStmt, *ast.UnlockTablesStmt, *ast.AnalyzeTableStmt,
		*ast.OptimizeTableStmt, *ast.CheckTableStmt, *ast.ChecksumTableStmt, *ast.RepairTableStmt,
		*ast.FlushStmt, *ast.KillStmt, *ast.ShutdownStmt, *ast.RestartStmt, *ast.XAStmt,
		*ast.TraceStmt, *ast.PlanReplayerStmt, *ast.LoadStatsStmt, *ast.CacheIndexStmt,
		*ast.LoadIndexIntoCacheStmt:
		return base.DML, false
	}

	return base.QueryTypeUnknown, false
}
