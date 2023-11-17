package v1

import (
	"context"
	"fmt"
	"path"
	"strconv"

	"golang.org/x/exp/slices"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// SchemaDesignService implements SchemaDesignServiceServer interface.
type SchemaDesignService struct {
	v1pb.UnimplementedSchemaDesignServiceServer
	store          *store.Store
	licenseService enterprise.LicenseService
}

// NewSchemaDesignService creates a new SchemaDesignService.
func NewSchemaDesignService(store *store.Store, licenseService enterprise.LicenseService) *SchemaDesignService {
	return &SchemaDesignService{
		store:          store,
		licenseService: licenseService,
	}
}

// GetSchemaDesign gets the schema design.
func (s *SchemaDesignService) GetSchemaDesign(ctx context.Context, request *v1pb.GetSchemaDesignRequest) (*v1pb.SchemaDesign, error) {
	_, sheetID, err := common.GetProjectResourceIDAndSchemaDesignSheetID(request.Name)
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

		LoadFull: true,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if err := s.checkSchemaDesignPermission(ctx, sheet.ProjectUID); err != nil {
		return nil, err
	}
	schemaDesign, err := s.convertSheetToSchemaDesign(ctx, sheet, v1pb.SchemaDesignView_SCHEMA_DESIGN_VIEW_FULL)
	if err != nil {
		return nil, err
	}
	return schemaDesign, nil
}

func (s *SchemaDesignService) checkSchemaDesignPermission(ctx context.Context, projectUID int) error {
	role, ok := ctx.Value(common.RoleContextKey).(api.Role)
	if !ok {
		return status.Errorf(codes.Internal, "role not found")
	}
	if isOwnerOrDBA(role) {
		return nil
	}

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return status.Errorf(codes.Internal, "principal ID not found")
	}
	policy, err := s.store.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{UID: &projectUID})
	if err != nil {
		return status.Errorf(codes.Internal, err.Error())
	}
	for _, binding := range policy.Bindings {
		if binding.Role == api.Developer || binding.Role == api.Owner {
			for _, member := range binding.Members {
				if member.ID == principalID || member.Email == api.AllUsers {
					return nil
				}
			}
		}
	}
	return status.Errorf(codes.PermissionDenied, "permission denied")
}

