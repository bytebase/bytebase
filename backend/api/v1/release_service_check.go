package v1

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	advisorpg "github.com/bytebase/bytebase/backend/plugin/advisor/pg"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg"
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
			matches, err := utils.GetMatchedDatabasesInDatabaseGroup(ctx, existedDatabaseGroup, groupDatabases)
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
	sanitizedFiles, err := validateAndSanitizeReleaseFiles(ctx, s.store, request.Release.Files, false)
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
		resp, err := s.checkReleaseDeclarative(ctx, sanitizedFiles, targetDatabases, request.CustomRules)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to check release declarative"))
		}
		response = resp
	case v1pb.Release_File_VERSIONED:
		resp, err := s.checkReleaseVersioned(ctx, sanitizedFiles, targetDatabases, request.CustomRules)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to check release versioned"))
		}
		response = resp
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unexpected release file type %q", releaseFileType.String()))
	}

	return connect.NewResponse(response), nil
}

func (s *ReleaseService) checkReleaseVersioned(ctx context.Context, files []*v1pb.Release_File, databases []*store.DatabaseMessage, customRules string) (*v1pb.CheckReleaseResponse, error) {
	resp := &v1pb.CheckReleaseResponse{}
	var errorAdviceCount, warningAdviceCount int
	var maxRiskLevel storepb.RiskLevel

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
		originCatalog, finalCatalog, err := catalog.NewCatalog(ctx, s.store, database.InstanceID, database.DatabaseName, engine, store.IsObjectCaseSensitive(instance), nil)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create catalog"))
		}
		// Batch fetch all revisions for this database
		revisions, err := s.store.ListRevisions(ctx, &store.FindRevisionMessage{
			InstanceID:   &database.InstanceID,
			DatabaseName: &database.DatabaseName,
			Type:         common.NewP(storepb.SchemaChangeType_VERSIONED),
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

		// Batch AI linting for all files in this database (if custom rules provided)
		var aiAdvicesMap map[string][]*v1pb.Advice
		if customRules != "" {
			var filesToLint []fileSchema
			for _, file := range files {
				// Skip files that have already been applied
				if _, ok := revisionMap[file.Version]; !ok {
					filesToLint = append(filesToLint, fileSchema{
						Path:    file.Path,
						Content: string(file.Statement),
					})
				}
			}
			if len(filesToLint) > 0 {
				slog.Info("Running batch AI-powered linting for versioned files", "database", database.DatabaseName, "filesCount", len(filesToLint))
				var err error
				aiAdvicesMap, err = s.runAIPoweredLintBatch(ctx, filesToLint, customRules)
				if err != nil {
					slog.Error("Batch AI linting failed for versioned files", "database", database.DatabaseName, "error", err)
					// Continue processing even if AI linting fails
				} else {
					slog.Info("Batch AI linting completed for versioned files", "database", database.DatabaseName, "filesWithAdvices", len(aiAdvicesMap))
				}
			}
		}

		for _, file := range files {
			// Check if file has been applied to database.
			if appliedRevision, ok := revisionMap[file.Version]; ok {
				// Check if the SHA256 matches
				if appliedRevision.Payload.SheetSha256 != file.SheetSha256 {
					// Add a warning advice if SHA256 mismatch
					checkResult := &v1pb.CheckReleaseResponse_CheckResult{
						File:   file.Path,
						Target: common.FormatDatabase(instance.ResourceID, database.DatabaseName),
						Advices: []*v1pb.Advice{
							{
								Status:  v1pb.Advice_WARNING,
								Code:    code.Internal.Int32(),
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
						return checkResult, nil
					}
				}

				changeType := storepb.PlanCheckRunConfig_DDL
				switch file.MigrationType {
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

					environmentID := ""
					if database.EffectiveEnvironmentID != nil {
						environmentID = *database.EffectiveEnvironmentID
					}
					commonArgs := map[string]any{
						common.CELAttributeResourceEnvironmentID: environmentID,
						common.CELAttributeResourceProjectID:     database.ProjectID,
						common.CELAttributeResourceInstanceID:    instance.ResourceID,
						common.CELAttributeResourceDatabaseName:  database.DatabaseName,
						common.CELAttributeResourceDBEngine:      engine.String(),
						common.CELAttributeStatementText:         statement,
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
					adviceStatus, sqlReviewAdvices, err := s.runSQLReviewCheckForFile(ctx, originCatalog, finalCatalog, instance, database, changeType, statement)
					if err != nil {
						return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to check SQL review"))
					}
					// If the advice status is not SUCCESS, we will add the file and advices to the response.
					if adviceStatus != storepb.Advice_SUCCESS {
						// Mark parser-based advices
						for _, advice := range sqlReviewAdvices {
							advice.RuleType = v1pb.Advice_PARSER_BASED
						}
						checkResult.Advices = append(checkResult.Advices, sqlReviewAdvices...)
					}
				}

				// Add AI-powered linting results from batch processing (if available)
				if aiAdvicesMap != nil {
					if aiAdvices, ok := aiAdvicesMap[file.Path]; ok && len(aiAdvices) > 0 {
						slog.Info("Adding AI linting results for versioned file", "file", file.Path, "advicesCount", len(aiAdvices))
						checkResult.Advices = append(checkResult.Advices, aiAdvices...)
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
							Code:    code.Internal.Int32(),
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

func (s *ReleaseService) checkReleaseDeclarative(ctx context.Context, files []*v1pb.Release_File, databases []*store.DatabaseMessage, customRules string) (*v1pb.CheckReleaseResponse, error) {
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
			Type:         common.NewP(storepb.SchemaChangeType_DECLARATIVE),
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
								Code:    code.Internal.Int32(),
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

		// Perform SDL style and integrity checks for PostgreSQL
		var sdlStyleAdvices map[string][]*storepb.Advice
		var sdlIntegrityAdvices map[string][]*storepb.Advice
		if engine == storepb.Engine_POSTGRES {
			fileContents := make(map[string]string)
			for _, file := range files {
				fileContents[file.Path] = string(file.Statement)
			}

			// Run SDL style checks (schema name requirements, index naming, etc.)
			sdlStyleAdvices = make(map[string][]*storepb.Advice)
			for filePath, content := range fileContents {
				advices, err := advisorpg.CheckSDLStyle(content)
				if err != nil {
					// Continue with other checks even if style check fails
					sdlStyleAdvices[filePath] = []*storepb.Advice{{
						Status:  storepb.Advice_ERROR,
						Code:    code.Internal.Int32(),
						Title:   "Failed to check SDL style",
						Content: err.Error(),
					}}
				} else {
					sdlStyleAdvices[filePath] = advices
				}
			}

			// Run SDL integrity checks (handles cross-file validation)
			var err error
			sdlIntegrityAdvices, err = advisorpg.CheckSDLIntegrity(fileContents)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to check SDL integrity"))
			}
		}

		// Batch AI linting for all declarative files (if custom rules provided)
		var aiAdvicesMap map[string][]*v1pb.Advice
		if customRules != "" {
			var filesToLint []fileSchema
			for _, file := range files {
				filesToLint = append(filesToLint, fileSchema{
					Path:    file.Path,
					Content: string(file.Statement),
				})
			}
			if len(filesToLint) > 0 {
				slog.Info("Running batch AI-powered linting for declarative files", "database", database.DatabaseName, "filesCount", len(filesToLint))
				var err error
				aiAdvicesMap, err = s.runAIPoweredLintBatch(ctx, filesToLint, customRules)
				if err != nil {
					slog.Error("Batch AI linting failed for declarative files", "database", database.DatabaseName, "error", err)
					// Continue processing even if AI linting fails
				} else {
					slog.Info("Batch AI linting completed for declarative files", "database", database.DatabaseName, "filesWithAdvices", len(aiAdvicesMap))
				}
			}
		}

		// Perform syntax check and statement type check for declarative files
		for _, file := range files {
			checkResult := &v1pb.CheckReleaseResponse_CheckResult{
				File:   file.Path,
				Target: common.FormatDatabase(instance.ResourceID, database.DatabaseName),
			}

			// statement is guaranteed to be populated by validateAndSanitizeReleaseFiles
			statement := string(file.Statement)

			// Check if any syntax error in the statement.
			if common.EngineSupportSyntaxCheck(engine) {
				asts, syntaxAdvices := s.sheetManager.GetASTsForChecks(engine, statement)
				if len(syntaxAdvices) > 0 {
					for _, advice := range syntaxAdvices {
						checkResult.Advices = append(checkResult.Advices, convertToV1Advice(advice))
					}
				} else {
					// Only check statement types if there are no syntax errors
					statementsWithPos, err := getStatementTypesWithPositionsForEngine(engine, asts)
					if err != nil {
						checkResult.Advices = append(checkResult.Advices, &v1pb.Advice{
							Status:  v1pb.Advice_ERROR,
							Code:    code.Internal.Int32(),
							Title:   "Failed to parse statement types",
							Content: err.Error(),
						})
					} else {
						// Check all statement types against whitelist and collect disallowed ones with positions
						for _, stmt := range statementsWithPos {
							if !isAllowedInSDL(stmt.Type) {
								// Create a separate advice for each disallowed statement with position
								advice := &v1pb.Advice{
									Status: v1pb.Advice_ERROR,
									Code:   code.StatementDisallowedInSDL.Int32(),
									Title:  "Disallowed statement in SDL file",
									Content: fmt.Sprintf(
										"Statement type '%s' is not allowed in SDL files.\n\n"+
											"SDL files should only contain CREATE and COMMENT statements to declare the desired schema.\n\n"+
											"Common fixes:\n"+
											"- To modify an object: Update its CREATE statement in the SDL file\n"+
											"- To rename an object: Change the name in its CREATE statement\n"+
											"- To remove an object: Delete its CREATE statement from the SDL file\n"+
											"- Do not use ALTER, RENAME, DROP, or DML statements (INSERT/UPDATE/DELETE)\n\n"+
											"Statement text:\n%s",
										stmt.Type,
										stmt.Text,
									),
								}
								// Set position information if available
								if stmt.Line > 0 {
									advice.StartPosition = &v1pb.Position{
										Line: int32(stmt.Line),
									}
								}
								checkResult.Advices = append(checkResult.Advices, advice)
							}
						}
					}

					// Add SDL style and integrity check results for this file (PostgreSQL only)
					if engine == storepb.Engine_POSTGRES && len(checkResult.Advices) == 0 {
						// Add SDL style check results
						if advices, exists := sdlStyleAdvices[file.Path]; exists {
							for _, advice := range advices {
								checkResult.Advices = append(checkResult.Advices, convertToV1Advice(advice))
							}
						}
						// Add SDL integrity check results
						if advices, exists := sdlIntegrityAdvices[file.Path]; exists {
							for _, advice := range advices {
								checkResult.Advices = append(checkResult.Advices, convertToV1Advice(advice))
							}
						}
					}
				}
			}

			// Mark parser-based advices
			for _, advice := range checkResult.Advices {
				if advice.RuleType == v1pb.Advice_RULE_TYPE_UNSPECIFIED {
					advice.RuleType = v1pb.Advice_PARSER_BASED
				}
			}

			// Add AI-powered linting results from batch processing (if available)
			if aiAdvicesMap != nil {
				if aiAdvices, ok := aiAdvicesMap[file.Path]; ok && len(aiAdvices) > 0 {
					slog.Info("Adding AI linting results for declarative file", "file", file.Path, "advicesCount", len(aiAdvices))
					checkResult.Advices = append(checkResult.Advices, aiAdvices...)
				}
			}

			// Only add to results if there are advices
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
	originCatalog *catalog.DatabaseState,
	finalCatalog *catalog.DatabaseState,
	instance *store.InstanceMessage,
	database *store.DatabaseMessage,
	changeType storepb.PlanCheckRunConfig_ChangeDatabaseType,
	statement string,
) (storepb.Advice_Status, []*v1pb.Advice, error) {
	dbSchema, err := s.store.GetDBSchema(ctx, &store.FindDBSchemaMessage{
		InstanceID:   database.InstanceID,
		DatabaseName: database.DatabaseName,
	})
	if err != nil {
		return storepb.Advice_ERROR, nil, errors.Wrapf(err, "failed to fetch database schema for database %s", database.String())
	}
	if dbSchema == nil {
		if err := s.schemaSyncer.SyncDatabaseSchema(ctx, database); err != nil {
			return storepb.Advice_ERROR, nil, errors.Wrapf(err, "failed to sync database schema for database %s", database.String())
		}
		dbSchema, err = s.store.GetDBSchema(ctx, &store.FindDBSchemaMessage{
			InstanceID:   database.InstanceID,
			DatabaseName: database.DatabaseName,
		})
		if err != nil {
			return storepb.Advice_ERROR, nil, errors.Wrapf(err, "failed to fetch database schema for database %s", database.String())
		}
		if dbSchema == nil {
			return storepb.Advice_ERROR, nil, errors.Wrapf(err, "cannot found schema for database %s", database.String())
		}
	}

	dbMetadata := dbSchema.GetMetadata()
	useDatabaseOwner, err := getUseDatabaseOwner(ctx, s.store, instance, database)
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
		OriginCatalog:            originCatalog,
		FinalCatalog:             finalCatalog,
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

func convertRiskLevel(riskLevel storepb.RiskLevel) (v1pb.RiskLevel, error) {
	switch riskLevel {
	case storepb.RiskLevel_RISK_LEVEL_UNSPECIFIED:
		return v1pb.RiskLevel_RISK_LEVEL_UNSPECIFIED, nil
	case storepb.RiskLevel_LOW:
		return v1pb.RiskLevel_LOW, nil
	case storepb.RiskLevel_MODERATE:
		return v1pb.RiskLevel_MODERATE, nil
	case storepb.RiskLevel_HIGH:
		return v1pb.RiskLevel_HIGH, nil
	default:
		return v1pb.RiskLevel_RISK_LEVEL_UNSPECIFIED, errors.Errorf("unexpected risk level %v", riskLevel)
	}
}

// allowedSDLStatementTypes defines the whitelist of statement types allowed in SDL files.
// SDL files should only contain CREATE and COMMENT statements to declare the desired schema.
// ALTER SEQUENCE is allowed for setting ownership (OWNED BY).
var allowedSDLStatementTypes = map[string]bool{
	// CREATE statements - declare new objects
	"CREATE_TABLE":     true,
	"CREATE_INDEX":     true,
	"CREATE_VIEW":      true,
	"CREATE_SEQUENCE":  true,
	"CREATE_FUNCTION":  true,
	"CREATE_PROCEDURE": true,
	"CREATE_SCHEMA":    true,

	// ALTER statements - limited to specific cases
	"ALTER_SEQUENCE": true, // Allowed for OWNED BY and sequence options

	// COMMENT - metadata annotations
	"COMMENT": true,
}

// isAllowedInSDL checks if a statement type is allowed in SDL files.
func isAllowedInSDL(stmtType string) bool {
	return allowedSDLStatementTypes[stmtType]
}

// statementTypeWithPosition contains statement type and its position information.
type statementTypeWithPosition struct {
	Type string
	// Line is the one-based line number where the statement ends.
	Line int
	Text string
}

// getStatementTypesWithPositionsForEngine returns statement types with position info for the given engine and ASTs.
// The line numbers are one-based.
// Currently only PostgreSQL is supported.
func getStatementTypesWithPositionsForEngine(engine storepb.Engine, asts any) ([]statementTypeWithPosition, error) {
	switch engine {
	case storepb.Engine_POSTGRES, storepb.Engine_COCKROACHDB, storepb.Engine_REDSHIFT:
		pgStmts, err := pg.GetStatementTypes(asts)
		if err != nil {
			return nil, err
		}
		// Convert pg.StatementTypeWithPosition to local statementTypeWithPosition
		result := make([]statementTypeWithPosition, len(pgStmts))
		for i, stmt := range pgStmts {
			result[i] = statementTypeWithPosition{
				Type: stmt.Type,
				Line: stmt.Line,
				Text: stmt.Text,
			}
		}
		return result, nil
	default:
		// For unsupported engines, return empty list (skip check)
		return []statementTypeWithPosition{}, nil
	}
}
