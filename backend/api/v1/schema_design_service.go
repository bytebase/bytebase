package v1

import (
	"context"
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
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"
	"github.com/pkg/errors"

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
	if err := checkDatabaseMetadata(schemaDesign.Engine, schemaDesign.SchemaMetadata); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid schema design: %v", err))
	}
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
	instanceID, _, changeHistoryIDStr, err := getInstanceDatabaseIDChangeHistory(schemaDesign.SchemaVersion)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
		ResourceID: &instanceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	changeHistory, err := s.store.GetInstanceChangeHistory(ctx, &store.FindInstanceChangeHistoryMessage{
		ID:         &changeHistoryIDStr,
		InstanceID: &instance.UID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if changeHistory == nil {
		return nil, status.Errorf(codes.NotFound, "schema version %s not found", changeHistoryIDStr)
	}
	schemaDesignSheetPayload := &storepb.SheetPayload{
		Type: storepb.SheetPayload_SCHEMA_DESIGN,
		SchemaDesign: &storepb.SheetPayload_SchemaDesign{
			BaselineChangeHistoryId: changeHistory.UID,
			Engine:                  storepb.Engine(schemaDesign.Engine),
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
	// Try to transform the schema string to database metadata to make sure it's valid.
	if _, err := transformSchemaStringToDatabaseMetadata(schemaDesign.Engine, schema); err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to transform schema string to database metadata: %v", err))
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
	currentPrincipalID := ctx.Value(common.PrincipalIDContextKey).(int)
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
	if len(request.UpdateMask.Paths) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask is required")
	}

	sheetUpdate := &store.PatchSheetMessage{
		UID:       sheetUID,
		UpdaterID: currentPrincipalID,
	}
	schemaDesign := request.SchemaDesign
	if slices.Contains(request.UpdateMask.Paths, "title") {
		sheetUpdate.Name = &schemaDesign.Title
	}
	if slices.Contains(request.UpdateMask.Paths, "schema") {
		schema, err := getDesignSchema(schemaDesign.Engine, schemaDesign.BaselineSchema, schemaDesign.SchemaMetadata)
		if err != nil {
			return nil, err
		}
		// Try to transform the schema string to database metadata to make sure it's valid.
		if _, err := transformSchemaStringToDatabaseMetadata(schemaDesign.Engine, schema); err != nil {
			return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to transform schema string to database metadata: %v", err))
		}
		sheetUpdate.Statement = &schema
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

// ParseSchemaString parses a schema string to database metadata.
func (*SchemaDesignService) ParseSchemaString(_ context.Context, request *v1pb.ParseSchemaStringRequest) (*v1pb.ParseSchemaStringResponse, error) {
	if request.SchemaString == "" {
		return nil, status.Errorf(codes.InvalidArgument, "schema_string is required")
	}
	metadata, err := transformSchemaStringToDatabaseMetadata(request.Engine, request.SchemaString)
	if err != nil {
		return nil, err
	}
	return &v1pb.ParseSchemaStringResponse{
		SchemaMetadata: metadata,
	}, nil
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

	baselineSchema := ""
	schemaVersion := ""
	if sheetPayload.SchemaDesign.BaselineChangeHistoryId != "" {
		changeHistory, err := s.store.GetInstanceChangeHistory(ctx, &store.FindInstanceChangeHistoryMessage{
			ID: &sheetPayload.SchemaDesign.BaselineChangeHistoryId,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to find change history: %v", err))
		}
		if changeHistory != nil {
			baselineSchema = changeHistory.Schema
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
	name        string
	columns     map[string]*columnState
	indexes     map[string]*indexState
	foreignKeys map[string]*foreignKeyState
}

func (t *tableState) toString(buf *strings.Builder) error {
	if _, err := buf.WriteString(fmt.Sprintf("CREATE TABLE `%s` (\n  ", t.name)); err != nil {
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
			if _, err := buf.WriteString(",\n  "); err != nil {
				return err
			}
		}
		if err := column.toString(buf); err != nil {
			return err
		}
	}

	indexes := []*indexState{}
	for _, index := range t.indexes {
		indexes = append(indexes, index)
	}
	sort.Slice(indexes, func(i, j int) bool {
		return indexes[i].name < indexes[j].name
	})

	for i, index := range indexes {
		if i+len(columns) > 0 {
			if _, err := buf.WriteString(",\n  "); err != nil {
				return err
			}
		}
		if err := index.toString(buf); err != nil {
			return err
		}
	}

	foreignKeys := []*foreignKeyState{}
	for _, fk := range t.foreignKeys {
		foreignKeys = append(foreignKeys, fk)
	}
	sort.Slice(foreignKeys, func(i, j int) bool {
		return foreignKeys[i].name < foreignKeys[j].name
	})

	for i, fk := range foreignKeys {
		if i+len(columns)+len(indexes) > 0 {
			if _, err := buf.WriteString(",\n  "); err != nil {
				return err
			}
		}
		if err := fk.toString(buf); err != nil {
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
		name:        name,
		columns:     make(map[string]*columnState),
		indexes:     make(map[string]*indexState),
		foreignKeys: make(map[string]*foreignKeyState),
	}
}

func convertToTableState(table *v1pb.TableMetadata) *tableState {
	state := newTableState(table.Name)
	for _, column := range table.Columns {
		state.columns[column.Name] = convertToColumnState(column)
	}
	for _, index := range table.Indexes {
		state.indexes[index.Name] = convertToIndexState(index)
	}
	for _, fk := range table.ForeignKeys {
		state.foreignKeys[fk.Name] = convertToForeignKeyState(fk)
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

	indexes := []*v1pb.IndexMetadata{}
	for _, index := range t.indexes {
		indexes = append(indexes, index.convertToIndexMetadata())
	}
	sort.Slice(indexes, func(i, j int) bool {
		return indexes[i].Name < indexes[j].Name
	})

	fks := []*v1pb.ForeignKeyMetadata{}
	for _, fk := range t.foreignKeys {
		fks = append(fks, fk.convertToForeignKeyMetadata())
	}
	sort.Slice(fks, func(i, j int) bool {
		return fks[i].Name < fks[j].Name
	})

	return &v1pb.TableMetadata{
		Name:        t.name,
		Columns:     columns,
		Indexes:     indexes,
		ForeignKeys: fks,
	}
}

type foreignKeyState struct {
	name              string
	columns           []string
	referencedTable   string
	referencedColumns []string
}

func (f *foreignKeyState) convertToForeignKeyMetadata() *v1pb.ForeignKeyMetadata {
	return &v1pb.ForeignKeyMetadata{
		Name:              f.name,
		Columns:           f.columns,
		ReferencedTable:   f.referencedTable,
		ReferencedColumns: f.referencedColumns,
	}
}

func convertToForeignKeyState(foreignKey *v1pb.ForeignKeyMetadata) *foreignKeyState {
	return &foreignKeyState{
		name:              foreignKey.Name,
		columns:           foreignKey.Columns,
		referencedTable:   foreignKey.ReferencedTable,
		referencedColumns: foreignKey.ReferencedColumns,
	}
}

func (f *foreignKeyState) toString(buf *strings.Builder) error {
	if _, err := buf.WriteString("CONSTRAINT `"); err != nil {
		return err
	}
	if _, err := buf.WriteString(f.name); err != nil {
		return err
	}
	if _, err := buf.WriteString("` FOREIGN KEY ("); err != nil {
		return err
	}
	for i, column := range f.columns {
		if i > 0 {
			if _, err := buf.WriteString(", "); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString("`"); err != nil {
			return err
		}
		if _, err := buf.WriteString(column); err != nil {
			return err
		}
		if _, err := buf.WriteString("`"); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString(") REFERENCES `"); err != nil {
		return err
	}
	if _, err := buf.WriteString(f.referencedTable); err != nil {
		return err
	}
	if _, err := buf.WriteString("` ("); err != nil {
		return err
	}
	for i, column := range f.referencedColumns {
		if i > 0 {
			if _, err := buf.WriteString(", "); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString("`"); err != nil {
			return err
		}
		if _, err := buf.WriteString(column); err != nil {
			return err
		}
		if _, err := buf.WriteString("`"); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString(")"); err != nil {
		return err
	}
	return nil
}

type indexState struct {
	name    string
	keys    []string
	primary bool
	unique  bool
}

func (i *indexState) convertToIndexMetadata() *v1pb.IndexMetadata {
	return &v1pb.IndexMetadata{
		Name:        i.name,
		Expressions: i.keys,
		Primary:     i.primary,
		Unique:      i.unique,
		// Unsupported, for tests only.
		Visible: true,
	}
}

func convertToIndexState(index *v1pb.IndexMetadata) *indexState {
	return &indexState{
		name:    index.Name,
		keys:    index.Expressions,
		primary: index.Primary,
		unique:  index.Unique,
	}
}

func (i *indexState) toString(buf *strings.Builder) error {
	if i.primary {
		if _, err := buf.WriteString("PRIMARY KEY ("); err != nil {
			return err
		}
		for i, key := range i.keys {
			if i > 0 {
				if _, err := buf.WriteString(", "); err != nil {
					return err
				}
			}
			if _, err := buf.WriteString(fmt.Sprintf("`%s`", key)); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString(")"); err != nil {
			return err
		}
	}
	// TODO: support other type indexes.
	return nil
}

type columnState struct {
	name         string
	tp           string
	defaultValue *string
	comment      string
	nullable     bool
}

func (c *columnState) toString(buf *strings.Builder) error {
	if _, err := buf.WriteString(fmt.Sprintf("`%s` %s", c.name, c.tp)); err != nil {
		return err
	}
	if c.nullable {
		if _, err := buf.WriteString(" NULL"); err != nil {
			return err
		}
	} else {
		if _, err := buf.WriteString(" NOT NULL"); err != nil {
			return err
		}
	}
	if c.defaultValue != nil {
		if _, err := buf.WriteString(fmt.Sprintf(" DEFAULT %s", *c.defaultValue)); err != nil {
			return err
		}
	}
	if c.comment != "" {
		if _, err := buf.WriteString(fmt.Sprintf(" COMMENT '%s'", c.comment)); err != nil {
			return err
		}
	}
	return nil
}

func (c *columnState) convertToColumnMetadata() *v1pb.ColumnMetadata {
	result := &v1pb.ColumnMetadata{
		Name:     c.name,
		Type:     c.tp,
		Nullable: c.nullable,
		Comment:  c.comment,
	}
	if c.defaultValue != nil {
		result.Default = &wrapperspb.StringValue{Value: *c.defaultValue}
	}
	return result
}

func convertToColumnState(column *v1pb.ColumnMetadata) *columnState {
	result := &columnState{
		name:     column.Name,
		tp:       column.Type,
		nullable: column.Nullable,
		comment:  column.Comment,
	}
	if column.Default != nil {
		result.defaultValue = &column.Default.Value
	}
	return result
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

// EnterTableConstraintDef is called when production tableConstraintDef is entered.
func (t *mysqlTransformer) EnterTableConstraintDef(ctx *mysql.TableConstraintDefContext) {
	if t.err != nil || t.currentTable == "" {
		return
	}

	if ctx.GetType_() != nil {
		switch strings.ToUpper(ctx.GetType_().GetText()) {
		case "PRIMARY":
			list := extractKeyListVariants(ctx.KeyListVariants())
			t.state.schemas[""].tables[t.currentTable].indexes["PRIMARY"] = &indexState{
				name:    "PRIMARY",
				keys:    list,
				primary: true,
				unique:  true,
			}
		case "FOREIGN":
			var name string
			if ctx.ConstraintName() != nil && ctx.ConstraintName().Identifier() != nil {
				name = parser.NormalizeMySQLIdentifier(ctx.ConstraintName().Identifier())
			} else if ctx.IndexName() != nil {
				name = parser.NormalizeMySQLIdentifier(ctx.IndexName().Identifier())
			}
			keys := extractKeyList(ctx.KeyList())
			table := t.state.schemas[""].tables[t.currentTable]
			if table.foreignKeys[name] != nil {
				t.err = errors.New("multiple foreign keys found: " + name)
				return
			}
			referencedTable, referencedColumns := extractReference(ctx.References())
			fk := &foreignKeyState{
				name:              name,
				columns:           keys,
				referencedTable:   referencedTable,
				referencedColumns: referencedColumns,
			}
			table.foreignKeys[name] = fk
		}
	}
}

func extractReference(ctx mysql.IReferencesContext) (string, []string) {
	_, table := parser.NormalizeMySQLTableRef(ctx.TableRef())
	if ctx.IdentifierListWithParentheses() != nil {
		columns := extractIdentifierList(ctx.IdentifierListWithParentheses().IdentifierList())
		return table, columns
	}
	return table, nil
}

func extractIdentifierList(ctx mysql.IIdentifierListContext) []string {
	var result []string
	for _, identifier := range ctx.AllIdentifier() {
		result = append(result, parser.NormalizeMySQLIdentifier(identifier))
	}
	return result
}

func extractKeyListVariants(ctx mysql.IKeyListVariantsContext) []string {
	if ctx.KeyList() != nil {
		return extractKeyList(ctx.KeyList())
	}
	if ctx.KeyListWithExpression() != nil {
		return extractKeyListWithExpression(ctx.KeyListWithExpression())
	}
	return nil
}

func extractKeyListWithExpression(ctx mysql.IKeyListWithExpressionContext) []string {
	var result []string
	for _, key := range ctx.AllKeyPartOrExpression() {
		if key.KeyPart() != nil {
			keyText := parser.NormalizeMySQLIdentifier(key.KeyPart().Identifier())
			result = append(result, keyText)
		} else if key.ExprWithParentheses() != nil {
			keyText := key.GetParser().GetTokenStream().GetTextFromRuleContext(key.ExprWithParentheses())
			result = append(result, keyText)
		}
	}
	return result
}

func extractKeyList(ctx mysql.IKeyListContext) []string {
	var result []string
	for _, key := range ctx.AllKeyPart() {
		keyText := parser.NormalizeMySQLIdentifier(key.Identifier())
		result = append(result, keyText)
	}
	return result
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
	columnState := &columnState{
		name:         columnName,
		tp:           dataType,
		defaultValue: nil,
		comment:      "",
		nullable:     true,
	}

	for _, attribute := range ctx.FieldDefinition().AllColumnAttribute() {
		switch {
		case attribute.NullLiteral() != nil && attribute.NOT_SYMBOL() != nil:
			columnState.nullable = false
		case attribute.DEFAULT_SYMBOL() != nil && attribute.SERIAL_SYMBOL() == nil:
			defaultValueStart := nextDefaultChannelTokenIndex(ctx.GetParser().GetTokenStream(), attribute.DEFAULT_SYMBOL().GetSymbol().GetTokenIndex())
			defaultValue := attribute.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
				Start: defaultValueStart,
				Stop:  attribute.GetStop().GetTokenIndex(),
			})
			columnState.defaultValue = &defaultValue
		case attribute.COMMENT_SYMBOL() != nil:
			commentStart := nextDefaultChannelTokenIndex(ctx.GetParser().GetTokenStream(), attribute.COMMENT_SYMBOL().GetSymbol().GetTokenIndex())
			comment := attribute.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
				Start: commentStart,
				Stop:  attribute.GetStop().GetTokenIndex(),
			})
			if comment != `''` && len(comment) > 2 {
				columnState.comment = comment[1 : len(comment)-1]
			}
		}
	}

	table.columns[columnName] = columnState
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

	to                  *databaseState
	result              strings.Builder
	currentTable        *tableState
	firstElementInTable bool
	columnDefine        strings.Builder
	tableConstraints    strings.Builder
	err                 error
}

// EnterCreateTable is called when production createTable is entered.
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

	table, ok := schema.tables[tableName]
	if !ok {
		return
	}

	g.currentTable = table
	g.firstElementInTable = true
	g.columnDefine.Reset()
	g.tableConstraints.Reset()

	delete(schema.tables, tableName)
	if _, err := g.result.WriteString("CREATE "); err != nil {
		g.err = err
		return
	}
	if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
		Start: ctx.GetStart().GetTokenIndex(),
		Stop:  ctx.TableElementList().GetStart().GetTokenIndex() - 1,
	})); err != nil {
		g.err = err
		return
	}
}

// ExitCreateTable is called when production createTable is exited.
func (g *mysqlDesignSchemaGenerator) ExitCreateTable(ctx *mysql.CreateTableContext) {
	if g.err != nil || g.currentTable == nil {
		return
	}

	var columnList []*columnState
	for _, column := range g.currentTable.columns {
		columnList = append(columnList, column)
	}
	sort.Slice(columnList, func(i, j int) bool {
		return columnList[i].name < columnList[j].name
	})
	for _, column := range columnList {
		if g.firstElementInTable {
			g.firstElementInTable = false
		} else {
			if _, err := g.columnDefine.WriteString(",\n  "); err != nil {
				g.err = err
				return
			}
		}
		if err := column.toString(&g.columnDefine); err != nil {
			g.err = err
			return
		}
	}

	if g.currentTable.indexes["PRIMARY"] != nil {
		if g.firstElementInTable {
			g.firstElementInTable = false
		} else {
			if _, err := g.columnDefine.WriteString(",\n  "); err != nil {
				g.err = err
				return
			}
		}
		if err := g.currentTable.indexes["PRIMARY"].toString(&g.tableConstraints); err != nil {
			return
		}
	}

	var fks []*foreignKeyState
	for _, fk := range g.currentTable.foreignKeys {
		fks = append(fks, fk)
	}
	sort.Slice(fks, func(i, j int) bool {
		return fks[i].name < fks[j].name
	})
	for _, fk := range fks {
		if g.firstElementInTable {
			g.firstElementInTable = false
		} else {
			if _, err := g.columnDefine.WriteString(",\n  "); err != nil {
				g.err = err
				return
			}
		}
		if err := fk.toString(&g.tableConstraints); err != nil {
			g.err = err
			return
		}
	}

	if _, err := g.result.WriteString(g.columnDefine.String()); err != nil {
		g.err = err
		return
	}

	if _, err := g.result.WriteString(g.tableConstraints.String()); err != nil {
		g.err = err
		return
	}

	if _, err := g.result.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
		Start: ctx.TableElementList().GetStop().GetTokenIndex() + 1,
		Stop:  ctx.GetStop().GetTokenIndex(),
	})); err != nil {
		g.err = err
		return
	}

	if _, err := g.result.WriteString(";\n"); err != nil {
		g.err = err
		return
	}

	g.currentTable = nil
	g.firstElementInTable = false
}

type columnAttr struct {
	text  string
	order int
}

var columnAttrOrder = map[string]int{
	"NULL":           1,
	"DEFAULT":        2,
	"VISIBLE":        3,
	"AUTO_INCREMENT": 4,
	"UNIQUE":         5,
	"KEY":            6,
	"COMMENT":        7,
	"COLLATE":        8,
	"COLUMN_FORMAT":  9,
	"SECONDARY":      10,
	"STORAGE":        11,
	"SERIAL":         12,
	"SRID":           13,
	"ON":             14,
	"CHECK":          15,
	"ENFORCED":       16,
}

func extractNewAttrs(column *columnState, attrs []mysql.IColumnAttributeContext) []columnAttr {
	var result []columnAttr
	nullExists := false
	defaultExists := false
	commentExists := false
	for _, attr := range attrs {
		if attr.GetValue() != nil {
			switch strings.ToUpper(attr.GetValue().GetText()) {
			case "DEFAULT":
				defaultExists = true
			case "COMMENT":
				defaultExists = true
			}
		} else if attr.NullLiteral() != nil {
			nullExists = true
		}
	}

	if !nullExists && !column.nullable {
		result = append(result, columnAttr{
			text:  "NOT NULL",
			order: columnAttrOrder["NULL"],
		})
	}
	if !defaultExists && column.defaultValue != nil {
		result = append(result, columnAttr{
			text:  "DEFAULT " + *column.defaultValue,
			order: columnAttrOrder["DEFAULT"],
		})
	}
	if !commentExists && column.comment != "" {
		result = append(result, columnAttr{
			text:  "COMMENT '" + column.comment + "'",
			order: columnAttrOrder["COMMENT"],
		})
	}
	return result
}

func getAttrOrder(attr mysql.IColumnAttributeContext) int {
	if attr.GetValue() != nil {
		switch strings.ToUpper(attr.GetValue().GetText()) {
		case "DEFAULT", "ON", "AUTO_INCREMENT", "SERIAL", "KEY", "UNIQUE", "COMMENT", "COLUMN_FORMAT", "STORAGE", "SRID":
			return columnAttrOrder[attr.GetValue().GetText()]
		}
	}
	if attr.NullLiteral() != nil {
		return columnAttrOrder["NULL"]
	}
	if attr.SECONDARY_SYMBOL() != nil {
		return columnAttrOrder["SECONDARY"]
	}
	if attr.Collate() != nil {
		return columnAttrOrder["COLLATE"]
	}
	if attr.CheckConstraint() != nil {
		return columnAttrOrder["CHECK"]
	}
	if attr.ConstraintEnforcement() != nil {
		return columnAttrOrder["ENFORCED"]
	}
	return len(columnAttrOrder) + 1
}

// EnterTableConstraintDef is called when production tableConstraintDef is entered.
func (g *mysqlDesignSchemaGenerator) EnterTableConstraintDef(ctx *mysql.TableConstraintDefContext) {
	if g.err != nil || g.currentTable == nil {
		return
	}

	if ctx.GetType_() == nil {
		if _, err := g.tableConstraints.WriteString(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)); err != nil {
			g.err = err
			return
		}
		return
	}

	switch strings.ToUpper(ctx.GetType_().GetText()) {
	case "PRIMARY":
		if g.currentTable.indexes["PRIMARY"] != nil {
			if g.firstElementInTable {
				g.firstElementInTable = false
			} else {
				if _, err := g.tableConstraints.WriteString(",\n  "); err != nil {
					g.err = err
					return
				}
			}

			keys := extractKeyListVariants(ctx.KeyListVariants())
			if equalKeys(keys, g.currentTable.indexes["PRIMARY"].keys) {
				if _, err := g.tableConstraints.WriteString(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)); err != nil {
					g.err = err
					return
				}
			} else {
				if err := g.currentTable.indexes["PRIMARY"].toString(&g.tableConstraints); err != nil {
					g.err = err
					return
				}
			}
			delete(g.currentTable.indexes, "PRIMARY")
		}
	case "FOREIGN":
		var name string
		if ctx.ConstraintName() != nil && ctx.ConstraintName().Identifier() != nil {
			name = parser.NormalizeMySQLIdentifier(ctx.ConstraintName().Identifier())
		} else if ctx.IndexName() != nil {
			name = parser.NormalizeMySQLIdentifier(ctx.IndexName().Identifier())
		}
		if g.currentTable.foreignKeys[name] != nil {
			if g.firstElementInTable {
				g.firstElementInTable = false
			} else {
				if _, err := g.tableConstraints.WriteString(",\n  "); err != nil {
					g.err = err
					return
				}
			}

			fk := g.currentTable.foreignKeys[name]

			columns := extractKeyList(ctx.KeyList())
			referencedTable, referencedColumns := extractReference(ctx.References())
			equal := equalKeys(columns, fk.columns) && referencedTable == fk.referencedTable && equalKeys(referencedColumns, fk.referencedColumns)
			if equal {
				if _, err := g.tableConstraints.WriteString(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)); err != nil {
					g.err = err
					return
				}
			} else {
				if err := fk.toString(&g.tableConstraints); err != nil {
					g.err = err
					return
				}
			}
			delete(g.currentTable.foreignKeys, name)
		}
	default:
		if g.firstElementInTable {
			g.firstElementInTable = false
		} else {
			if _, err := g.tableConstraints.WriteString(",\n  "); err != nil {
				g.err = err
				return
			}
		}
		if _, err := g.tableConstraints.WriteString(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)); err != nil {
			g.err = err
			return
		}
	}
}