// ListSchemaDesigns lists schema designs.
func (s *SchemaDesignService) ListSchemaDesigns(ctx context.Context, request *v1pb.ListSchemaDesignsRequest) (*v1pb.ListSchemaDesignsResponse, error) {
	projectID, err := common.GetProjectID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	schemaDesignSheetType := storepb.SheetPayload_SCHEMA_DESIGN.String()
	sheetFind := &store.FindSheetMessage{
		PayloadType: &schemaDesignSheetType,
	}
	if request.View == v1pb.SchemaDesignView_SCHEMA_DESIGN_VIEW_FULL {
		sheetFind.LoadFull = true
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
		if err := s.checkSchemaDesignPermission(ctx, sheet.ProjectUID); err != nil {
			st := status.Convert(err)
			if st.Code() == codes.PermissionDenied {
				continue
			}
			return nil, err
		}
		schemaDesign, err := s.convertSheetToSchemaDesign(ctx, sheet, request.View)
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
	projectID, err := common.GetProjectID(request.Parent)
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
	if err := s.checkSchemaDesignPermission(ctx, project.UID); err != nil {
		return nil, err
	}

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	schemaDesign := request.SchemaDesign
	if schemaDesign == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty branch design")
	}

	schemaDesignSheetType := storepb.SheetPayload_SCHEMA_DESIGN.String()
	sheet, err := s.store.GetSheet(ctx, &store.FindSheetMessage{ProjectUID: &project.UID, Title: &schemaDesign.Title, PayloadType: &schemaDesignSheetType}, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get existing branch design, error %v", err)
	}
	if sheet != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "branch %q has already existed", schemaDesign.Title)
	}

	// Branch protection check.
	if err := s.checkProtectionRules(ctx, project, schemaDesign, principalID); err != nil {
		return nil, err
	}

	sanitizeSchemaDesignSchemaMetadata(schemaDesign)
	if err := checkDatabaseMetadata(schemaDesign.Engine, schemaDesign.SchemaMetadata); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid schema design: %v", err))
	}
	instanceID, databaseName, err := common.GetInstanceDatabaseID(schemaDesign.BaselineDatabase)
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
		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		find.IgnoreCaseSensitive = store.IgnoreDatabaseAndTableCaseSensitive(instance)
	}
	database, err := s.store.GetDatabaseV2(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if database == nil {
		return nil, status.Errorf(codes.NotFound, "database %q not found", databaseName)
	}

	schema, err := getDesignSchema(schemaDesign.Engine, schemaDesign.BaselineSchema, schemaDesign.SchemaMetadata)
	if err != nil {
		return nil, err
	}
	// Try to transform the schema string to database metadata to make sure it's valid.
	if _, err := transformSchemaStringToDatabaseMetadata(schemaDesign.Engine, schema); err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to transform schema string to database metadata: %v", err))
	}

	_, baselineSheetUID, err := common.GetProjectResourceIDSheetUID(schemaDesign.BaselineSheetName)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	schemaDesignType := storepb.SheetPayload_SchemaDesign_Type(schemaDesign.Type)
	schemaDesignSheetPayload := &storepb.SheetPayload{
		Type: storepb.SheetPayload_SCHEMA_DESIGN,
		SchemaDesign: &storepb.SheetPayload_SchemaDesign{
			Type:       schemaDesignType,
			Engine:     storepb.Engine(schemaDesign.Engine),
			Protection: convertProtectionToStore(schemaDesign.Protection),
		},
		DatabaseConfig: convertV1DatabaseConfig(&v1pb.DatabaseConfig{
			Name:          databaseName,
			SchemaConfigs: schemaDesign.SchemaMetadata.SchemaConfigs,
		}),
	}
	if baselineMetadata := schemaDesign.BaselineSchemaMetadata; baselineMetadata != nil {
		schemaDesignSheetPayload.BaselineDatabaseConfig = convertV1DatabaseConfig(&v1pb.DatabaseConfig{
			Name:          databaseName,
			SchemaConfigs: baselineMetadata.SchemaConfigs,
		})
	}

	if schemaDesignType == storepb.SheetPayload_SchemaDesign_MAIN_BRANCH {
		schemaDesignSheetPayload.SchemaDesign.BaselineSheetId = fmt.Sprintf("%d", baselineSheetUID)
	} else if schemaDesignType == storepb.SheetPayload_SchemaDesign_PERSONAL_DRAFT {
		baselineSheetCreate := &store.SheetMessage{
			Title:       schemaDesign.Title,
			ProjectUID:  project.UID,
			DatabaseUID: &database.UID,
			Statement:   schemaDesign.BaselineSchema,
			Visibility:  store.ProjectSheet,
			Source:      store.SheetFromBytebaseArtifact,
			Type:        store.SheetForSQL,
			CreatorID:   principalID,
			UpdaterID:   principalID,
		}
		sheet, err := s.store.CreateSheet(ctx, baselineSheetCreate)
		if err != nil {
			return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to create sheet: %v", err))
		}
		schemaDesignSheetPayload.SchemaDesign.BaselineSheetId = strconv.Itoa(sheet.UID)
		// baselineSheetID is a reference to the baseline schema design.
		schemaDesignSheetPayload.SchemaDesign.BaselineSchemaDesignId = fmt.Sprintf("%d", baselineSheetUID)
	}
	if schemaDesign.BaselineChangeHistoryId != nil {
		schemaDesignSheetPayload.SchemaDesign.BaselineChangeHistoryId = *schemaDesign.BaselineChangeHistoryId
	}

	sheetCreate := &store.SheetMessage{
		Title:       schemaDesign.Title,
		ProjectUID:  project.UID,
		DatabaseUID: &database.UID,
		Statement:   schema,
		Visibility:  store.ProjectSheet,
		Source:      store.SheetFromBytebaseArtifact,
		Type:        store.SheetForSQL,
		CreatorID:   principalID,
		UpdaterID:   principalID,
		Payload:     schemaDesignSheetPayload,
	}
	sheet, err = s.store.CreateSheet(ctx, sheetCreate)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to create sheet: %v", err))
	}
	schemaDesign, err = s.convertSheetToSchemaDesign(ctx, sheet, v1pb.SchemaDesignView_SCHEMA_DESIGN_VIEW_FULL)
	if err != nil {
		return nil, err
	}
	return schemaDesign, nil
}

