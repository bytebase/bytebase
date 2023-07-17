package v1

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/exp/slices"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/common"
	enterpriseAPI "github.com/bytebase/bytebase/backend/enterprise/api"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// SchemaDesignService implements SchemaDesignServiceServer interface.
type SchemaDesignService struct {
	v1pb.UnimplementedSchemaDesignServiceServer
	store          *store.Store
	licenseService enterpriseAPI.LicenseService
}

// NewSchemaDesignService creates a new SchemaDesignService.
func NewSchemaDesignService(store *store.Store, licenseService enterpriseAPI.LicenseService) *SchemaDesignService {
	return &SchemaDesignService{
		store:          store,
		licenseService: licenseService,
	}
}

// GetSchemaDesign gets the schema design.
func (s *SchemaDesignService) GetSchemaDesign(ctx context.Context, request *v1pb.GetSchemaDesignRequest) (*v1pb.SchemaDesign, error) {
	_, sheetID, err := getProjectResourceIDAndSchemaDesignSheetID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	sheetUID, err := strconv.Atoi(sheetID)
	if err != nil || sheetUID <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid sheet id %s, must be positive integer", sheetID))
	}
	schemaDesignSheetType := storepb.SheetPayload_SCHEMA_DESIGN.String()
	sheet, err := s.getSheet(ctx, &store.FindSheetMessage{
		UID:         &sheetUID,
		PayloadType: &schemaDesignSheetType,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	schemaDesign, err := s.convertSheetToSchemaDesign(ctx, sheet)
	if err != nil {
		return nil, err
	}
	return schemaDesign, nil
}

// ListSchemaDesigns lists schema designs.
func (s *SchemaDesignService) ListSchemaDesigns(ctx context.Context, request *v1pb.ListSchemaDesignsRequest) (*v1pb.ListSchemaDesignsResponse, error) {
	projectID, err := getProjectID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	schemaDesignSheetType := storepb.SheetPayload_SCHEMA_DESIGN.String()
	sheetFind := &store.FindSheetMessage{
		PayloadType: &schemaDesignSheetType,
	}
	if projectID != "-" {
		project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
			ResourceID: &projectID,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get project: %v", err))
		}
		if project == nil {
			return nil, status.Errorf(codes.NotFound, fmt.Sprintf("project not found: %v", projectID))
		}
		sheetFind.ProjectUID = &project.UID
	}
	sheets, err := s.listSheets(ctx, sheetFind)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to list sheet: %v", err))
	}

	schemaDesigns := make([]*v1pb.SchemaDesign, 0)
	for _, sheet := range sheets {
		schemaDesign, err := s.convertSheetToSchemaDesign(ctx, sheet)
		if err != nil {
			return nil, err
		}
		schemaDesigns = append(schemaDesigns, schemaDesign)
	}
	response := &v1pb.ListSchemaDesignsResponse{
		SchemaDesigns: schemaDesigns,
	}
	return response, nil
}