func equalKeys(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, key := range a {
		if key != b[i] {
			return false
		}
	}
	return true
}

// EnterColumnDefinition is called when production columnDefinition is entered.
func (g *mysqlDesignSchemaGenerator) EnterColumnDefinition(ctx *mysql.ColumnDefinitionContext) {
	if g.err != nil || g.currentTable == nil {
		return
	}

	_, _, columnName := parser.NormalizeMySQLColumnName(ctx.ColumnName())
	column, ok := g.currentTable.columns[columnName]
	if !ok {
		return
	}

	delete(g.currentTable.columns, columnName)

	if g.firstElementInTable {
		g.firstElementInTable = false
	} else {
		if _, err := g.columnDefine.WriteString(",\n  "); err != nil {
			g.err = err
			return
		}
	}

	// compare column type
	typeCtx := ctx.FieldDefinition().DataType()
	columnType := ctx.GetParser().GetTokenStream().GetTextFromRuleContext(typeCtx)
	if columnType != column.tp {
		if _, err := g.columnDefine.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
			Start: ctx.GetStart().GetTokenIndex(),
			Stop:  typeCtx.GetStart().GetTokenIndex() - 1,
		})); err != nil {
			g.err = err
			return
		}
		if _, err := g.columnDefine.WriteString(column.tp); err != nil {
			g.err = err
			return
		}
	} else {
		if _, err := g.columnDefine.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
			Start: ctx.GetStart().GetTokenIndex(),
			Stop:  typeCtx.GetStop().GetTokenIndex(),
		})); err != nil {
			g.err = err
			return
		}
	}
	startPos := typeCtx.GetStop().GetTokenIndex() + 1

	newAttr := extractNewAttrs(column, ctx.FieldDefinition().AllColumnAttribute())

	for _, attribute := range ctx.FieldDefinition().AllColumnAttribute() {
		attrOrder := getAttrOrder(attribute)
		for ; len(newAttr) > 0 && newAttr[0].order < attrOrder; newAttr = newAttr[1:] {
			if _, err := g.columnDefine.WriteString(" " + newAttr[0].text); err != nil {
				g.err = err
				return
			}
		}
		switch {
		// nullable
		case attribute.NullLiteral() != nil:
			sameNullable := attribute.NOT_SYMBOL() == nil && column.nullable
			sameNullable = sameNullable || (attribute.NOT_SYMBOL() != nil && !column.nullable)
			if sameNullable {
				if _, err := g.columnDefine.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
					Start: startPos,
					Stop:  attribute.GetStop().GetTokenIndex(),
				})); err != nil {
					g.err = err
					return
				}
			} else {
				if _, err := g.columnDefine.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
					Start: startPos,
					Stop:  attribute.GetStart().GetTokenIndex() - 1,
				})); err != nil {
					g.err = err
					return
				}
				if column.nullable {
					if _, err := g.columnDefine.WriteString(" NULL"); err != nil {
						g.err = err
						return
					}
				} else {
					if _, err := g.columnDefine.WriteString(" NOT NULL"); err != nil {
						g.err = err
						return
					}
				}
			}
		case attribute.DEFAULT_SYMBOL() != nil && attribute.SERIAL_SYMBOL() == nil:
			defaultValueStart := nextDefaultChannelTokenIndex(attribute.GetParser().GetTokenStream(), attribute.DEFAULT_SYMBOL().GetSymbol().GetTokenIndex())
			defaultValue := attribute.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
				Start: defaultValueStart,
				Stop:  attribute.GetStop().GetTokenIndex(),
			})
			if column.defaultValue != nil && *column.defaultValue == defaultValue {
				if _, err := g.columnDefine.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
					Start: startPos,
					Stop:  attribute.GetStop().GetTokenIndex(),
				})); err != nil {
					g.err = err
					return
				}
			} else if column.defaultValue != nil {
				if _, err := g.columnDefine.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
					Start: startPos,
					Stop:  defaultValueStart - 1,
				})); err != nil {
					g.err = err
					return
				}
				if _, err := g.columnDefine.WriteString(*column.defaultValue); err != nil {
					g.err = err
					return
				}
			}
		case attribute.COMMENT_SYMBOL() != nil:
			commentStart := nextDefaultChannelTokenIndex(attribute.GetParser().GetTokenStream(), attribute.COMMENT_SYMBOL().GetSymbol().GetTokenIndex())
			commentValue := attribute.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
				Start: commentStart,
				Stop:  attribute.GetStop().GetTokenIndex(),
			})
			if commentValue != `''` && len(commentValue) > 2 && column.comment == commentValue[1:len(commentValue)-1] {
				if _, err := g.columnDefine.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
					Start: startPos,
					Stop:  attribute.GetStop().GetTokenIndex(),
				})); err != nil {
					g.err = err
					return
				}
			} else if column.comment != "" {
				if _, err := g.columnDefine.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
					Start: startPos,
					Stop:  commentStart - 1,
				})); err != nil {
					g.err = err
					return
				}
				if _, err := g.columnDefine.WriteString(fmt.Sprintf("'%s'", column.comment)); err != nil {
					g.err = err
					return
				}
			}
		default:
			if _, err := g.columnDefine.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
				Start: startPos,
				Stop:  attribute.GetStop().GetTokenIndex(),
			})); err != nil {
				g.err = err
				return
			}
		}
		startPos = attribute.GetStop().GetTokenIndex() + 1
	}

	for _, attr := range newAttr {
		if _, err := g.columnDefine.WriteString(" " + attr.text); err != nil {
			g.err = err
			return
		}
	}

	if _, err := g.columnDefine.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
		Start: startPos,
		Stop:  ctx.GetStop().GetTokenIndex(),
	})); err != nil {
		g.err = err
		return
	}
}

