package mongodb

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"

	"github.com/bytebase/parser/mongodb"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// GetQueryType parses a MongoDB shell statement and returns its query type.
func GetQueryType(statement string) base.QueryType {
	stmts, err := ParseMongoShell(statement)
	if err != nil || len(stmts) == 0 {
		return base.DML
	}

	ast, ok := stmts[0].AST.(*base.ANTLRAST)
	if !ok || ast.Tree == nil {
		return base.DML
	}

	l := &queryTypeListener{
		result: base.DML,
	}
	antlr.ParseTreeWalkerDefault.Walk(l, ast.Tree)
	return l.result
}

type queryTypeListener struct {
	*mongodb.BaseMongoShellParserListener

	result base.QueryType
}

// Shell commands.

func (l *queryTypeListener) EnterShowDatabases(_ *mongodb.ShowDatabasesContext) {
	l.result = base.SelectInfoSchema
}

func (l *queryTypeListener) EnterShowCollections(_ *mongodb.ShowCollectionsContext) {
	l.result = base.SelectInfoSchema
}

// Database methods: info/read-only.

func (l *queryTypeListener) EnterGetCollectionNames(_ *mongodb.GetCollectionNamesContext) {
	l.result = base.SelectInfoSchema
}

func (l *queryTypeListener) EnterGetCollectionInfos(_ *mongodb.GetCollectionInfosContext) {
	l.result = base.SelectInfoSchema
}

//nolint:staticcheck // Method name must match ANTLR-generated listener interface.
func (l *queryTypeListener) EnterDbStats(_ *mongodb.DbStatsContext) {
	l.result = base.SelectInfoSchema
}

func (l *queryTypeListener) EnterServerStatus(_ *mongodb.ServerStatusContext) {
	l.result = base.SelectInfoSchema
}

func (l *queryTypeListener) EnterServerBuildInfo(_ *mongodb.ServerBuildInfoContext) {
	l.result = base.SelectInfoSchema
}

//nolint:staticcheck // Method name must match ANTLR-generated listener interface.
func (l *queryTypeListener) EnterDbVersion(_ *mongodb.DbVersionContext) {
	l.result = base.SelectInfoSchema
}

func (l *queryTypeListener) EnterHostInfo(_ *mongodb.HostInfoContext) {
	l.result = base.SelectInfoSchema
}

func (l *queryTypeListener) EnterListCommands(_ *mongodb.ListCommandsContext) {
	l.result = base.SelectInfoSchema
}

func (l *queryTypeListener) EnterGetName(_ *mongodb.GetNameContext) {
	l.result = base.SelectInfoSchema
}

// Database methods: DDL.

func (l *queryTypeListener) EnterCreateCollection(_ *mongodb.CreateCollectionContext) {
	l.result = base.DDL
}

func (l *queryTypeListener) EnterDropDatabase(_ *mongodb.DropDatabaseContext) {
	l.result = base.DDL
}

// Database methods: DML.

func (l *queryTypeListener) EnterGetSiblingDB(_ *mongodb.GetSiblingDBContext) {
	l.result = base.DML
}

func (l *queryTypeListener) EnterGetMongo(_ *mongodb.GetMongoContext) {
	l.result = base.DML
}

// runCommand / adminCommand: classify by argument text.

func (l *queryTypeListener) EnterRunCommand(ctx *mongodb.RunCommandContext) {
	l.result = classifyCommandArguments(ctx.Arguments())
}

func (l *queryTypeListener) EnterAdminCommand(ctx *mongodb.AdminCommandContext) {
	l.result = classifyCommandArguments(ctx.Arguments())
}

func classifyCommandArguments(args mongodb.IArgumentsContext) base.QueryType {
	if args == nil {
		return base.DML
	}
	argText := args.GetText()

	// Read-only query commands.
	for _, keyword := range []string{"find", "aggregate", "count", "distinct"} {
		if strings.Contains(argText, keyword) {
			return base.Select
		}
	}
	// Info/metadata commands.
	for _, keyword := range []string{
		"serverStatus", "listCollections", "listIndexes", "listDatabases",
		"collStats", "dbStats", "hostInfo", "buildInfo", "connectionStatus",
	} {
		if strings.Contains(argText, keyword) {
			return base.SelectInfoSchema
		}
	}
	// DDL commands.
	for _, keyword := range []string{
		"create", "drop", "createIndexes", "dropIndexes",
		"renameCollection", "collMod",
	} {
		if strings.Contains(argText, keyword) {
			return base.DDL
		}
	}

	return base.DML
}