// CreateSchemaDesign creates a new schema design.
func (s *SchemaDesignService) CreateSchemaDesign(ctx context.Context, request *v1pb.CreateSchemaDesignRequest) (*v1pb.SchemaDesign, error) {
	projectID, err := getProjectID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get project: %v", err))
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("project not found: %v", projectID))
	}
	currentPrincipalID := ctx.Value(common.PrincipalIDContextKey).(int)
	schemaDesign := request.SchemaDesign
	instanceID, databaseName, err := getInstanceDatabaseID(schemaDesign.BaselineDatabase)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	find := &store.FindDatabaseMessage{}
	databaseUID, isNumber := isNumber(databaseName)
	if isNumber {
		// Expected format: "instances/{ignored_value}/database/{uid}"
		find.UID = &databaseUID
	} else {
		// Expected format: "instances/{instance}/database/{database}"
		find.InstanceID = &instanceID
		find.DatabaseName = &databaseName
	}
	database, err := s.store.GetDatabaseV2(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if database == nil {
		return nil, status.Errorf(codes.NotFound, "database %q not found", databaseName)
	}
	schemaVersionUID, err := strconv.ParseInt(schemaDesign.SchemaVersion, 10, 64)
	if err != nil || schemaVersionUID <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid schema version %s, must be positive integer", schemaDesign.SchemaVersion))
	}
	changeHistory, err := s.store.GetInstanceChangeHistory(ctx, &store.FindInstanceChangeHistoryMessage{
		ID: &schemaVersionUID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if changeHistory == nil {
		return nil, status.Errorf(codes.NotFound, "schema version %d not found", schemaVersionUID)
	}
	schemaDesignSheetPayload := &storepb.SheetPayload{
		Type: storepb.SheetPayload_SCHEMA_DESIGN,
		SchemaDesign: &storepb.SheetPayload_SchemaDesign{
			BaselineSheetId: int64(*changeHistory.SheetID),
			Engine:          storepb.Engine(schemaDesign.Engine),
		},
	}
	payloadBytes, err := protojson.Marshal(schemaDesignSheetPayload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to marshal schema design sheet payload: %v", err))
	}
	schema, err := getDesignSchema(schemaDesign.Engine, schemaDesign.BaselineSchema, schemaDesign.SchemaMetadata)
	if err != nil {
		return nil, err
	}

	sheetCreate := &store.SheetMessage{
		Name:        schemaDesign.Title,
		ProjectUID:  project.UID,
		DatabaseUID: &database.UID,
		Statement:   schema,
		Visibility:  store.ProjectSheet,
		Source:      store.SheetFromBytebaseArtifact,
		Type:        store.SheetForSQL,
		CreatorID:   currentPrincipalID,
		UpdaterID:   currentPrincipalID,
		Payload:     string(payloadBytes),
	}
	sheet, err := s.store.CreateSheet(ctx, sheetCreate)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to create sheet: %v", err))
	}
	schemaDesign, err = s.convertSheetToSchemaDesign(ctx, sheet)
	if err != nil {
		return nil, err
	}
	return schemaDesign, nil
}

// UpdateSchemaDesign updates an existing schema design.
func (s *SchemaDesignService) UpdateSchemaDesign(ctx context.Context, request *v1pb.UpdateSchemaDesignRequest) (*v1pb.SchemaDesign, error) {
	_, sheetID, err := getProjectResourceIDAndSchemaDesignSheetID(request.SchemaDesign.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	sheetUID, err := strconv.Atoi(sheetID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid sheet id %s, must be positive integer", sheetID))
	}
	if request.UpdateMask == nil {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask is required")
	}
	if !slices.Contains(request.UpdateMask.Paths, "schema") {
		return nil, status.Errorf(codes.InvalidArgument, "schema is required")
	}

	schemaDesign := request.SchemaDesign
	schema, err := getDesignSchema(schemaDesign.Engine, schemaDesign.BaselineSchema, schemaDesign.SchemaMetadata)
	if err != nil {
		return nil, err
	}
	sheetUpdate := &store.PatchSheetMessage{
		UID:       sheetUID,
		Statement: &schema,
	}
	sheet, err := s.store.PatchSheet(ctx, sheetUpdate)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to update sheet: %v", err))
	}
	schemaDesign, err = s.convertSheetToSchemaDesign(ctx, sheet)
	if err != nil {
		return nil, err
	}
	return schemaDesign, nil
}

// DeleteSchemaDesign deletes an existing schema design.
func (s *SchemaDesignService) DeleteSchemaDesign(ctx context.Context, request *v1pb.DeleteSchemaDesignRequest) (*emptypb.Empty, error) {
	_, sheetID, err := getProjectResourceIDAndSchemaDesignSheetID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	sheetUID, err := strconv.Atoi(sheetID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid sheet id %s, must be positive integer", sheetID))
	}
	err = s.store.DeleteSheet(ctx, sheetUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to delete sheet: %v", err))
	}
	return &emptypb.Empty{}, nil
}

func (s *SchemaDesignService) listSheets(ctx context.Context, find *store.FindSheetMessage) ([]*store.SheetMessage, error) {
	currentPrincipalID := ctx.Value(common.PrincipalIDContextKey).(int)
	list, err := s.store.ListSheets(ctx, find, currentPrincipalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get sheet: %v", err))
	}
	return list, nil
}

func (s *SchemaDesignService) getSheet(ctx context.Context, find *store.FindSheetMessage) (*store.SheetMessage, error) {
	currentPrincipalID := ctx.Value(common.PrincipalIDContextKey).(int)
	sheet, err := s.store.GetSheet(ctx, find, currentPrincipalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get sheet: %v", err))
	}
	if sheet == nil {
		return nil, status.Errorf(codes.NotFound, "cannot find the sheet")
	}
	return sheet, nil
}