func nextDefaultChannelTokenIndex(tokens antlr.TokenStream, currentIndex int) int {
	for i := currentIndex + 1; i < tokens.Size(); i++ {
		if tokens.Get(i).GetChannel() == antlr.TokenDefaultChannel {
			return i
		}
	}
	return 0
}

func checkDatabaseMetadata(engine v1pb.Engine, metadata *v1pb.DatabaseMetadata) error {
	type fkMetadata struct {
		name                string
		tableName           string
		referencedTableName string
		referencedColumns   []string
	}
	fkMap := make(map[string][]*fkMetadata)
	if engine != v1pb.Engine_MYSQL {
		return errors.Errorf("only mysql is supported")
	}
	for _, schema := range metadata.Schemas {
		if schema.Name != "" {
			return errors.Errorf("schema name should be empty for MySQL")
		}
		tableNameMap := make(map[string]bool)
		for _, table := range schema.Tables {
			if table.Name == "" {
				return errors.Errorf("table name should not be empty")
			}
			if _, ok := tableNameMap[table.Name]; ok {
				return errors.Errorf("duplicate table name %s", table.Name)
			}
			tableNameMap[table.Name] = true
			columnNameMap := make(map[string]bool)
			for _, column := range table.Columns {
				if column.Name == "" {
					return errors.Errorf("column name should not be empty in table %s", table.Name)
				}
				if _, ok := columnNameMap[column.Name]; ok {
					return errors.Errorf("duplicate column name %s in table %s", column.Name, table.Name)
				}

				columnNameMap[column.Name] = true
				if column.Type == "" {
					return errors.Errorf("column %s type should not be empty in table %s", column.Name, table.Name)
				}

				if !checkMySQLColumnType(column.Type) {
					return errors.Errorf("column %s type %s is invalid in table %s", column.Name, column.Type, table.Name)
				}
			}

			indexNameMap := make(map[string]bool)
			for _, index := range table.Indexes {
				if index.Name == "" {
					return errors.Errorf("index name should not be empty in table %s", table.Name)
				}
				if _, ok := indexNameMap[index.Name]; ok {
					return errors.Errorf("duplicate index name %s in table %s", index.Name, table.Name)
				}
				indexNameMap[index.Name] = true
				if index.Primary {
					for _, key := range index.Expressions {
						if _, ok := columnNameMap[key]; !ok {
							return errors.Errorf("primary key column %s not found in table %s", key, table.Name)
						}
					}
				}
			}

			for _, fk := range table.ForeignKeys {
				if fk.Name == "" {
					return errors.Errorf("foreign key name should not be empty in table %s", table.Name)
				}
				if _, ok := indexNameMap[fk.Name]; ok {
					return errors.Errorf("duplicate foreign key name %s in table %s", fk.Name, table.Name)
				}
				indexNameMap[fk.Name] = true
				for _, key := range fk.Columns {
					if _, ok := columnNameMap[key]; !ok {
						return errors.Errorf("foreign key column %s not found in table %s", key, table.Name)
					}
				}
				fks, ok := fkMap[fk.ReferencedTable]
				if !ok {
					fks = []*fkMetadata{}
					fkMap[fk.ReferencedTable] = fks
				}
				meta := &fkMetadata{
					name:                fk.Name,
					tableName:           table.Name,
					referencedTableName: fk.ReferencedTable,
					referencedColumns:   fk.ReferencedColumns,
				}
				fks = append(fks, meta)
				fkMap[fk.ReferencedTable] = fks
			}
		}

		// check foreign key reference
		for _, table := range schema.Tables {
			fks, ok := fkMap[table.Name]
			if !ok {
				continue
			}
			columnNameMap := make(map[string]bool)
			for _, column := range table.Columns {
				columnNameMap[column.Name] = true
			}
			for _, fk := range fks {
				for _, key := range fk.referencedColumns {
					if _, ok := columnNameMap[key]; !ok {
						return errors.Errorf("foreign key %s in table %s references column %s in table %s but not found", fk.name, fk.tableName, key, fk.referencedTableName)
					}
				}
				hasIndex := false
				for _, index := range table.Indexes {
					if len(index.Expressions) < len(fk.referencedColumns) {
						continue
					}
					for i, key := range fk.referencedColumns {
						if index.Expressions[i] != key {
							break
						}
						if i == len(fk.referencedColumns)-1 {
							hasIndex = true
						}
					}
				}
				if !hasIndex {
					return errors.Errorf("missing index for foreign key %s for table %s in the referenced table '%s'", fk.name, fk.tableName, fk.referencedTableName)
				}
			}
			delete(fkMap, table.Name)
		}

		for _, fks := range fkMap {
			if len(fks) == 0 {
				continue
			}
			return errors.Errorf("foreign key %s in table %s references table %s but not found", fks[0].name, fks[0].tableName, fks[0].referencedTableName)
		}
	}
	return nil
}

func checkMySQLColumnType(tp string) bool {
	_, err := parser.ParseMySQL(fmt.Sprintf("CREATE TABLE t (a %s NOT NULL)", tp))
	return err == nil
}