// Collection operations: db.collection.method().

func (l *queryTypeListener) EnterCollectionOperation(ctx *mongodb.CollectionOperationContext) {
	chain := ctx.MethodChain()
	if chain == nil {
		l.result = base.DML
		return
	}
	l.result = classifyMethodChain(chain)
}

func classifyMethodChain(chain mongodb.IMethodChainContext) base.QueryType {
	// Check if .explain() prefix exists (db.collection.explain().find())
	if chain.CollectionExplainMethod() != nil {
		return base.Explain
	}

	// Check cursor methods for explain().
	for _, cursor := range chain.AllCursorMethodCall() {
		if cursor.ExplainMethod() != nil {
			return base.Explain
		}
	}

	// Classify based on the collection method call.
	mc := chain.CollectionMethodCall()
	if mc != nil {
		return classifyCollectionMethodCall(mc)
	}

	return base.DML
}

func classifyCollectionMethodCall(mc mongodb.ICollectionMethodCallContext) base.QueryType {
	switch {
	// Read methods -> Select.
	case mc.FindMethod() != nil,
		mc.FindOneMethod() != nil,
		mc.CountDocumentsMethod() != nil,
		mc.EstimatedDocumentCountMethod() != nil,
		mc.CollectionCountMethod() != nil,
		mc.DistinctMethod() != nil:
		return base.Select

	// Aggregate: Select by default, DML if $out or $merge.
	case mc.AggregateMethod() != nil:
		return classifyAggregate(mc.AggregateMethod())

	// Write methods -> DML.
	case mc.InsertOneMethod() != nil,
		mc.InsertManyMethod() != nil,
		mc.UpdateOneMethod() != nil,
		mc.UpdateManyMethod() != nil,
		mc.DeleteOneMethod() != nil,
		mc.DeleteManyMethod() != nil,
		mc.ReplaceOneMethod() != nil,
		mc.FindOneAndUpdateMethod() != nil,
		mc.FindOneAndReplaceMethod() != nil,
		mc.FindOneAndDeleteMethod() != nil,
		mc.CollectionInsertMethod() != nil,
		mc.CollectionRemoveMethod() != nil,
		mc.UpdateMethod() != nil,
		mc.BulkWriteMethod() != nil,
		mc.FindAndModifyMethod() != nil,
		mc.MapReduceMethod() != nil:
		return base.DML

	// DDL methods.
	case mc.CreateIndexMethod() != nil,
		mc.CreateIndexesMethod() != nil,
		mc.DropIndexMethod() != nil,
		mc.DropIndexesMethod() != nil,
		mc.DropMethod() != nil,
		mc.RenameCollectionMethod() != nil,
		mc.HideIndexMethod() != nil,
		mc.UnhideIndexMethod() != nil,
		mc.ReIndexMethod() != nil,
		mc.CreateSearchIndexMethod() != nil,
		mc.CreateSearchIndexesMethod() != nil,
		mc.DropSearchIndexMethod() != nil,
		mc.UpdateSearchIndexMethod() != nil:
		return base.DDL

	// Info methods -> SelectInfoSchema.
	case mc.GetIndexesMethod() != nil,
		mc.StatsMethod() != nil,
		mc.StorageSizeMethod() != nil,
		mc.TotalIndexSizeMethod() != nil,
		mc.TotalSizeMethod() != nil,
		mc.DataSizeMethod() != nil,
		mc.IsCappedMethod() != nil,
		mc.ValidateMethod() != nil,
		mc.LatencyStatsMethod() != nil,
		mc.GetShardDistributionMethod() != nil,
		mc.GetShardVersionMethod() != nil,
		mc.AnalyzeShardKeyMethod() != nil:
		return base.SelectInfoSchema

	// Explain.
	case mc.CollectionExplainMethod() != nil:
		return base.Explain

	default:
		return base.DML
	}
}

func classifyAggregate(ctx mongodb.IAggregateMethodContext) base.QueryType {
	if ctx == nil {
		return base.Select
	}
	args := ctx.Arguments()
	if args != nil {
		argText := args.GetText()
		if strings.Contains(argText, "$out") || strings.Contains(argText, "$merge") {
			return base.DML
		}
	}
	return base.Select
}