func (s *SchemaDesignService) convertSheetToSchemaDesign(ctx context.Context, sheet *store.SheetMessage) (*v1pb.SchemaDesign, error) {
	sheetPayload := &storepb.SheetPayload{}
	err := protojson.Unmarshal([]byte(sheet.Payload), sheetPayload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to unmarshal sheet payload: %v", err))
	}
	if sheetPayload.Type != storepb.SheetPayload_SCHEMA_DESIGN || sheetPayload.SchemaDesign == nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("unwanted sheet type: %v", sheetPayload.Type))
	}

	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		UID: &sheet.ProjectUID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get project: %v", err))
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("cannot find the project: %d", sheet.ProjectUID))
	}
	name := fmt.Sprintf("%s%s/%s%v", projectNamePrefix, project.ResourceID, schemaDesignPrefix, sheet.UID)

	database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		UID: sheet.DatabaseUID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get database: %v", err))
	}
	if database == nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("cannot find the database: %d", sheet.DatabaseUID))
	}

	creator, err := s.store.GetUserByID(ctx, sheet.CreatorID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get creator: %v", err))
	}
	if creator == nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("cannot find the creator: %d", sheet.CreatorID))
	}
	updater, err := s.store.GetUserByID(ctx, sheet.UpdaterID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get updater: %v", err))
	}
	if updater == nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("cannot find the updater: %d", sheet.UpdaterID))
	}

	engine := v1pb.Engine(sheetPayload.SchemaDesign.Engine)
	schema := sheet.Statement
	schemaMetadata, err := transformSchemaStringToDatabaseMetadata(engine, schema)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to transform schema string to database metadata: %v", err))
	}

	baselineSheetID := int(sheetPayload.SchemaDesign.BaselineSheetId)
	baselineSheet, err := s.getSheet(ctx, &store.FindSheetMessage{
		UID: &baselineSheetID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to find sheet: %v", err))
	}
	baselineSchema := ""
	schemaVersion := ""
	if baselineSheet != nil {
		baselineSchema = baselineSheet.Statement
		changeHistory, err := s.store.GetInstanceChangeHistory(ctx, &store.FindInstanceChangeHistoryMessage{
			SheetID: &baselineSheet.UID,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to find change history: %v", err))
		}
		if changeHistory != nil {
			schemaVersion = changeHistory.UID
		}
	}
	baselineSchemaMetadata, err := transformSchemaStringToDatabaseMetadata(engine, baselineSchema)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to transform schema string to database metadata: %v", err))
	}

	return &v1pb.SchemaDesign{
		Name:                   name,
		Title:                  sheet.Name,
		Schema:                 schema,
		SchemaMetadata:         schemaMetadata,
		BaselineSchema:         baselineSchema,
		BaselineSchemaMetadata: baselineSchemaMetadata,
		Engine:                 engine,
		BaselineDatabase:       fmt.Sprintf("%s%s/%s%s", instanceNamePrefix, database.InstanceID, databaseIDPrefix, database.DatabaseName),
		SchemaVersion:          schemaVersion,
		Creator:                fmt.Sprintf("users/%s", creator.Email),
		Updater:                fmt.Sprintf("users/%s", updater.Email),
		CreateTime:             timestamppb.New(sheet.CreatedTime),
		UpdateTime:             timestamppb.New(sheet.UpdatedTime),
	}, nil
}

func transformSchemaStringToDatabaseMetadata(engine v1pb.Engine, schema string) (*v1pb.DatabaseMetadata, error) {
	switch engine {
	case v1pb.Engine_MYSQL:
		return parseMySQLSchemaStringToDatabaseMetadata(schema)
	default:
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("unsupported engine: %v", engine))
	}
}

func parseMySQLSchemaStringToDatabaseMetadata(schema string) (*v1pb.DatabaseMetadata, error) {
	list, err := parser.ParseMySQL(schema)
	if err != nil {
		return nil, err
	}

	listener := &mysqlTransformer{
		state: newDatabaseState(),
	}
	listener.state.schemas[""] = newSchemaState()

	for _, stmt := range list {
		antlr.ParseTreeWalkerDefault.Walk(listener, stmt.Tree)
	}

	return listener.state.convertToDatabaseMetadata(), listener.err
}

