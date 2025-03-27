package v1

import (
	"context"
	"fmt"
	"sort"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/google/cel-go/cel"
	"github.com/pkg/errors"

	celtypes "github.com/google/cel-go/common/types"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/runner/plancheck"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func (s *ReleaseService) CheckRelease(ctx context.Context, request *v1pb.CheckReleaseRequest) (*v1pb.CheckReleaseResponse, error) {
	if len(request.Targets) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "targets cannot be empty")
	}

	var targetDatabases []*store.DatabaseMessage
	for _, target := range request.Targets {
		// Handle database target.
		if instanceID, databaseName, err := common.GetInstanceDatabaseID(target); err == nil {
			instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
				ResourceID: &instanceID,
			})
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to get instance, error: %v", err)
			}
			if instance == nil {
				return nil, status.Errorf(codes.NotFound, "instance %q not found", instanceID)
			}
			database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
				InstanceID:   &instanceID,
				DatabaseName: &databaseName,
			})
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to get database, error: %v", err)
			}
			if database == nil {
				return nil, status.Errorf(codes.NotFound, "database %q not found", target)
			}
			targetDatabases = append(targetDatabases, database)
		}

		// Handle database group target. Extract all matched databases in the database group.
		if projectResourceID, databaseGroupResourceID, err := common.GetProjectIDDatabaseGroupID(target); err == nil {
			project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
				ResourceID: &projectResourceID,
			})
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}
			if project == nil {
				return nil, status.Errorf(codes.NotFound, "project %q not found", projectResourceID)
			}
			if project.Deleted {
				return nil, status.Errorf(codes.NotFound, "project %q has been deleted", projectResourceID)
			}
			existedDatabaseGroup, err := s.store.GetDatabaseGroup(ctx, &store.FindDatabaseGroupMessage{
				ProjectID:  &project.ResourceID,
				ResourceID: &databaseGroupResourceID,
			})
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}
			if existedDatabaseGroup == nil {
				return nil, status.Errorf(codes.NotFound, "database group %q not found", databaseGroupResourceID)
			}
			groupDatabases, err := s.store.ListDatabases(ctx, &store.FindDatabaseMessage{
				ProjectID: &projectResourceID,
			})
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}
			// Filter out databases that are matched with the database group.
			matches, _, err := utils.GetMatchedAndUnmatchedDatabasesInDatabaseGroup(ctx, existedDatabaseGroup, groupDatabases)
			if err != nil {
				return nil, err
			}
			targetDatabases = append(targetDatabases, matches...)
		}
	}

	// Validate and sanitize release files.
	var err error
	request.Release.Files, err = validateAndSanitizeReleaseFiles(request.Release.Files)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid release files, err: %v", err)
	}

	response := &v1pb.CheckReleaseResponse{}
	var errorAdviceCount, warningAdviceCount int
	var stopChecking bool
	var maxRiskLevel int32
	for _, database := range targetDatabases {
		if stopChecking {
			break
		}
		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
			ResourceID: &database.InstanceID,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get instance, error: %v", err)
		}
		if instance == nil {
			return nil, status.Errorf(codes.NotFound, "instance %q not found", database.InstanceID)
		}
		// Continue if the instance is not supported by SQL review.
		if !isSQLReviewSupported(instance.Metadata.GetEngine()) {
			continue
		}

		catalog, err := catalog.NewCatalog(ctx, s.store, database.InstanceID, database.DatabaseName, instance.Metadata.GetEngine(), store.IsObjectCaseSensitive(instance), nil)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create catalog: %v", err)
		}
		for _, file := range request.Release.Files {
			if stopChecking {
				break
			}

			// Check if file has been applied to database.
			revisions, err := s.store.ListRevisions(ctx, &store.FindRevisionMessage{
				InstanceID:   &database.InstanceID,
				DatabaseName: &database.DatabaseName,
				Version:      &file.Version,
				ShowDeleted:  false,
			})
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to list revisions: %v", err)
			}
			if len(revisions) > 0 {
				// Skip the file if it has been applied to the database.
				continue
			}

			checkResult := &v1pb.CheckReleaseResponse_CheckResult{
				File:   file.Path,
				Target: fmt.Sprintf("instances/%s/databases/%s", instance.ResourceID, database.DatabaseName),
			}

			// Check if any syntax error in the statement.
			_, syntaxAdvices := s.sheetManager.GetASTsForChecks(instance.Metadata.GetEngine(), file.Statement)
			if len(syntaxAdvices) > 0 {
				for _, advice := range syntaxAdvices {
					checkResult.Advices = append(checkResult.Advices, convertToV1Advice(advice))
				}
			} else {
				changeType := storepb.PlanCheckRunConfig_DDL
				switch file.ChangeType {
				case v1pb.Release_File_DDL_GHOST:
					changeType = storepb.PlanCheckRunConfig_DDL_GHOST
				case v1pb.Release_File_DML:
					changeType = storepb.PlanCheckRunConfig_DML
				}

				// Get SQL summary report for the statement and target database.
				// Including affected rows.
				summaryReport, err := plancheck.GetSQLSummaryReport(ctx, s.store, s.sheetManager, s.dbFactory, database, file.Statement)
				if err != nil {
					return nil, status.Errorf(codes.Internal, "failed to get SQL summary report, error: %v", err)
				}
				if summaryReport != nil {
					checkResult.AffectedRows = summaryReport.AffectedRows
					response.AffectedRows += summaryReport.AffectedRows

					riskLevel, err := s.calculateRiskLevel(
						ctx,
						instance,
						database,
						changeType,
						summaryReport,
						file.Statement,
					)
					if err != nil {
						return nil, status.Errorf(codes.Internal, "failed to calculate risk level, error: %v", err)
					}
					if riskLevel > maxRiskLevel {
						maxRiskLevel = riskLevel
					}
					riskLevelEnum, err := convertRiskLevel(riskLevel)
					if err != nil {
						return nil, status.Errorf(codes.Internal, "failed to convert risk level, error: %v", err)
					}
					checkResult.RiskLevel = riskLevelEnum
				}
				adviceStatus, sqlReviewAdvices, err := s.runSQLReviewCheckForFile(ctx, catalog, instance, database, changeType, file.Statement)
				if err != nil {
					return nil, status.Errorf(codes.Internal, "failed to check SQL review: %v", err)
				}
				// If the advice status is not SUCCESS, we will add the file and advices to the response.
				if adviceStatus != storepb.Advice_SUCCESS {
					checkResult.Advices = sqlReviewAdvices
				}
			}

			response.Results = append(response.Results, checkResult)
			for _, advice := range checkResult.Advices {
				switch advice.Status {
				case v1pb.Advice_ERROR:
					if errorAdviceCount < common.MaximumAdvicePerStatus {
						errorAdviceCount++
					}
				case v1pb.Advice_WARNING:
					if warningAdviceCount < common.MaximumAdvicePerStatus {
						warningAdviceCount++
					}
				default:
				}
			}
			// Check if we need to stop checking for the rest of files.
			// If we have reached the maximum number of advices for both error and warning, we will stop checking.
			if errorAdviceCount >= common.MaximumAdvicePerStatus && warningAdviceCount >= common.MaximumAdvicePerStatus {
				stopChecking = true
			}
		}
	}

	riskLevelEnum, err := convertRiskLevel(maxRiskLevel)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert risk level, error: %v", err)
	}
	response.RiskLevel = riskLevelEnum
	return response, nil
}