func (s *SchemaDesignService) checkProtectionRules(ctx context.Context, project *store.ProjectMessage, schemaDesign *v1pb.SchemaDesign, currentPrincipalID int) error {
	if project.Setting == nil {
		return nil
	}
	user, err := s.store.GetUserByID(ctx, currentPrincipalID)
	if err != nil {
		return err
	}
	policy, err := s.store.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{ProjectID: &project.ResourceID})
	if err != nil {
		return err
	}
	// Skip protection check for workspace owner and DBA.
	if isOwnerOrDBA(user.Role) {
		return nil
	}

	for _, rule := range project.Setting.ProtectionRules {
		if rule.Target != storepb.ProtectionRule_BRANCH {
			continue
		}
		ok, err := path.Match(rule.NameFilter, schemaDesign.Title)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		pass := false
		for _, binding := range policy.Bindings {
			matchUser := false
			for _, member := range binding.Members {
				if member.Email == user.Email {
					matchUser = true
					break
				}
			}
			if matchUser {
				for _, role := range rule.CreateAllowedRoles {
					// Convert role format.
					if role == convertToProjectRole(binding.Role) {
						pass = true
						break
					}
				}
			}
			if pass {
				break
			}
		}
		if !pass {
			return status.Errorf(codes.InvalidArgument, "not allowed to create branch by project protection rules")
		}
	}
	return nil
}

// UpdateSchemaDesign updates an existing schema design.
func (s *SchemaDesignService) UpdateSchemaDesign(ctx context.Context, request *v1pb.UpdateSchemaDesignRequest) (*v1pb.SchemaDesign, error) {
	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	_, sheetID, err := common.GetProjectResourceIDAndSchemaDesignSheetID(request.SchemaDesign.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid schema design name: %v", err))
	}
	sheetUID, err := strconv.Atoi(sheetID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid sheet id %s, must be positive integer", sheetID))
	}
	if request.UpdateMask == nil || len(request.UpdateMask.Paths) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask is required")
	}

	sheet, err := s.getSheet(ctx, &store.FindSheetMessage{
		UID:      &sheetUID,
		LoadFull: true,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get sheet: %v", err))
	}
	if err := s.checkSchemaDesignPermission(ctx, sheet.ProjectUID); err != nil {
		return nil, err
	}

	sheetUpdate := &store.PatchSheetMessage{
		UID:       sheetUID,
		UpdaterID: principalID,
	}
	schemaDesign := request.SchemaDesign
	_, databaseName, err := common.GetInstanceDatabaseID(schemaDesign.BaselineDatabase)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid baseline database: %v", err))
	}

	if slices.Contains(request.UpdateMask.Paths, "title") {
		sheetUpdate.Title = &schemaDesign.Title
	}
	if slices.Contains(request.UpdateMask.Paths, "schema") {
		sheetUpdate.Statement = &schemaDesign.Schema
	}
	if slices.Contains(request.UpdateMask.Paths, "metadata") {
		sanitizeSchemaDesignSchemaMetadata(schemaDesign)
		schema, err := getDesignSchema(schemaDesign.Engine, schemaDesign.BaselineSchema, schemaDesign.SchemaMetadata)
		if err != nil {
			return nil, err
		}
		sheetUpdate.Statement = &schema
		sheet.Payload.DatabaseConfig = convertV1DatabaseConfig(&v1pb.DatabaseConfig{
			Name:          databaseName,
			SchemaConfigs: schemaDesign.SchemaMetadata.SchemaConfigs,
		})
		sheetUpdate.Payload = sheet.Payload
	}
	// Update baseline schema design id for personal draft schema design.
	if slices.Contains(request.UpdateMask.Paths, "baseline_sheet_name") {
		_, sheetUID, err := common.GetProjectResourceIDSheetUID(schemaDesign.BaselineSheetName)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid baseline sheet name: %v", err))
		}
		sheet.Payload.SchemaDesign.BaselineSheetId = fmt.Sprintf("%d", sheetUID)
		sheetUpdate.Payload = sheet.Payload
	}
	if slices.Contains(request.UpdateMask.Paths, "baseline_database_config") {
		sheet.Payload.BaselineDatabaseConfig = convertV1DatabaseConfig(&v1pb.DatabaseConfig{
			Name:          databaseName,
			SchemaConfigs: schemaDesign.BaselineSchemaMetadata.SchemaConfigs,
		})
		sheetUpdate.Payload = sheet.Payload
	}

	// If the schema is updated, we need to make sure the schema string is valid.
	if sheetUpdate.Statement != nil {
		// Try to transform the schema string to database metadata to make sure it's valid.
		if _, err := transformSchemaStringToDatabaseMetadata(schemaDesign.Engine, *sheetUpdate.Statement); err != nil {
			return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to transform schema string to database metadata: %v", err))
		}
	}

	sheet, err = s.store.PatchSheet(ctx, sheetUpdate)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to update sheet: %v", err))
	}
	schemaDesign, err = s.convertSheetToSchemaDesign(ctx, sheet, v1pb.SchemaDesignView_SCHEMA_DESIGN_VIEW_FULL)
	if err != nil {
		return nil, err
	}
	return schemaDesign, nil
}

