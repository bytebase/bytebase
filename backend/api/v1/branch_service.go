package v1

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// NewBranchService implements BranchServiceServer interface.
type BranchService struct {
	v1pb.UnimplementedBranchServiceServer
}

// NewBranchService creates a new BranchService.
func NewBranchService() *BranchService {
	return &BranchService{}
}

func (*BranchService) DiffMetadata(ctx context.Context, request *v1pb.DiffMetadataRequest) (*v1pb.DiffMetadataResponse, error) {
	switch request.Engine {
	case v1pb.Engine_MYSQL, v1pb.Engine_POSTGRES, v1pb.Engine_TIDB, v1pb.Engine_ORACLE:
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unsupported engine: %v", request.Engine)
	}
	if request.SourceMetadata == nil || request.TargetMetadata == nil {
		return nil, status.Errorf(codes.InvalidArgument, "source_metadata and target_metadata are required")
	}
	storeSourceMetadata, err := convertV1DatabaseMetadata(request.SourceMetadata)
	if err != nil {
		return nil, err
	}
	sourceConfig := convertV1DatabaseConfig(
		ctx,
		&v1pb.DatabaseConfig{
			Name:          request.SourceMetadata.Name,
			SchemaConfigs: request.SourceMetadata.SchemaConfigs,
		},
		nil, /* optionalStores */
	)
	sanitizeCommentForSchemaMetadata(storeSourceMetadata, model.NewDatabaseConfig(sourceConfig), request.ClassificationFromConfig)

	storeTargetMetadata, err := convertV1DatabaseMetadata(request.TargetMetadata)
	if err != nil {
		return nil, err
	}
	targetConfig := convertV1DatabaseConfig(
		ctx,
		&v1pb.DatabaseConfig{
			Name:          request.TargetMetadata.Name,
			SchemaConfigs: request.TargetMetadata.SchemaConfigs,
		},
		nil, /* optionalStores */
	)
	if err := checkDatabaseMetadata(storepb.Engine(request.Engine), storeTargetMetadata); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid target metadata: %v", err)
	}
	sanitizeCommentForSchemaMetadata(storeTargetMetadata, model.NewDatabaseConfig(targetConfig), request.ClassificationFromConfig)

	storeSourceMetadata, storeTargetMetadata = trimDatabaseMetadata(storeSourceMetadata, storeTargetMetadata)
	if err := checkDatabaseMetadataColumnType(storepb.Engine(request.Engine), storeTargetMetadata); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid target metadata: %v", err)
	}

	sourceSchema, err := schema.GetDesignSchema(storepb.Engine(request.Engine), storeSourceMetadata)
	if err != nil {
		return nil, err
	}
	targetSchema, err := schema.GetDesignSchema(storepb.Engine(request.Engine), storeTargetMetadata)
	if err != nil {
		return nil, err
	}

	diff, err := base.SchemaDiff(convertEngine(request.Engine), base.DiffContext{
		IgnoreCaseSensitive: false,
		StrictMode:          true,
	}, sourceSchema, targetSchema)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to compute diff between source and target schemas, error: %v", err)
	}

	return &v1pb.DiffMetadataResponse{
		Diff: diff,
	}, nil
}
func trimDatabaseMetadata(sourceMetadata *storepb.DatabaseSchemaMetadata, targetMetadata *storepb.DatabaseSchemaMetadata) (*storepb.DatabaseSchemaMetadata, *storepb.DatabaseSchemaMetadata) {
	// TODO(d): handle indexes, etc.
	sourceModel, targetModel := model.NewDatabaseMetadata(sourceMetadata), model.NewDatabaseMetadata(targetMetadata)
	s, t := &storepb.DatabaseSchemaMetadata{}, &storepb.DatabaseSchemaMetadata{}
	for _, schema := range sourceMetadata.GetSchemas() {
		ts := targetModel.GetSchema(schema.GetName())
		if ts == nil {
			s.Schemas = append(s.Schemas, schema)
			continue
		}
		trimSchema := &storepb.SchemaMetadata{Name: schema.GetName()}
		for _, table := range schema.GetTables() {
			tt := ts.GetTable(table.GetName())
			if tt == nil {
				trimSchema.Tables = append(trimSchema.Tables, table)
				continue
			}

			if !common.EqualTable(table, tt.GetProto()) {
				trimSchema.Tables = append(trimSchema.Tables, table)
				continue
			}
		}
		for _, view := range schema.GetViews() {
			tv := ts.GetView(view.GetName())
			if tv == nil {
				trimSchema.Views = append(trimSchema.Views, view)
				continue
			}
			if view.GetComment() != tv.GetProto().GetComment() {
				trimSchema.Views = append(trimSchema.Views, view)
				continue
			}
			if view.GetDefinition() != tv.Definition {
				trimSchema.Views = append(trimSchema.Views, view)
				continue
			}
		}
		for _, function := range schema.GetFunctions() {
			tf := ts.GetFunction(function.GetName())
			if tf == nil {
				trimSchema.Functions = append(trimSchema.Functions, function)
				continue
			}
			if function.GetDefinition() != tf.Definition {
				trimSchema.Functions = append(trimSchema.Functions, function)
				continue
			}
		}
		for _, procedure := range schema.GetProcedures() {
			tp := ts.GetProcedure(procedure.GetName())
			if tp == nil {
				trimSchema.Procedures = append(trimSchema.Procedures, procedure)
				continue
			}
			if procedure.GetDefinition() != tp.Definition {
				trimSchema.Procedures = append(trimSchema.Procedures, procedure)
				continue
			}
		}
		// Always append empty schema to avoid creating schema duplicates.
		s.Schemas = append(s.Schemas, trimSchema)
	}

	for _, schema := range targetMetadata.GetSchemas() {
		ts := sourceModel.GetSchema(schema.GetName())
		if ts == nil {
			t.Schemas = append(t.Schemas, schema)
			continue
		}
		trimSchema := &storepb.SchemaMetadata{Name: schema.GetName()}
		for _, table := range schema.GetTables() {
			tt := ts.GetTable(table.GetName())
			if tt == nil {
				trimSchema.Tables = append(trimSchema.Tables, table)
				continue
			}

			if !common.EqualTable(table, tt.GetProto()) {
				trimSchema.Tables = append(trimSchema.Tables, table)
				continue
			}
		}
		for _, view := range schema.GetViews() {
			tv := ts.GetView(view.GetName())
			if tv == nil {
				trimSchema.Views = append(trimSchema.Views, view)
				continue
			}
			if view.GetDefinition() != tv.Definition {
				trimSchema.Views = append(trimSchema.Views, view)
				continue
			}
		}
		for _, function := range schema.GetFunctions() {
			tf := ts.GetFunction(function.GetName())
			if tf == nil {
				trimSchema.Functions = append(trimSchema.Functions, function)
				continue
			}
			if function.GetDefinition() != tf.Definition {
				trimSchema.Functions = append(trimSchema.Functions, function)
				continue
			}
		}
		for _, procedure := range schema.GetProcedures() {
			tp := ts.GetProcedure(procedure.GetName())
			if tp == nil {
				trimSchema.Procedures = append(trimSchema.Procedures, procedure)
				continue
			}
			if procedure.GetDefinition() != tp.Definition {
				trimSchema.Procedures = append(trimSchema.Procedures, procedure)
				continue
			}
		}
		// Always append empty schema to avoid creating schema duplicates.
		t.Schemas = append(t.Schemas, trimSchema)
	}

	return s, t
}