func (s *ReleaseService) runSQLReviewCheckForFile(
	ctx context.Context,
	catalog *catalog.Catalog,
	instance *store.InstanceMessage,
	database *store.DatabaseMessage,
	changeType storepb.PlanCheckRunConfig_ChangeDatabaseType,
	statement string,
) (storepb.Advice_Status, []*v1pb.Advice, error) {
	dbSchema, err := s.store.GetDBSchema(ctx, database.InstanceID, database.DatabaseName)
	if err != nil {
		return storepb.Advice_ERROR, nil, errors.Wrapf(err, "failed to fetch database schema for database %s", database.String())
	}
	if dbSchema == nil {
		if err := s.schemaSyncer.SyncDatabaseSchema(ctx, database); err != nil {
			return storepb.Advice_ERROR, nil, errors.Wrapf(err, "failed to sync database schema for database %s", database.String())
		}
		dbSchema, err = s.store.GetDBSchema(ctx, database.InstanceID, database.DatabaseName)
		if err != nil {
			return storepb.Advice_ERROR, nil, errors.Wrapf(err, "failed to fetch database schema for database %s", database.String())
		}
		if dbSchema == nil {
			return storepb.Advice_ERROR, nil, errors.Wrapf(err, "cannot found schema for database %s", database.String())
		}
	}

	dbMetadata := dbSchema.GetMetadata()
	useDatabaseOwner, err := getUseDatabaseOwner(ctx, s.store, instance, database, changeType)
	if err != nil {
		return storepb.Advice_ERROR, nil, status.Errorf(codes.Internal, "failed to get use database owner: %v", err)
	}
	driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, database, db.ConnectionContext{
		OperationalComponent: "sql-review-check-context",
	})
	if err != nil {
		return storepb.Advice_ERROR, nil, status.Errorf(codes.Internal, "failed to get database driver: %v", err)
	}
	defer driver.Close(ctx)
	connection := driver.GetDB()

	classificationConfig := getClassificationByProject(ctx, s.store, database.ProjectID)
	context := advisor.SQLReviewCheckContext{
		Charset:                  dbMetadata.CharacterSet,
		Collation:                dbMetadata.Collation,
		ChangeType:               changeType,
		DBSchema:                 dbMetadata,
		DBType:                   instance.Metadata.GetEngine(),
		Catalog:                  catalog,
		Driver:                   connection,
		CurrentDatabase:          database.DatabaseName,
		ClassificationConfig:     classificationConfig,
		UsePostgresDatabaseOwner: useDatabaseOwner,
		ListDatabaseNamesFunc:    BuildListDatabaseNamesFunc(s.store),
		InstanceID:               instance.ResourceID,
		IsObjectCaseSensitive:    store.IsObjectCaseSensitive(instance),
	}

	reviewConfig, err := s.store.GetReviewConfigForDatabase(ctx, database)
	if err != nil {
		if e, ok := err.(*common.Error); ok && e.Code == common.NotFound {
			// Continue to check the builtin rules.
			reviewConfig = &storepb.ReviewConfigPayload{}
		} else {
			return storepb.Advice_ERROR, nil, status.Errorf(codes.Internal, "failed to get SQL review policy with error: %v", err)
		}
	}

	res, err := advisor.SQLReviewCheck(ctx, s.sheetManager, statement, reviewConfig.SqlReviewRules, context)
	if err != nil {
		return storepb.Advice_ERROR, nil, status.Errorf(codes.Internal, "failed to exec SQL review with error: %v", err)
	}

	adviceLevel := storepb.Advice_SUCCESS
	var advices []*v1pb.Advice
	for _, advice := range res {
		switch advice.Status {
		case storepb.Advice_WARNING:
			if adviceLevel != storepb.Advice_ERROR {
				adviceLevel = storepb.Advice_WARNING
			}
		case storepb.Advice_ERROR:
			adviceLevel = storepb.Advice_ERROR
		case storepb.Advice_SUCCESS, storepb.Advice_STATUS_UNSPECIFIED:
			continue
		}
		advices = append(advices, convertToV1Advice(advice))
	}
	return adviceLevel, advices, nil
}

