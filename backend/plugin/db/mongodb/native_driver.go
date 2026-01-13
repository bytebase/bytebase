package mongodb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"google.golang.org/protobuf/types/known/durationpb"

	mongoshparser "github.com/bytebase/parser/mongodb"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

// queryWithNativeDriver executes a MongoDB query using the native Go driver instead of mongosh.
// MVP: Only supports db.collection.find() without filter.
func (d *Driver) queryWithNativeDriver(ctx context.Context, statement string, queryContext db.QueryContext) []*v1pb.QueryResult {
	statement = strings.TrimSpace(statement)
	startTime := time.Now()

	// Parse the statement
	tree, parseErrors := parseMongoShell(statement)
	if len(parseErrors) > 0 {
		return []*v1pb.QueryResult{{
			Error:     fmt.Sprintf("Parse error: %s", parseErrors[0]),
			Statement: statement,
			Latency:   durationpb.New(time.Since(startTime)),
		}}
	}

	// Use visitor to extract operation
	visitor := &mongoShellVisitor{}
	result := visitor.Visit(tree.(antlr.ParseTree))
	if visitor.err != nil {
		return []*v1pb.QueryResult{{
			Error:     visitor.err.Error(),
			Statement: statement,
			Latency:   durationpb.New(time.Since(startTime)),
		}}
	}

	op, ok := result.(*mongoOperation)
	if !ok || op == nil {
		return []*v1pb.QueryResult{{
			Error:     "Failed to parse MongoDB statement",
			Statement: statement,
			Latency:   durationpb.New(time.Since(startTime)),
		}}
	}

	// Execute the operation
	queryResult, err := d.executeOperation(ctx, op, queryContext)
	if err != nil {
		return []*v1pb.QueryResult{{
			Error:     err.Error(),
			Statement: statement,
			Latency:   durationpb.New(time.Since(startTime)),
		}}
	}

	queryResult.Statement = statement
	queryResult.Latency = durationpb.New(time.Since(startTime))
	return []*v1pb.QueryResult{queryResult}
}

// parseMongoShell parses a MongoDB shell statement and returns the parse tree.
func parseMongoShell(statement string) (antlr.Tree, []string) {
	is := antlr.NewInputStream(statement)
	lexer := mongoshparser.NewMongoShellLexer(is)

	lexerErrors := &errorListener{}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(lexerErrors)

	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := mongoshparser.NewMongoShellParser(stream)

	parserErrors := &errorListener{}
	parser.RemoveErrorListeners()
	parser.AddErrorListener(parserErrors)

	tree := parser.Program()

	var allErrors []string
	allErrors = append(allErrors, lexerErrors.errors...)
	allErrors = append(allErrors, parserErrors.errors...)

	return tree, allErrors
}

// errorListener collects parse errors.
type errorListener struct {
	antlr.DefaultErrorListener
	errors []string
}

func (l *errorListener) SyntaxError(_ antlr.Recognizer, _ any, line, column int, msg string, _ antlr.RecognitionException) {
	l.errors = append(l.errors, fmt.Sprintf("line %d:%d %s", line, column, msg))
}

// operationType represents the type of MongoDB operation.
type operationType int

const (
	opUnknown operationType = iota
	opFind
)

// mongoOperation represents a parsed MongoDB operation.
type mongoOperation struct {
	opType     operationType
	collection string
	// For find operations - MVP: filter is always empty
	filter bson.D
	// Cursor modifiers - MVP: not yet supported
	sort       bson.D
	limit      *int64
	skip       *int64
	projection bson.D
}

// mongoShellVisitor visits the parse tree and extracts the operation.
type mongoShellVisitor struct {
	mongoshparser.BaseMongoShellParserVisitor
	err error
}

func (v *mongoShellVisitor) Visit(tree antlr.ParseTree) any {
	return tree.Accept(v)
}

func (v *mongoShellVisitor) VisitChildren(node antlr.RuleNode) any {
	for _, child := range node.GetChildren() {
		if pt, ok := child.(antlr.ParseTree); ok {
			result := pt.Accept(v)
			if result != nil {
				return result
			}
		}
	}
	return nil
}

func (v *mongoShellVisitor) VisitProgram(ctx *mongoshparser.ProgramContext) any {
	for _, stmt := range ctx.AllStatement() {
		result := stmt.Accept(v)
		if result != nil {
			return result
		}
	}
	return nil
}

func (v *mongoShellVisitor) VisitStatement(ctx *mongoshparser.StatementContext) any {
	if dbStmt := ctx.DbStatement(); dbStmt != nil {
		return dbStmt.Accept(v)
	}
	return nil
}

