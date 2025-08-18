package v1

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/runner/plancheck"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/store/model"
	"github.com/bytebase/bytebase/backend/utils"
)

func (s *ReleaseService) CheckRelease(ctx context.Context, req *connect.Request[v1pb.CheckReleaseRequest]) (*connect.Response[v1pb.CheckReleaseResponse], error) {
	request := req.Msg
	if len(request.Targets) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("targets cannot be empty"))
	}

	projectID, err := common.GetProjectID(request.GetParent())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID:  &projectID,
		ShowDeleted: true,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %q not found", projectID))
	}

	var targetDatabases []*store.DatabaseMessage
	for _, target := range request.Targets {
		// Handle database target.
		if _, _, err := common.GetInstanceDatabaseID(target); err == nil {
			database, err := getDatabaseMessage(ctx, s.store, target)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to found database %v", target))
			}
			if database == nil || database.Deleted {
				return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database %v not found", target))
			}
			targetDatabases = append(targetDatabases, database)
		}

		// Handle database group target. Extract all matched databases in the database group.
		if projectResourceID, databaseGroupResourceID, err := common.GetProjectIDDatabaseGroupID(target); err == nil {
			project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
				ResourceID: &projectResourceID,
			})
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
			if project == nil {
				return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %q not found", projectResourceID))
			}
			if project.Deleted {
				return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %q has been deleted", projectResourceID))
			}
			existedDatabaseGroup, err := s.store.GetDatabaseGroup(ctx, &store.FindDatabaseGroupMessage{
				ProjectID:  &project.ResourceID,
				ResourceID: &databaseGroupResourceID,
			})
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
			if existedDatabaseGroup == nil {
				return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("database group %q not found", databaseGroupResourceID))
			}
			groupDatabases, err := s.store.ListDatabases(ctx, &store.FindDatabaseMessage{
				ProjectID: &projectResourceID,
			})
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
			// Filter out databases that are matched with the database group.
			matches, _, err := utils.GetMatchedAndUnmatchedDatabasesInDatabaseGroup(ctx, existedDatabaseGroup, groupDatabases)
			if err != nil {
				return nil, err
			}
			targetDatabases = append(targetDatabases, matches...)
		}
	}

	if project.Setting.GetCiSamplingSize() > 0 && len(targetDatabases) > int(project.Setting.GetCiSamplingSize()) {
		targetDatabases = targetDatabases[:project.Setting.GetCiSamplingSize()]
	}

	// Validate and sanitize release files.
	sanitizedFiles, err := validateAndSanitizeReleaseFiles(ctx, s.store, request.Release.Files)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "invalid release files"))
	}
	if len(sanitizedFiles) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("release files cannot be empty"))
	}

	releaseFileType := sanitizedFiles[0].Type

	var response *v1pb.CheckReleaseResponse
	switch releaseFileType {
	case v1pb.Release_File_DECLARATIVE:
		// To check declarative files, we use the original files.
		// Because the sanitized files are merged into one file for the declarative case and
		// we lose file path information.
		resp, err := s.checkReleaseDeclarative(ctx, request.Release.Files, targetDatabases)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to check release declarative"))
		}
		response = resp
	case v1pb.Release_File_VERSIONED:
		resp, err := s.checkReleaseVersioned(ctx, sanitizedFiles, targetDatabases)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to check release versioned"))
		}
		response = resp
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unexpected release file type %q", releaseFileType.String()))
	}

	return connect.NewResponse(response), nil
}