type mysqlTransformer struct {
	*mysql.BaseMySQLParserListener

	state        *databaseState
	currentTable string
	err          error
}

type databaseState struct {
	name    string
	schemas map[string]*schemaState
}

func newDatabaseState() *databaseState {
	return &databaseState{
		schemas: make(map[string]*schemaState),
	}
}

func convertToDatabaseState(database *v1pb.DatabaseMetadata) *databaseState {
	state := newDatabaseState()
	state.name = database.Name
	for _, schema := range database.Schemas {
		state.schemas[schema.Name] = convertToSchemaState(schema)
	}
	return state
}

func (s *databaseState) convertToDatabaseMetadata() *v1pb.DatabaseMetadata {
	schemas := []*v1pb.SchemaMetadata{}
	for _, schema := range s.schemas {
		schemas = append(schemas, schema.convertToSchemaMetadata())
	}
	sort.Slice(schemas, func(i, j int) bool {
		return schemas[i].Name < schemas[j].Name
	})
	return &v1pb.DatabaseMetadata{
		Name:    s.name,
		Schemas: schemas,
		// Unsupported, for tests only.
		Extensions: []*v1pb.ExtensionMetadata{},
	}
}

type schemaState struct {
	name   string
	tables map[string]*tableState
}

func newSchemaState() *schemaState {
	return &schemaState{
		tables: make(map[string]*tableState),
	}
}

func convertToSchemaState(schema *v1pb.SchemaMetadata) *schemaState {
	state := newSchemaState()
	state.name = schema.Name
	for _, table := range schema.Tables {
		state.tables[table.Name] = convertToTableState(table)
	}
	return state
}

func (s *schemaState) convertToSchemaMetadata() *v1pb.SchemaMetadata {
	tables := []*v1pb.TableMetadata{}
	for _, table := range s.tables {
		tables = append(tables, table.convertToTableMetadata())
	}
	sort.Slice(tables, func(i, j int) bool {
		return tables[i].Name < tables[j].Name
	})
	return &v1pb.SchemaMetadata{
		Name:   s.name,
		Tables: tables,
		// Unsupported, for tests only.
		Views:     []*v1pb.ViewMetadata{},
		Functions: []*v1pb.FunctionMetadata{},
		Streams:   []*v1pb.StreamMetadata{},
		Tasks:     []*v1pb.TaskMetadata{},
	}
}

type tableState struct {
	name    string
	columns map[string]*columnState
}