// MergeSchemaDesign merges a personal draft schema design to the target schema design.
func (s *SchemaDesignService) MergeSchemaDesign(ctx context.Context, request *v1pb.MergeSchemaDesignRequest) (*v1pb.SchemaDesign, error) {
	_, sheetID, err := common.GetProjectResourceIDAndSchemaDesignSheetID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	sheetUID, err := strconv.Atoi(sheetID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	schemaDesignSheetType := storepb.SheetPayload_SCHEMA_DESIGN.String()
	sheet, err := s.getSheet(ctx, &store.FindSheetMessage{
		UID:         &sheetUID,
		PayloadType: &schemaDesignSheetType,
		LoadFull:    true,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get sheet: %v", err))
	}
	schemaDesign, err := s.convertSheetToSchemaDesign(ctx, sheet, v1pb.SchemaDesignView_SCHEMA_DESIGN_VIEW_FULL)
	if err != nil {
		return nil, err
	}

	_, targetSheetID, err := common.GetProjectResourceIDAndSchemaDesignSheetID(request.TargetName)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	targetSheetUID, err := strconv.Atoi(targetSheetID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid sheet id %s, must be positive integer", targetSheetID))
	}
	targetSheet, err := s.getSheet(ctx, &store.FindSheetMessage{
		UID:         &targetSheetUID,
		PayloadType: &schemaDesignSheetType,
		LoadFull:    true,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get target sheet: %v", err))
	}
	targetSchemaDesign, err := s.convertSheetToSchemaDesign(ctx, targetSheet, v1pb.SchemaDesignView_SCHEMA_DESIGN_VIEW_FULL)
	if err != nil {
		return nil, err
	}

	baselineEtag := generateEtag([]byte(schemaDesign.BaselineSchema))
	// Restrict merging only when the target schema design is not updated.
	// Maybe we can support auto-merging in the future.
	mergedTargetSchema := schemaDesign.Schema
	if baselineEtag != targetSchemaDesign.Etag {
		mergedTarget, err := tryMerge(schemaDesign.BaselineSchemaMetadata, schemaDesign.SchemaMetadata, targetSchemaDesign.SchemaMetadata)
		if err != nil {
			return nil, status.Errorf(codes.FailedPrecondition, fmt.Sprintf("failed to merge schema design: %v", err))
		}
		if mergedTarget == nil {
			return nil, status.Errorf(codes.FailedPrecondition, "failed to merge schema design: no change")
		}
		mergedTargetSchema, err = getDesignSchema(schemaDesign.Engine, targetSchemaDesign.Schema, mergedTarget)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert merged metadata to schema string, %v", err)
		}
	}

	// To merge from one schema design to another, we focus on the three schema string(map to database metadata):
	// Head Schema, Baseline of Head Schema, and Target Schema.
	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	targetSheet.Payload.DatabaseConfig = sheet.Payload.DatabaseConfig
	targetSheet.Payload.BaselineDatabaseConfig = sheet.Payload.BaselineDatabaseConfig
	sheetUpdate := &store.PatchSheetMessage{
		UID:       targetSheetUID,
		UpdaterID: principalID,
		Statement: &mergedTargetSchema,
		Payload:   targetSheet.Payload,
	}
	// Update main branch schema design.
	targetSheet, err = s.store.PatchSheet(ctx, sheetUpdate)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to update main branch schema design: %v", err))
	}
	targetSchemaDesign, err = s.convertSheetToSchemaDesign(ctx, targetSheet, v1pb.SchemaDesignView_SCHEMA_DESIGN_VIEW_FULL)
	if err != nil {
		return nil, err
	}
	return targetSchemaDesign, nil
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
	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	_, sheetID, err := common.GetProjectResourceIDAndSchemaDesignSheetID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	sheetUID, err := strconv.Atoi(sheetID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid sheet id %s, must be positive integer", sheetID))
	}
	sheet, err := s.getSheet(ctx, &store.FindSheetMessage{
		UID:      &sheetUID,
		LoadFull: true,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get sheet: %v", err))
	}
	if err := s.checkSchemaDesignPermission(ctx, sheet.ProjectUID); err != nil {
		return nil, err
	}
	schemaDesign, err := s.convertSheetToSchemaDesign(ctx, sheet, v1pb.SchemaDesignView_SCHEMA_DESIGN_VIEW_FULL)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to convert sheet to schema design: %v", err))
	}
	// Clear the baselineSchemaDesignId field for all schema designs which have this schema design as baseline.
	if schemaDesign.Type == v1pb.SchemaDesign_MAIN_BRANCH {
		filter := fmt.Sprintf("(sheet.payload->>'schemaDesign')::jsonb->>'baselineSchemaDesignId' = '%d'", sheet.UID)
		sheets, err := s.store.ListSheets(ctx, &store.FindSheetMessage{
			SchemaDesignFilter: &filter,
		}, principalID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to list sheets: %v", err))
		}
		for _, sheet := range sheets {
			sheet.Payload.SchemaDesign.BaselineSchemaDesignId = ""
			_, err := s.store.PatchSheet(ctx, &store.PatchSheetMessage{
				UID:       sheet.UID,
				UpdaterID: principalID,
				Payload:   sheet.Payload,
			})
			if err != nil {
				return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to patch sheet: %v", err))
			}
		}
	}
	// Find and delete the baseline sheet if it exists.
	if sheet.Payload.SchemaDesign != nil && sheet.Payload.SchemaDesign.BaselineSheetId != "" {
		baselineSheetUID, err := strconv.Atoi(sheet.Payload.SchemaDesign.BaselineSheetId)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid sheet id %s, must be positive integer", sheetID))
		}
		err = s.store.DeleteSheet(ctx, baselineSheetUID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to delete baseline sheet: %v", err))
		}
	}

	err = s.store.DeleteSheet(ctx, sheetUID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to delete sheet: %v", err))
	}
	return &emptypb.Empty{}, nil
}