// Bulk operations.

func (l *queryTypeListener) EnterBulkStatement(_ *mongodb.BulkStatementContext) {
	l.result = base.DML
}

// RS statements.

func (l *queryTypeListener) EnterRsStatement(ctx *mongodb.RsStatementContext) {
	id := ctx.Identifier()
	if id == nil {
		l.result = base.DML
		return
	}
	name := id.GetText()
	switch name {
	case "status", "printReplicationInfo", "printSecondaryReplicationInfo", "help", "conf":
		l.result = base.SelectInfoSchema
	default:
		l.result = base.DML
	}
}

// SH statements.

func (l *queryTypeListener) EnterShStatement(ctx *mongodb.ShStatementContext) {
	id := ctx.Identifier()
	if id == nil {
		l.result = base.DML
		return
	}
	name := id.GetText()
	switch name {
	case "status", "getBalancerState", "isBalancerRunning",
		"getShardedDataDistribution", "isConfigShardEnabled",
		"listShards", "help", "balancerCollectionStatus",
		"checkMetadataConsistency":
		l.result = base.SelectInfoSchema
	default:
		l.result = base.DML
	}
}

// Encryption statements.

func (l *queryTypeListener) EnterKeyVaultStatement(ctx *mongodb.KeyVaultStatementContext) {
	l.result = classifyEncryptionIdentifiers(ctx.AllIdentifier())
}

func (l *queryTypeListener) EnterClientEncryptionStatement(ctx *mongodb.ClientEncryptionStatementContext) {
	l.result = classifyEncryptionIdentifiers(ctx.AllIdentifier())
}

func classifyEncryptionIdentifiers(identifiers []mongodb.IIdentifierContext) base.QueryType {
	if len(identifiers) == 0 {
		return base.DML
	}
	// Use the last identifier in the chain.
	last := identifiers[len(identifiers)-1]
	name := last.GetText()
	switch name {
	case "getKey", "getKeyByAltName", "getKeys", "decrypt", "encrypt", "encryptExpression":
		return base.SelectInfoSchema
	default:
		return base.DML
	}
}

// Plan cache statements.

func (l *queryTypeListener) EnterPlanCacheStatement(ctx *mongodb.PlanCacheStatementContext) {
	identifiers := ctx.AllIdentifier()
	if len(identifiers) == 0 {
		l.result = base.DML
		return
	}
	// Check identifiers in the method chain after getPlanCache().
	for _, id := range identifiers {
		name := id.GetText()
		switch name {
		case "list", "help":
			l.result = base.SelectInfoSchema
			return
		case "clear", "clearPlansByQuery":
			l.result = base.DML
			return
		}
	}
	l.result = base.DML
}

// SP statements.

func (l *queryTypeListener) EnterSpStatement(ctx *mongodb.SpStatementContext) {
	identifiers := ctx.AllIdentifier()
	if len(identifiers) == 0 {
		l.result = base.DML
		return
	}

	// Single-identifier form: sp.method()
	if len(identifiers) == 1 {
		name := identifiers[0].GetText()
		switch name {
		case "listConnections", "listStreamProcessors":
			l.result = base.SelectInfoSchema
		default:
			l.result = base.DML
		}
		return
	}

	// Compound form: sp.X.Y()
	secondName := identifiers[1].GetText()
	switch secondName {
	case "stats", "sample":
		l.result = base.SelectInfoSchema
	default:
		l.result = base.DML
	}
}

// Native function calls.

func (l *queryTypeListener) EnterNativeFunctionCall(_ *mongodb.NativeFunctionCallContext) {
	l.result = base.DML
}

// Connection statements.

func (l *queryTypeListener) EnterMongoConnection(_ *mongodb.MongoConnectionContext) {
	l.result = base.DML
}

func (l *queryTypeListener) EnterConnectCall(_ *mongodb.ConnectCallContext) {
	l.result = base.DML
}

//nolint:staticcheck // Method name must match ANTLR-generated listener interface.
func (l *queryTypeListener) EnterDbGetMongoChain(_ *mongodb.DbGetMongoChainContext) {
	l.result = base.DML
}