func (t *tableState) toString(buf *strings.Builder) error {
	if _, err := buf.WriteString(fmt.Sprintf("CREATE TABLE `%s` (\n", t.name)); err != nil {
		return err
	}
	columns := []*columnState{}
	for _, column := range t.columns {
		columns = append(columns, column)
	}
	sort.Slice(columns, func(i, j int) bool {
		return columns[i].name < columns[j].name
	})
	for i, column := range columns {
		if i > 0 {
			if _, err := buf.WriteString(",\n"); err != nil {
				return err
			}
		}
		if err := column.toString(buf); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString("\n);\n"); err != nil {
		return err
	}
	return nil
}

func newTableState(name string) *tableState {
	return &tableState{
		name:    name,
		columns: make(map[string]*columnState),
	}
}

func convertToTableState(table *v1pb.TableMetadata) *tableState {
	state := newTableState(table.Name)
	for _, column := range table.Columns {
		state.columns[column.Name] = convertToColumnState(column)
	}
	return state
}

func (t *tableState) convertToTableMetadata() *v1pb.TableMetadata {
	columns := []*v1pb.ColumnMetadata{}
	for _, column := range t.columns {
		columns = append(columns, column.convertToColumnMetadata())
	}
	sort.Slice(columns, func(i, j int) bool {
		return columns[i].Name < columns[j].Name
	})
	return &v1pb.TableMetadata{
		Name:    t.name,
		Columns: columns,
		// Unsupported, for tests only.
		Indexes:     []*v1pb.IndexMetadata{},
		ForeignKeys: []*v1pb.ForeignKeyMetadata{},
	}
}

type columnState struct {
	name string
	tp   string
}

func (c *columnState) toString(buf *strings.Builder) error {
	if _, err := buf.WriteString(fmt.Sprintf("  `%s` %s", c.name, c.tp)); err != nil {
		return err
	}
	return nil
}

func (c *columnState) convertToColumnMetadata() *v1pb.ColumnMetadata {
	return &v1pb.ColumnMetadata{
		Name: c.name,
		Type: c.tp,
	}
}

func convertToColumnState(column *v1pb.ColumnMetadata) *columnState {
	return &columnState{
		name: column.Name,
		tp:   column.Type,
	}
}

// EnterCreateTable is called when production createTable is entered.
func (t *mysqlTransformer) EnterCreateTable(ctx *mysql.CreateTableContext) {
	if t.err != nil {
		return
	}
	databaseName, tableName := parser.NormalizeMySQLTableName(ctx.TableName())
	if databaseName != "" {
		if t.state.name == "" {
			t.state.name = databaseName
		} else if t.state.name != databaseName {
			t.err = errors.New("multiple database names found: " + t.state.name + ", " + databaseName)
			return
		}
	}

	schema := t.state.schemas[""]
	if _, ok := schema.tables[tableName]; ok {
		t.err = errors.New("multiple table names found: " + tableName)
		return
	}

	schema.tables[tableName] = newTableState(tableName)
	t.currentTable = tableName
}

// ExitCreateTable is called when production createTable is exited.
func (t *mysqlTransformer) ExitCreateTable(_ *mysql.CreateTableContext) {
	t.currentTable = ""
}

// EnterColumnDefinition is called when production columnDefinition is entered.
func (t *mysqlTransformer) EnterColumnDefinition(ctx *mysql.ColumnDefinitionContext) {
	if t.err != nil || t.currentTable == "" {
		return
	}

	_, _, columnName := parser.NormalizeMySQLColumnName(ctx.ColumnName())
	dataType := ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.FieldDefinition().DataType())
	table := t.state.schemas[""].tables[t.currentTable]
	if _, ok := table.columns[columnName]; ok {
		t.err = errors.New("multiple column names found: " + columnName + " in table " + t.currentTable)
		return
	}

	table.columns[columnName] = &columnState{
		name: columnName,
		tp:   dataType,
	}
}

func getDesignSchema(engine v1pb.Engine, baselineSchema string, to *v1pb.DatabaseMetadata) (string, error) {
	switch engine {
	case v1pb.Engine_MYSQL:
		return getMySQLDesignSchema(baselineSchema, to)
	default:
		return "", status.Errorf(codes.InvalidArgument, fmt.Sprintf("unsupported engine: %v", engine))
	}
}

func getMySQLDesignSchema(baselineSchema string, to *v1pb.DatabaseMetadata) (string, error) {
	toState := convertToDatabaseState(to)
	list, err := parser.ParseMySQL(baselineSchema)
	if err != nil {
		return "", err
	}

	listener := &mysqlDesignSchemaGenerator{
		to: toState,
	}

	for _, stmt := range list {
		antlr.ParseTreeWalkerDefault.Walk(listener, stmt.Tree)
	}
	if listener.err != nil {
		return "", listener.err
	}

	for _, schema := range to.Schemas {
		schemaState, ok := toState.schemas[schema.Name]
		if !ok {
			continue
		}
		for _, table := range schema.Tables {
			table, ok := schemaState.tables[table.Name]
			if !ok {
				continue
			}
			if err := table.toString(&listener.result); err != nil {
				return "", err
			}
		}
	}

	return listener.result.String(), nil
}

type mysqlDesignSchemaGenerator struct {
	*mysql.BaseMySQLParserListener

	to     *databaseState
	result strings.Builder
	err    error
}

func (g *mysqlDesignSchemaGenerator) EnterCreateTable(ctx *mysql.CreateTableContext) {
	if g.err != nil {
		return
	}
	databaseName, tableName := parser.NormalizeMySQLTableName(ctx.TableName())
	if databaseName != "" && g.to.name != "" && databaseName != g.to.name {
		g.err = errors.New("multiple database names found: " + g.to.name + ", " + databaseName)
		return
	}

	schema, ok := g.to.schemas[""]
	if !ok || schema == nil {
		return
	}

	if _, ok := schema.tables[tableName]; !ok {
		return
	}

	// TODO: compare table
	delete(schema.tables, tableName)
	if _, err := g.result.WriteString("CREATE "); err != nil {
		g.err = err
		return
	}
	if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)); err != nil {
		g.err = err
		return
	}

	if _, err := g.result.WriteString(";\n"); err != nil {
		g.err = err
		return
	}
}