func (*SchemaDesignService) DiffMetadata(_ context.Context, request *v1pb.DiffMetadataRequest) (*v1pb.DiffMetadataResponse, error) {
	switch request.Engine {
	case v1pb.Engine_MYSQL, v1pb.Engine_POSTGRES, v1pb.Engine_TIDB:
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unsupported engine: %v", request.Engine)
	}
	if request.SourceMetadata == nil || request.TargetMetadata == nil {
		return nil, status.Errorf(codes.InvalidArgument, "source_metadata and target_metadata are required")
	}

	if err := checkDatabaseMetadata(request.Engine, request.SourceMetadata); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid source metadata: %v", err))
	}
	if err := checkDatabaseMetadata(request.Engine, request.TargetMetadata); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid target metadata: %v", err))
	}

	sanitizeCommentForSchemaMetadata(request.SourceMetadata)
	sanitizeCommentForSchemaMetadata(request.TargetMetadata)

	sourceSchema, err := transformDatabaseMetadataToSchemaString(request.Engine, request.SourceMetadata)
	if err != nil {
		return nil, err
	}
	targetSchema, err := transformDatabaseMetadataToSchemaString(request.Engine, request.TargetMetadata)
	if err != nil {
		return nil, err
	}

	diff, err := base.SchemaDiff(convertEngine(request.Engine), sourceSchema, targetSchema, false /* ignoreCaseSensitive */)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to compute diff between source and target schemas, error: %v", err)
	}

	return &v1pb.DiffMetadataResponse{
		Diff: diff,
	}, nil
}