func (s *ReleaseService) calculateRiskLevel(
	ctx context.Context,
	instance *store.InstanceMessage,
	database *store.DatabaseMessage,
	changeType storepb.PlanCheckRunConfig_ChangeDatabaseType,
	summaryReport *storepb.PlanCheckRunResult_Result_SqlSummaryReport,
	statement string,
) (int32, error) {
	risks, err := s.store.ListRisks(ctx)
	if err != nil {
		return 0, errors.Wrap(err, "failed to list risks")
	}
	// sort by level DESC, higher risks go first.
	sort.Slice(risks, func(i, j int) bool {
		return risks[i].Level > risks[j].Level
	})

	riskSource := getRiskSourceFromChangeType(changeType)
	if riskSource == store.RiskSourceUnknown {
		return 0, nil
	}

	risk, err := func() (int32, error) {
		for _, risk := range risks {
			if !risk.Active {
				continue
			}
			if risk.Source != riskSource {
				continue
			}
			if risk.Expression == nil || risk.Expression.Expression == "" {
				continue
			}
			e, err := cel.NewEnv(common.RiskFactors...)
			if err != nil {
				return 0, errors.Wrapf(err, "failed to create cel environment")
			}
			ast, issues := e.Parse(risk.Expression.Expression)
			if issues != nil && issues.Err() != nil {
				return 0, errors.Errorf("failed to parse expression: %v", issues.Err())
			}
			prg, err := e.Program(ast, cel.EvalOptions(cel.OptPartialEval))
			if err != nil {
				return 0, err
			}
			args := map[string]any{
				"environment_id": instance.EnvironmentID,
				"project_id":     database.ProjectID,
				"database_name":  database.DatabaseName,
				// convert to string type otherwise cel-go will complain that storepb.Engine is not string type.
				"db_engine":     instance.Metadata.GetEngine().String(),
				"sql_statement": statement,
			}

			vars, err := e.PartialVars(args)
			if err != nil {
				return 0, errors.Wrapf(err, "failed to get vars")
			}
			out, _, err := prg.Eval(vars)
			if err != nil {
				return 0, errors.Wrapf(err, "failed to eval expression")
			}
			if res, ok := out.Equal(celtypes.True).Value().(bool); ok && res {
				return risk.Level, nil
			}

			var tableRows int64
			for _, db := range summaryReport.GetChangedResources().GetDatabases() {
				for _, sc := range db.GetSchemas() {
					for _, tb := range sc.GetTables() {
						tableRows += tb.GetTableRows()
					}
				}
			}
			args["affected_rows"] = summaryReport.AffectedRows
			args["table_rows"] = tableRows

			var tableNames []string
			for _, db := range summaryReport.GetChangedResources().GetDatabases() {
				for _, schema := range db.GetSchemas() {
					for _, table := range schema.GetTables() {
						tableNames = append(tableNames, table.Name)
					}
				}
			}
			for _, statementType := range summaryReport.StatementTypes {
				args["sql_type"] = statementType
				for _, tableName := range tableNames {
					args["table_name"] = tableName
					out, _, err := prg.Eval(args)
					if err != nil {
						return 0, err
					}
					if res, ok := out.Equal(celtypes.True).Value().(bool); ok && res {
						return risk.Level, nil
					}
				}
			}
		}
		return 0, nil
	}()
	if err != nil {
		return 0, errors.Wrap(err, "failed to calculate risk level")
	}
	return risk, nil
}

func getRiskSourceFromChangeType(changeType storepb.PlanCheckRunConfig_ChangeDatabaseType) store.RiskSource {
	switch changeType {
	case storepb.PlanCheckRunConfig_DDL, storepb.PlanCheckRunConfig_DDL_GHOST:
		return store.RiskSourceDatabaseSchemaUpdate
	case storepb.PlanCheckRunConfig_DML:
		return store.RiskSourceDatabaseDataUpdate
	default:
		return store.RiskSourceUnknown
	}
}

func convertRiskLevel(riskLevel int32) (v1pb.CheckReleaseResponse_RiskLevel, error) {
	switch riskLevel {
	case 0:
		return v1pb.CheckReleaseResponse_RISK_LEVEL_UNSPECIFIED, nil
	case 100:
		return v1pb.CheckReleaseResponse_LOW, nil
	case 200:
		return v1pb.CheckReleaseResponse_MODERATE, nil
	case 300:
		return v1pb.CheckReleaseResponse_HIGH, nil
	}
	return v1pb.CheckReleaseResponse_RISK_LEVEL_UNSPECIFIED, errors.Errorf("unexpected risk level %d", riskLevel)
}
