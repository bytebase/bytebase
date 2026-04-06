package mongodb

import (
	"slices"

	"github.com/bytebase/omni/mongo/analysis"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// GetQueryType parses a MongoDB shell statement and returns its query type.
func GetQueryType(statement string) base.QueryType {
	stmts, err := ParseMongoShell(statement)
	if err != nil || len(stmts) == 0 {
		return base.DML
	}

	node, ok := GetOmniNode(stmts[0].AST)
	if !ok || node == nil {
		return base.DML
	}

	a := analysis.Analyze(node)
	if a == nil {
		return base.DML
	}

	return omniOperationToQueryType(a)
}

// omniOperationToQueryType maps an omni StatementAnalysis to a base.QueryType.
func omniOperationToQueryType(a *analysis.StatementAnalysis) base.QueryType {
	switch a.Operation {
	case analysis.OpFind, analysis.OpFindOne, analysis.OpCount, analysis.OpDistinct:
		return base.Select
	case analysis.OpAggregate:
		if slices.Contains(a.PipelineStages, "$out") || slices.Contains(a.PipelineStages, "$merge") {
			return base.DML
		}
		return base.Select
	case analysis.OpRead:
		// OpRead covers runCommand({find:...}) and metadata reads (getIndexes, stats, etc.).
		// For runCommand/adminCommand, treat as Select (data access, not just metadata).
		// For collection metadata methods, treat as SelectInfoSchema.
		if a.MethodName == "runCommand" || a.MethodName == "adminCommand" {
			return base.Select
		}
		return base.SelectInfoSchema
	case analysis.OpInfo:
		// OpInfo covers show commands, db info methods, rs/sh info methods,
		// and runCommand with info commands. Also the default for unknown db
		// methods in omni — use SelectInfoSchema only for known info methods;
		// default to DML for safety on unknowns.
		switch a.MethodName {
		case "show", "status",
			"getCollectionNames", "getCollectionInfos",
			"serverStatus", "serverBuildInfo", "version",
			"hostInfo", "getName", "listCommands", "stats",
			"conf", "config", "printReplicationInfo", "printSecondaryReplicationInfo",
			"getBalancerState", "isBalancerRunning":
			return base.SelectInfoSchema
		case "runCommand", "adminCommand":
			// runCommand({serverStatus:1}) etc. — info commands routed through OpInfo.
			return base.SelectInfoSchema
		default:
			return base.DML
		}
	case analysis.OpWrite:
		return base.DML
	case analysis.OpAdmin:
		return base.DDL
	case analysis.OpExplain:
		return base.Explain
	default:
		return base.DML
	}
}