func (*SchemaDesignService) StringifyMetadata(_ context.Context, request *v1pb.StringifyMetadataRequest) (*v1pb.StringifyMetadataResponse, error) {
	switch request.Engine {
	case v1pb.Engine_MYSQL, v1pb.Engine_POSTGRES, v1pb.Engine_TIDB:
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unsupported engine: %v", request.Engine)
	}

	if request.Metadata == nil {
		return nil, status.Errorf(codes.InvalidArgument, "metadata is required")
	}
	if err := checkDatabaseMetadata(request.Engine, request.Metadata); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid metadata: %v", err))
	}

	sanitizeCommentForSchemaMetadata(request.Metadata)
	schemaString, err := transformDatabaseMetadataToSchemaString(request.Engine, request.Metadata)
	if err != nil {
		return nil, err
	}

	return &v1pb.StringifyMetadataResponse{
		SchemaString: schemaString,
	}, nil
}

func (s *SchemaDesignService) listSheets(ctx context.Context, find *store.FindSheetMessage) ([]*store.SheetMessage, error) {
	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	list, err := s.store.ListSheets(ctx, find, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get sheet: %v", err))
	}
	return list, nil
}

func (s *SchemaDesignService) getSheet(ctx context.Context, find *store.FindSheetMessage) (*store.SheetMessage, error) {
	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	sheet, err := s.store.GetSheet(ctx, find, principalID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get sheet: %v", err))
	}
	if sheet == nil {
		return nil, status.Errorf(codes.NotFound, "cannot find the sheet")
	}
	return sheet, nil
}

func (s *SchemaDesignService) convertSheetToSchemaDesign(ctx context.Context, sheet *store.SheetMessage, view v1pb.SchemaDesignView) (*v1pb.SchemaDesign, error) {
	if sheet.Payload.Type != storepb.SheetPayload_SCHEMA_DESIGN || sheet.Payload.SchemaDesign == nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("unwanted sheet type: %v", sheet.Payload.Type))
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

	database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
		UID:         sheet.DatabaseUID,
		ShowDeleted: true,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get database: %v", err))
	}
	if database == nil {
		return nil, status.Errorf(codes.NotFound, fmt.Sprintf("cannot find the database: %d", *sheet.DatabaseUID))
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

	engine := v1pb.Engine(sheet.Payload.SchemaDesign.Engine)
	schemaDesignType := v1pb.SchemaDesign_Type(sheet.Payload.SchemaDesign.Type)
	// For backward compatibility, we default to MAIN_BRANCH if the type is not specified.
	if schemaDesignType == v1pb.SchemaDesign_TYPE_UNSPECIFIED {
		schemaDesignType = v1pb.SchemaDesign_MAIN_BRANCH
	}
	name := fmt.Sprintf("%s%s/%s%v", common.ProjectNamePrefix, project.ResourceID, common.SchemaDesignPrefix, sheet.UID)

	var baselineSheetName string
	if schemaDesignType == v1pb.SchemaDesign_MAIN_BRANCH {
		if sheet.Payload.SchemaDesign.BaselineSheetId != "" {
			baselineSheetName = fmt.Sprintf("%s%s/%s%v", common.ProjectNamePrefix, project.ResourceID, common.SheetIDPrefix, sheet.Payload.SchemaDesign.BaselineSheetId)
		}
	} else {
		if sheet.Payload.SchemaDesign.BaselineSchemaDesignId != "" {
			baselineSheetName = fmt.Sprintf("%s%s/%s%v", common.ProjectNamePrefix, project.ResourceID, common.SchemaDesignPrefix, sheet.Payload.SchemaDesign.BaselineSchemaDesignId)
		}
	}
	schemaDesign := &v1pb.SchemaDesign{
		Name:                   name,
		Title:                  sheet.Title,
		Etag:                   "",
		Schema:                 "",
		SchemaMetadata:         nil,
		BaselineSchema:         "",
		BaselineSchemaMetadata: nil,
		BaselineSheetName:      baselineSheetName,
		Engine:                 engine,
		BaselineDatabase:       fmt.Sprintf("%s%s/%s%s", common.InstanceNamePrefix, database.InstanceID, common.DatabaseIDPrefix, database.DatabaseName),
		Type:                   schemaDesignType,
		Protection:             convertProtectionFromStore(sheet.Payload.SchemaDesign.Protection),
		Creator:                common.FormatUserEmail(creator.Email),
		Updater:                common.FormatUserEmail(updater.Email),
		CreateTime:             timestamppb.New(sheet.CreatedTime),
		UpdateTime:             timestamppb.New(sheet.UpdatedTime),
	}
	baselineChangeHistoryID := sheet.Payload.SchemaDesign.BaselineChangeHistoryId
	if baselineChangeHistoryID != "" {
		schemaDesign.BaselineChangeHistoryId = &baselineChangeHistoryID
	}

	if view == v1pb.SchemaDesignView_SCHEMA_DESIGN_VIEW_BASIC || view == v1pb.SchemaDesignView_SCHEMA_DESIGN_VIEW_UNSPECIFIED {
		return schemaDesign, nil
	}

	schema := sheet.Statement
	schemaMetadata, err := transformSchemaStringToDatabaseMetadata(engine, schema)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to transform schema string to database metadata: %v", err))
	}
	if config := convertDatabaseConfig(sheet.Payload.DatabaseConfig, nil /* filter */); config != nil {
		schemaMetadata.SchemaConfigs = config.SchemaConfigs
	}
	baselineSchema := ""
	if sheet.Payload.SchemaDesign.BaselineSheetId != "" {
		sheetUID, err := strconv.Atoi(sheet.Payload.SchemaDesign.BaselineSheetId)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("invalid sheet id %s, must be positive integer", sheet.Payload.SchemaDesign.BaselineSheetId))
		}
		baselineSheet, err := s.getSheet(ctx, &store.FindSheetMessage{
			UID:      &sheetUID,
			LoadFull: true,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to get sheet: %v", err))
		}
		baselineSchema = baselineSheet.Statement
	}

	// If the baseline schema is not found or empty, we use the current schema as the baseline schema.
	if baselineSchema == "" {
		baselineSchema = schema
	}
	baselineSchemaMetadata, err := transformSchemaStringToDatabaseMetadata(engine, baselineSchema)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("failed to transform schema string to database metadata: %v", err))
	}
	if config := convertDatabaseConfig(sheet.Payload.BaselineDatabaseConfig, nil /* filter */); config != nil {
		baselineSchemaMetadata.SchemaConfigs = config.SchemaConfigs
	}
	schemaDesign.Etag = generateEtag([]byte(schema))
	schemaDesign.Schema = schema
	schemaDesign.SchemaMetadata = schemaMetadata
	schemaDesign.BaselineSchema = baselineSchema
	schemaDesign.BaselineSchemaMetadata = baselineSchemaMetadata

	return schemaDesign, nil
}