func (s *ReleaseService) checkReleaseVersioned(ctx context.Context, files []*v1pb.Release_File, databases []*store.DatabaseMessage) (*v1pb.CheckReleaseResponse, error) {
	resp := &v1pb.CheckReleaseResponse{}
	var errorAdviceCount, warningAdviceCount int
	var maxRiskLevel int32

	risks, err := s.store.ListRisks(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to list risks"))
	}

	releaseFileVersions := make([]string, 0, len(files))
	for _, file := range files {
		releaseFileVersions = append(releaseFileVersions, file.Version)
	}

loop:
	for _, database := range databases {
		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
			ResourceID: &database.InstanceID,
		})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get instance"))
		}
		if instance == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("instance %q not found", database.InstanceID))
		}

		engine := instance.Metadata.GetEngine()
		catalog, err := catalog.NewCatalog(ctx, s.store, database.InstanceID, database.DatabaseName, engine, store.IsObjectCaseSensitive(instance), nil)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create catalog"))
		}
		// Batch fetch all revisions for this database
		revisions, err := s.store.ListRevisions(ctx, &store.FindRevisionMessage{
			InstanceID:   &database.InstanceID,
			DatabaseName: &database.DatabaseName,
			Type:         common.NewP(storepb.RevisionPayload_VERSIONED),
			Versions:     &releaseFileVersions,
			ShowDeleted:  false,
		})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to list revisions"))
		}

		// Create a map for quick lookup
		revisionMap := make(map[string]*store.RevisionMessage)
		for _, revision := range revisions {
			revisionMap[revision.Version] = revision
		}

		for _, file := range files {
			// Check if file has been applied to database.
			if appliedRevision, ok := revisionMap[file.Version]; ok {
				// Check if the SHA256 matches
				if appliedRevision.Payload.SheetSha256 != file.SheetSha256 {
					// Add a warning advice if SHA256 mismatch
					checkResult := &v1pb.CheckReleaseResponse_CheckResult{
						File:   file.Path,
						Target: fmt.Sprintf("instances/%s/databases/%s", instance.ResourceID, database.DatabaseName),
						Advices: []*v1pb.Advice{
							{
								Status:  v1pb.Advice_WARNING,
								Code:    advisor.Internal.Int32(),
								Title:   "Applied file has been modified",
								Content: fmt.Sprintf("The file %q with version %q has already been applied to the database, but its content has been modified. Applied SHA256: %s, Release SHA256: %s", file.Path, file.Version, appliedRevision.Payload.SheetSha256, file.SheetSha256),
							},
						},
					}
					resp.Results = append(resp.Results, checkResult)
					warningAdviceCount++
				}
				// Skip the file since it has been applied to the database.
				continue
			}

			checkResult, err := func() (*v1pb.CheckReleaseResponse_CheckResult, error) {
				checkResult := &v1pb.CheckReleaseResponse_CheckResult{
					File:   file.Path,
					Target: fmt.Sprintf("instances/%s/databases/%s", instance.ResourceID, database.DatabaseName),
				}
				// statement is guaranteed to be populated by validateAndSanitizeReleaseFiles
				statement := string(file.Statement)
				// Check if any syntax error in the statement.
				if common.EngineSupportSyntaxCheck(engine) {
					_, syntaxAdvices := s.sheetManager.GetASTsForChecks(engine, statement)
					if len(syntaxAdvices) > 0 {
						for _, advice := range syntaxAdvices {
							checkResult.Advices = append(checkResult.Advices, convertToV1Advice(advice))
						}
						return checkResult, nil
					}
				}

				changeType := storepb.PlanCheckRunConfig_DDL
				switch file.ChangeType {
				case v1pb.Release_File_DDL_GHOST:
					changeType = storepb.PlanCheckRunConfig_DDL_GHOST
				case v1pb.Release_File_DML:
					changeType = storepb.PlanCheckRunConfig_DML
				default:
					// Keep DDL as default change type
				}
				// Get SQL summary report for the statement and target database.
				// Including affected rows.
				summaryReport, err := plancheck.GetSQLSummaryReport(ctx, s.store, s.sheetManager, s.dbFactory, database, statement)
				if err != nil {
					return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get SQL summary report"))
				}
				if summaryReport != nil {
					checkResult.AffectedRows = summaryReport.AffectedRows
					resp.AffectedRows += summaryReport.AffectedRows

					commonArgs := map[string]any{
						"environment_id": "",
						"project_id":     database.ProjectID,
						"database_name":  database.DatabaseName,
						// convert to string type otherwise cel-go will complain that storepb.Engine is not string type.
						"db_engine":     engine.String(),
						"sql_statement": statement,
					}
					if database.EffectiveEnvironmentID != nil {
						commonArgs["environment_id"] = *database.EffectiveEnvironmentID
					}
					riskLevel, err := CalculateRiskLevelWithOptionalSummaryReport(ctx, risks, commonArgs, getRiskSourceFromChangeType(changeType), summaryReport)
					if err != nil {
						return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to calculate risk level"))
					}
					if riskLevel > maxRiskLevel {
						maxRiskLevel = riskLevel
					}
					riskLevelEnum, err := convertRiskLevel(riskLevel)
					if err != nil {
						return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert risk level"))
					}
					checkResult.RiskLevel = riskLevelEnum
				}
				if common.EngineSupportSQLReview(engine) {
					adviceStatus, sqlReviewAdvices, err := s.runSQLReviewCheckForFile(ctx, catalog, instance, database, changeType, statement)
					if err != nil {
						return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to check SQL review"))
					}
					// If the advice status is not SUCCESS, we will add the file and advices to the response.
					if adviceStatus != storepb.Advice_SUCCESS {
						checkResult.Advices = sqlReviewAdvices
					}
				}
				return checkResult, nil
			}()

			if err != nil {
				checkResult = &v1pb.CheckReleaseResponse_CheckResult{
					File:   file.Path,
					Target: common.FormatDatabase(instance.ResourceID, database.DatabaseName),
					Advices: []*v1pb.Advice{
						{
							Status:  v1pb.Advice_ERROR,
							Code:    advisor.Internal.Int32(),
							Title:   "Failed to check",
							Content: err.Error(),
						},
					},
				}
			}

			resp.Results = append(resp.Results, checkResult)
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
				break loop
			}
		}
	}

	riskLevelEnum, err := convertRiskLevel(maxRiskLevel)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert risk level"))
	}
	resp.RiskLevel = riskLevelEnum

	return resp, nil
}