func (v *mongoShellVisitor) VisitCollectionOperation(ctx *mongoshparser.CollectionOperationContext) any {
	op := &mongoOperation{}

	// Extract collection name from CollectionAccess
	if collAccess := ctx.CollectionAccess(); collAccess != nil {
		op.collection = v.extractCollectionName(collAccess)
	}

	// Extract method chain
	if methodChain := ctx.MethodChain(); methodChain != nil {
		v.extractMethodChain(methodChain, op)
	}

	return op
}

func (*mongoShellVisitor) extractCollectionName(ctx mongoshparser.ICollectionAccessContext) string {
	switch c := ctx.(type) {
	case *mongoshparser.DotAccessContext:
		if ident := c.Identifier(); ident != nil {
			return ident.GetText()
		}
	case *mongoshparser.BracketAccessContext:
		if strLit := c.StringLiteral(); strLit != nil {
			s := strLit.GetText()
			// Remove quotes
			return s[1 : len(s)-1]
		}
	case *mongoshparser.GetCollectionAccessContext:
		if strLit := c.StringLiteral(); strLit != nil {
			s := strLit.GetText()
			return s[1 : len(s)-1]
		}
	}
	return ""
}

func (v *mongoShellVisitor) extractMethodChain(ctx mongoshparser.IMethodChainContext, op *mongoOperation) {
	for _, methodCall := range ctx.AllMethodCall() {
		v.extractMethodCall(methodCall, op)
	}
}

func (v *mongoShellVisitor) extractMethodCall(ctx mongoshparser.IMethodCallContext, op *mongoOperation) {
	switch {
	case ctx.FindMethod() != nil:
		op.opType = opFind
		// MVP: find() without filter - we'll support filter in later iterations
	case ctx.FindOneMethod() != nil:
		op.opType = opFind
		one := int64(1)
		op.limit = &one
	case ctx.SortMethod() != nil,
		ctx.LimitMethod() != nil,
		ctx.SkipMethod() != nil,
		ctx.ProjectionMethod() != nil:
		// MVP: cursor modifiers not yet supported, ignore for now
	default:
		if genericMethod := ctx.GenericMethod(); genericMethod != nil {
			methodName := ""
			if ident := genericMethod.Identifier(); ident != nil {
				methodName = ident.GetText()
			}
			v.err = errors.Errorf("unsupported method: %s", methodName)
		}
	}
}

// executeOperation executes a parsed MongoDB operation.
func (d *Driver) executeOperation(ctx context.Context, op *mongoOperation, queryContext db.QueryContext) (*v1pb.QueryResult, error) {
	if op == nil {
		return nil, errors.New("no operation to execute")
	}

	switch op.opType {
	case opFind:
		return d.executeFind(ctx, op, queryContext)
	default:
		return nil, errors.New("unsupported operation type")
	}
}

// executeFind executes a find operation.
func (d *Driver) executeFind(ctx context.Context, op *mongoOperation, queryContext db.QueryContext) (*v1pb.QueryResult, error) {
	collection := d.client.Database(d.databaseName).Collection(op.collection)

	filter := op.filter
	if filter == nil {
		filter = bson.D{}
	}

	opts := options.Find()
	if op.sort != nil {
		opts.SetSort(op.sort)
	}
	if op.projection != nil {
		opts.SetProjection(op.projection)
	}
	if op.skip != nil {
		opts.SetSkip(*op.skip)
	}

	// Apply limit from operation or query context
	limit := int64(0)
	if op.limit != nil {
		limit = *op.limit
	}
	if queryContext.Limit > 0 && (limit == 0 || int64(queryContext.Limit) < limit) {
		limit = int64(queryContext.Limit)
	}
	if limit > 0 {
		opts.SetLimit(limit)
	}

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute find")
	}
	defer cursor.Close(ctx)

	var results []bson.M
	if err := cursor.All(ctx, &results); err != nil {
		return nil, errors.Wrap(err, "failed to read cursor")
	}

	// Format results using Extended JSON (relaxed mode) - same as existing mongosh code path
	result := &v1pb.QueryResult{
		ColumnNames:     []string{"result"},
		ColumnTypeNames: []string{"TEXT"},
	}

	for _, doc := range results {
		// Use relaxed EJSON format (canonical=false), same as convertRowsResult in mongodb.go
		formatted, err := bson.MarshalExtJSONIndent(doc, false, false, "", "  ")
		if err != nil {
			return nil, errors.Wrap(err, "failed to format document")
		}
		result.Rows = append(result.Rows, &v1pb.QueryRow{
			Values: []*v1pb.RowValue{
				{Kind: &v1pb.RowValue_StringValue{StringValue: string(formatted)}},
			},
		})
	}

	return result, nil
}