func convertProtectionToStore(protection *v1pb.SchemaDesign_Protection) *storepb.SheetPayload_SchemaDesign_Protection {
	if protection == nil {
		return &storepb.SheetPayload_SchemaDesign_Protection{}
	}

	return &storepb.SheetPayload_SchemaDesign_Protection{
		AllowForcePushes: protection.AllowForcePushes,
	}
}

func convertProtectionFromStore(protection *storepb.SheetPayload_SchemaDesign_Protection) *v1pb.SchemaDesign_Protection {
	if protection == nil {
		return &v1pb.SchemaDesign_Protection{}
	}

	return &v1pb.SchemaDesign_Protection{
		AllowForcePushes: protection.AllowForcePushes,
	}
}

func sanitizeSchemaDesignSchemaMetadata(design *v1pb.SchemaDesign) {
	if dbSchema := design.GetBaselineSchemaMetadata(); dbSchema != nil {
		sanitizeCommentForSchemaMetadata(dbSchema)
	}
	if dbSchema := design.GetSchemaMetadata(); dbSchema != nil {
		sanitizeCommentForSchemaMetadata(dbSchema)
	}
}

func sanitizeCommentForSchemaMetadata(dbSchema *v1pb.DatabaseMetadata) {
	for _, schema := range dbSchema.Schemas {
		for _, table := range schema.Tables {
			table.Comment = common.GetCommentFromClassificationAndUserComment(table.Classification, table.UserComment)
			for _, col := range table.Columns {
				col.Comment = common.GetCommentFromClassificationAndUserComment(col.Classification, col.UserComment)
			}
		}
	}
}

func setClassificationAndUserCommentFromComment(dbSchema *v1pb.DatabaseMetadata) {
	for _, schema := range dbSchema.Schemas {
		for _, table := range schema.Tables {
			table.Classification, table.UserComment = common.GetClassificationAndUserComment(table.Comment)
			for _, col := range table.Columns {
				col.Classification, col.UserComment = common.GetClassificationAndUserComment(col.Comment)
			}
		}
	}
}