func (s *ReleaseService) checkReleaseDeclarative(ctx context.Context, files []*v1pb.Release_File, databases []*store.DatabaseMessage) (*v1pb.CheckReleaseResponse, error) {
	var results []*v1pb.CheckReleaseResponse_CheckResult
	var errorAdviceCount, warningAdviceCount int
	for _, database := range databases {
		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{
			ResourceID: &database.InstanceID,
		})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get instance"))
		}
		if instance == nil {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("instance %q not found", database.InstanceID))
		}

		engine := instance.Metadata.GetEngine()
		revisions, err := s.store.ListRevisions(ctx, &store.FindRevisionMessage{
			InstanceID:   &database.InstanceID,
			DatabaseName: &database.DatabaseName,
			Type:         common.NewP(storepb.RevisionPayload_DECLARATIVE),
			Limit:        common.NewP(1),
			ShowDeleted:  false,
		})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to list revisions"))
		}
		if len(revisions) > 0 {
			rv, err := model.NewVersion(revisions[0].Version)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to parse version %q", revisions[0].Version))
			}
			for _, file := range files {
				fv, err := model.NewVersion(file.Version)
				if err != nil {
					return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to parse version %q", file.Version))
				}
				if fv.LessThanOrEqual(rv) {
					checkResult := &v1pb.CheckReleaseResponse_CheckResult{
						File:   file.Path,
						Target: common.FormatDatabase(instance.ResourceID, database.DatabaseName),
						Advices: []*v1pb.Advice{
							{
								Status:  v1pb.Advice_WARNING,
								Code:    advisor.Internal.Int32(),
								Title:   "Applied file has been modified",
								Content: fmt.Sprintf("The file %q has version %q, but there is an equal or higher version %q applied", file.Path, file.Version, revisions[0].Version),
							},
						},
					}
					results = append(results, checkResult)
					warningAdviceCount++
				}
			}
		}

		// Perform syntax check for declarative files
		for _, file := range files {
			checkResult := &v1pb.CheckReleaseResponse_CheckResult{
				File:   file.Path,
				Target: common.FormatDatabase(instance.ResourceID, database.DatabaseName),
			}

			// statement is guaranteed to be populated by validateAndSanitizeReleaseFiles
			statement := string(file.Statement)

			// Check if any syntax error in the statement.
			if common.EngineSupportSyntaxCheck(engine) {
				_, syntaxAdvices := s.sheetManager.GetASTsForChecks(engine, statement)
				if len(syntaxAdvices) > 0 {
					for _, advice := range syntaxAdvices {
						checkResult.Advices = append(checkResult.Advices, convertToV1Advice(advice))
					}
				}
			}

			// Only add to results if there are syntax errors
			if len(checkResult.Advices) > 0 {
				results = append(results, checkResult)
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
					break
				}
			}
		}
	}
	return &v1pb.CheckReleaseResponse{
		Results: results,
	}, nil
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
		return storepb.Advice_ERROR, nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get use database owner"))
	}
	driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, database, db.ConnectionContext{})
	if err != nil {
		return storepb.Advice_ERROR, nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get database driver"))
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
			return storepb.Advice_ERROR, nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get SQL review policy"))
		}
	}

	res, err := advisor.SQLReviewCheck(ctx, s.sheetManager, statement, reviewConfig.SqlReviewRules, context)
	if err != nil {
		return storepb.Advice_ERROR, nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to exec SQL review"))
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
		default:
			// Other advice statuses
		}
		advices = append(advices, convertToV1Advice(advice))
	}
	return adviceLevel, advices, nil
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
	default:
		return v1pb.CheckReleaseResponse_RISK_LEVEL_UNSPECIFIED, errors.Errorf("unexpected risk level %d", riskLevel)
	}
}
